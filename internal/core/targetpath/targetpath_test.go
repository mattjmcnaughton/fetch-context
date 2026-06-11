package targetpath

import (
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/core/repoid"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		name     string
		repoRoot string
		target   string
		want     string
	}{
		{"default target", "/ws", ".agentic/sources", "/ws/.agentic/sources"},
		{"profile override", "/ws", ".agentic/backend", "/ws/.agentic/backend"},
		{"nested repo root", "/home/u/proj", ".agentic/sources", "/home/u/proj/.agentic/sources"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.repoRoot, c.target); got != c.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", c.repoRoot, c.target, got, c.want)
			}
		})
	}
}

func TestRepoDir(t *testing.T) {
	cases := []struct {
		name string
		ref  string
		want string
	}{
		{"host/owner/repo nesting", "github.com/foo/bar", "/t/repos/github.com/foo/bar"},
		{"subgroup path preserved", "gitlab.com/acme/platform/team/utils", "/t/repos/gitlab.com/acme/platform/team/utils"},
		{"host with port", "http://127.0.0.1:5000/fixture/hello.git", "/t/repos/127.0.0.1:5000/fixture/hello"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec, err := repoid.Parse(c.ref)
			if err != nil {
				t.Fatal(err)
			}
			if got := RepoDir("/t", spec); got != c.want {
				t.Errorf("RepoDir(/t, %q) = %q, want %q", c.ref, got, c.want)
			}
		})
	}
}

func TestSubtrees(t *testing.T) {
	if got := ReposRoot("/t"); got != "/t/repos" {
		t.Errorf("ReposRoot = %q", got)
	}
	if got := URLsRoot("/t"); got != "/t/urls" {
		t.Errorf("URLsRoot = %q", got)
	}
	if got := URLFile("/t", "example.com/blog/post.md"); got != "/t/urls/example.com/blog/post.md" {
		t.Errorf("URLFile = %q", got)
	}
	if got := Gitignore("/t"); got != "/t/.gitignore" {
		t.Errorf("Gitignore = %q", got)
	}
}
