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
)

// fcbin is the absolute path of the binary under test.
var fcbin string

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
	return m.Run(), nil
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
