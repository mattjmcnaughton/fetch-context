//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helloDest is the derived destination for the hello fixture repo in w.
func helloDest(w *workspace) string {
	return w.target("repos", fixture.Host(), "fixture", "hello")
}

func TestAC_REPO_01_SinglePublicRepoLayout(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := helloDest(w)
	if !isGit(dest) {
		t.Errorf("%s is not a git repo", dest)
	}
	if !exists(filepath.Join(dest, "MARKER")) {
		t.Errorf("MARKER missing in clone")
	}
}

func TestAC_REPO_02_CloneIsShallowOnDefaultBranch(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := helloDest(w)
	if !isShallow(t, dest) {
		t.Error("clone is not shallow")
	}
	branch := strings.TrimSpace(mustGit(t, dest, "symbolic-ref", "--short", "HEAD"))
	if branch != "main" {
		t.Errorf("checked-out branch = %q, want the remote default (main)", branch)
	}
}

func TestAC_REPO_03_AutoGitignoreWritten(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	b, err := os.ReadFile(w.target(".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "*" {
		t.Errorf(".gitignore content = %q, want exactly %q", b, "*")
	}
}

func TestAC_REPO_04_TreeIsActuallyIgnored(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !treeClean(t, w.dir) {
		t.Errorf("git status in workspace is not empty:\n%s", mustGit(t, w.dir, "status", "--porcelain"))
	}
}

func TestAC_REPO_05_MultipleReposOneInvocation(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"), fixture.CloneURL("fixture/other"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) {
		t.Error("hello clone missing")
	}
	if !isGit(w.target("repos", fixture.Host(), "fixture", "other")) {
		t.Error("other clone missing")
	}
}

func TestAC_REPO_06_FullCloneURLNormalized(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello")) // explicit .git suffix
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) {
		t.Error("clone did not land at the normalized host/owner/repo path")
	}
	if exists(helloDest(w) + ".git") {
		t.Error("a destination with a .git suffix was created; suffix should be normalized away")
	}
}

func TestAC_REPO_07_RerunFetchesAndHardResets(t *testing.T) {
	w := newWorkspace(t)
	url := fixture.CloneURL("fixture/refresh")
	if res := w.run("repo", url); res.code != 0 {
		t.Fatalf("first run: exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := w.target("repos", fixture.Host(), "fixture", "refresh")

	// Inject local state and advance the remote.
	writeFile(t, filepath.Join(dest, "MARKER"), "local edit")
	writeFile(t, filepath.Join(dest, "UNTRACKED"), "x")
	if err := fixture.Commit("fixture/refresh", map[string]string{"ADVANCED": "v2\n"}); err != nil {
		t.Fatal(err)
	}

	res := w.run("repo", url)
	if res.code != 0 {
		t.Fatalf("re-run: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !treeClean(t, dest) {
		t.Error("clone tree not clean after re-run")
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "MARKER")); strings.Contains(string(b), "local edit") {
		t.Error("local edit survived the hard reset")
	}
	if exists(filepath.Join(dest, "UNTRACKED")) {
		t.Error("untracked file survived the re-run")
	}
	if !exists(filepath.Join(dest, "ADVANCED")) {
		t.Error("working tree does not match remote HEAD (new commit missing)")
	}
}

func TestAC_REPO_08_NonGitDestinationRefused(t *testing.T) {
	w := newWorkspace(t)
	dest := helloDest(w)
	sentinel := filepath.Join(dest, "SENTINEL")
	writeFile(t, sentinel, "keep")

	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "not a managed clone") {
		t.Errorf("stderr does not explain the refusal:\n%s", res.stderr)
	}
	if b, err := os.ReadFile(sentinel); err != nil || string(b) != "keep" {
		t.Errorf("sentinel was touched: %q, %v", b, err)
	}
	// isGit would find the workspace's .git above dest; check for clone
	// artifacts directly.
	if exists(filepath.Join(dest, ".git")) || exists(filepath.Join(dest, "MARKER")) {
		t.Error("a clone was performed over the unmanaged destination")
	}
}

func TestAC_REPO_09_BadRepoReference(t *testing.T) {
	w := newWorkspace(t)
	url := fixture.CloneURL("fixture/does-not-exist-xyz")
	res := w.run("repo", url)
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "does-not-exist-xyz") {
		t.Errorf("stderr does not report the clone failure:\n%s", res.stderr)
	}
	if exists(w.target("repos", fixture.Host(), "fixture", "does-not-exist-xyz")) {
		t.Error("partial directory left at the destination")
	}
}

func TestAC_REPO_10_MixedBatchContinuesOnError(t *testing.T) {
	w := newWorkspace(t)
	bad := fixture.CloneURL("fixture/does-not-exist-xyz")
	res := w.run("repo", fixture.CloneURL("fixture/hello"), bad)
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) || !exists(filepath.Join(helloDest(w), "MARKER")) {
		t.Error("the good clone is not fully present")
	}
	if !strings.Contains(res.stderr, "does-not-exist-xyz") {
		t.Errorf("stderr does not list the failed item:\n%s", res.stderr)
	}
	if exists(w.target("repos", fixture.Host(), "fixture", "does-not-exist-xyz")) {
		t.Error("failed entry left a partial directory")
	}
}

func TestAC_REPO_11_EquivalentFormsCollapse(t *testing.T) {
	w := newWorkspace(t)
	base := fixture.URL() + "/fixture/hello"
	res := w.run("repo", base+".git", base, base+"/")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	ownerDir := w.target("repos", fixture.Host(), "fixture")
	entries, err := os.ReadDir(ownerDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "hello" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("owner dir entries = %v, want exactly [hello]", names)
	}
}

// deepDest is the derived destination for the three-commit deep fixture.
func deepDest(w *workspace) string {
	return w.target("repos", fixture.Host(), "fixture", "deep")
}

func TestAC_REPO_12_DepthZeroClonesFullHistory(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", "--depth", "0", fixture.CloneURL("fixture/deep"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := deepDest(w)
	if isShallow(t, dest) {
		t.Error("clone is shallow, want full history")
	}
	if got := strings.TrimSpace(mustGit(t, dest, "rev-list", "--count", "HEAD")); got != "3" {
		t.Errorf("commit count = %s, want 3 (the remote's full history)", got)
	}
}

func TestAC_REPO_13_BranchClonesNamedBranch(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", "--branch", "develop", fixture.CloneURL("fixture/branchy"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := w.target("repos", fixture.Host(), "fixture", "branchy")
	if branch := strings.TrimSpace(mustGit(t, dest, "symbolic-ref", "--short", "HEAD")); branch != "develop" {
		t.Errorf("checked-out branch = %q, want develop", branch)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "MARKER")); string(b) != "develop\n" {
		t.Errorf("MARKER = %q, want the branch's content", b)
	}
}

func TestAC_REPO_14_RerunWithDepthZeroKeepsFullHistory(t *testing.T) {
	w := newWorkspace(t)
	url := fixture.CloneURL("fixture/deep-rerun")
	if err := fixture.Seed("fixture/deep-rerun", map[string]string{"MARKER": "v1\n"}); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Commit("fixture/deep-rerun", map[string]string{"MARKER": "v2\n"}); err != nil {
		t.Fatal(err)
	}
	if res := w.run("repo", "--depth", "0", url); res.code != 0 {
		t.Fatalf("first run: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if err := fixture.Commit("fixture/deep-rerun", map[string]string{"MARKER": "v3\n"}); err != nil {
		t.Fatal(err)
	}

	res := w.run("repo", "--depth", "0", url)
	if res.code != 0 {
		t.Fatalf("re-run: exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := w.target("repos", fixture.Host(), "fixture", "deep-rerun")
	if isShallow(t, dest) {
		t.Error("re-run re-shallowed the full-history clone")
	}
	if got := strings.TrimSpace(mustGit(t, dest, "rev-list", "--count", "HEAD")); got != "3" {
		t.Errorf("commit count = %s, want 3 (remote latest with full history)", got)
	}
}
