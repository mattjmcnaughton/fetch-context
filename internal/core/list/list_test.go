package list

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/fakes"
)

type listFixture struct {
	config  *fakes.FakeConfigStore
	fs      *fakes.FakeFileStore
	locator *fakes.FakeHostRepoLocator
	uc      *List
}

func newListFixture() *listFixture {
	f := &listFixture{
		config:  fakes.NewConfigStore(),
		fs:      fakes.NewFileStore(),
		locator: &fakes.FakeHostRepoLocator{Root: "/ws"},
	}
	f.uc = New(f.config, f.fs, f.locator, slog.New(slog.DiscardHandler))
	return f
}

func TestListShowsProfilesAndContents(t *testing.T) {
	f := newListFixture()
	f.config.Cfg.Profiles["backend"] = ports.Profile{
		Repos:  fakes.RepoEntries("github.com/redis/redis"),
		Groups: []string{"gitlab.com/acme"},
		URLs:   []string{"https://example.com/blog/post"},
	}
	f.config.Cfg.Profiles["web-stack"] = ports.Profile{
		Repos: fakes.RepoEntries("github.com/foo/bar"),
	}

	out, err := f.uc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"backend", "web-stack",
		"github.com/redis/redis", "gitlab.com/acme", "https://example.com/blog/post",
		"github.com/foo/bar",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestListReportsMaterializedState(t *testing.T) {
	f := newListFixture()
	f.config.Cfg.Profiles["backend"] = ports.Profile{
		Repos: fakes.RepoEntries("github.com/redis/redis", "github.com/not/yet"),
		URLs:  []string{"https://example.com/blog/post"},
	}
	for _, p := range []string{
		"/ws/.agentic/sources/repos/github.com/redis/redis/README",
		"/ws/.agentic/sources/urls/example.com/blog/post.md",
	} {
		if err := f.fs.WriteFile(p, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}

	out, err := f.uc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out, "\n")
	requireLine := func(substr, marker string) {
		t.Helper()
		for _, line := range lines {
			if strings.Contains(line, substr) {
				if !strings.Contains(line, marker) {
					t.Errorf("line %q should carry %q", line, marker)
				}
				return
			}
		}
		t.Errorf("no line mentions %q:\n%s", substr, out)
	}
	requireLine("github.com/redis/redis", "[materialized]")
	requireLine("github.com/not/yet", "[absent]")
	requireLine("https://example.com/blog/post", "[materialized]")
}

func TestListEmptyConfigIsFriendly(t *testing.T) {
	f := newListFixture()
	out, err := f.uc.Run(context.Background())
	if err != nil {
		t.Fatalf("empty config must not error (AC-LIST-03): %v", err)
	}
	if !strings.Contains(out, "no profiles") {
		t.Errorf("output should say there are no profiles:\n%s", out)
	}
}

func TestListHonorsPerProfileTargetForMaterializedState(t *testing.T) {
	f := newListFixture()
	f.config.Cfg.Profiles["backend"] = ports.Profile{
		Target: ".agentic/backend",
		Repos:  fakes.RepoEntries("github.com/foo/bar"),
	}
	if err := f.fs.WriteFile("/ws/.agentic/backend/repos/github.com/foo/bar/x", []byte("x")); err != nil {
		t.Fatal(err)
	}
	out, err := f.uc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[materialized]") {
		t.Errorf("entry under the profile target should count as materialized:\n%s", out)
	}
}
