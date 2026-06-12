//go:build e2e

package e2e

import (
	"os"
	"testing"
)

func TestAC_LAYOUT_01_ReposAndURLsAreSiblings(t *testing.T) {
	w := newWorkspace(t)
	if res := w.run("repo", fixture.CloneURL("fixture/hello")); res.code != 0 {
		t.Fatalf("repo: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if res := w.run("url", "http://example.test/page"); res.code != 0 {
		t.Fatalf("url: exit = %d, stderr: %s", res.code, res.stderr)
	}
	repos, err := os.Stat(w.target("repos"))
	if err != nil || !repos.IsDir() {
		t.Errorf("repos/ not directly under the target: %v", err)
	}
	urls, err := os.Stat(w.target("urls"))
	if err != nil || !urls.IsDir() {
		t.Errorf("urls/ not directly under the target: %v", err)
	}
}

func TestAC_LAYOUT_02_HostOwnerRepoNesting(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	want := w.target("repos", fixture.Host(), "fixture", "hello")
	if !isGit(want) {
		t.Errorf("clone not at repos/<host>/<owner>/<repo>: %s missing", want)
	}
}

func TestAC_LAYOUT_03_GitignoreIdempotent(t *testing.T) {
	w := newWorkspace(t)
	// Pre-create the gitignore with the expected content.
	writeFile(t, w.target(".gitignore"), "*")

	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	b, err := os.ReadFile(w.target(".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "*" {
		t.Errorf(".gitignore = %q after re-materialization, want exactly %q (not duplicated/appended)", b, "*")
	}
}
