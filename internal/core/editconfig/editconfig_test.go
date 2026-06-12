package editconfig

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/testing/fakes"
)

type editFixture struct {
	editor *fakes.FakeEditor
	config *fakes.FakeConfigStore
	fs     *fakes.FakeFileStore
	uc     *Edit
}

func newEditFixture() *editFixture {
	f := &editFixture{
		editor: &fakes.FakeEditor{},
		config: fakes.NewConfigStore(),
		fs:     fakes.NewFileStore(),
	}
	f.uc = New(f.editor, f.config, f.fs, slog.New(slog.DiscardHandler))
	return f
}

func TestEditOpensConfigPathAndReloads(t *testing.T) {
	f := newEditFixture()
	if err := f.uc.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.editor.Edited) != 1 || f.editor.Edited[0] != f.config.Path() {
		t.Errorf("Edited = %v, want the config path", f.editor.Edited)
	}
	// The parent directory must exist so a fresh config can be saved.
	if exists, _ := f.fs.Exists("/home/u/.config/fetch-context"); !exists {
		t.Error("config parent dir not created before editing")
	}
}

func TestEditInvalidResultErrorsAndKeepsFile(t *testing.T) {
	f := newEditFixture()
	f.editor.OnEdit = func(path string) error {
		// Simulate the user saving malformed YAML: subsequent loads fail.
		f.config.LoadErr = errors.New("parsing config: yaml: line 2: mapping values are not allowed")
		return nil
	}

	err := f.uc.Run(context.Background())
	if err == nil {
		t.Fatal("want validation error after a bad edit (AC-EDIT-02)")
	}
	if !strings.Contains(err.Error(), "yaml") || !strings.Contains(err.Error(), "left in place") {
		t.Errorf("error %q should report the validation problem and that the file is preserved", err)
	}
}

func TestEditorFailureSurfaces(t *testing.T) {
	f := newEditFixture()
	f.editor.Err = errors.New("editor exited 1")
	if err := f.uc.Run(context.Background()); err == nil {
		t.Fatal("want editor failure surfaced")
	}
}
