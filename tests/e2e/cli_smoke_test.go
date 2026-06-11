//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

func TestAC_VERSION_01_VersionPrints(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("version")
	if res.code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", res.code, res.stderr)
	}
	if strings.TrimSpace(res.stdout) == "" {
		t.Fatal("stdout is empty, want a non-empty version string")
	}
}

func TestAC_USAGE_01_NoArgsPrintsUsage(t *testing.T) {
	w := newWorkspace(t)
	res := w.run()
	if res.code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "Usage:") {
		t.Fatalf("stderr does not contain usage text:\n%s", res.stderr)
	}
}

func TestAC_USAGE_02_UnknownSubcommand(t *testing.T) {
	w := newWorkspace(t)
	res := w.run("frobnicate")
	if res.code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "frobnicate") {
		t.Fatalf("stderr does not name the unknown command:\n%s", res.stderr)
	}
}
