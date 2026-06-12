//go:build contract

package pagereader

import (
	"context"
	"log/slog"
	"testing"
)

// Shape/protocol contract against the real reader proxy: the literal
// wrapped-URL form (<base>/<origin>) is accepted and returns non-empty
// markdown. Run via `just test-contract`.
func TestPageReaderContractAgainstRealJina(t *testing.T) {
	r := New(DefaultBase, slog.New(slog.DiscardHandler))
	body, err := r.Fetch(context.Background(), "https://example.com/")
	if err != nil {
		t.Fatalf("fetch through %s failed: %v", DefaultBase, err)
	}
	if len(body) == 0 {
		t.Fatal("reader returned an empty body")
	}
}
