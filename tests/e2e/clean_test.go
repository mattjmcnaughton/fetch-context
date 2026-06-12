//go:build e2e

package e2e

import (
	"testing"
)

// materializeBoth puts a repo clone and a url file under the default target.
func materializeBoth(t *testing.T, w *workspace) {
	t.Helper()
	if res := w.run("repo", fixture.CloneURL("fixture/hello")); res.code != 0 {
		t.Fatalf("repo: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if res := w.run("url", "http://example.test/blog/post"); res.code != 0 {
		t.Fatalf("url: exit = %d, stderr: %s", res.code, res.stderr)
	}
}

func TestAC_CLEAN_01_RemovesWholeTarget(t *testing.T) {
	w := newWorkspace(t)
	materializeBoth(t, w)
	res := w.run("clean")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if exists(w.target("repos")) || exists(w.target("urls")) {
		t.Error("repos/ or urls/ still present after clean")
	}
}

func TestAC_CLEAN_02_ScopedReposOnly(t *testing.T) {
	w := newWorkspace(t)
	materializeBoth(t, w)
	res := w.run("clean", "repos")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if exists(w.target("repos")) {
		t.Error("repos/ still present")
	}
	if !exists(w.target("urls", "example.test", "blog", "post.md")) {
		t.Error("urls/ was touched by clean repos")
	}
}

func TestAC_CLEAN_03_ScopedURLsOnly(t *testing.T) {
	w := newWorkspace(t)
	materializeBoth(t, w)
	res := w.run("clean", "urls")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if exists(w.target("urls")) {
		t.Error("urls/ still present")
	}
	if !isGit(helloDest(w)) {
		t.Error("repos/ was touched by clean urls")
	}
}

func TestAC_CLEAN_04_NeverDeletesOutsideTarget(t *testing.T) {
	w := newWorkspace(t)
	writeFile(t, w.path("keep.txt"), "precious")
	materializeBoth(t, w)

	res := w.run("clean")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !exists(w.path("keep.txt")) {
		t.Error("clean deleted a file outside the resolved target")
	}
}

func TestAC_CLEAN_05_ProfileCleanClearsOnlyItsTarget(t *testing.T) {
	w := newWorkspace(t)
	hello := fixture.CloneURL("fixture/hello")
	other := fixture.CloneURL("fixture/other")
	w.writeConfig(t, `
profiles:
  backend:
    target: .agentic/backend
    repos:
      - "`+hello+`"
  web:
    target: .agentic/web
    repos:
      - "`+other+`"
`)
	for _, name := range []string{"backend", "web"} {
		if res := w.run("load", name); res.code != 0 {
			t.Fatalf("load %s: exit = %d, stderr: %s", name, res.code, res.stderr)
		}
	}
	// Ad-hoc content under the global target too.
	if res := w.run("repo", hello); res.code != 0 {
		t.Fatalf("repo: exit = %d, stderr: %s", res.code, res.stderr)
	}

	res := w.run("clean", "backend")
	if res.code != 0 {
		t.Fatalf("clean backend: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if exists(w.path(".agentic", "backend")) {
		t.Error(".agentic/backend still present")
	}
	if !exists(w.path(".agentic", "web")) {
		t.Error(".agentic/web was removed — clean must never auto-discover other profiles' targets")
	}
	if !exists(w.target()) {
		t.Error(".agentic/sources was removed — global target is out of scope for a profile clean")
	}
}
