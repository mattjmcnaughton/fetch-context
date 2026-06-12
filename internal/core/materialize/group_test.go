package materialize

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/fakes"
)

type groupFixture struct {
	github  *fakes.FakeForgeEnumerator
	gitlab  *fakes.FakeForgeEnumerator
	git     *fakes.FakeGitRepo
	fs      *fakes.FakeFileStore
	locator *fakes.FakeHostRepoLocator
	uc      *Group
}

func newGroupFixture() *groupFixture {
	github := fakes.NewForgeEnumerator()
	gitlab := fakes.NewForgeEnumerator()
	git := fakes.NewGitRepo()
	fs := fakes.NewFileStore()
	git.FS = fs
	locator := &fakes.FakeHostRepoLocator{Root: "/ws"}
	enums := map[string]ports.ForgeEnumerator{
		"github.com": github,
		"gitlab.com": gitlab,
	}
	return &groupFixture{
		github:  github,
		gitlab:  gitlab,
		git:     git,
		fs:      fs,
		locator: locator,
		uc:      NewGroup(enums, git, fs, locator, slog.New(slog.DiscardHandler)),
	}
}

func (f *groupFixture) run(t *testing.T, refs ...string) error {
	t.Helper()
	return f.uc.Materialize(context.Background(), GroupRequest{Refs: refs, Target: ".agentic/sources", Depth: 1})
}

func TestGroupClonesEveryEnumeratedRepo(t *testing.T) {
	f := newGroupFixture()
	f.github.Repos["my-org"] = []ports.GroupRepo{
		{Path: "alpha", CloneURL: "http://git.test/my-org/alpha.git"},
		{Path: "beta", CloneURL: "http://git.test/my-org/beta.git"},
		{Path: "gamma", CloneURL: "http://git.test/my-org/gamma.git"},
	}
	if err := f.run(t, "github.com/my-org"); err != nil {
		t.Fatal(err)
	}
	wantDests := []string{
		"/ws/.agentic/sources/repos/github.com/my-org/alpha",
		"/ws/.agentic/sources/repos/github.com/my-org/beta",
		"/ws/.agentic/sources/repos/github.com/my-org/gamma",
	}
	if len(f.git.Clones) != 3 {
		t.Fatalf("Clones = %+v, want 3", f.git.Clones)
	}
	for i, want := range wantDests {
		if f.git.Clones[i].Dest != want {
			t.Errorf("clone %d dest = %q, want %q", i, f.git.Clones[i].Dest, want)
		}
	}
}

func TestGroupPreservesSubgroupPath(t *testing.T) {
	f := newGroupFixture()
	f.gitlab.Repos["acme/platform"] = []ports.GroupRepo{
		{Path: "top", CloneURL: "http://git.test/acme/platform/top.git"},
		{Path: "sub/nested", CloneURL: "http://git.test/acme/platform/sub/nested.git"},
	}
	if err := f.run(t, "gitlab.com/acme/platform"); err != nil {
		t.Fatal(err)
	}
	want := "/ws/.agentic/sources/repos/gitlab.com/acme/platform/sub/nested"
	found := false
	for _, c := range f.git.Clones {
		if c.Dest == want {
			found = true
		}
	}
	if !found {
		t.Errorf("Clones = %+v, want a clone at %q (AC-GROUP-02)", f.git.Clones, want)
	}
}

func TestGroupEnumerationFailureSurfaces(t *testing.T) {
	f := newGroupFixture()
	f.github.Errs["private-org"] = errors.New("401 Unauthorized: authentication required")

	err := f.run(t, "github.com/private-org")
	if err == nil {
		t.Fatal("enumeration failure must not be silently skipped (AC-GROUP-05)")
	}
	if !strings.Contains(err.Error(), "private-org") || !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q must name the group and the auth failure", err)
	}
	if len(f.git.Clones) != 0 {
		t.Errorf("Clones = %+v, want none", f.git.Clones)
	}
}

func TestGroupOneBadCloneDoesNotAbortRest(t *testing.T) {
	f := newGroupFixture()
	f.github.Repos["my-org"] = []ports.GroupRepo{
		{Path: "alpha", CloneURL: "http://git.test/my-org/alpha.git"},
		{Path: "beta", CloneURL: "http://git.test/my-org/beta.git"},
		{Path: "gamma", CloneURL: "http://git.test/my-org/gamma.git"},
	}
	f.git.CloneErrs["http://git.test/my-org/beta.git"] = errors.New("remote returned 404")

	err := f.run(t, "github.com/my-org")
	if err == nil {
		t.Fatal("want error when one clone fails (R3)")
	}
	if len(f.git.Clones) != 3 {
		t.Errorf("Clones = %+v, want all three attempted (AC-GROUP-06)", f.git.Clones)
	}
	if !strings.Contains(err.Error(), "beta") {
		t.Errorf("error %q must name the failed repo", err)
	}
	if strings.Contains(err.Error(), "alpha") || strings.Contains(err.Error(), "gamma") {
		t.Errorf("error %q must not blame successful repos", err)
	}
}

func TestGroupUnknownHostIsUsageError(t *testing.T) {
	f := newGroupFixture()
	err := f.run(t, "codeberg.org/some-org")
	if !usageerr.IsUsage(err) {
		t.Fatalf("err = %v, want a usage error for an unknown forge host (ADR-0001)", err)
	}
	if !strings.Contains(err.Error(), "codeberg.org") {
		t.Errorf("error %q must name the host", err)
	}
	if len(f.git.Clones) != 0 {
		t.Error("nothing must be materialized on a usage error")
	}
}

func TestGroupExistingClonesAreRefreshed(t *testing.T) {
	f := newGroupFixture()
	f.github.Repos["my-org"] = []ports.GroupRepo{
		{Path: "alpha", CloneURL: "http://git.test/my-org/alpha.git"},
	}
	dest := "/ws/.agentic/sources/repos/github.com/my-org/alpha"
	if err := f.fs.MkdirAll(dest); err != nil {
		t.Fatal(err)
	}
	f.git.ManagedDirs[dest] = true

	if err := f.run(t, "github.com/my-org"); err != nil {
		t.Fatal(err)
	}
	if len(f.git.Refreshes) != 1 || f.git.Refreshes[0].Dest != dest {
		t.Errorf("Refreshes = %v, want [%s] (AC-GROUP-04)", f.git.Refreshes, dest)
	}
}

func TestGroupDepthReachesEveryClone(t *testing.T) {
	f := newGroupFixture()
	f.github.Repos["my-org"] = []ports.GroupRepo{
		{Path: "alpha", CloneURL: "https://github.com/my-org/alpha.git"},
		{Path: "beta", CloneURL: "https://github.com/my-org/beta.git"},
	}
	if err := f.uc.Materialize(context.Background(), GroupRequest{
		Refs: []string{"github.com/my-org"}, Target: ".agentic/sources", Depth: 0,
	}); err != nil {
		t.Fatal(err)
	}
	if len(f.git.Clones) == 0 {
		t.Fatal("no clones recorded")
	}
	for _, c := range f.git.Clones {
		if c.Options != (ports.CloneOptions{Depth: 0}) {
			t.Errorf("clone %s options = %+v, want depth 0, no branch", c.URL, c.Options)
		}
	}
}

func TestGroupParallelFailuresStayInInputOrder(t *testing.T) {
	f := newGroupFixture()
	f.github.Repos["my-org"] = []ports.GroupRepo{
		{Path: "bad1", CloneURL: "https://github.com/my-org/bad1.git"},
		{Path: "ok", CloneURL: "https://github.com/my-org/ok.git"},
		{Path: "bad2", CloneURL: "https://github.com/my-org/bad2.git"},
	}
	f.git.CloneErrs["https://github.com/my-org/bad1.git"] = errors.New("boom1")
	f.git.CloneErrs["https://github.com/my-org/bad2.git"] = errors.New("boom2")

	err := f.uc.Materialize(context.Background(), GroupRequest{
		Refs: []string{"github.com/my-org"}, Target: ".agentic/sources", Depth: 1, Parallel: 3,
	})
	if err == nil {
		t.Fatal("want error")
	}
	var batch *BatchError
	if !errors.As(err, &batch) {
		t.Fatalf("err = %T, want *BatchError", err)
	}
	if len(batch.Items) != 2 ||
		batch.Items[0].Ref != "github.com/my-org: bad1" ||
		batch.Items[1].Ref != "github.com/my-org: bad2" {
		t.Errorf("failures = %+v, want bad1 then bad2 (enumeration order)", batch.Items)
	}
	if len(f.git.Clones) != 3 {
		t.Errorf("Clones = %d, want all three attempted", len(f.git.Clones))
	}
}
