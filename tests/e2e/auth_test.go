//go:build e2e

package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAC_AUTH_01_PublicRepoNeedsNoToken(t *testing.T) {
	w := newWorkspace(t) // baseEnv carries no tokens
	res := w.run("repo", fixture.CloneURL("fixture/hello"))
	if res.code != 0 {
		t.Fatalf("exit = %d, want 0 without a token; stderr: %s", res.code, res.stderr)
	}
}

func TestAC_AUTH_02_TokenUsedForPrivateRepo(t *testing.T) {
	w := newWorkspace(t)
	w.setEnv("GITHUB_TOKEN=" + privateToken)
	res := w.run("repo", fixture.CloneURL("private/secret"))
	if res.code != 0 {
		t.Fatalf("exit = %d, want 0 with token; stderr: %s", res.code, res.stderr)
	}
	dest := w.target("repos", fixture.Host(), "private", "secret")
	if !isGit(dest) || !exists(filepath.Join(dest, "SECRET")) {
		t.Error("private repo was not cloned")
	}
}

func TestAC_AUTH_03_AuthFailureSurfaced(t *testing.T) {
	w := newWorkspace(t) // no token
	res := w.run("repo", fixture.CloneURL("private/secret"))
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(strings.ToLower(res.stderr), "auth") {
		t.Errorf("stderr does not state the auth problem:\n%s", res.stderr)
	}
}
