//go:build integration

package filestore

import (
	"io"
	"path/filepath"
	"sort"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New()

	sub := filepath.Join(dir, "a", "b")
	if err := s.MkdirAll(sub); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(sub, "f.txt")
	if err := s.WriteFile(file, []byte("hello")); err != nil {
		t.Fatal(err)
	}

	exists, err := s.Exists(file)
	if err != nil || !exists {
		t.Fatalf("Exists(%s) = %v, %v; want true", file, exists, err)
	}

	r, err := s.OpenForRead(file)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	r.Close()
	if err != nil || string(b) != "hello" {
		t.Fatalf("read = %q, %v", b, err)
	}
}

func TestWriteFileCreatesParents(t *testing.T) {
	dir := t.TempDir()
	s := New()
	file := filepath.Join(dir, "x", "y", "z.md")
	if err := s.WriteFile(file, []byte("m")); err != nil {
		t.Fatalf("WriteFile should create parent dirs: %v", err)
	}
}

func TestRemoveIsRecursive(t *testing.T) {
	dir := t.TempDir()
	s := New()
	if err := s.WriteFile(filepath.Join(dir, "tree", "deep", "f"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(filepath.Join(dir, "tree")); err != nil {
		t.Fatal(err)
	}
	exists, _ := s.Exists(filepath.Join(dir, "tree"))
	if exists {
		t.Error("tree still exists after Remove")
	}
	// Removing a missing path is not an error (idempotent cleanup).
	if err := s.Remove(filepath.Join(dir, "tree")); err != nil {
		t.Errorf("Remove of missing path: %v", err)
	}
}

func TestWalk(t *testing.T) {
	dir := t.TempDir()
	s := New()
	for _, rel := range []string{"a/f1", "a/b/f2", "f3"} {
		if err := s.WriteFile(filepath.Join(dir, rel), []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	var files []string
	err := s.Walk(dir, func(path string, isDir bool) error {
		if !isDir {
			rel, _ := filepath.Rel(dir, path)
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	want := []string{"a/b/f2", "a/f1", "f3"}
	if len(files) != 3 || files[0] != want[0] || files[1] != want[1] || files[2] != want[2] {
		t.Errorf("Walk files = %v, want %v", files, want)
	}
}
