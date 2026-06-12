//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

func TestAC_LIST_01_ShowsProfilesAndContents(t *testing.T) {
	w := newWorkspace(t)
	w.writeConfig(t, `
profiles:
  backend:
    repos:
      - github.com/redis/redis
    groups:
      - gitlab.com/acme
    urls:
      - "https://example.com/blog/some-post"
  web-stack:
    repos:
      - github.com/foo/bar
`)
	res := w.run("list")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	for _, want := range []string{
		"backend", "web-stack",
		"github.com/redis/redis", "gitlab.com/acme",
		"https://example.com/blog/some-post", "github.com/foo/bar",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Errorf("stdout missing %q:\n%s", want, res.stdout)
		}
	}
}

func TestAC_LIST_02_ReportsMaterializedState(t *testing.T) {
	w := newWorkspace(t)
	hello := fixture.CloneURL("fixture/hello")
	other := fixture.CloneURL("fixture/other")
	w.writeConfig(t, `
profiles:
  backend:
    repos:
      - "`+hello+`"
    urls:
      - "http://example.test/blog/post"
  untouched:
    repos:
      - "`+other+`"
`)
	if res := w.run("load", "backend"); res.code != 0 {
		t.Fatalf("load: exit = %d, stderr: %s", res.code, res.stderr)
	}

	res := w.run("list")
	if res.code != 0 {
		t.Fatalf("list: exit = %d, stderr: %s", res.code, res.stderr)
	}
	requireLineWith(t, res.stdout, hello, "[materialized]")
	requireLineWith(t, res.stdout, "http://example.test/blog/post", "[materialized]")
	requireLineWith(t, res.stdout, other, "[absent]")
}

func TestAC_LIST_03_EmptyConfigIsFriendly(t *testing.T) {
	w := newWorkspace(t) // no config file at all
	res := w.run("list")
	if res.code != 0 {
		t.Fatalf("exit = %d, want 0 for empty config; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "no profiles") {
		t.Errorf("stdout should clearly say there are no profiles:\n%s", res.stdout)
	}
}

// requireLineWith asserts that the line mentioning needle also carries
// marker.
func requireLineWith(t *testing.T, out, needle, marker string) {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, needle) {
			if !strings.Contains(line, marker) {
				t.Errorf("line %q should carry %q", line, marker)
			}
			return
		}
	}
	t.Errorf("no line mentions %q:\n%s", needle, out)
}
