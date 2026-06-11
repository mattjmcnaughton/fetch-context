//go:build integration

package gitfixture

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The fixture itself is behavior, so it gets a smoke test: serve a bare
// repo, clone it with the real git client, see the seeded file.
func TestServeAndClone(t *testing.T) {
	s := newServer(t)
	if err := s.Seed("fixture/hello", map[string]string{"MARKER": "marker\n"}); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "clone")
	clone(t, s.CloneURL("fixture/hello"), dest, nil)
	if _, err := os.Stat(filepath.Join(dest, "MARKER")); err != nil {
		t.Errorf("MARKER missing in clone: %v", err)
	}
}

func TestFailSwitch(t *testing.T) {
	s := newServer(t)
	if err := s.Seed("fixture/beta", map[string]string{"f": "x"}); err != nil {
		t.Fatal(err)
	}
	s.SetFailing("fixture/beta", true)
	dest := filepath.Join(t.TempDir(), "clone")
	if err := tryClone(s.CloneURL("fixture/beta"), dest, nil); err == nil {
		t.Fatal("clone succeeded, want failure while fail switch is on")
	}
	s.SetFailing("fixture/beta", false)
	clone(t, s.CloneURL("fixture/beta"), dest, nil)
}

func TestPrivateRepoRequiresToken(t *testing.T) {
	s := newServer(t)
	if err := s.SeedPrivate("private/secret", "s3cret", map[string]string{"f": "x"}); err != nil {
		t.Fatal(err)
	}

	noAuth := filepath.Join(t.TempDir(), "noauth")
	if err := tryClone(s.CloneURL("private/secret"), noAuth, nil); err == nil {
		t.Fatal("unauthenticated clone succeeded, want 401 failure")
	}

	authed := filepath.Join(t.TempDir(), "authed")
	header := basicAuthHeader(t, "s3cret")
	clone(t, s.CloneURL("private/secret"), authed, []string{"-c", "http.extraHeader=" + header})
}

func TestCommitAdvancesRemote(t *testing.T) {
	s := newServer(t)
	if err := s.Seed("fixture/hello", map[string]string{"MARKER": "v1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit("fixture/hello", map[string]string{"NEW": "v2"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "clone")
	clone(t, s.CloneURL("fixture/hello"), dest, nil)
	if _, err := os.Stat(filepath.Join(dest, "NEW")); err != nil {
		t.Errorf("NEW missing after Commit: %v", err)
	}
}

func newServer(t *testing.T) *Server {
	t.Helper()
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func clone(t *testing.T, url, dest string, extraCfg []string) {
	t.Helper()
	if err := tryClone(url, dest, extraCfg); err != nil {
		t.Fatalf("clone %s: %v", url, err)
	}
}

func tryClone(url, dest string, extraCfg []string) error {
	args := append(extraCfg, "clone", "-q", "--depth=1", url, dest)
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &cloneError{err: err, out: string(out)}
	}
	return nil
}

type cloneError struct {
	err error
	out string
}

func (e *cloneError) Error() string { return e.err.Error() + ": " + e.out }

func basicAuthHeader(t *testing.T, token string) string {
	t.Helper()
	return "Authorization: Basic " + BasicCredential("token", token)
}
