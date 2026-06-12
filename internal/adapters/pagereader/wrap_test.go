package pagereader

import "testing"

// The wrapping rule is the one genuinely tricky pure helper in this adapter
// (see docs/testing.md): the origin URL is appended literally, never
// percent-encoded.
func TestWrapURL(t *testing.T) {
	cases := []struct {
		name   string
		base   string
		origin string
		want   string
	}{
		{"plain", "https://r.jina.ai", "https://example.com/blog/post", "https://r.jina.ai/https://example.com/blog/post"},
		{"base trailing slash collapsed", "https://r.jina.ai/", "http://example.test/x", "https://r.jina.ai/http://example.test/x"},
		{"query survives literally", "http://127.0.0.1:9", "http://e.test/p?x=1&y=2", "http://127.0.0.1:9/http://e.test/p?x=1&y=2"},
		{"no percent-encoding", "http://b", "http://e.test/a b", "http://b/http://e.test/a b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := wrapURL(c.base, c.origin); got != c.want {
				t.Errorf("wrapURL(%q, %q) = %q, want %q", c.base, c.origin, got, c.want)
			}
		})
	}
}
