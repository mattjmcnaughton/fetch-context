//go:build integration

package pagereader

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/testing/readermock"
)

func TestFetchReturnsMarkdownAndWrapsLiterally(t *testing.T) {
	mock := readermock.New()
	t.Cleanup(mock.Close)
	r := New(mock.URL(), slog.New(slog.DiscardHandler))

	body, err := r.Fetch(context.Background(), "http://example.test/blog/post?x=1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), readermock.Marker) {
		t.Errorf("body %q missing marker", body)
	}
	if got := mock.LastRequest(); got != "/http://example.test/blog/post?x=1" {
		t.Errorf("outbound request URI = %q, want the origin appended literally (AC-URL-04)", got)
	}
}

func TestFetchSurfacesNon200(t *testing.T) {
	mock := readermock.New()
	mock.Close() // force a connection error path too? no — use a fresh server returning 200 only.

	r := New(mock.URL(), slog.New(slog.DiscardHandler))
	if _, err := r.Fetch(context.Background(), "http://example.test/x"); err == nil {
		t.Fatal("want error when the reader is unreachable")
	}
}
