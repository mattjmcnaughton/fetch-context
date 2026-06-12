//go:build integration

package configstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, home, content string) string {
	t.Helper()
	path := filepath.Join(home, ".config", "fetch-context", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathUnderConfigHome(t *testing.T) {
	s := New("/sandbox")
	want := "/sandbox/.config/fetch-context/config.yaml"
	if s.Path() != want {
		t.Errorf("Path() = %q, want %q (AC-CONFIG-01)", s.Path(), want)
	}
}

func TestLoadMissingFileYieldsDefaults(t *testing.T) {
	cfg, err := New(t.TempDir()).Load()
	if err != nil {
		t.Fatalf("missing config must not be fatal (AC-CONFIG-04): %v", err)
	}
	if cfg.Target != ".agentic/sources" {
		t.Errorf("Target = %q, want default", cfg.Target)
	}
	if len(cfg.Profiles) != 0 {
		t.Errorf("Profiles = %v, want empty", cfg.Profiles)
	}
}

func TestLoadEmptyFileYieldsDefaults(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, "")
	cfg, err := New(home).Load()
	if err != nil {
		t.Fatalf("empty config must not be fatal: %v", err)
	}
	if cfg.Target != ".agentic/sources" {
		t.Errorf("Target = %q", cfg.Target)
	}
}

func TestLoadFullConfig(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, `
target: .agentic/ctx
profiles:
  backend:
    target: .agentic/backend
    repos:
      - github.com/redis/redis
    groups:
      - gitlab.com/acme/platform
    urls:
      - https://example.com/blog/post
  web-stack:
    repos:
      - github.com/foo/bar
      - github.com/foo/baz
`)
	cfg, err := New(home).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Target != ".agentic/ctx" {
		t.Errorf("global target = %q (AC-CONFIG-02 input)", cfg.Target)
	}
	backend := cfg.Profiles["backend"]
	if backend.Target != ".agentic/backend" {
		t.Errorf("backend target = %q", backend.Target)
	}
	if len(backend.Repos) != 1 || len(backend.Groups) != 1 || len(backend.URLs) != 1 {
		t.Errorf("backend = %+v", backend)
	}
	if web := cfg.Profiles["web-stack"]; len(web.Repos) != 2 || web.Target != "" {
		t.Errorf("web-stack = %+v", web)
	}
}

func TestLoadMalformedYAMLErrorsPrecisely(t *testing.T) {
	home := t.TempDir()
	path := writeConfig(t, home, "profiles:\n  backend: [unclosed\n")
	_, err := New(home).Load()
	if err == nil {
		t.Fatal("want parse error (AC-CONFIG-03)")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "yaml") {
		t.Errorf("error %q should name the file and the YAML problem", err)
	}
}

func TestLoadUnknownFieldRejected(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, "profiles:\n  backend:\n    repositories: [a/b]\n")
	if _, err := New(home).Load(); err == nil {
		t.Fatal("unknown field must be rejected loudly, not ignored")
	}
}
