//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"
)

func TestAC_CONFIG_01_PathHonorsFetchContextHome(t *testing.T) {
	w := newWorkspace(t)
	script := editorScript(t, `printf '# touched-by-test\n' > "$1"`)
	w.setEnv("EDITOR=" + script)

	res := w.run("edit")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	b, err := os.ReadFile(w.configPath())
	if err != nil {
		t.Fatalf("config not at $FETCH_CONTEXT_HOME/.config/fetch-context/config.yaml: %v", err)
	}
	if !strings.Contains(string(b), "touched-by-test") {
		t.Errorf("file at the sandboxed path was not the one edited: %q", b)
	}
}

func TestAC_CONFIG_02_GlobalTargetOverrideHonored(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, "target: .agentic/ctx\n")
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(w.path(".agentic", "ctx", "repos", fixture.Host(), "fixture", "hello")) {
		t.Error("clone not under the overridden global target")
	}
	if exists(w.target()) {
		t.Error("default target was used despite the override")
	}
}

func TestAC_CONFIG_03_MalformedConfigErrorsClearly(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, "profiles:\n  backend: [unclosed\n")
	res := w.run("list")
	if res.code == 0 {
		t.Fatal("exit = 0, want non-zero for malformed config")
	}
	if !strings.Contains(res.stderr, "yaml") && !strings.Contains(res.stderr, "pars") {
		t.Errorf("stderr does not identify the parse error:\n%s", res.stderr)
	}
}

func TestAC_CONFIG_04_MissingConfigNotFatalForOneOffs(t *testing.T) {
	w := newWorkspace(t) // fresh FETCH_CONTEXT_HOME: no config file exists
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, want 0 with no config; stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) {
		t.Error("default target was not used")
	}
}

func TestAC_CONFIG_05_GlobalCloneDepthHonored(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, "clone:\n  depth: 0\n")
	res := w.run("repo", fixture.CloneURL("fixture/deep"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	dest := w.target("repos", fixture.Host(), "fixture", "deep")
	if isShallow(t, dest) {
		t.Error("clone is shallow despite clone.depth: 0 in config")
	}
}

func TestAC_CONFIG_06_UnknownRepoEntryFieldErrors(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, "profiles:\n  p:\n    repos:\n      - ref: a/b\n        brnch: oops\n")
	res := w.run("list")
	if res.code == 0 {
		t.Fatal("unknown repo-entry field must fail loudly")
	}
	if !strings.Contains(res.stderr, "brnch") {
		t.Errorf("stderr does not name the unknown field:\n%s", res.stderr)
	}
}

func TestAC_CONFIG_07_CloneParallelValidatedAndEffective(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, "clone:\n  parallel: 2\n")
	res := w.run("group", "github.com/fixture-org")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if !isGit(w.target("repos", "github.com", "fixture-org", name)) {
			t.Errorf("%s missing with clone.parallel: 2", name)
		}
	}

	w2 := newWorkspace(t)
	w2.writeConfig(t, "clone:\n  parallel: 0\n")
	res = w2.run("list")
	if res.code == 0 {
		t.Fatal("clone.parallel: 0 must fail loudly")
	}
	if !strings.Contains(res.stderr, "parallel") {
		t.Errorf("stderr does not name the invalid setting:\n%s", res.stderr)
	}
}
