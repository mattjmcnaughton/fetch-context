// Package filestore implements ports.FileStore over afero.OsFs. The core
// sees only the narrow port, never afero.
package filestore

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

type Store struct {
	fs afero.Fs
}

func New() *Store { return &Store{fs: afero.NewOsFs()} }

func (s *Store) MkdirAll(path string) error {
	return s.fs.MkdirAll(path, 0o755)
}

// WriteFile writes data, creating missing parent directories.
func (s *Store) WriteFile(path string, data []byte) error {
	if err := s.fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return afero.WriteFile(s.fs, path, data, 0o644)
}

func (s *Store) OpenForRead(path string) (io.ReadCloser, error) {
	return s.fs.Open(path)
}

// Remove removes path recursively; a missing path is not an error.
func (s *Store) Remove(path string) error {
	return s.fs.RemoveAll(path)
}

func (s *Store) Exists(path string) (bool, error) {
	return afero.Exists(s.fs, path)
}

func (s *Store) Walk(root string, fn ports.WalkFunc) error {
	return afero.Walk(s.fs, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return fn(path, info.IsDir())
	})
}

var _ ports.FileStore = (*Store)(nil)
