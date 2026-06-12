//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"
)

// AC-SANDBOX-01 is the sandboxing view of AC-CONFIG-01: the config the
// binary reads lives under FETCH_CONTEXT_HOME.
func TestAC_SANDBOX_01_FetchContextHomeRedirectsConfig(t *testing.T) {
	w := newWorkspace(t) // home is a fresh sandbox dir
	w.writeConfig(t, "profiles:\n  sandboxed-profile:\n    repos: [\"a/b\"]\n")
	res := w.run("list")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "sandboxed-profile") {
		t.Errorf("config under FETCH_CONTEXT_HOME was not read:\n%s", res.stdout)
	}
}

func TestAC_SANDBOX_02_TargetStaysRepoLocal(t *testing.T) {
	w := newWorkspace(t) // FETCH_CONTEXT_HOME already points at an unrelated dir
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) {
		t.Error("clone not under the workspace")
	}
	entries, err := os.ReadDir(w.home)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == ".agentic" || e.Name() == "repos" {
			t.Errorf("materialized content leaked into FETCH_CONTEXT_HOME: %s", e.Name())
		}
	}
}
