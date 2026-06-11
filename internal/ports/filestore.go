package ports

import "io"

// FileStore is the narrow filesystem surface the core needs. The real
// adapter wraps afero.OsFs; the fake wraps afero.MemMapFs. The core never
// sees afero.
type FileStore interface {
	MkdirAll(path string) error
	// WriteFile writes data, creating missing parent directories.
	WriteFile(path string, data []byte) error
	// OpenForRead opens a file for reading; callers must Close it.
	OpenForRead(path string) (io.ReadCloser, error)
	// Remove removes path and, for directories, everything beneath it.
	Remove(path string) error
	Exists(path string) (bool, error)
	// Walk visits path and its descendants depth-first.
	Walk(root string, fn WalkFunc) error
}

// WalkFunc is called for each path visited by FileStore.Walk.
type WalkFunc func(path string, isDir bool) error
