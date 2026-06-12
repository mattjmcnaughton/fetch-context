package materialize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
	"github.com/mattjmcnaughton/fetch-context/internal/core/urlmap"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// URLRequest is one `url` invocation.
type URLRequest struct {
	URLs   []string
	Target string
}

// URL is the MaterializeURL use case: fetch pages as markdown and write
// them beneath urls/ (overwriting on re-fetch, AC-URL-03).
type URL struct {
	reader  ports.PageReader
	fs      ports.FileStore
	locator ports.HostRepoLocator
	log     *slog.Logger
}

func NewURL(reader ports.PageReader, fs ports.FileStore, locator ports.HostRepoLocator, log *slog.Logger) *URL {
	return &URL{reader: reader, fs: fs, locator: locator, log: log}
}

func (m *URL) Materialize(ctx context.Context, req URLRequest) error {
	root, err := m.locator.RepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}
	targetAbs := targetpath.Resolve(root, req.Target)

	var failures []ItemError
	prepared := false
	for _, raw := range req.URLs {
		mapped, err := urlmap.Map(raw)
		if err != nil {
			failures = append(failures, ItemError{Ref: raw, Err: err})
			continue
		}
		if !prepared {
			if err := ensureTarget(m.fs, targetAbs); err != nil {
				return fmt.Errorf("preparing target %s: %w", targetAbs, err)
			}
			prepared = true
		}
		if err := m.materializeOne(ctx, raw, targetpath.URLFile(targetAbs, mapped)); err != nil {
			m.log.Warn("url materialization failed", "url", raw, "error", err)
			failures = append(failures, ItemError{Ref: raw, Err: err})
		}
	}
	return errorOrNil(failures)
}

func (m *URL) materializeOne(ctx context.Context, raw, dest string) error {
	page, err := m.reader.Fetch(ctx, raw)
	if err != nil {
		return err
	}
	m.log.Debug("writing fetched page", "url", raw, "dest", dest)
	return m.fs.WriteFile(dest, page)
}
