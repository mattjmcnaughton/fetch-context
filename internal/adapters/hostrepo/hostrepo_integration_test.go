//go:build integration

package hostrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoRootFromNestedCWD(t *testing.T) {
	ws := t.TempDir()
	mustRun(t, ws, "git", "init", "-q")
	nested := filepath.Join(ws, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	got, err := New().RepoRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(t, got, ws) {
		t.Errorf("RepoRoot = %q, want %q", got, ws)
	}
}

func TestRepoRootOutsideGitRepoErrors(t *testing.T) {
	plain := t.TempDir()
	t.Chdir(plain)

	_, err := New().RepoRoot(context.Background())
	if err == nil {
		t.Fatal("want error outside a git repo")
	}
	if !strings.Contains(err.Error(), "repo root") {
		t.Errorf("error %q should explain that a repo root could not be resolved", err)
	}
}

func samePath(t *testing.T, a, b string) bool {
	t.Helper()
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		t.Fatal(err)
	}
	return ra == rb
}

func mustRun(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
