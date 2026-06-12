// Package targetpath derives on-disk paths beneath the resolved target from
// repo and URL specs (AC-LAYOUT-*, AC-ROOT-01).
package targetpath

import (
	"path/filepath"

	"github.com/mattjmcnaughton/fetch-context/internal/core/repoid"
)

// DefaultTarget is the install target relative to the repo root when no
// config overrides it.
const DefaultTarget = ".agentic/sources"

// Resolve joins the host repo root with the (relative) configured target.
func Resolve(repoRoot, target string) string {
	return filepath.Join(repoRoot, target)
}

// ReposRoot is the subtree holding clones.
func ReposRoot(targetAbs string) string {
	return filepath.Join(targetAbs, "repos")
}

// URLsRoot is the subtree holding fetched markdown.
func URLsRoot(targetAbs string) string {
	return filepath.Join(targetAbs, "urls")
}

// RepoDir maps a repo spec to repos/<host>/<owner>/<repo>. Owner may span
// several path segments (GitLab subgroups).
func RepoDir(targetAbs string, spec repoid.Spec) string {
	return filepath.Join(ReposRoot(targetAbs), spec.Host, spec.Owner, spec.Repo)
}

// GroupRepoDir maps one enumerated group repo to
// repos/<host>/<group-slug>/<repo-path>, preserving subgroup segments in
// both slug and path (AC-GROUP-02).
func GroupRepoDir(targetAbs string, group repoid.GroupSpec, repoPath string) string {
	return filepath.Join(ReposRoot(targetAbs), group.Host, group.Slug, repoPath)
}

// URLFile maps an already-mapped relative markdown path (see core/urlmap)
// beneath urls/.
func URLFile(targetAbs, mapped string) string {
	return filepath.Join(URLsRoot(targetAbs), mapped)
}

// Gitignore is the auto-written ignore file at the target root.
func Gitignore(targetAbs string) string {
	return filepath.Join(targetAbs, ".gitignore")
}
