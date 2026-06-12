// Package gitfixture serves bare fixture repos over HTTP for integration
// and e2e tests, mirroring §1.3 of docs/acceptance.md. It fronts
// `git http-backend` through net/http/cgi, with two control surfaces:
// per-repo fail switches (AC-GROUP-06) and per-repo Basic-auth token gating
// (AC-AUTH-02/03). See ADR-0001 decision 3.
package gitfixture

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Server serves bare repos beneath a temporary root.
type Server struct {
	httpSrv *httptest.Server
	root    string

	mu      sync.Mutex
	failing map[string]bool
	tokens  map[string]string
}

// New starts an empty fixture server. Call Seed/SeedPrivate to add repos
// and Close when done.
func New() (*Server, error) {
	root, err := os.MkdirTemp("", "gitfixture-")
	if err != nil {
		return nil, err
	}
	backendPath, err := gitHTTPBackendPath()
	if err != nil {
		os.RemoveAll(root)
		return nil, err
	}
	s := &Server{
		root:    root,
		failing: make(map[string]bool),
		tokens:  make(map[string]string),
	}
	backend := &cgi.Handler{
		Path: backendPath,
		Env: []string{
			"GIT_PROJECT_ROOT=" + root,
			"GIT_HTTP_EXPORT_ALL=1",
		},
	}
	s.httpSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := repoName(r.URL.Path)
		s.mu.Lock()
		failing := s.failing[name]
		token, private := s.tokens[name]
		s.mu.Unlock()

		if failing {
			http.NotFound(w, r)
			return
		}
		if private {
			_, pass, ok := r.BasicAuth()
			if !ok || pass != token {
				w.Header().Set("WWW-Authenticate", `Basic realm="gitfixture"`)
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
		}
		backend.ServeHTTP(w, r)
	}))
	return s, nil
}

// Close stops the server and removes the seeded repos.
func (s *Server) Close() {
	s.httpSrv.Close()
	os.RemoveAll(s.root)
}

// URL is the server base, e.g. http://127.0.0.1:PORT.
func (s *Server) URL() string { return s.httpSrv.URL }

// Host is the host:port the server listens on — the host segment clones
// land under (per §1.3, paths are asserted against the fixture host).
func (s *Server) Host() string {
	return strings.TrimPrefix(s.httpSrv.URL, "http://")
}

// CloneURL is the clonable URL for a seeded repo name like "fixture/hello".
func (s *Server) CloneURL(name string) string {
	return s.httpSrv.URL + "/" + name + ".git"
}

// SetFailing makes every request for the named repo return 404 (the
// AC-GROUP-06 "git server fails clones of beta" switch).
func (s *Server) SetFailing(name string, failing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failing[name] = failing
}

// Seed creates a public bare repo (default branch main) containing files.
func (s *Server) Seed(name string, files map[string]string) error {
	return s.seed(name, files)
}

// SeedPrivate creates a token-gated bare repo: requests must carry HTTP
// Basic auth whose password equals token (username is ignored).
func (s *Server) SeedPrivate(name, token string, files map[string]string) error {
	if err := s.seed(name, files); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[name] = token
	return nil
}

// Commit adds (or updates) files in an existing repo with a new commit, so
// tests can observe refresh-to-latest behavior.
func (s *Server) Commit(name string, files map[string]string) error {
	work, err := os.MkdirTemp("", "gitfixture-work-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	bare := filepath.Join(s.root, name+".git")
	if err := runGit(work, "clone", "-q", bare, "."); err != nil {
		return err
	}
	return commitAndPush(work, bare, files)
}

// CommitOnBranch adds (or updates) files in an existing repo with a new
// commit on the named branch, creating the branch from the repo's current
// default tip when absent — so tests can exercise branch-pinned clones.
func (s *Server) CommitOnBranch(name, branch string, files map[string]string) error {
	work, err := os.MkdirTemp("", "gitfixture-work-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	bare := filepath.Join(s.root, name+".git")
	if err := runGit(work, "clone", "-q", bare, "."); err != nil {
		return err
	}
	if err := runGit(work, "checkout", "-q", "-B", branch); err != nil {
		return err
	}
	for rel, content := range files {
		path := filepath.Join(work, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if err := runGit(work, "add", "-A"); err != nil {
		return err
	}
	if err := runGit(work, "commit", "-q", "-m", "fixture commit", "--allow-empty"); err != nil {
		return err
	}
	return runGit(work, "push", "-q", bare, branch+":"+branch)
}

// seed builds the bare repo at root/<name>.git from a throwaway worktree.
func (s *Server) seed(name string, files map[string]string) error {
	bare := filepath.Join(s.root, name+".git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		return err
	}
	if err := runGit(bare, "init", "-q", "--bare", "-b", "main"); err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "gitfixture-work-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	if err := runGit(work, "init", "-q", "-b", "main"); err != nil {
		return err
	}
	return commitAndPush(work, bare, files)
}

// commitAndPush writes files in work, commits, and pushes main to bare.
func commitAndPush(work, bare string, files map[string]string) error {
	for rel, content := range files {
		path := filepath.Join(work, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if err := runGit(work, "add", "-A"); err != nil {
		return err
	}
	if err := runGit(work, "commit", "-q", "-m", "fixture commit", "--allow-empty"); err != nil {
		return err
	}
	return runGit(work, "push", "-q", bare, "main:main")
}

// runGit runs git in dir with a hermetic identity and no global config.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git",
		append([]string{
			"-c", "user.name=fixture",
			"-c", "user.email=fixture@example.test",
			"-c", "init.defaultBranch=main",
		}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %v in %s: %w\n%s", args, dir, err, out)
	}
	return nil
}

// BasicCredential encodes an HTTP Basic credential. The fixture's auth gate
// ignores the username and matches the password against the repo's token.
func BasicCredential(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

// repoName extracts "owner/repo" from a request path like
// "/owner/repo.git/info/refs".
func repoName(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	if i := strings.Index(trimmed, ".git"); i >= 0 {
		return trimmed[:i]
	}
	return trimmed
}

// gitHTTPBackendPath locates the git-http-backend CGI binary.
func gitHTTPBackendPath() (string, error) {
	out, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		return "", fmt.Errorf("git --exec-path: %w", err)
	}
	p := filepath.Join(strings.TrimSpace(string(out)), "git-http-backend")
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("git-http-backend not found at %s: %w", p, err)
	}
	return p, nil
}
