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

func TestLoadCloneDefaultsAppliedWhenAbsent(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, "target: .agentic/ctx\n")
	cfg, err := New(home).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Clone.Depth != 1 || cfg.Clone.Parallel != 4 {
		t.Errorf("Clone = %+v, want defaults {Depth:1 Parallel:4}", cfg.Clone)
	}
}

func TestLoadCloneSection(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, "clone:\n  depth: 0\n  parallel: 2\n")
	cfg, err := New(home).Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Clone.Depth != 0 || cfg.Clone.Parallel != 2 {
		t.Errorf("Clone = %+v, want {Depth:0 Parallel:2} (explicit depth 0 = full history)", cfg.Clone)
	}
}

func TestLoadCloneSectionRejectsInvalidValues(t *testing.T) {
	for name, content := range map[string]string{
		"negative depth":    "clone:\n  depth: -1\n",
		"zero parallel":     "clone:\n  parallel: 0\n",
		"negative parallel": "clone:\n  parallel: -3\n",
	} {
		home := t.TempDir()
		writeConfig(t, home, content)
		if _, err := New(home).Load(); err == nil {
			t.Errorf("%s: want loud config error", name)
		}
	}
}

func TestLoadRepoEntryForms(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, `
profiles:
  mixed:
    repos:
      - github.com/foo/scalar
      - ref: github.com/foo/full
        depth: 0
        branch: develop
      - ref: github.com/foo/plain-mapping
`)
	cfg, err := New(home).Load()
	if err != nil {
		t.Fatal(err)
	}
	repos := cfg.Profiles["mixed"].Repos
	if len(repos) != 3 {
		t.Fatalf("repos = %+v, want 3 entries", repos)
	}
	if repos[0].Ref != "github.com/foo/scalar" || repos[0].Depth != nil || repos[0].Branch != "" {
		t.Errorf("scalar entry = %+v", repos[0])
	}
	if repos[1].Ref != "github.com/foo/full" || repos[1].Depth == nil || *repos[1].Depth != 0 || repos[1].Branch != "develop" {
		t.Errorf("mapping entry = %+v, want depth 0 (explicit) and branch develop", repos[1])
	}
	if repos[2].Ref != "github.com/foo/plain-mapping" || repos[2].Depth != nil {
		t.Errorf("ref-only mapping = %+v, want nil depth (inherit)", repos[2])
	}
}

func TestLoadRepoEntryUnknownFieldRejectedWithLine(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, `profiles:
  p:
    repos:
      - ref: a/b
        brnch: oops
`)
	_, err := New(home).Load()
	if err == nil {
		t.Fatal("unknown repo-entry field must be rejected loudly")
	}
	if !strings.Contains(err.Error(), "brnch") || !strings.Contains(err.Error(), "line 5") {
		t.Errorf("error %q should name the unknown field and its line", err)
	}
}

func TestLoadRepoEntryRequiresRef(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, "profiles:\n  p:\n    repos:\n      - depth: 1\n")
	if _, err := New(home).Load(); err == nil {
		t.Fatal("mapping repo entry without ref must be rejected")
	}
}

func TestLoadRepoEntryRejectsNegativeDepth(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, "profiles:\n  p:\n    repos:\n      - ref: a/b\n        depth: -2\n")
	if _, err := New(home).Load(); err == nil {
		t.Fatal("negative per-repo depth must be rejected")
	}
}

func TestLoadRepoEntryRejectsNonScalarNonMapping(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, "profiles:\n  p:\n    repos:\n      - [a/b]\n")
	if _, err := New(home).Load(); err == nil {
		t.Fatal("sequence repo entry must be rejected")
	}
}
