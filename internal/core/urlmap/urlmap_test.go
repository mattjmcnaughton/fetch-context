package urlmap

import (
	"strings"
	"testing"
)

func TestMap(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"path", "http://example.test/blog/post", "example.test/blog/post.md"},
		{"https path", "https://docs.example.com/changelog", "docs.example.com/changelog.md"},
		{"nested path", "http://example.test/a/b/c", "example.test/a/b/c.md"},
		{"root no slash", "http://example.test", "example.test/index.md"},
		{"root with slash", "http://example.test/", "example.test/index.md"},
		{"trailing slash stripped", "http://example.test/blog/", "example.test/blog.md"},
		{"trailing slash equals clean", "http://example.test/blog", "example.test/blog.md"},
		{"host with port", "http://127.0.0.1:8080/x", "127.0.0.1:8080/x.md"},
		{"dots kept in segments", "http://example.test/docs/v1.2/intro", "example.test/docs/v1.2/intro.md"},
		{"percent-decoded then sanitized", "http://example.test/a%20b/c", "example.test/a_b/c.md"},
		{"unsafe chars sanitized", "http://example.test/a*b<c>", "example.test/a_b_c_.md"},
		{"dot-dot segment neutralized", "http://example.test/%2e%2e/etc/passwd", "example.test/_/etc/passwd.md"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Map(c.in)
			if err != nil {
				t.Fatalf("Map(%q) error: %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("Map(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestMapQueryHashSuffix(t *testing.T) {
	clean, err := Map("http://example.test/blog/post")
	if err != nil {
		t.Fatal(err)
	}
	q1a, err := Map("http://example.test/blog/post?x=1")
	if err != nil {
		t.Fatal(err)
	}
	q1b, err := Map("http://example.test/blog/post?x=1")
	if err != nil {
		t.Fatal(err)
	}
	q2, err := Map("http://example.test/blog/post?x=2")
	if err != nil {
		t.Fatal(err)
	}

	if clean != "example.test/blog/post.md" {
		t.Errorf("clean path affected by query support: %q", clean)
	}
	if q1a != q1b {
		t.Errorf("hash not deterministic: %q vs %q", q1a, q1b)
	}
	if q1a == q2 || q1a == clean || q2 == clean {
		t.Errorf("query variants must be distinct: clean=%q q1=%q q2=%q", clean, q1a, q2)
	}
	wantPrefix := "example.test/blog/post__"
	if !strings.HasPrefix(q1a, wantPrefix) || !strings.HasSuffix(q1a, ".md") {
		t.Errorf("q1 = %q, want %s<8-hex>.md", q1a, wantPrefix)
	}
	hash := strings.TrimSuffix(strings.TrimPrefix(q1a, wantPrefix), ".md")
	if len(hash) != 8 {
		t.Errorf("hash suffix = %q, want 8 hex chars", hash)
	}
	for _, r := range hash {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Errorf("hash %q contains non-hex %q", hash, r)
		}
	}
}

func TestMapInvalid(t *testing.T) {
	for _, in := range []string{"", "not-a-url", "ftp://example.test/x", "http://"} {
		t.Run(in, func(t *testing.T) {
			if got, err := Map(in); err == nil {
				t.Errorf("Map(%q) = %q, want error", in, got)
			}
		})
	}
}
