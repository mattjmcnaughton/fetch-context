package clean

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/fakes"
)

type cleanFixture struct {
	config  *fakes.FakeConfigStore
	fs      *fakes.FakeFileStore
	locator *fakes.FakeHostRepoLocator
	uc      *Clean
}

func newCleanFixture(t *testing.T) *cleanFixture {
	t.Helper()
	f := &cleanFixture{
		config:  fakes.NewConfigStore(),
		fs:      fakes.NewFileStore(),
		locator: &fakes.FakeHostRepoLocator{Root: "/ws"},
	}
	// Materialized content under the global target plus a bystander file.
	for _, p := range []string{
		"/ws/.agentic/sources/repos/github.com/foo/bar/MARKER",
		"/ws/.agentic/sources/urls/example.test/post.md",
		"/ws/keep.txt",
	} {
		if err := f.fs.WriteFile(p, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	f.uc = New(f.config, f.fs, f.locator, slog.New(slog.DiscardHandler))
	return f
}

func (f *cleanFixture) mustExist(t *testing.T, path string, want bool) {
	t.Helper()
	got, err := f.fs.Exists(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Exists(%s) = %v, want %v", path, got, want)
	}
}

func TestCleanNoArgRemovesWholeTarget(t *testing.T) {
	f := newCleanFixture(t)
	if err := f.uc.Run(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	f.mustExist(t, "/ws/.agentic/sources/repos", false)
	f.mustExist(t, "/ws/.agentic/sources/urls", false)
	f.mustExist(t, "/ws/keep.txt", true)
}

func TestCleanReposScope(t *testing.T) {
	f := newCleanFixture(t)
	if err := f.uc.Run(context.Background(), "repos"); err != nil {
		t.Fatal(err)
	}
	f.mustExist(t, "/ws/.agentic/sources/repos", false)
	f.mustExist(t, "/ws/.agentic/sources/urls/example.test/post.md", true)
}

func TestCleanURLsScope(t *testing.T) {
	f := newCleanFixture(t)
	if err := f.uc.Run(context.Background(), "urls"); err != nil {
		t.Fatal(err)
	}
	f.mustExist(t, "/ws/.agentic/sources/urls", false)
	f.mustExist(t, "/ws/.agentic/sources/repos/github.com/foo/bar/MARKER", true)
}

func TestCleanProfileClearsOnlyThatProfilesTarget(t *testing.T) {
	f := newCleanFixture(t)
	f.config.Cfg.Profiles["backend"] = ports.Profile{Target: ".agentic/backend"}
	f.config.Cfg.Profiles["web"] = ports.Profile{Target: ".agentic/web"}
	for _, p := range []string{
		"/ws/.agentic/backend/repos/x/MARKER",
		"/ws/.agentic/web/repos/y/MARKER",
	} {
		if err := f.fs.WriteFile(p, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}

	if err := f.uc.Run(context.Background(), "backend"); err != nil {
		t.Fatal(err)
	}
	f.mustExist(t, "/ws/.agentic/backend", false)
	// Never auto-discovers other profiles' targets (R7, AC-CLEAN-05).
	f.mustExist(t, "/ws/.agentic/web/repos/y/MARKER", true)
	f.mustExist(t, "/ws/.agentic/sources/repos/github.com/foo/bar/MARKER", true)
}

func TestCleanProfileWithoutOverrideUsesGlobalTarget(t *testing.T) {
	f := newCleanFixture(t)
	f.config.Cfg.Profiles["plain"] = ports.Profile{Repos: []string{"a/b"}}
	if err := f.uc.Run(context.Background(), "plain"); err != nil {
		t.Fatal(err)
	}
	f.mustExist(t, "/ws/.agentic/sources", false)
}

func TestCleanUnknownScopeIsUsageError(t *testing.T) {
	f := newCleanFixture(t)
	err := f.uc.Run(context.Background(), "no-such-thing")
	if !usageerr.IsUsage(err) {
		t.Fatalf("err = %v, want usage error", err)
	}
	f.mustExist(t, "/ws/.agentic/sources/repos", true)
}

func TestCleanRefusesPathsOutsideTheRepoRoot(t *testing.T) {
	f := newCleanFixture(t)
	f.config.Cfg.Profiles["evil"] = ports.Profile{Target: "../elsewhere"}
	err := f.uc.Run(context.Background(), "evil")
	if err == nil {
		t.Fatal("want refusal for a target escaping the repo root (AC-CLEAN-04)")
	}
	if !strings.Contains(err.Error(), "refus") {
		t.Errorf("error %q should state the refusal", err)
	}
}

func TestCleanRefusesTargetEqualToRepoRoot(t *testing.T) {
	f := newCleanFixture(t)
	f.config.Cfg.Target = "."
	err := f.uc.Run(context.Background(), "")
	if err == nil {
		t.Fatal("want refusal when the target resolves to the repo root itself")
	}
	f.mustExist(t, "/ws/keep.txt", true)
}
