package profile

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/fakes"
)

type loadFixture struct {
	config *fakes.FakeConfigStore
	repos  *fakes.FakeMaterializer[materialize.RepoRequest]
	groups *fakes.FakeMaterializer[materialize.GroupRequest]
	urls   *fakes.FakeMaterializer[materialize.URLRequest]
	uc     *Load
}

func newLoadFixture() *loadFixture {
	f := &loadFixture{
		config: fakes.NewConfigStore(),
		repos:  &fakes.FakeMaterializer[materialize.RepoRequest]{},
		groups: &fakes.FakeMaterializer[materialize.GroupRequest]{},
		urls:   &fakes.FakeMaterializer[materialize.URLRequest]{},
	}
	f.config.Cfg.Profiles["backend"] = ports.Profile{
		Repos:  fakes.RepoEntries("github.com/redis/redis"),
		Groups: []string{"github.com/my-org"},
		URLs:   []string{"https://example.com/blog/post"},
	}
	f.uc = NewLoad(f.config, f.repos, f.groups, f.urls, slog.New(slog.DiscardHandler))
	return f
}

func TestLoadMaterializesAllKeys(t *testing.T) {
	f := newLoadFixture()
	if err := f.uc.Run(context.Background(), "backend"); err != nil {
		t.Fatal(err)
	}
	if len(f.repos.Requests) != 1 || f.repos.Requests[0].Refs[0] != "github.com/redis/redis" {
		t.Errorf("repo requests = %+v", f.repos.Requests)
	}
	if len(f.groups.Requests) != 1 || f.groups.Requests[0].Refs[0] != "github.com/my-org" {
		t.Errorf("group requests = %+v", f.groups.Requests)
	}
	if len(f.urls.Requests) != 1 || f.urls.Requests[0].URLs[0] != "https://example.com/blog/post" {
		t.Errorf("url requests = %+v", f.urls.Requests)
	}
	// Without a per-profile override, the global target is used.
	if f.repos.Requests[0].Target != ".agentic/sources" {
		t.Errorf("repo target = %q", f.repos.Requests[0].Target)
	}
}

func TestLoadPerProfileTargetOverride(t *testing.T) {
	f := newLoadFixture()
	f.config.Cfg.Profiles["backend"] = ports.Profile{
		Target: ".agentic/backend",
		Repos:  fakes.RepoEntries("a/b"),
	}
	if err := f.uc.Run(context.Background(), "backend"); err != nil {
		t.Fatal(err)
	}
	if f.repos.Requests[0].Target != ".agentic/backend" {
		t.Errorf("target = %q, want the profile override (AC-LOAD-02)", f.repos.Requests[0].Target)
	}
}

func TestLoadUnknownProfileIsUsageError(t *testing.T) {
	f := newLoadFixture()
	err := f.uc.Run(context.Background(), "no-such-profile")
	if !usageerr.IsUsage(err) {
		t.Fatalf("err = %v, want usage error (exit 2, AC-LOAD-04)", err)
	}
	if !strings.Contains(err.Error(), "no-such-profile") {
		t.Errorf("error %q must name the unknown profile", err)
	}
	if len(f.repos.Requests)+len(f.groups.Requests)+len(f.urls.Requests) != 0 {
		t.Error("nothing must be materialized for an unknown profile")
	}
}

func TestLoadContinuesAcrossKeysOnError(t *testing.T) {
	f := newLoadFixture()
	f.repos.Err = errors.New("1 item(s) failed:\n  bad/repo: clone failed")

	err := f.uc.Run(context.Background(), "backend")
	if err == nil {
		t.Fatal("want error when a key fails (R3, AC-LOAD-06)")
	}
	if len(f.groups.Requests) != 1 || len(f.urls.Requests) != 1 {
		t.Error("a failing repos key must not abort groups/urls")
	}
	if !strings.Contains(err.Error(), "bad/repo") {
		t.Errorf("error %q must carry the per-item detail", err)
	}
}

func TestLoadSkipsEmptyKeys(t *testing.T) {
	f := newLoadFixture()
	f.config.Cfg.Profiles["urls-only"] = ports.Profile{URLs: []string{"https://e.test/x"}}
	if err := f.uc.Run(context.Background(), "urls-only"); err != nil {
		t.Fatal(err)
	}
	if len(f.repos.Requests) != 0 || len(f.groups.Requests) != 0 {
		t.Error("empty keys must not trigger materializer calls")
	}
	if len(f.urls.Requests) != 1 {
		t.Error("urls key not materialized")
	}
}

func TestLoadConfigErrorSurfaces(t *testing.T) {
	f := newLoadFixture()
	f.config.LoadErr = errors.New("parsing config: yaml: line 3")
	if err := f.uc.Run(context.Background(), "backend"); err == nil {
		t.Fatal("want config error surfaced")
	}
}
