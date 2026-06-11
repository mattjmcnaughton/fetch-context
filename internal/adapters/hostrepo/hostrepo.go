// Package hostrepo locates the root of the repo the user is working in via
// `git rev-parse --show-toplevel`. It implements ports.HostRepoLocator.
package hostrepo

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

type Locator struct{}

func New() *Locator { return &Locator{} }

// RepoRoot resolves the toplevel of the git repo containing the CWD. Outside
// any repo it errors (R4): the repo-local target model is part of the
// contract, so there is no CWD fallback.
func (l *Locator) RepoRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("could not resolve a repo root: the current directory is not inside a git repository (%s)",
			strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

var _ ports.HostRepoLocator = (*Locator)(nil)
