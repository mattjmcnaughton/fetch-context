package materialize

import (
	"io"

	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// gitignoreContent is the exact content of the auto-written ignore file:
// the whole target tree is ignored (AC-REPO-03).
const gitignoreContent = "*"

// ensureTarget creates the target root and writes its .gitignore
// idempotently: an existing file with the exact expected content is left
// untouched (AC-LAYOUT-03).
func ensureTarget(fs ports.FileStore, targetAbs string) error {
	if err := fs.MkdirAll(targetAbs); err != nil {
		return err
	}
	path := targetpath.Gitignore(targetAbs)
	if current, err := readAll(fs, path); err == nil && current == gitignoreContent {
		return nil
	}
	return fs.WriteFile(path, []byte(gitignoreContent))
}

// readAll reads a whole file through the port.
func readAll(fs ports.FileStore, path string) (string, error) {
	r, err := fs.OpenForRead(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
