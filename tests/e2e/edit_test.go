//go:build e2e

package e2e

import (
	"os"
	"strings"
	"testing"
)

func TestAC_EDIT_01_OpensEditorAndReloadsValidEdit(t *testing.T) {
	w := newWorkspace(t)
	script := editorScript(t, `cat > "$1" <<'EOF'
profiles:
  added-by-editor:
    repos:
      - github.com/foo/bar
EOF`)
	w.setEnv("EDITOR=" + script)

	res := w.run("edit")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	listRes := w.run("list")
	if listRes.code != 0 {
		t.Fatalf("list after edit: exit = %d, stderr: %s", listRes.code, listRes.stderr)
	}
	if !strings.Contains(listRes.stdout, "added-by-editor") {
		t.Errorf("list does not show the edited-in profile:\n%s", listRes.stdout)
	}
}

func TestAC_EDIT_02_InvalidEditRejectedFilePreserved(t *testing.T) {
	w := newWorkspace(t)
	script := editorScript(t, `printf '::: not yaml {{{\n' > "$1"`)
	w.setEnv("EDITOR=" + script)

	res := w.run("edit")
	if res.code == 0 {
		t.Fatal("exit = 0, want non-zero for an invalid edit")
	}
	if !strings.Contains(res.stderr, "yaml") && !strings.Contains(res.stderr, "invalid") {
		t.Errorf("stderr does not report the validation error:\n%s", res.stderr)
	}
	b, err := os.ReadFile(w.configPath())
	if err != nil {
		t.Fatalf("malformed file was removed: %v", err)
	}
	if !strings.Contains(string(b), "::: not yaml") {
		t.Errorf("malformed content not preserved on disk: %q", b)
	}
}

func TestAC_EDIT_03_VisualTakesPrecedenceOverEditor(t *testing.T) {
	w := newWorkspace(t)
	visualMarker := w.path("visual-ran")
	editorMarker := w.path("editor-ran")
	w.setEnv(
		"VISUAL="+editorScript(t, `touch "`+visualMarker+`"`),
		"EDITOR="+editorScript(t, `touch "`+editorMarker+`"`),
	)

	res := w.run("edit")
	if res.code != 0 {
		t.Fatalf("exit = %d, stderr: %s", res.code, res.stderr)
	}
	if !exists(visualMarker) {
		t.Error("the VISUAL script did not run")
	}
	if exists(editorMarker) {
		t.Error("the EDITOR script ran despite VISUAL being set")
	}
}
