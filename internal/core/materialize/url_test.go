package materialize

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/testing/fakes"
)

type urlFixture struct {
	reader  *fakes.FakePageReader
	fs      *fakes.FakeFileStore
	locator *fakes.FakeHostRepoLocator
	uc      *URL
}

func newURLFixture() *urlFixture {
	reader := fakes.NewPageReader()
	fs := fakes.NewFileStore()
	locator := &fakes.FakeHostRepoLocator{Root: "/ws"}
	return &urlFixture{
		reader:  reader,
		fs:      fs,
		locator: locator,
		uc:      NewURL(reader, fs, locator, slog.New(slog.DiscardHandler)),
	}
}

func (f *urlFixture) run(t *testing.T, urls ...string) error {
	t.Helper()
	return f.uc.Materialize(context.Background(), URLRequest{URLs: urls, Target: ".agentic/sources"})
}

func TestURLWritesMappedFile(t *testing.T) {
	f := newURLFixture()
	f.reader.Pages["http://example.test/blog/post"] = []byte("PAGE-CONTENT")

	if err := f.run(t, "http://example.test/blog/post"); err != nil {
		t.Fatal(err)
	}
	got := f.fs.ReadString("/ws/.agentic/sources/urls/example.test/blog/post.md")
	if got != "PAGE-CONTENT" {
		t.Errorf("written content = %q", got)
	}
}

func TestURLOverwritesOnRefetch(t *testing.T) {
	f := newURLFixture()
	path := "/ws/.agentic/sources/urls/example.test/blog/post.md"
	if err := f.fs.WriteFile(path, []byte("STALE")); err != nil {
		t.Fatal(err)
	}
	f.reader.Pages["http://example.test/blog/post"] = []byte("FRESH")

	if err := f.run(t, "http://example.test/blog/post"); err != nil {
		t.Fatal(err)
	}
	if got := f.fs.ReadString(path); got != "FRESH" {
		t.Errorf("content = %q, want overwritten (AC-URL-03)", got)
	}
}

func TestURLBatchContinuesOnError(t *testing.T) {
	f := newURLFixture()
	f.reader.Errs["http://bad.test/x"] = errors.New("reader returned 502")

	err := f.run(t, "http://good.test/a", "http://bad.test/x", "http://good.test/b")
	if err == nil {
		t.Fatal("want error when any item fails (R3)")
	}
	if len(f.reader.Fetched) != 3 {
		t.Errorf("Fetched = %v, want all three attempted", f.reader.Fetched)
	}
	for _, page := range []string{"/ws/.agentic/sources/urls/good.test/a.md", "/ws/.agentic/sources/urls/good.test/b.md"} {
		if got := f.fs.ReadString(page); got == "" {
			t.Errorf("good page %s not written", page)
		}
	}
	if !strings.Contains(err.Error(), "bad.test") || !strings.Contains(err.Error(), "502") {
		t.Errorf("error %q must name the failed URL and reason", err)
	}
}

func TestURLInvalidURLIsItemFailure(t *testing.T) {
	f := newURLFixture()
	err := f.run(t, "ftp://nope/x", "http://good.test/a")
	if err == nil {
		t.Fatal("want error for the invalid URL")
	}
	if len(f.reader.Fetched) != 1 {
		t.Errorf("Fetched = %v, want only the good URL", f.reader.Fetched)
	}
}

func TestURLWritesGitignore(t *testing.T) {
	f := newURLFixture()
	if err := f.run(t, "http://example.test/a"); err != nil {
		t.Fatal(err)
	}
	if got := f.fs.ReadString("/ws/.agentic/sources/.gitignore"); got != "*" {
		t.Errorf("gitignore = %q, want %q", got, "*")
	}
}

func TestURLLocatorFailureCreatesNothing(t *testing.T) {
	f := newURLFixture()
	f.locator.Err = errors.New("not inside a git repository")
	if err := f.run(t, "http://example.test/a"); err == nil {
		t.Fatal("want locator error surfaced")
	}
	if len(f.reader.Fetched) != 0 {
		t.Error("reader was called despite locator failure")
	}
}
