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

// RepoRequest is one `repo` invocation: refs to materialize beneath a
// target (relative to the host repo root).
type RepoRequest struct {
	Refs   []string
	Target string
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

	specs, badRefs := repoid.ParseAll(req.Refs)
	var failures []ItemError
	for _, bad := range badRefs {
		failures = append(failures, ItemError{Ref: bad.Ref, Err: bad.Err})
	}

	if len(specs) > 0 {
		if err := ensureTarget(m.fs, targetAbs); err != nil {
			return fmt.Errorf("preparing target %s: %w", targetAbs, err)
		}
	}
	for _, spec := range specs {
		if err := m.materializeOne(ctx, spec, targetAbs); err != nil {
			m.log.Warn("repo materialization failed", "ref", spec.Ref, "error", err)
			failures = append(failures, ItemError{Ref: spec.Ref, Err: err})
		}
	}
	return errorOrNil(failures)
}

// materializeOne clones into the derived destination, or refreshes an
// existing managed clone; an existing unmanaged path is refused untouched
// (AC-REPO-07/08).
func (m *Repo) materializeOne(ctx context.Context, spec repoid.Spec, targetAbs string) error {
	dest := targetpath.RepoDir(targetAbs, spec)
	exists, err := m.fs.Exists(dest)
	if err != nil {
		return err
	}
	if exists {
		managed, err := m.git.IsManagedClone(ctx, dest)
		if err != nil {
			return err
		}
		if !managed {
			return fmt.Errorf("destination %s exists and is not a managed clone; refusing to touch it", dest)
		}
		m.log.Debug("refreshing managed clone", "dest", dest)
		return m.git.Refresh(ctx, dest)
	}
	m.log.Debug("cloning", "url", spec.CloneURL(), "dest", dest)
	return m.git.Clone(ctx, spec.CloneURL(), dest)
}
