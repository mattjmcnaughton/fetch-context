package repoid

import "testing"

func TestParseGroup(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantHost string
		wantSlug string
	}{
		{"github org", "github.com/my-org", "github.com", "my-org"},
		{"gitlab nested group", "gitlab.com/acme/platform", "gitlab.com", "acme/platform"},
		{"bare slug defaults to github.com", "my-org", "github.com", "my-org"},
		{"trailing slash stripped", "github.com/my-org/", "github.com", "my-org"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseGroup(c.in)
			if err != nil {
				t.Fatalf("ParseGroup(%q) error: %v", c.in, err)
			}
			if got.Host != c.wantHost || got.Slug != c.wantSlug {
				t.Errorf("ParseGroup(%q) = %+v, want host %q slug %q", c.in, got, c.wantHost, c.wantSlug)
			}
		})
	}
}

func TestParseGroupInvalid(t *testing.T) {
	for _, in := range []string{"", "   ", "github.com", "github.com/"} {
		t.Run(in, func(t *testing.T) {
			if got, err := ParseGroup(in); err == nil {
				t.Errorf("ParseGroup(%q) = %+v, want error", in, got)
			}
		})
	}
}
