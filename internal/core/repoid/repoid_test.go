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

func TestParseAllDedupesEquivalentForms(t *testing.T) {
	refs := []string{"foo/bar", "foo/bar/", "foo/bar.git", "https://github.com/foo/bar.git"}
	specs, bad := ParseAll(refs)
	if len(bad) != 0 {
		t.Fatalf("unexpected parse failures: %v", bad)
	}
	if len(specs) != 1 {
		t.Fatalf("len(specs) = %d, want 1 (equivalent forms must collapse); got %+v", len(specs), specs)
	}
	if specs[0].Key() != "github.com/foo/bar" {
		t.Errorf("Key() = %q", specs[0].Key())
	}
}

func TestParseAllKeepsOrderAndReportsFailures(t *testing.T) {
	refs := []string{"a/one", "not_a_ref", "b/two"}
	specs, bad := ParseAll(refs)
	if len(specs) != 2 || specs[0].Repo != "one" || specs[1].Repo != "two" {
		t.Fatalf("specs = %+v, want one then two", specs)
	}
	if len(bad) != 1 || bad[0].Ref != "not_a_ref" || bad[0].Err == nil {
		t.Fatalf("bad = %+v, want a single failure for not_a_ref", bad)
	}
}
