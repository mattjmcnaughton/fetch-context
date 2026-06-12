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

	"github.com/mattjmcnaughton/fetch-context/internal/testing/gitfixture"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/readermock"
)

// fcbin is the absolute path of the binary under test.
var fcbin string

// fixture is the suite-wide loopback git server (acceptance.md §1.3).
var fixture *gitfixture.Server

// reader is the suite-wide mock reader proxy.
var reader *readermock.Server

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

	return m.Run(), nil
}

// seedFixture creates the §1.3 seed repos.
func seedFixture() error {
	public := map[string]map[string]string{
		"fixture/hello":   {"MARKER": "hello marker\n"},
		"fixture/other":   {"OTHER": "other\n"},
		"fixture/refresh": {"MARKER": "v1\n"},
	}
	for name, files := range public {
		if err := fixture.Seed(name, files); err != nil {
			return err
		}
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
