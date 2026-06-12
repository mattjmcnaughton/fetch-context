// Package forgemock mocks the GitHub and GitLab list-repos endpoints as one
// httptest.Server (R2 in docs/acceptance.md): exact URL paths, query
// parameters, pagination Link headers, JSON shapes, and auth error codes.
// The contract-twin tests in adapters/forge keep this mock honest against
// the real APIs.
package forgemock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Repo is one seeded repo. Path is relative to the org/group (subgroup
// segments allowed for GitLab).
type Repo struct {
	Path     string
	CloneURL string
}

type Server struct {
	httpSrv *httptest.Server

	mu             sync.Mutex
	githubOrgs     map[string][]Repo
	gitlabGroups   map[string][]Repo
	requiredGitHub map[string]string // org → token
	requiredGitLab map[string]string // group → token
	validTokens    map[string]bool
	maxPageSize    int
}

func New() *Server {
	s := &Server{
		githubOrgs:     make(map[string][]Repo),
		gitlabGroups:   make(map[string][]Repo),
		requiredGitHub: make(map[string]string),
		requiredGitLab: make(map[string]string),
		validTokens:    make(map[string]bool),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/orgs/", s.handleGitHub)
	mux.HandleFunc("/groups/", s.handleGitLab)
	s.httpSrv = httptest.NewServer(mux)
	return s
}

func (s *Server) Close()      { s.httpSrv.Close() }
func (s *Server) URL() string { return s.httpSrv.URL }

// SeedGitHubOrg registers a (public) GitHub org and its repos.
func (s *Server) SeedGitHubOrg(org string, repos []Repo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.githubOrgs[org] = repos
}

// SeedGitLabGroup registers a (public) GitLab group and its repos.
func (s *Server) SeedGitLabGroup(group string, repos []Repo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gitlabGroups[group] = repos
}

// RequireGitHubToken makes the org private: unauthenticated requests get
// 401. The token also becomes a recognized credential.
func (s *Server) RequireGitHubToken(org, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requiredGitHub[org] = token
	s.validTokens[token] = true
}

// RequireGitLabToken is the GitLab equivalent of RequireGitHubToken.
func (s *Server) RequireGitLabToken(group, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requiredGitLab[group] = token
	s.validTokens[token] = true
}

// SetMaxPageSize caps the page size regardless of per_page, to force
// pagination with small fixtures (AC-GROUP-03).
func (s *Server) SetMaxPageSize(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maxPageSize = n
}

// handleGitHub implements GET /orgs/{org}/repos.
func (s *Server) handleGitHub(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "orgs" || parts[2] != "repos" {
		http.NotFound(w, r)
		return
	}
	org := parts[1]

	s.mu.Lock()
	repos, known := s.githubOrgs[org]
	required, private := s.requiredGitHub[org]
	s.mu.Unlock()

	token := bearerToken(r)
	if token != "" && !s.tokenValid(token) {
		githubError(w, http.StatusUnauthorized, "Bad credentials")
		return
	}
	if private {
		if token == "" {
			githubError(w, http.StatusUnauthorized, "Requires authentication")
			return
		}
		if token != required {
			githubError(w, http.StatusNotFound, "Not Found") // wrong identity: GitHub hides private orgs
			return
		}
	}
	if !known && !private {
		githubError(w, http.StatusNotFound, "Not Found")
		return
	}

	page, items := s.paginate(w, r, repos)
	type ghRepo struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	}
	out := make([]ghRepo, 0, len(items))
	for _, repo := range items {
		out = append(out, ghRepo{Name: repo.Path, FullName: org + "/" + repo.Path, CloneURL: repo.CloneURL})
	}
	_ = page
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// handleGitLab implements GET /groups/{group}/projects, honoring
// include_subgroups.
func (s *Server) handleGitLab(w http.ResponseWriter, r *http.Request) {
	escaped := strings.TrimPrefix(r.URL.EscapedPath(), "/groups/")
	escaped, ok := strings.CutSuffix(escaped, "/projects")
	if !ok {
		http.NotFound(w, r)
		return
	}
	group, err := url.PathUnescape(escaped)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	s.mu.Lock()
	repos, known := s.gitlabGroups[group]
	required, private := s.requiredGitLab[group]
	s.mu.Unlock()

	token := r.Header.Get("PRIVATE-TOKEN")
	if token != "" && !s.tokenValid(token) {
		gitlabError(w, http.StatusUnauthorized, "401 Unauthorized")
		return
	}
	if private && token != required {
		if token == "" {
			gitlabError(w, http.StatusUnauthorized, "401 Unauthorized")
		} else {
			gitlabError(w, http.StatusNotFound, "404 Group Not Found")
		}
		return
	}
	if !known && !private {
		gitlabError(w, http.StatusNotFound, "404 Group Not Found")
		return
	}

	if r.URL.Query().Get("include_subgroups") != "true" {
		topLevel := repos[:0:0]
		for _, repo := range repos {
			if !strings.Contains(repo.Path, "/") {
				topLevel = append(topLevel, repo)
			}
		}
		repos = topLevel
	}

	_, items := s.paginate(w, r, repos)
	type glProject struct {
		PathWithNamespace string `json:"path_with_namespace"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
	}
	out := make([]glProject, 0, len(items))
	for _, repo := range items {
		out = append(out, glProject{PathWithNamespace: group + "/" + repo.Path, HTTPURLToRepo: repo.CloneURL})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// paginate slices repos per page/per_page and writes the Link header.
func (s *Server) paginate(w http.ResponseWriter, r *http.Request, repos []Repo) (int, []Repo) {
	q := r.URL.Query()
	perPage := intParam(q.Get("per_page"), 30)
	s.mu.Lock()
	if s.maxPageSize > 0 && perPage > s.maxPageSize {
		perPage = s.maxPageSize
	}
	s.mu.Unlock()
	page := intParam(q.Get("page"), 1)

	start := (page - 1) * perPage
	end := start + perPage
	if start > len(repos) {
		start = len(repos)
	}
	if end > len(repos) {
		end = len(repos)
	}

	lastPage := (len(repos) + perPage - 1) / perPage
	if lastPage < 1 {
		lastPage = 1
	}
	var links []string
	pageURL := func(n int) string {
		u := *r.URL
		qq := u.Query()
		qq.Set("page", strconv.Itoa(n))
		u.RawQuery = qq.Encode()
		return s.httpSrv.URL + u.String()
	}
	if page < lastPage {
		links = append(links, fmt.Sprintf(`<%s>; rel="next"`, pageURL(page+1)))
	}
	links = append(links, fmt.Sprintf(`<%s>; rel="last"`, pageURL(lastPage)))
	w.Header().Set("Link", strings.Join(links, ", "))
	return page, repos[start:end]
}

func (s *Server) tokenValid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.validTokens[token]
}

// bearerToken extracts the token from "Authorization: Bearer X" or
// "Authorization: token X".
func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	for _, prefix := range []string{"Bearer ", "token "} {
		if strings.HasPrefix(auth, prefix) {
			return strings.TrimPrefix(auth, prefix)
		}
	}
	return ""
}

func githubError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func gitlabError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func intParam(raw string, def int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	return n
}
