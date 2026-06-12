// Package materialize holds the use cases behind the one-off commands:
// MaterializeRepo (repo), MaterializeGroup (group), MaterializeURL (url).
package materialize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mattjmcnaughton/fetch-context/internal/core/repoid"
	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// RepoItem is one repo to materialize, with fully resolved clone options —
// callers (CLI flags, profile entries) apply precedence before building it.
type RepoItem struct {
	Ref string
	// Depth is the history depth; 0 = full history.
	Depth int
	// Branch pins the branch to clone; "" = remote default.
	Branch string
}

// RepoRequest is one `repo` invocation: items to materialize beneath a
// target (relative to the host repo root).
type RepoRequest struct {
	Items  []RepoItem
	Target string
	// Parallel caps concurrent clones; <= 1 runs sequentially.
	Parallel int
}

// ItemsFromRefs builds items with uniform options — the CLI path, where one
// set of flags applies to every ref.
func ItemsFromRefs(refs []string, depth int, branch string) []RepoItem {
	items := make([]RepoItem, 0, len(refs))
	for _, ref := range refs {
		items = append(items, RepoItem{Ref: ref, Depth: depth, Branch: branch})
	}
	return items
}

// Repo is the MaterializeRepo use case.
type Repo struct {
	git     ports.GitRepo
	fs      ports.FileStore
	locator ports.HostRepoLocator
	log     *slog.Logger
}

func NewRepo(git ports.GitRepo, fs ports.FileStore, locator ports.HostRepoLocator, log *slog.Logger) *Repo {
	return &Repo{git: git, fs: fs, locator: locator, log: log}
}

// Materialize clones (or refreshes) every ref with continue-on-error
// semantics (R3). It returns a *BatchError when any item failed.
func (m *Repo) Materialize(ctx context.Context, req RepoRequest) error {
	root, err := m.locator.RepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}
	targetAbs := targetpath.Resolve(root, req.Target)

	// Parse and dedup by repo identity, keeping the first occurrence's
	// options — equivalent forms still collapse to one clone (AC-REPO-11).
	type job struct {
		spec repoid.Spec
		opts ports.CloneOptions
	}
	var jobs []job
	var failures []ItemError
	seen := make(map[string]bool)
	for _, item := range req.Items {
		spec, err := repoid.Parse(item.Ref)
		if err != nil {
			failures = append(failures, ItemError{Ref: item.Ref, Err: err})
			continue
		}
		if seen[spec.Key()] {
			continue
		}
		seen[spec.Key()] = true
		jobs = append(jobs, job{spec: spec, opts: ports.CloneOptions{Depth: item.Depth, Branch: item.Branch}})
	}

	if len(jobs) > 0 {
		if err := ensureTarget(m.fs, targetAbs); err != nil {
			return fmt.Errorf("preparing target %s: %w", targetAbs, err)
		}
	}
	for _, j := range jobs {
		if err := m.materializeOne(ctx, j.spec, targetAbs, j.opts); err != nil {
			m.log.Warn("repo materialization failed", "ref", j.spec.Ref, "error", err)
			failures = append(failures, ItemError{Ref: j.spec.Ref, Err: err})
		}
	}
	return errorOrNil(failures)
}

// materializeOne clones into the derived destination, or refreshes an
// existing managed clone; an existing unmanaged path is refused untouched
// (AC-REPO-07/08).
func (m *Repo) materializeOne(ctx context.Context, spec repoid.Spec, targetAbs string, opts ports.CloneOptions) error {
	return cloneOrRefresh(ctx, m.git, m.fs, m.log, spec.CloneURL(), targetpath.RepoDir(targetAbs, spec), opts)
}

// cloneOrRefresh applies the shared destination rules: clone when absent,
// refresh when a managed clone (converging it toward opts), refuse
// untouched otherwise. `group` repos obey the same rules as `repo`
// (AC-GROUP-04).
func cloneOrRefresh(ctx context.Context, git ports.GitRepo, fs ports.FileStore, log *slog.Logger, cloneURL, dest string, opts ports.CloneOptions) error {
	exists, err := fs.Exists(dest)
	if err != nil {
		return err
	}
	if exists {
		managed, err := git.IsManagedClone(ctx, dest)
		if err != nil {
			return err
		}
		if !managed {
			return fmt.Errorf("destination %s exists and is not a managed clone; refusing to touch it", dest)
		}
		log.Debug("refreshing managed clone", "dest", dest)
		return git.Refresh(ctx, dest, opts)
	}
	log.Debug("cloning", "url", cloneURL, "dest", dest)
	return git.Clone(ctx, cloneURL, dest, opts)
}
