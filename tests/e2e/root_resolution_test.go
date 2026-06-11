//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAC_ROOT_01_TargetResolvedAgainstRepoRoot(t *testing.T) {
	w := newWorkspace(t)
	nested := filepath.Join(w.dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	res := w.runIn(nested, "repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(helloDest(w)) {
		t.Error("clone not under the repo root's .agentic/sources")
	}
	if exists(filepath.Join(nested, ".agentic")) {
		t.Error(".agentic was created under the CWD instead of the repo root")
	}
}

func TestAC_ROOT_02_OutsideGitRepoErrors(t *testing.T) {
	w := newWorkspace(t)
	plain := t.TempDir() // not a git repo

	res := w.runIn(plain, "repo", fixture.CloneURL("fixture/hello"))
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "repo root") {
		t.Errorf("stderr does not explain the unresolved repo root:\n%s", res.stderr)
	}
	if exists(filepath.Join(plain, ".agentic")) {
		t.Error(".agentic was created outside a git repo")
	}
}
