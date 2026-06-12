//go:build e2e

package e2e

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// workspace is the per-scenario clean slate from acceptance.md §1.4: a fresh
// git-initialized directory acting as "the current repo", plus an isolated
// FETCH_CONTEXT_HOME.
type workspace struct {
	t *testing.T
	// dir is the workspace root (the "current repo").
	dir string
	// home is the isolated FETCH_CONTEXT_HOME.
	home string
	// extraEnv appends to the environment of every run (KEY=VALUE).
	extraEnv []string
}

func newWorkspace(t *testing.T) *workspace {
	t.Helper()
	w := &workspace{
		t:    t,
		dir:  t.TempDir(),
		home: t.TempDir(),
	}
	mustGit(t, w.dir, "init", "-q")
	// Contract seams (acceptance.md §1.2) point at the loopback fixtures.
	w.setEnv(
		"JINA_BASE_URL="+reader.URL(),
		"GITHUB_API_URL="+forge.URL(),
		"GITLAB_API_URL="+forge.URL(),
	)
	return w
}

// result captures one binary invocation.
type result struct {
	stdout string
	stderr string
	code   int
}

// run invokes $FCBIN with args, CWD = workspace root.
func (w *workspace) run(args ...string) result {
	return w.runIn(w.dir, args...)
}

// runIn invokes $FCBIN with args from an arbitrary CWD.
func (w *workspace) runIn(cwd string, args ...string) result {
	w.t.Helper()
	cmd := exec.Command(fcbin, args...)
	cmd.Dir = cwd
	cmd.Env = append(baseEnv(), append([]string{"FETCH_CONTEXT_HOME=" + w.home}, w.extraEnv...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			w.t.Fatalf("running %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// baseEnv is a minimal environment: enough for git/go to function, no
// inherited tokens or seams that could leak host state into a scenario.
func baseEnv() []string {
	keep := []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOPATH"}
	env := make([]string, 0, len(keep))
	for _, k := range keep {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

// setEnv adds KEY=VALUE pairs to every subsequent run.
func (w *workspace) setEnv(kv ...string) {
	w.extraEnv = append(w.extraEnv, kv...)
}

// target returns a path beneath the default resolved target.
func (w *workspace) target(parts ...string) string {
	return filepath.Join(append([]string{w.dir, ".agentic", "sources"}, parts...)...)
}

// mustGit runs git in dir and fails the test on error.
func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitOut(dir, args...)
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return out
}

// gitOut runs git in dir, returning combined output.
func gitOut(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// writeFile writes content to a path, creating parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// exists reports whether a path exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
