// Package gitrepo manages upstream clones by shelling out to the git
// binary. It implements ports.GitRepo.
//
// Credentials (ADR-0001 decisions 3/4): clones are attempted
// unauthenticated first; on an auth-shaped failure each configured
// credential is retried once via an http.extraHeader Basic credential.
// Nothing is written to disk and tokens never enter the saved remote URL.
package gitrepo

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// Credential kinds select the Basic-auth username convention of the forge.
const (
	KindGitHub = "github"
	KindGitLab = "gitlab"
)

type Credential struct {
	Kind  string
	Token string
}

// basicHeader renders the credential as an http.extraHeader value.
func (c Credential) basicHeader() string {
	user := "token"
	switch c.Kind {
	case KindGitHub:
		user = "x-access-token"
	case KindGitLab:
		user = "oauth2"
	}
	cred := base64.StdEncoding.EncodeToString([]byte(user + ":" + c.Token))
	return "Authorization: Basic " + cred
}

type Adapter struct {
	log   *slog.Logger
	creds []Credential
}

func New(log *slog.Logger, creds ...Credential) *Adapter {
	return &Adapter{log: log, creds: creds}
}

// Clone clones cloneURL into dest honoring opts (depth, branch). A failed
// clone leaves no partial directory (AC-REPO-09); auth failures are retried
// with each configured credential and surfaced explicitly when they stick
// (AC-AUTH-03).
func (a *Adapter) Clone(ctx context.Context, cloneURL, dest string, opts ports.CloneOptions) error {
	preExisted := pathExists(dest)
	cleanup := func() {
		if !preExisted {
			os.RemoveAll(dest)
		}
	}

	args := []string{"clone", "-q"}
	if opts.Depth > 0 {
		args = append(args, fmt.Sprintf("--depth=%d", opts.Depth))
	}
	if opts.Branch != "" {
		args = append(args, "--branch="+opts.Branch)
	}
	args = append(args, cloneURL, dest)

	stderr, err := a.git(ctx, nil, args...)
	if err == nil {
		return nil
	}
	if isAuthFailure(stderr) {
		for _, cred := range a.creds {
			cleanup()
			a.log.Debug("retrying clone with credential", "url", cloneURL, "kind", cred.Kind)
			retryStderr, retryErr := a.git(ctx, &cred, args...)
			if retryErr == nil {
				return nil
			}
			stderr = retryStderr
		}
	}
	cleanup()
	if isAuthFailure(stderr) {
		return fmt.Errorf("authentication failed cloning %s (set GITHUB_TOKEN/GITLAB_TOKEN with access to the repo): %s",
			cloneURL, summarize(stderr))
	}
	return fmt.Errorf("cloning %s: %s", cloneURL, summarize(stderr))
}

// Refresh discards local state in a managed clone and resets it to the
// remote's latest (AC-REPO-07), converging the clone toward opts: depth N
// keeps (or re-trims) the history at N, depth 0 unshallows, and a pinned
// branch different from the checked-out one is switched to.
func (a *Adapter) Refresh(ctx context.Context, dest string, opts ports.CloneOptions) error {
	branchOut, err := a.gitIn(ctx, dest, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("resolving branch of %s: %s", dest, summarize(branchOut))
	}
	current := strings.TrimSpace(branchOut)
	branch := opts.Branch
	if branch == "" {
		branch = current
	}

	fetch := []string{"fetch", "-q"}
	if opts.Depth > 0 {
		fetch = append(fetch, fmt.Sprintf("--depth=%d", opts.Depth))
	} else if a.isShallow(ctx, dest) {
		// `--unshallow` errors on an already-full clone, so it is applied
		// only when converging a shallow clone to full history.
		fetch = append(fetch, "--unshallow")
	}
	fetch = append(fetch, "origin", branch)
	if stderr, err := a.gitAuthRetry(ctx, dest, fetch...); err != nil {
		return fmt.Errorf("fetching %s: %s", dest, summarize(stderr))
	}
	if out, err := a.gitIn(ctx, dest, "reset", "-q", "--hard", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("resetting %s: %s", dest, summarize(out))
	}
	if branch != current {
		if out, err := a.gitIn(ctx, dest, "checkout", "-q", "-B", branch); err != nil {
			return fmt.Errorf("switching %s to branch %s: %s", dest, branch, summarize(out))
		}
	}
	if out, err := a.gitIn(ctx, dest, "clean", "-q", "-fdx"); err != nil {
		return fmt.Errorf("cleaning %s: %s", dest, summarize(out))
	}
	return nil
}

// isShallow reports whether the clone at dest has truncated history.
func (a *Adapter) isShallow(ctx context.Context, dest string) bool {
	out, err := a.gitIn(ctx, dest, "rev-parse", "--is-shallow-repository")
	return err == nil && strings.TrimSpace(out) == "true"
}

// IsManagedClone reports whether dest is itself a working-tree root
// (ADR-0001 decision 5). A plain directory inside the host repo resolves to
// the host's .git and is therefore not managed.
func (a *Adapter) IsManagedClone(ctx context.Context, dest string) (bool, error) {
	out, err := a.gitIn(ctx, dest, "rev-parse", "--absolute-git-dir")
	if err != nil {
		// Not a git repo at all (or unreadable): not managed.
		return false, nil
	}
	gitDir, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	if err != nil {
		return false, nil
	}
	want, err := filepath.EvalSymlinks(filepath.Join(dest, ".git"))
	if err != nil {
		return false, nil
	}
	return gitDir == want, nil
}

// git runs a git command (optionally with a credential header), returning
// stderr output for diagnosis.
func (a *Adapter) git(ctx context.Context, cred *Credential, args ...string) (string, error) {
	full := args
	if cred != nil {
		full = append([]string{"-c", "http.extraHeader=" + cred.basicHeader()}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.String(), err
}

// gitIn runs a git command inside dir, returning combined output.
func (a *Adapter) gitIn(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// gitAuthRetry runs a git command inside dir, retrying once per credential
// on auth-shaped failures (private-repo refresh needs auth too).
func (a *Adapter) gitAuthRetry(ctx context.Context, dir string, args ...string) (string, error) {
	withDir := append([]string{"-C", dir}, args...)
	stderr, err := a.git(ctx, nil, withDir...)
	if err == nil || !isAuthFailure(stderr) {
		return stderr, err
	}
	for _, cred := range a.creds {
		retryStderr, retryErr := a.git(ctx, &cred, withDir...)
		if retryErr == nil {
			return retryStderr, nil
		}
		stderr = retryStderr
	}
	return stderr, err
}

// isAuthFailure recognizes git's auth-shaped failure messages.
func isAuthFailure(stderr string) bool {
	lower := strings.ToLower(stderr)
	for _, marker := range []string{
		"authentication failed",
		"could not read username",
		"could not read password",
		"401",
		"403",
		"unauthorized",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// summarize trims git stderr to its informative tail.
func summarize(out string) string {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return "git failed with no output"
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}
	return strings.Join(lines, "; ")
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

var _ ports.GitRepo = (*Adapter)(nil)
