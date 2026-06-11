//go:build e2e

package e2e

import (
	"os"
	"testing"
)

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
