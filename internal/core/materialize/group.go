package materialize

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/core/parallel"
	"github.com/mattjmcnaughton/fetch-context/internal/core/repoid"
	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// GroupRequest is one `group` invocation. Depth applies to every enumerated
// repo (groups always clone the remote default branch).
type GroupRequest struct {
	Refs   []string
	Target string
	// Depth is the history depth for every clone; 0 = full history.
	Depth int
	// Parallel caps concurrent clones; <= 1 runs sequentially.
	Parallel int
}

// Group is the MaterializeGroup use case: enumerate an org/group via its
// forge API and clone every repo with the same rules as `repo`.
type Group struct {
	// enumerators is keyed by forge host (ADR-0001 decision 1); the wiring
	// provides one per supported forge.
	enumerators map[string]ports.ForgeEnumerator
	git         ports.GitRepo
	fs          ports.FileStore
	locator     ports.HostRepoLocator
	log         *slog.Logger
}

func NewGroup(enumerators map[string]ports.ForgeEnumerator, git ports.GitRepo, fs ports.FileStore, locator ports.HostRepoLocator, log *slog.Logger) *Group {
	return &Group{enumerators: enumerators, git: git, fs: fs, locator: locator, log: log}
}

func (m *Group) Materialize(ctx context.Context, req GroupRequest) error {
	// Argument validation happens up front: a malformed ref or unknown
	// forge host is the caller's mistake (exit 2) and nothing is touched.
	specs := make([]repoid.GroupSpec, 0, len(req.Refs))
	for _, ref := range req.Refs {
		spec, err := repoid.ParseGroup(ref)
		if err != nil {
			return usageerr.Wrap(err)
		}
		if _, ok := m.enumerators[spec.Host]; !ok {
			return usageerr.Newf("no forge adapter for host %q (supported: %s)", spec.Host, strings.Join(m.supportedHosts(), ", "))
		}
		specs = append(specs, spec)
	}

	root, err := m.locator.RepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}
	targetAbs := targetpath.Resolve(root, req.Target)

	var failures []ItemError
	prepared := false
	for _, spec := range specs {
		repos, err := m.enumerators[spec.Host].Enumerate(ctx, spec.Slug)
		if err != nil {
			// Enumeration failures (auth included) are reported, never
			// silently skipped (AC-GROUP-05).
			m.log.Warn("group enumeration failed", "group", spec.Ref, "error", err)
			failures = append(failures, ItemError{Ref: spec.Ref, Err: err})
			continue
		}
		m.log.Debug("group enumerated", "group", spec.Ref, "repos", len(repos))
		if !prepared && len(repos) > 0 {
			if err := ensureTarget(m.fs, targetAbs); err != nil {
				return fmt.Errorf("preparing target %s: %w", targetAbs, err)
			}
			prepared = true
		}
		// Fan this spec's clones out (bounded, ADR-0002); enumeration and
		// target prep stay sequential. Results come back in input order, so
		// the failure summary is deterministic.
		type cloneJob struct {
			ref, url, dest string
		}
		jobs := make([]cloneJob, 0, len(repos))
		for _, repo := range repos {
			jobs = append(jobs, cloneJob{
				ref:  spec.Ref + ": " + repo.Path,
				url:  repo.CloneURL,
				dest: targetpath.GroupRepoDir(targetAbs, spec, repo.Path),
			})
		}
		results := parallel.Map(ctx, req.Parallel, jobs, func(ctx context.Context, j cloneJob) error {
			return cloneOrRefresh(ctx, m.git, m.fs, m.log, j.url, j.dest, ports.CloneOptions{Depth: req.Depth})
		})
		for i, err := range results {
			if err != nil {
				failures = append(failures, ItemError{Ref: jobs[i].ref, Err: err})
			}
		}
	}
	return errorOrNil(failures)
}

func (m *Group) supportedHosts() []string {
	hosts := make([]string, 0, len(m.enumerators))
	for h := range m.enumerators {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	return hosts
}
