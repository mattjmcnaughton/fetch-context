package repoid

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Spec
	}{
		{
			"bare shorthand defaults to github.com",
			"foo/bar",
			Spec{Ref: "foo/bar", Host: "github.com", Owner: "foo", Repo: "bar"},
		},
		{
			"trailing slash stripped",
			"foo/bar/",
			Spec{Ref: "foo/bar/", Host: "github.com", Owner: "foo", Repo: "bar"},
		},
		{
			".git suffix stripped",
			"foo/bar.git",
			Spec{Ref: "foo/bar.git", Host: "github.com", Owner: "foo", Repo: "bar"},
		},
		{
			"host-qualified shorthand",
			"github.com/foo/bar",
			Spec{Ref: "github.com/foo/bar", Host: "github.com", Owner: "foo", Repo: "bar"},
		},
		{
			"host-qualified with subgroup path",
			"gitlab.com/acme/platform/team/utils",
			Spec{Ref: "gitlab.com/acme/platform/team/utils", Host: "gitlab.com", Owner: "acme/platform/team", Repo: "utils"},
		},
		{
			"full https URL with .git",
			"https://github.com/foo/bar.git",
			Spec{Ref: "https://github.com/foo/bar.git", Scheme: "https", Host: "github.com", Owner: "foo", Repo: "bar"},
		},
		{
			"full http URL with port",
			"http://127.0.0.1:5000/fixture/hello.git",
			Spec{Ref: "http://127.0.0.1:5000/fixture/hello.git", Scheme: "http", Host: "127.0.0.1:5000", Owner: "fixture", Repo: "hello"},
		},
		{
			"full URL without .git",
			"https://github.com/foo/bar",
			Spec{Ref: "https://github.com/foo/bar", Scheme: "https", Host: "github.com", Owner: "foo", Repo: "bar"},
		},
		{
			"scp-like SSH ref",
			"git@github.com:foo/bar.git",
			Spec{Ref: "git@github.com:foo/bar.git", Scheme: "ssh", User: "git", Host: "github.com", Owner: "foo", Repo: "bar", scpLike: true},
		},
		{
			"scp-like SSH ref with subgroup path",
			"git@gitlab.com:acme/platform/team/utils.git",
			Spec{Ref: "git@gitlab.com:acme/platform/team/utils.git", Scheme: "ssh", User: "git", Host: "gitlab.com", Owner: "acme/platform/team", Repo: "utils", scpLike: true},
		},
		{
			"ssh:// URL",
			"ssh://git@github.com/foo/bar.git",
			Spec{Ref: "ssh://git@github.com/foo/bar.git", Scheme: "ssh", User: "git", Host: "github.com", Owner: "foo", Repo: "bar"},
		},
		{
			"ssh:// URL with port",
			"ssh://git@github.com:2222/foo/bar.git",
			Spec{Ref: "ssh://git@github.com:2222/foo/bar.git", Scheme: "ssh", User: "git", Host: "github.com:2222", Owner: "foo", Repo: "bar"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", c.in, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Parse(%q) = %+v, want %+v", c.in, got, c.want)
			}
		})
	}
}

func TestKeyCollapsesAcrossProtocols(t *testing.T) {
	// SSH and HTTPS forms of the same repo must dedup to one clone and land
	// at the same host/owner/repo destination (AC-REPO-11 philosophy).
	forms := []string{
		"foo/bar",
		"github.com/foo/bar",
		"https://github.com/foo/bar.git",
		"git@github.com:foo/bar.git",
		"ssh://git@github.com/foo/bar.git",
	}
	want := "github.com/foo/bar"
	for _, in := range forms {
		spec, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", in, err)
		}
		if got := spec.Key(); got != want {
			t.Errorf("Parse(%q).Key() = %q, want %q", in, got, want)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"single segment", "foo"},
		{"host-qualified missing repo", "github.com/foo"},
		{"unsupported scheme", "ftp://host/a/b"},
		{"URL with single path segment", "https://host/justrepo.git"},
		{"whitespace only", "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, err := Parse(c.in); err == nil {
				t.Errorf("Parse(%q) = %+v, want error", c.in, got)
			}
		})
	}
}

func TestCloneURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"shorthand gets https and .git", "foo/bar", "https://github.com/foo/bar.git"},
		{"host-qualified gets https", "gitlab.com/acme/lib", "https://gitlab.com/acme/lib.git"},
		{"http scheme preserved", "http://127.0.0.1:5000/fixture/hello", "http://127.0.0.1:5000/fixture/hello.git"},
		{"equivalent forms share a clone URL", "https://github.com/foo/bar.git", "https://github.com/foo/bar.git"},
		{"scp-like SSH round-trips", "git@github.com:foo/bar.git", "git@github.com:foo/bar.git"},
		{"scp-like SSH without .git gets .git", "git@github.com:foo/bar", "git@github.com:foo/bar.git"},
		{"ssh:// URL preserved", "ssh://git@github.com/foo/bar.git", "ssh://git@github.com/foo/bar.git"},
		{"ssh:// URL with port preserved", "ssh://git@github.com:2222/foo/bar.git", "ssh://git@github.com:2222/foo/bar.git"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", c.in, err)
			}
			if got := spec.CloneURL(); got != c.want {
				t.Errorf("CloneURL() = %q, want %q", got, c.want)
			}
		})
	}
}
