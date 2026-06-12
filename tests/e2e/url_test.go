//go:build e2e

package e2e

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestAC_URL_01_FetchToMarkdownAtHostPath(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("url", "http://example.test/blog/post")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	b, err := os.ReadFile(w.target("urls", "example.test", "blog", "post.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 || !strings.Contains(string(b), "ACCEPTANCE-MARKER") {
		t.Errorf("page content = %q, want non-empty with marker", b)
	}
}

func TestAC_URL_02_RootURLToIndexMD(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("url", "http://example.test")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	b, err := os.ReadFile(w.target("urls", "example.test", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Error("index.md is empty")
	}
}

func TestAC_URL_03_RefetchOverwrites(t *testing.T) {
	w := newWorkspace(t)
	if res := w.run("url", "http://example.test/blog/post"); res.code != 0 {
		t.Fatalf("first fetch: exit = %d, stderr: %s", res.code, res.stderr)
	}
	page := w.target("urls", "example.test", "blog", "post.md")
	writeFile(t, page, "STALE")

	res := w.run("url", "http://example.test/blog/post")
	if res.code != 0 {
		t.Fatalf("re-fetch: exit = %d, stderr: %s", res.code, res.stderr)
	}
	b, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "STALE") || !strings.Contains(string(b), "ACCEPTANCE-MARKER") {
		t.Errorf("content = %q, want STALE overwritten by fresh fetch", b)
	}
}

func TestAC_URL_04_ProxyWrapsLiterally(t *testing.T) {
	w := newWorkspace(t)
	origin := "http://example.test/x"
	res := w.run("url", origin)
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if got := reader.LastRequest(); got != "/"+origin {
		t.Errorf("outbound request URI = %q, want %q (origin appended literally, not percent-encoded)", got, "/"+origin)
	}
}

func TestAC_URL_05_MultipleURLsOneInvocation(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("url", "http://example.test/blog/post", "http://example.test")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !exists(w.target("urls", "example.test", "blog", "post.md")) {
		t.Error("page file missing")
	}
	if !exists(w.target("urls", "example.test", "index.md")) {
		t.Error("index file missing")
	}
}

func TestAC_URL_06_QueryHashSuffix(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("url",
		"http://example.test/blog/post",
		"http://example.test/blog/post?x=1",
		"http://example.test/blog/post?x=2",
	)
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	hash := func(q string) string {
		sum := sha256.Sum256([]byte(q))
		return hex.EncodeToString(sum[:])[:8]
	}
	for _, file := range []string{
		"post.md",
		"post__" + hash("x=1") + ".md",
		"post__" + hash("x=2") + ".md",
	} {
		if !exists(w.target("urls", "example.test", "blog", file)) {
			t.Errorf("expected file missing: %s", file)
		}
	}
	entries, err := os.ReadDir(w.target("urls", "example.test", "blog"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("got %d files, want 3 distinct (no silent overwrites)", len(entries))
	}
}

func TestAC_URL_07_TrailingSlashCollapses(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("url", "http://example.test/blog", "http://example.test/blog/")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	entries, err := os.ReadDir(w.target("urls", "example.test"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "blog.md" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("entries = %v, want exactly [blog.md]", names)
	}
}
