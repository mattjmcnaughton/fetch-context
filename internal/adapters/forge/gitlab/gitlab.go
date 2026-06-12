// Package gitlab enumerates GitLab groups via the REST API. Groups are
// recursive: /groups/{id}/projects?include_subgroups=true walks the group
// and all subgroups; the subgroup path is preserved in each project's
// path_with_namespace.
package gitlab

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
const DefaultBase = "https://gitlab.com/api/v4"

type Enumerator struct {
	base   string
	token  string
	client *http.Client
	log    *slog.Logger
}

func New(base, token string, log *slog.Logger) *Enumerator {
	if base == "" {
		base = DefaultBase
	}
	return &Enumerator{base: strings.TrimSuffix(base, "/"), token: token, client: http.DefaultClient, log: log}
}

func (e *Enumerator) Enumerate(ctx context.Context, slug string) ([]ports.GroupRepo, error) {
	next := fmt.Sprintf("%s/groups/%s/projects?per_page=100&include_subgroups=true",
		e.base, url.PathEscape(slug))
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
	if e.token != "" {
		req.Header.Set("PRIVATE-TOKEN", e.token)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("enumerating GitLab group %q: %w", slug, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", statusError(slug, resp)
	}

	var items []struct {
		PathWithNamespace string `json:"path_with_namespace"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, "", fmt.Errorf("enumerating GitLab group %q: decoding response: %w", slug, err)
	}
	repos := make([]ports.GroupRepo, 0, len(items))
	for _, item := range items {
		repos = append(repos, ports.GroupRepo{
			Path:     relativePath(slug, item.PathWithNamespace),
			CloneURL: item.HTTPURLToRepo,
		})
	}
	return repos, linkheader.Next(resp.Header.Get("Link")), nil
}

// relativePath strips the group slug from a project's path_with_namespace,
// keeping subgroup segments ("acme/sub/nested" under "acme" → "sub/nested").
func relativePath(slug, pathWithNamespace string) string {
	if rel, ok := strings.CutPrefix(pathWithNamespace, slug+"/"); ok {
		return rel
	}
	return pathWithNamespace
}

func statusError(slug string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	hint := ""
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		hint = " — authentication/permission problem? Private groups require GITLAB_TOKEN with access"
	}
	return fmt.Errorf("enumerating GitLab group %q: API returned %s%s (%s)",
		slug, resp.Status, hint, strings.TrimSpace(string(body)))
}

var _ ports.ForgeEnumerator = (*Enumerator)(nil)
