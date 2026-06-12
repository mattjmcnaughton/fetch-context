package linkheader

import "testing"

func TestNext(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"empty", "", ""},
		{"no next", `<https://x/page=1>; rel="prev"`, ""},
		{
			"github shape",
			`<https://api.github.com/orgs/o/repos?page=2>; rel="next", <https://api.github.com/orgs/o/repos?page=5>; rel="last"`,
			"https://api.github.com/orgs/o/repos?page=2",
		},
		{
			"next not first",
			`<https://x?page=5>; rel="last", <https://x?page=2>; rel="next"`,
			"https://x?page=2",
		},
		{"unquoted rel", `<https://x?page=2>; rel=next`, "https://x?page=2"},
		{"garbage", `not a link header`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Next(c.header); got != c.want {
				t.Errorf("Next(%q) = %q, want %q", c.header, got, c.want)
			}
		})
	}
}
