// Package editor launches the user's editor ($VISUAL > $EDITOR > vi) on a
// path and blocks until it exits. It implements ports.Editor.
package editor

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/envx"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

type Editor struct {
	env envx.Env
}

func New(env envx.Env) *Editor {
	return &Editor{env: env}
}

func (e *Editor) Edit(ctx context.Context, path string) error {
	command := chooseEditor(e.env)
	// The editor value may carry arguments ("code --wait"), so run it
	// through the shell with the path appended as a safely-quoted "$1".
	cmd := exec.CommandContext(ctx, "sh", "-c", command+` "$1"`, "sh", path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %q on %s: %w", command, path, err)
	}
	return nil
}

// chooseEditor applies the $VISUAL > $EDITOR > vi precedence (AC-EDIT-03).
func chooseEditor(env envx.Env) string {
	if v, ok := env.Get("VISUAL"); ok && v != "" {
		return v
	}
	if v, ok := env.Get("EDITOR"); ok && v != "" {
		return v
	}
	return "vi"
}

var _ ports.Editor = (*Editor)(nil)
