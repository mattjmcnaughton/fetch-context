//go:build integration

package gitrepo

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/gitfixture"
)

// depth1 is the pre-CloneOptions default: shallow, remote default branch.
var depth1 = ports.CloneOptions{Depth: 1}

func newFixture(t *testing.T) *gitfixture.Server {
	t.Helper()
	s, err := gitfixture.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func newAdapter(creds ...Credential) *Adapter {
	return New(slog.New(slog.DiscardHandler), creds...)
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestCloneShallowDefaultBranch(t *testing.T) {
	s := newFixture(t)
	if err := s.Seed("fixture/hello", map[string]string{"MARKER": "m\n"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "sub", "hello")

	if err := newAdapter().Clone(context.Background(), s.CloneURL("fixture/hello"), dest, depth1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "MARKER")); err != nil {
		t.Errorf("MARKER missing: %v", err)
	}
	if got := gitIn(t, dest, "rev-parse", "--is-shallow-repository"); got != "true" {
		t.Errorf("is-shallow = %s, want true (AC-REPO-02)", got)
	}
	if got := gitIn(t, dest, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("branch = %s, want main (remote default)", got)
	}
}

func TestCloneFailureLeavesNoPartialDir(t *testing.T) {
	s := newFixture(t)
	dest := filepath.Join(t.TempDir(), "missing")

	err := newAdapter().Clone(context.Background(), s.CloneURL("fixture/does-not-exist-xyz"), dest, depth1)
	if err == nil {
		t.Fatal("want clone failure")
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Errorf("partial directory left at %s (AC-REPO-09)", dest)
	}
}

func TestRefreshDiscardsLocalStateAndPullsLatest(t *testing.T) {
	s := newFixture(t)
	if err := s.Seed("fixture/hello", map[string]string{"MARKER": "v1\n"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "hello")
	a := newAdapter()
	if err := a.Clone(context.Background(), s.CloneURL("fixture/hello"), dest, depth1); err != nil {
		t.Fatal(err)
	}

	// Inject a local edit and an untracked file; advance the remote.
	if err := os.WriteFile(filepath.Join(dest, "MARKER"), []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "UNTRACKED"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit("fixture/hello", map[string]string{"NEW": "v2\n"}); err != nil {
		t.Fatal(err)
	}

	if err := a.Refresh(context.Background(), dest, depth1); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dest, "status", "--porcelain"); got != "" {
		t.Errorf("tree not clean after refresh: %q (AC-REPO-07)", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "MARKER")); string(b) != "v1\n" {
		t.Errorf("MARKER = %q, want remote content restored", b)
	}
	if _, err := os.Stat(filepath.Join(dest, "UNTRACKED")); !os.IsNotExist(err) {
		t.Error("UNTRACKED survived refresh")
	}
	if _, err := os.Stat(filepath.Join(dest, "NEW")); err != nil {
		t.Errorf("NEW missing — refresh did not advance to remote latest: %v", err)
	}
}

func TestIsManagedClone(t *testing.T) {
	s := newFixture(t)
	if err := s.Seed("fixture/hello", map[string]string{"MARKER": "m"}); err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	gitIn(t, ws, "init", "-q")
	a := newAdapter()
	ctx := context.Background()

	cloneDest := filepath.Join(ws, "sources", "repos", "hello")
	if err := a.Clone(ctx, s.CloneURL("fixture/hello"), cloneDest, depth1); err != nil {
		t.Fatal(err)
	}
	if got, err := a.IsManagedClone(ctx, cloneDest); err != nil || !got {
		t.Errorf("IsManagedClone(clone) = %v, %v; want true", got, err)
	}

	// A plain directory nested inside the host repo is NOT a managed clone,
	// even though rev-parse finds the host's .git above it.
	plain := filepath.Join(ws, "sources", "repos", "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := a.IsManagedClone(ctx, plain); err != nil || got {
		t.Errorf("IsManagedClone(plain dir in host repo) = %v, %v; want false", got, err)
	}
}

func TestPrivateCloneWithCredential(t *testing.T) {
	s := newFixture(t)
	if err := s.SeedPrivate("private/secret", "s3cret", map[string]string{"f": "x"}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	noAuthDest := filepath.Join(t.TempDir(), "noauth")
	err := newAdapter().Clone(ctx, s.CloneURL("private/secret"), noAuthDest, depth1)
	if err == nil {
		t.Fatal("unauthenticated clone of private repo succeeded")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "auth") {
		t.Errorf("error %q should state the auth problem (AC-AUTH-03)", err)
	}

	authedDest := filepath.Join(t.TempDir(), "authed")
	a := newAdapter(Credential{Kind: KindGitHub, Token: "s3cret"})
	if err := a.Clone(ctx, s.CloneURL("private/secret"), authedDest, depth1); err != nil {
		t.Fatalf("authenticated clone failed: %v", err)
	}
}

func TestCloneHonorsContextCancellation(t *testing.T) {
	s := newFixture(t)
	if err := s.Seed("fixture/hello", map[string]string{"MARKER": "m"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := newAdapter().Clone(ctx, s.CloneURL("fixture/hello"), filepath.Join(t.TempDir(), "c"), depth1)
	if err == nil {
		t.Fatal("clone with canceled context succeeded")
	}
}

// seedWithHistory seeds name with commits successive content versions so
// depth assertions have more than one commit to count.
func seedWithHistory(t *testing.T, s *gitfixture.Server, name string, commits int) {
	t.Helper()
	if err := s.Seed(name, map[string]string{"MARKER": "v0\n"}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < commits; i++ {
		if err := s.Commit(name, map[string]string{"MARKER": fmt.Sprintf("v%d\n", i)}); err != nil {
			t.Fatal(err)
		}
	}
}

func commitCount(t *testing.T, dest string) int {
	t.Helper()
	n, err := strconv.Atoi(gitIn(t, dest, "rev-list", "--count", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCloneDepthN(t *testing.T) {
	s := newFixture(t)
	seedWithHistory(t, s, "fixture/deep", 3)
	dest := filepath.Join(t.TempDir(), "deep")

	if err := newAdapter().Clone(context.Background(), s.CloneURL("fixture/deep"), dest, ports.CloneOptions{Depth: 2}); err != nil {
		t.Fatal(err)
	}
	if got := commitCount(t, dest); got != 2 {
		t.Errorf("commit count = %d, want 2", got)
	}
	if got := gitIn(t, dest, "rev-parse", "--is-shallow-repository"); got != "true" {
		t.Errorf("is-shallow = %s, want true", got)
	}
}

func TestCloneDepthZeroIsFullHistory(t *testing.T) {
	s := newFixture(t)
	seedWithHistory(t, s, "fixture/deep", 3)
	dest := filepath.Join(t.TempDir(), "deep")

	if err := newAdapter().Clone(context.Background(), s.CloneURL("fixture/deep"), dest, ports.CloneOptions{Depth: 0}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dest, "rev-parse", "--is-shallow-repository"); got != "false" {
		t.Errorf("is-shallow = %s, want false", got)
	}
	if got := commitCount(t, dest); got != 3 {
		t.Errorf("commit count = %d, want 3", got)
	}
}

func TestCloneBranch(t *testing.T) {
	s := newFixture(t)
	if err := s.Seed("fixture/branchy", map[string]string{"MARKER": "main\n"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitOnBranch("fixture/branchy", "develop", map[string]string{"MARKER": "develop\n"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "branchy")

	if err := newAdapter().Clone(context.Background(), s.CloneURL("fixture/branchy"), dest, ports.CloneOptions{Depth: 1, Branch: "develop"}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dest, "symbolic-ref", "--short", "HEAD"); got != "develop" {
		t.Errorf("branch = %s, want develop", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "MARKER")); string(b) != "develop\n" {
		t.Errorf("MARKER = %q, want branch content", b)
	}
}

func TestRefreshKeepsFullHistory(t *testing.T) {
	s := newFixture(t)
	seedWithHistory(t, s, "fixture/deep", 2)
	dest := filepath.Join(t.TempDir(), "deep")
	a := newAdapter()
	full := ports.CloneOptions{Depth: 0}
	if err := a.Clone(context.Background(), s.CloneURL("fixture/deep"), dest, full); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit("fixture/deep", map[string]string{"MARKER": "v2\n"}); err != nil {
		t.Fatal(err)
	}

	if err := a.Refresh(context.Background(), dest, full); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dest, "rev-parse", "--is-shallow-repository"); got != "false" {
		t.Errorf("refresh re-shallowed a full clone: is-shallow = %s", got)
	}
	if got := commitCount(t, dest); got != 3 {
		t.Errorf("commit count = %d, want 3 (remote latest, full history)", got)
	}
}

func TestRefreshUnshallowsWhenDepthBecomesZero(t *testing.T) {
	s := newFixture(t)
	seedWithHistory(t, s, "fixture/deep", 3)
	dest := filepath.Join(t.TempDir(), "deep")
	a := newAdapter()
	if err := a.Clone(context.Background(), s.CloneURL("fixture/deep"), dest, ports.CloneOptions{Depth: 1}); err != nil {
		t.Fatal(err)
	}

	if err := a.Refresh(context.Background(), dest, ports.CloneOptions{Depth: 0}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dest, "rev-parse", "--is-shallow-repository"); got != "false" {
		t.Errorf("is-shallow = %s, want false after converging to depth 0", got)
	}
	if got := commitCount(t, dest); got != 3 {
		t.Errorf("commit count = %d, want 3", got)
	}
}

func TestRefreshKeepsDepthNShallow(t *testing.T) {
	s := newFixture(t)
	seedWithHistory(t, s, "fixture/deep", 3)
	dest := filepath.Join(t.TempDir(), "deep")
	a := newAdapter()
	d1 := ports.CloneOptions{Depth: 1}
	if err := a.Clone(context.Background(), s.CloneURL("fixture/deep"), dest, d1); err != nil {
		t.Fatal(err)
	}
	if err := s.Commit("fixture/deep", map[string]string{"MARKER": "v3\n"}); err != nil {
		t.Fatal(err)
	}

	if err := a.Refresh(context.Background(), dest, d1); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dest, "rev-parse", "--is-shallow-repository"); got != "true" {
		t.Errorf("is-shallow = %s, want true", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "MARKER")); string(b) != "v3\n" {
		t.Errorf("MARKER = %q, want remote latest", b)
	}
}

func TestRefreshSwitchesToPinnedBranch(t *testing.T) {
	s := newFixture(t)
	if err := s.Seed("fixture/branchy", map[string]string{"MARKER": "main\n"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitOnBranch("fixture/branchy", "develop", map[string]string{"MARKER": "develop\n"}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "branchy")
	a := newAdapter()
	if err := a.Clone(context.Background(), s.CloneURL("fixture/branchy"), dest, depth1); err != nil {
		t.Fatal(err)
	}

	if err := a.Refresh(context.Background(), dest, ports.CloneOptions{Depth: 1, Branch: "develop"}); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, dest, "symbolic-ref", "--short", "HEAD"); got != "develop" {
		t.Errorf("branch = %s, want develop after converging", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dest, "MARKER")); string(b) != "develop\n" {
		t.Errorf("MARKER = %q, want branch content", b)
	}
}
