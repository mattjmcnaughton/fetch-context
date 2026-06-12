//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC-SAFE-01 is the conflict-behavior view of AC-REPO-07: a destination that
// is a managed clone gets refreshed, never refused.
func TestAC_SAFE_01_ManagedCloneIsRefreshed(t *testing.T) {
	w := newWorkspace(t)
	url := fixture.CloneURL("fixture/hello")
	if res := w.run("repo", url); res.code != 0 {
		t.Fatalf("first run: exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := helloDest(w)
	writeFile(t, filepath.Join(dest, "MARKER"), "dirty")

	res := w.run("repo", url)
	if res.code != 0 {
		t.Fatalf("re-run on managed clone refused: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !treeClean(t, dest) {
		t.Error("managed clone not refreshed to a clean tree")
	}
}

// AC-SAFE-03 is the conflict-behavior view of AC-URL-03: url markdown is
// overwritten on re-fetch, never refused.
func TestAC_SAFE_03_URLMarkdownOverwrittenOnRefetch(t *testing.T) {
	w := newWorkspace(t)
	if res := w.run("url", "http://example.test/page"); res.code != 0 {
		t.Fatalf("first fetch: exit = %d, stderr: %s", res.code, res.stderr)
	}
	page := w.target("urls", "example.test", "page.md")
	writeFile(t, page, "STALE")

	res := w.run("url", "http://example.test/page")
	if res.code != 0 {
		t.Fatalf("re-fetch refused: exit = %d, stderr: %s", res.code, res.stderr)
	}
	b, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "STALE") {
		t.Error("stale content not overwritten")
	}
}

// AC-SAFE-04 is the conflict-behavior view of AC-CLEAN-04: clean is
// target-scoped and never deletes outside the resolved target.
func TestAC_SAFE_04_CleanIsTargetScoped(t *testing.T) {
	w := newWorkspace(t)
	writeFile(t, w.path("keep.txt"), "precious")
	if res := w.run("repo", fixture.CloneURL("fixture/hello")); res.code != 0 {
		t.Fatalf("repo: exit = %d, stderr: %s", res.code, res.stderr)
	}

	res := w.run("clean")
	if res.code != 0 {
		t.Fatalf("clean: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !exists(w.path("keep.txt")) {
		t.Error("clean removed a path outside its target tree")
	}
	if exists(w.target()) {
		t.Error("target tree not removed")
	}
}

// AC-SAFE-02 is the conflict-behavior view of AC-REPO-08: an unmanaged
// directory is never clobbered.
func TestAC_SAFE_02_UnmanagedDirectoryNeverClobbered(t *testing.T) {
	w := newWorkspace(t)
	dest := helloDest(w)
	sentinel := filepath.Join(dest, "SENTINEL")
	writeFile(t, sentinel, "keep")

	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	if !exists(sentinel) {
		t.Error("sentinel destroyed")
	}
	if !strings.Contains(res.stderr, "refusing") && !strings.Contains(res.stderr, "not a managed clone") {
		t.Errorf("stderr does not state the refusal:\n%s", res.stderr)
	}
}
