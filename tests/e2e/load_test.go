//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

func TestAC_LOAD_01_MaterializesAllProfileKeys(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, `
profiles:
  backend:
    repos:
      - "`+fixture.CloneURL("fixture/hello")+`"
    groups:
      - github.com/fixture-org
    urls:
      - "http://example.test/blog/post"
`)
	res := w.run("load", "backend")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) {
		t.Error("repos entry not cloned")
	}
	if !isGit(w.target("repos", "github.com", "fixture-org", "alpha")) {
		t.Error("groups entry not cloned")
	}
	if !exists(w.target("urls", "example.test", "blog", "post.md")) {
		t.Error("urls entry not written")
	}
}

func TestAC_LOAD_02_PerProfileTargetOverride(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, `
profiles:
  backend:
    target: .agentic/backend
    repos:
      - "`+fixture.CloneURL("fixture/hello")+`"
    urls:
      - "http://example.test/blog/post"
`)
	res := w.run("load", "backend")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(w.path(".agentic", "backend", "repos", fixture.Host(), "fixture", "hello")) {
		t.Error("repo not under the overridden target")
	}
	if !exists(w.path(".agentic", "backend", "urls", "example.test", "blog", "post.md")) {
		t.Error("url not under the overridden target")
	}
	if exists(w.target()) {
		t.Error("content leaked into the default .agentic/sources target")
	}
}

func TestAC_LOAD_03_MissingProfileName(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("load")
	if res.code != 2 {
		t.Fatalf("exit = %d, want 2; stderr: %s", res.code, res.stderr)
	}
	if strings.TrimSpace(res.stderr) == "" {
		t.Error("stderr should carry a usage/error message")
	}
}

func TestAC_LOAD_04_UnknownProfile(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, "profiles:\n  backend:\n    repos: [\"a/b\"]\n")
	res := w.run("load", "no-such-profile")
	if res.code != 2 {
		t.Fatalf("exit = %d, want 2; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "no-such-profile") {
		t.Errorf("stderr does not name the unknown profile:\n%s", res.stderr)
	}
	if exists(w.path(".agentic")) {
		t.Error("something was materialized for an unknown profile")
	}
}

func TestAC_LOAD_05_NoImplicitProfile(t *testing.T) {
	w := newWorkspace(t)
	// Even with exactly one profile defined, bare `load` never guesses.
	w.writeConfig(t, `
profiles:
  only-one:
    repos:
      - "`+fixture.CloneURL("fixture/hello")+`"
`)
	res := w.run("load")
	if res.code != 2 {
		t.Fatalf("exit = %d, want 2 (no auto-selection); stderr: %s", res.code, res.stderr)
	}
	if exists(w.path(".agentic")) {
		t.Error("a profile was implicitly loaded")
	}
}

func TestAC_LOAD_06_PartialFailureReportedNotFatalOnFirst(t *testing.T) {
	w := newWorkspace(t)
	bad := fixture.CloneURL("fixture/does-not-exist-xyz")
	w.writeConfig(t, `
profiles:
  backend:
    repos:
      - "`+fixture.CloneURL("fixture/hello")+`"
      - "`+bad+`"
    urls:
      - "http://example.test/blog/post"
`)
	res := w.run("load", "backend")
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) {
		t.Error("good repo not cloned")
	}
	if !exists(w.target("urls", "example.test", "blog", "post.md")) {
		t.Error("good url not written")
	}
	if !strings.Contains(res.stderr, "does-not-exist-xyz") {
		t.Errorf("stderr does not name the bad repo:\n%s", res.stderr)
	}
	if exists(w.target("repos", fixture.Host(), "fixture", "does-not-exist-xyz")) {
		t.Error("partial directory left for the bad repo")
	}
}

// path joins parts beneath the workspace root.
func (w *workspace) path(parts ...string) string {
	return joinPath(w.dir, parts...)
}

func TestAC_LOAD_07_RepoEntryMappingFormHonored(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, `
profiles:
  mixed:
    repos:
      - `+fixture.CloneURL("fixture/hello")+`
      - ref: `+fixture.CloneURL("fixture/branchy")+`
        depth: 0
        branch: develop
`)
	res := w.run("load", "mixed")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}

	hello := w.target("repos", fixture.Host(), "fixture", "hello")
	if !isShallow(t, hello) {
		t.Error("plain entry should stay shallow (global default)")
	}
	if branch := strings.TrimSpace(mustGit(t, hello, "symbolic-ref", "--short", "HEAD")); branch != "main" {
		t.Errorf("plain entry branch = %q, want main", branch)
	}

	branchy := w.target("repos", fixture.Host(), "fixture", "branchy")
	if isShallow(t, branchy) {
		t.Error("mapping entry with depth 0 should have full history")
	}
	if branch := strings.TrimSpace(mustGit(t, branchy, "symbolic-ref", "--short", "HEAD")); branch != "develop" {
		t.Errorf("mapping entry branch = %q, want develop", branch)
	}
}
