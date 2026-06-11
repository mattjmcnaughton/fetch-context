package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/version"
)

func executeRoot(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRoot()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errBuf.String(), err
}

func TestRootNoArgsIsUsageError(t *testing.T) {
	_, _, err := executeRoot(t)
	if !usageerr.IsUsage(err) {
		t.Fatalf("err = %v, want a usage error", err)
	}
}

func TestRootUnknownSubcommandIsUsageErrorNamingIt(t *testing.T) {
	_, _, err := executeRoot(t, "frobnicate")
	if !usageerr.IsUsage(err) {
		t.Fatalf("err = %v, want a usage error", err)
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("error %q does not name the unknown command", err)
	}
}

func TestRootUnknownFlagIsUsageError(t *testing.T) {
	_, _, err := executeRoot(t, "--bogus")
	if !usageerr.IsUsage(err) {
		t.Fatalf("err = %v, want a usage error", err)
	}
}

func TestVersionSubcommandPrintsVersion(t *testing.T) {
	stdout, _, err := executeRoot(t, "version")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if !strings.Contains(stdout, version.Version) {
		t.Errorf("stdout %q does not contain version %q", stdout, version.Version)
	}
}
