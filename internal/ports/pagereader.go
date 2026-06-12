package ports

import "context"

// PageReader fetches a URL rendered as clean markdown (via the jina reader
// proxy in production).
type PageReader interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}
