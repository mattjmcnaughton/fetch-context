//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

// Helper assertions mirroring docs/acceptance.md §1.6.

// isGit reports whether dir is a git repository.
func isGit(dir string) bool {
	_, err := gitOut(dir, "rev-parse", "--git-dir")
	return err == nil
}

// isShallow reports whether dir is a shallow git repository.
func isShallow(t *testing.T, dir string) bool {
	t.Helper()
	out := mustGit(t, dir, "rev-parse", "--is-shallow-repository")
	return strings.TrimSpace(out) == "true"
}

// treeClean reports whether `git status --porcelain` is empty in dir.
func treeClean(t *testing.T, dir string) bool {
	t.Helper()
	out := mustGit(t, dir, "status", "--porcelain")
	return strings.TrimSpace(out) == ""
}
