// Package pagereader fetches pages through the jina reader proxy. It
// implements ports.PageReader.
package pagereader

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// DefaultBase is the production reader proxy.
const DefaultBase = "https://r.jina.ai"

type Reader struct {
	base   string
	client *http.Client
	log    *slog.Logger
}

func New(base string, log *slog.Logger) *Reader {
	if base == "" {
		base = DefaultBase
	}
	return &Reader{base: base, client: http.DefaultClient, log: log}
}

// Fetch GETs the wrapped origin URL and returns the markdown body.
func (r *Reader) Fetch(ctx context.Context, origin string) ([]byte, error) {
	wrapped := wrapURL(r.base, origin)
	r.log.Debug("fetching page", "url", wrapped)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wrapped, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", origin, err)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", origin, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: reader returned %s", origin, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading page body for %s: %w", origin, err)
	}
	return body, nil
}

// wrapURL appends the origin URL literally to the reader base — no
// percent-encoding (AC-URL-04): <base>/<origin>.
func wrapURL(base, origin string) string {
	return strings.TrimSuffix(base, "/") + "/" + origin
}

var _ ports.PageReader = (*Reader)(nil)
