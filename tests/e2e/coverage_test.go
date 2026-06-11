//go:build e2e

package e2e

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestACCoverage enforces the 1:1 mapping from docs/testing.md: every AC ID
// defined in docs/acceptance.md has exactly one e2e test named
// TestAC_<CATEGORY>_<NN>_<ShortName>, and every such test maps back to a
// defined AC ID.
func TestACCoverage(t *testing.T) {
	acIDs, err := acceptanceIDs(filepath.Join(repoRoot(), "docs", "acceptance.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(acIDs) == 0 {
		t.Fatal("no AC IDs parsed from docs/acceptance.md")
	}
	testIDs, err := e2eTestIDs(".")
	if err != nil {
		t.Fatal(err)
	}

	for id := range acIDs {
		switch len(testIDs[id]) {
		case 0:
			t.Errorf("%s has no e2e test (expected one Test%s_*)", id, idToTestPrefix(id))
		case 1:
		default:
			t.Errorf("%s has %d e2e tests, want exactly 1: %v", id, len(testIDs[id]), testIDs[id])
		}
	}
	for id, names := range testIDs {
		if !acIDs[id] {
			t.Errorf("test(s) %v map to %s, which is not defined in docs/acceptance.md", names, id)
		}
	}
}

// acDefRe matches an AC definition line: "**AC-REPO-03 — auto-gitignore...**".
var acDefRe = regexp.MustCompile(`^\*\*(AC-[A-Z]+-[0-9]+)\b`)

// acceptanceIDs parses the set of AC IDs defined in the acceptance doc.
func acceptanceIDs(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ids := make(map[string]bool)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := acDefRe.FindStringSubmatch(sc.Text()); m != nil {
			ids[m[1]] = true
		}
	}
	return ids, sc.Err()
}

// acTestRe matches a conforming e2e test name: TestAC_REPO_07_ShortName.
var acTestRe = regexp.MustCompile(`^TestAC_([A-Z]+)_([0-9]+)_`)

// e2eTestIDs maps AC ID → the test function names in dir that encode it.
func e2eTestIDs(dir string) (map[string][]string, error) {
	fset := token.NewFileSet()
	byID := make(map[string][]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if m := acTestRe.FindStringSubmatch(fn.Name.Name); m != nil {
				id := fmt.Sprintf("AC-%s-%s", m[1], m[2])
				byID[id] = append(byID[id], fn.Name.Name)
			}
		}
	}
	return byID, nil
}

// idToTestPrefix converts AC-REPO-07 to AC_REPO_07.
func idToTestPrefix(id string) string {
	return strings.ReplaceAll(id, "-", "_")
}
