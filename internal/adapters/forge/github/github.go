// Package github enumerates GitHub orgs via the REST API. GitHub orgs are
// flat: /orgs/{org}/repos lists everything, paginated via Link headers.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/forge/linkheader"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// DefaultBase is the production API base URL.
const DefaultBase = "https://api.github.com"

type Enumerator struct {
	base   string
	token  string
	client *http.Client
	log    *slog.Logger
}

// New builds an enumerator. The token is a constructor argument (read from
// the environment in wiring); the core never sees it.
func New(base, token string, log *slog.Logger) *Enumerator {
	if base == "" {
		base = DefaultBase
	}
	return &Enumerator{base: strings.TrimSuffix(base, "/"), token: token, client: http.DefaultClient, log: log}
}

func (e *Enumerator) Enumerate(ctx context.Context, slug string) ([]ports.GroupRepo, error) {
	next := fmt.Sprintf("%s/orgs/%s/repos?per_page=100", e.base, url.PathEscape(slug))
	var out []ports.GroupRepo
	for next != "" {
		repos, nextURL, err := e.fetchPage(ctx, slug, next)
		if err != nil {
			return nil, err
		}
		out = append(out, repos...)
		next = nextURL
	}
	return out, nil
}

func (e *Enumerator) fetchPage(ctx context.Context, slug, pageURL string) ([]ports.GroupRepo, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("enumerating GitHub org %q: %w", slug, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", statusError("GitHub", "org", slug, "GITHUB_TOKEN", resp)
	}

	var items []struct {
		Name     string `json:"name"`
		CloneURL string `json:"clone_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, "", fmt.Errorf("enumerating GitHub org %q: decoding response: %w", slug, err)
	}
	repos := make([]ports.GroupRepo, 0, len(items))
	for _, item := range items {
		repos = append(repos, ports.GroupRepo{Path: item.Name, CloneURL: item.CloneURL})
	}
	return repos, linkheader.Next(resp.Header.Get("Link")), nil
}

// statusError renders a non-200 as an explicit error; auth-shaped statuses
// say so (AC-GROUP-05: never silently skipped).
func statusError(forge, kind, slug, tokenVar string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	hint := ""
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		hint = fmt.Sprintf(" — authentication/permission problem? Private %ss require %s with access", kind, tokenVar)
	}
	return fmt.Errorf("enumerating %s %s %q: API returned %s%s (%s)",
		forge, kind, slug, resp.Status, hint, strings.TrimSpace(string(body)))
}

var _ ports.ForgeEnumerator = (*Enumerator)(nil)
