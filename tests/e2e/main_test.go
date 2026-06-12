//go:build e2e

// Package e2e exercises the compiled fetch-context binary against the
// acceptance criteria in docs/acceptance.md. It is black-box: the only
// internal imports allowed are the fixtures under internal/testing/.
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/testing/forgemock"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/gitfixture"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/readermock"
)

// fcbin is the absolute path of the binary under test.
var fcbin string

// fixture is the suite-wide loopback git server (acceptance.md §1.3).
var fixture *gitfixture.Server

// reader is the suite-wide mock reader proxy.
var reader *readermock.Server

// forge is the suite-wide mock GitHub/GitLab API.
var forge *forgemock.Server

// privateToken gates the private fixture repo (AC-AUTH-02/03).
const privateToken = "s3cret-token"

func TestMain(m *testing.M) {
	code, err := runSuite(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e setup:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

// runSuite builds the binary unless $FCBIN is already set (the just/Docker
// path pre-builds it), boots the loopback fixtures, and runs the tests.
func runSuite(m *testing.M) (int, error) {
	fcbin = os.Getenv("FCBIN")
	if fcbin == "" {
		dir, err := os.MkdirTemp("", "fcbin-")
		if err != nil {
			return 0, err
		}
		defer os.RemoveAll(dir)
		fcbin = filepath.Join(dir, "fetch-context")
		build := exec.Command("go", "build", "-o", fcbin, "./cmd/fetch-context")
		build.Dir = repoRoot()
		if out, err := build.CombinedOutput(); err != nil {
			return 0, fmt.Errorf("building binary under test: %v\n%s", err, out)
		}
		os.Setenv("FCBIN", fcbin)
	}

	var err error
	fixture, err = gitfixture.New()
	if err != nil {
		return 0, fmt.Errorf("starting gitfixture: %w", err)
	}
	defer fixture.Close()
	if err := seedFixture(); err != nil {
		return 0, fmt.Errorf("seeding gitfixture: %w", err)
	}

	reader = readermock.New()
	defer reader.Close()

	forge = forgemock.New()
	defer forge.Close()
	seedForge()

	return m.Run(), nil
}

// seedForge wires the mock forge's orgs/groups to clone URLs on the git
// fixture (§1.3: every entry's clone URL points back at the git server).
func seedForge() {
	repo := func(name string) forgemock.Repo {
		return forgemock.Repo{Path: lastSegment(name), CloneURL: fixture.CloneURL(name)}
	}
	forge.SeedGitHubOrg("fixture-org", []forgemock.Repo{
		repo("fixture/alpha"), repo("fixture/beta"), repo("fixture/gamma"),
	})
	forge.SeedGitHubOrg("big-org", []forgemock.Repo{
		repo("big/pg1"), repo("big/pg2"), repo("big/pg3"), repo("big/pg4"), repo("big/pg5"),
	})
	forge.RequireGitHubToken("private-org", privateToken)
	forge.SeedGitHubOrg("private-org", []forgemock.Repo{repo("private/secret")})
	forge.SeedGitLabGroup("acme", []forgemock.Repo{
		{Path: "top", CloneURL: fixture.CloneURL("acme/top")},
		{Path: "sub/nested", CloneURL: fixture.CloneURL("acme/sub/nested")},
	})
	// Small pages force pagination across the suite (AC-GROUP-03).
	forge.SetMaxPageSize(2)
}

func lastSegment(name string) string {
	return filepath.Base(name)
}

// seedFixture creates the §1.3 seed repos.
func seedFixture() error {
	public := map[string]map[string]string{
		"fixture/hello":   {"MARKER": "hello marker\n"},
		"fixture/other":   {"OTHER": "other\n"},
		"fixture/refresh": {"MARKER": "v1\n"},
		"fixture/alpha":   {"MARKER": "alpha\n"},
		"fixture/beta":    {"MARKER": "beta\n"},
		"fixture/gamma":   {"MARKER": "gamma\n"},
		"big/pg1":         {"N": "1"},
		"big/pg2":         {"N": "2"},
		"big/pg3":         {"N": "3"},
		"big/pg4":         {"N": "4"},
		"big/pg5":         {"N": "5"},
		"acme/top":        {"MARKER": "top\n"},
		"acme/sub/nested": {"MARKER": "nested\n"},
	}
	for name, files := range public {
		if err := fixture.Seed(name, files); err != nil {
			return err
		}
	}
	// fixture/deep carries three commits on main so depth/full-history
	// assertions have history to count (AC-REPO-12/14, AC-CONFIG-05).
	if err := fixture.Seed("fixture/deep", map[string]string{"MARKER": "v1\n"}); err != nil {
		return err
	}
	for _, v := range []string{"v2\n", "v3\n"} {
		if err := fixture.Commit("fixture/deep", map[string]string{"MARKER": v}); err != nil {
			return err
		}
	}
	// fixture/branchy has a develop branch with distinct content
	// (AC-REPO-13, AC-LOAD-07).
	if err := fixture.Seed("fixture/branchy", map[string]string{"MARKER": "main\n"}); err != nil {
		return err
	}
	if err := fixture.CommitOnBranch("fixture/branchy", "develop", map[string]string{"MARKER": "develop\n"}); err != nil {
		return err
	}
	return fixture.SeedPrivate("private/secret", privateToken, map[string]string{"SECRET": "s\n"})
}

// repoRoot walks up from the test's working directory to the directory
// containing go.mod.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found above " + dir)
		}
		dir = parent
	}
}
