// Package core holds pure use cases and domain services. This test
// mechanically enforces the purity rule from docs/architecture.md: nothing
// under internal/core/ may import infrastructure.
package core

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/mattjmcnaughton/fetch-context"

// TestCorePurity asserts that no non-test file under internal/core/ imports
// internal/adapters/..., os, net/http, os/exec, or any third-party package.
// Allowed: the rest of the stdlib, internal/ports, and internal/core itself.
func TestCorePurity(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if reason := forbidden(p); reason != "" {
				t.Errorf("internal/core/%s imports %q: %s", path, p, reason)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// forbidden returns a non-empty reason if the import path is not allowed in
// the core.
func forbidden(p string) string {
	if p == modulePath || strings.HasPrefix(p, modulePath+"/") {
		rest := strings.TrimPrefix(p, modulePath+"/")
		if strings.HasPrefix(rest, "internal/ports") || strings.HasPrefix(rest, "internal/core") {
			return ""
		}
		return "core may import only internal/ports and internal/core from this module"
	}
	first, _, _ := strings.Cut(p, "/")
	if strings.Contains(first, ".") {
		return "third-party packages are not allowed in the core"
	}
	switch {
	case p == "os":
		return "the core must not touch the OS directly; use a port"
	case p == "os/exec" || strings.HasPrefix(p, "os/exec/"):
		return "the core must not exec processes; use a port"
	case p == "net/http" || strings.HasPrefix(p, "net/http/"):
		return "the core must not speak HTTP; use a port"
	}
	return ""
}
