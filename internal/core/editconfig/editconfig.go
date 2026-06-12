// Package editconfig opens the config in the user's editor and validates
// the result. An invalid edit errors but leaves the file on disk for the
// user to fix (AC-EDIT-02).
package editconfig

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

type Edit struct {
	editor ports.Editor
	config ports.ConfigStore
	fs     ports.FileStore
	log    *slog.Logger
}

func New(editor ports.Editor, config ports.ConfigStore, fs ports.FileStore, log *slog.Logger) *Edit {
	return &Edit{editor: editor, config: config, fs: fs, log: log}
}

func (e *Edit) Run(ctx context.Context) error {
	path := e.config.Path()
	if err := e.fs.MkdirAll(filepath.Dir(path)); err != nil {
		return fmt.Errorf("preparing config directory: %w", err)
	}
	if err := e.editor.Edit(ctx, path); err != nil {
		return fmt.Errorf("running editor: %w", err)
	}
	if _, err := e.config.Load(); err != nil {
		return fmt.Errorf("config is invalid after the edit (the file is left in place at %s for you to fix): %w", path, err)
	}
	e.log.Debug("config edited and reloaded", "path", path)
	return nil
}
