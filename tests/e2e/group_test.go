//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAC_GROUP_01_GitHubOrgEnumeratesFlat(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("group", "github.com/fixture-org")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	orgDir := w.target("repos", "github.com", "fixture-org")
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if !isGit(filepath.Join(orgDir, name)) {
			t.Errorf("clone for %s missing at %s", name, filepath.Join(orgDir, name))
		}
	}
	entries, err := os.ReadDir(orgDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("org dir has %d entries, want exactly 3 (no extra repos)", len(entries))
	}
}

func TestAC_GROUP_02_GitLabGroupRecursesPreservingSubgroupPath(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("group", "gitlab.com/acme")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !isGit(w.target("repos", "gitlab.com", "acme", "top")) {
		t.Error("top-level repo missing")
	}
	if !isGit(w.target("repos", "gitlab.com", "acme", "sub", "nested")) {
		t.Error("subgroup path not preserved: repos/gitlab.com/acme/sub/nested missing")
	}
}

func TestAC_GROUP_03_Pagination(t *testing.T) {
	w := newWorkspace(t)
	// The mock forge serves pages of at most 2; big-org holds 5 repos.
	res := w.run("group", "github.com/big-org")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	entries, err := os.ReadDir(w.target("repos", "github.com", "big-org"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Errorf("cloned %d repos, want all 5 across pages", len(entries))
	}
}

func TestAC_GROUP_04_EnumeratedReposObeyRepoRules(t *testing.T) {
	w := newWorkspace(t)
	if res := w.run("group", "github.com/fixture-org"); res.code != 0 {
		t.Fatalf("first run: exit = %d, stderr: %s", res.code, res.stderr)
	}
	alpha := w.target("repos", "github.com", "fixture-org", "alpha")
	if !isShallow(t, alpha) {
		t.Error("group clone not shallow")
	}
	writeFile(t, filepath.Join(alpha, "MARKER"), "mutated")

	res := w.run("group", "github.com/fixture-org")
	if res.code != 0 {
		t.Fatalf("re-run: exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !treeClean(t, alpha) {
		t.Error("mutated clone not hard-reset on re-run (per AC-REPO-07)")
	}
}

func TestAC_GROUP_05_MissingTokenReportedNotSkipped(t *testing.T) {
	w := newWorkspace(t) // no GITHUB_TOKEN in env
	res := w.run("group", "github.com/private-org")
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stdout: %s stderr: %s", res.code, res.stdout, res.stderr)
	}
	lower := strings.ToLower(res.stderr)
	if !strings.Contains(lower, "auth") && !strings.Contains(lower, "permission") {
		t.Errorf("stderr does not name an auth/permission failure:\n%s", res.stderr)
	}
}

func TestAC_GROUP_06_OneBadRepoDoesNotAbortRest(t *testing.T) {
	w := newWorkspace(t)
	fixture.SetFailing("fixture/beta", true)
	defer fixture.SetFailing("fixture/beta", false)

	res := w.run("group", "github.com/fixture-org")
	if res.code != 1 {
		t.Fatalf("exit = %d, want 1; stderr: %s", res.code, res.stderr)
	}
	orgDir := w.target("repos", "github.com", "fixture-org")
	for _, good := range []string{"alpha", "gamma"} {
		dir := filepath.Join(orgDir, good)
		if !isGit(dir) || !exists(filepath.Join(dir, "MARKER")) {
			t.Errorf("%s not fully cloned despite beta's failure", good)
		}
	}
	if !strings.Contains(res.stderr, "beta") {
		t.Errorf("stderr does not name beta as failed:\n%s", res.stderr)
	}
	if exists(filepath.Join(orgDir, "beta")) {
		t.Error("partial directory left for beta")
	}
}
