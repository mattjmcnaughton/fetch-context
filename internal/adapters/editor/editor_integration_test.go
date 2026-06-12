//go:build integration

package editor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/envx"
)

func writeScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-editor.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEditRunsRealEditorScript(t *testing.T) {
	script := writeScript(t, `echo "edited-by-script" >> "$1"`)
	target := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(target, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New(envx.Fake{"EDITOR": script})
	if err := e.Edit(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "original") || !strings.Contains(string(b), "edited-by-script") {
		t.Errorf("file content = %q", b)
	}
}

func TestEditSurfacesEditorFailure(t *testing.T) {
	script := writeScript(t, "exit 3")
	e := New(envx.Fake{"EDITOR": script})
	if err := e.Edit(context.Background(), filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal("want error when the editor exits non-zero")
	}
}
