//go:build e2e

package e2e

import (
	"testing"
)

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
