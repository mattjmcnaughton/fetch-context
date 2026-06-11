//go:build e2e

package e2e

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestAC_SCOPE_01_NoLockfileEverWritten(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"), fixture.CloneURL("fixture/other"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	for _, root := range []string{w.dir, w.home} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".lock") {
				t.Errorf("lockfile found: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAC_SCOPE_02_NoCommitPinning(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := helloDest(w)
	if !isShallow(t, dest) {
		t.Error("clone not shallow")
	}
	// Tracking a branch, not a detached-HEAD pin.
	if _, err := gitOut(dest, "symbolic-ref", "HEAD"); err != nil {
		t.Error("HEAD is detached; clones must track a branch")
	}
}

func TestAC_SCOPE_03_NothingCommittedToHostRepo(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !treeClean(t, w.dir) {
		t.Error("workspace tree not clean")
	}
	out, _ := gitOut(w.dir, "rev-list", "--all", "--count")
	if strings.TrimSpace(out) != "0" {
		t.Errorf("commits were created in the host repo: rev-list count = %s", out)
	}
}
