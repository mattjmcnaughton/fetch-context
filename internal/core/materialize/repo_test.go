package materialize

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
	"github.com/mattjmcnaughton/fetch-context/internal/testing/fakes"
)

type repoFixture struct {
	git     *fakes.FakeGitRepo
	fs      *fakes.FakeFileStore
	locator *fakes.FakeHostRepoLocator
	uc      *Repo
}

func newRepoFixture() *repoFixture {
	git := fakes.NewGitRepo()
	fs := fakes.NewFileStore()
	git.FS = fs
	locator := &fakes.FakeHostRepoLocator{Root: "/ws"}
	return &repoFixture{
		git:     git,
		fs:      fs,
		locator: locator,
		uc:      NewRepo(git, fs, locator, slog.New(slog.DiscardHandler)),
	}
}

func (f *repoFixture) run(t *testing.T, refs ...string) error {
	t.Helper()
	return f.uc.Materialize(context.Background(), RepoRequest{Refs: refs, Target: ".agentic/sources"})
}

func TestRepoClonesToDerivedPath(t *testing.T) {
	f := newRepoFixture()
	if err := f.run(t, "github.com/foo/bar"); err != nil {
		t.Fatal(err)
	}
	want := fakes.CloneCall{URL: "https://github.com/foo/bar.git", Dest: "/ws/.agentic/sources/repos/github.com/foo/bar", Options: ports.CloneOptions{Depth: 1}}
	if len(f.git.Clones) != 1 || f.git.Clones[0] != want {
		t.Errorf("Clones = %+v, want [%+v]", f.git.Clones, want)
	}
}

func TestRepoExistingManagedCloneIsRefreshed(t *testing.T) {
	f := newRepoFixture()
	dest := "/ws/.agentic/sources/repos/github.com/foo/bar"
	if err := f.fs.MkdirAll(dest); err != nil {
		t.Fatal(err)
	}
	f.git.ManagedDirs[dest] = true

	if err := f.run(t, "github.com/foo/bar"); err != nil {
		t.Fatal(err)
	}
	if len(f.git.Clones) != 0 {
		t.Errorf("Clones = %+v, want none", f.git.Clones)
	}
	if len(f.git.Refreshes) != 1 || f.git.Refreshes[0].Dest != dest {
		t.Errorf("Refreshes = %+v, want [%s]", f.git.Refreshes, dest)
	}
}

func TestRepoExistingNonCloneRefusedAndUntouched(t *testing.T) {
	f := newRepoFixture()
	dest := "/ws/.agentic/sources/repos/github.com/foo/bar"
	sentinel := dest + "/SENTINEL"
	if err := f.fs.WriteFile(sentinel, []byte("keep")); err != nil {
		t.Fatal(err)
	}

	err := f.run(t, "github.com/foo/bar")
	if err == nil {
		t.Fatal("want error for unmanaged destination")
	}
	if !strings.Contains(err.Error(), "not a managed clone") {
		t.Errorf("error %q does not explain the refusal", err)
	}
	if len(f.git.Clones)+len(f.git.Refreshes) != 0 {
		t.Errorf("git was invoked: clones %v refreshes %v", f.git.Clones, f.git.Refreshes)
	}
	if got := f.fs.ReadString(sentinel); got != "keep" {
		t.Errorf("sentinel content = %q, want untouched", got)
	}
}

func TestRepoBatchContinuesOnErrorAndAggregates(t *testing.T) {
	f := newRepoFixture()
	f.git.CloneErrs["https://github.com/foo/bad.git"] = errors.New("remote returned 404")

	err := f.run(t, "foo/one", "foo/bad", "foo/two")
	if err == nil {
		t.Fatal("want error when any item fails (R3)")
	}
	if len(f.git.Clones) != 3 {
		t.Errorf("Clones = %+v, want all three attempted", f.git.Clones)
	}
	if !strings.Contains(err.Error(), "foo/bad") || !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q must name the failed item and its reason", err)
	}
	if strings.Contains(err.Error(), "foo/one") || strings.Contains(err.Error(), "foo/two") {
		t.Errorf("error %q must not blame the successful items", err)
	}
}

func TestRepoInvalidRefIsAnItemFailureNotFatal(t *testing.T) {
	f := newRepoFixture()
	err := f.run(t, "good/one", "not_a_ref")
	if err == nil {
		t.Fatal("want error for the invalid ref")
	}
	if len(f.git.Clones) != 1 {
		t.Errorf("Clones = %+v, want the good item cloned", f.git.Clones)
	}
	if !strings.Contains(err.Error(), "not_a_ref") {
		t.Errorf("error %q must name the invalid ref", err)
	}
}

func TestRepoDedupesEquivalentForms(t *testing.T) {
	f := newRepoFixture()
	if err := f.run(t, "foo/bar", "foo/bar/", "foo/bar.git", "https://github.com/foo/bar.git"); err != nil {
		t.Fatal(err)
	}
	if len(f.git.Clones) != 1 {
		t.Errorf("Clones = %+v, want exactly one (AC-REPO-11)", f.git.Clones)
	}
}

func TestRepoWritesGitignoreIdempotently(t *testing.T) {
	f := newRepoFixture()
	if err := f.run(t, "foo/bar"); err != nil {
		t.Fatal(err)
	}
	gi := "/ws/.agentic/sources/.gitignore"
	if got := f.fs.ReadString(gi); got != "*" {
		t.Fatalf("gitignore content = %q, want exactly %q", got, "*")
	}
	// Second run leaves it alone (still exactly `*`, not appended).
	if err := f.run(t, "foo/bar"); err != nil {
		t.Fatal(err)
	}
	if got := f.fs.ReadString(gi); got != "*" {
		t.Errorf("gitignore after second run = %q, want %q", got, "*")
	}
}

func TestRepoLocatorFailureCreatesNothing(t *testing.T) {
	f := newRepoFixture()
	f.locator.Err = errors.New("not inside a git repository")

	err := f.run(t, "foo/bar")
	if err == nil || !strings.Contains(err.Error(), "not inside a git repository") {
		t.Fatalf("err = %v, want the locator failure surfaced", err)
	}
	if len(f.git.Clones) != 0 {
		t.Errorf("Clones = %+v, want none", f.git.Clones)
	}
	if exists, _ := f.fs.Exists("/ws/.agentic"); exists {
		t.Error(".agentic was created despite the locator failure (AC-ROOT-02)")
	}
}

func TestRepoCloneFailureForOneItemStillExitsNonNil(t *testing.T) {
	f := newRepoFixture()
	f.git.CloneErrs["https://github.com/foo/bad.git"] = errors.New("boom")
	err := f.run(t, "foo/bad")
	if err == nil {
		t.Fatal("want error")
	}
	var batch *BatchError
	if !errors.As(err, &batch) {
		t.Fatalf("err = %T, want *BatchError", err)
	}
	if len(batch.Items) != 1 || batch.Items[0].Ref != "foo/bad" {
		t.Errorf("batch items = %+v", batch.Items)
	}
}
