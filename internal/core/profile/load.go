// Package profile holds the LoadProfile use case: materialize every key of
// a named profile using the rules of the corresponding one-off command.
package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// The three materializer interfaces are declared here, by the consumer
// (docs/architecture.md): the real wiring binds them to *materialize.Repo,
// *materialize.Group, *materialize.URL, which satisfy them directly; unit
// tests bind three tiny fakes.

type RepoMaterializer interface {
	Materialize(ctx context.Context, req materialize.RepoRequest) error
}

type GroupMaterializer interface {
	Materialize(ctx context.Context, req materialize.GroupRequest) error
}

type URLMaterializer interface {
	Materialize(ctx context.Context, req materialize.URLRequest) error
}

// Load is the LoadProfile use case.
type Load struct {
	config ports.ConfigStore
	repos  RepoMaterializer
	groups GroupMaterializer
	urls   URLMaterializer
	log    *slog.Logger
}

func NewLoad(config ports.ConfigStore, repos RepoMaterializer, groups GroupMaterializer, urls URLMaterializer, log *slog.Logger) *Load {
	return &Load{config: config, repos: repos, groups: groups, urls: urls, log: log}
}

// Run materializes the named profile. Keys are attempted independently
// (continue-on-error, R3/AC-LOAD-06); per-item detail from the underlying
// batch errors is preserved.
func (l *Load) Run(ctx context.Context, name string) error {
	cfg, err := l.config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	prof, ok := cfg.Profiles[name]
	if !ok {
		return usageerr.Newf("unknown profile %q (defined profiles: %s)", name, profileNames(cfg))
	}
	target := prof.Target
	if target == "" {
		target = cfg.Target
	}
	l.log.Debug("loading profile", "profile", name, "target", target)

	var errs []error
	if len(prof.Repos) > 0 {
		// Per-entry overrides win over the global clone defaults.
		items := make([]materialize.RepoItem, 0, len(prof.Repos))
		for _, entry := range prof.Repos {
			depth := cfg.Clone.Depth
			if entry.Depth != nil {
				depth = *entry.Depth
			}
			items = append(items, materialize.RepoItem{Ref: entry.Ref, Depth: depth, Branch: entry.Branch})
		}
		if err := l.repos.Materialize(ctx, materialize.RepoRequest{Items: items, Target: target, Parallel: cfg.Clone.Parallel}); err != nil {
			errs = append(errs, fmt.Errorf("repos: %w", err))
		}
	}
	if len(prof.Groups) > 0 {
		if err := l.groups.Materialize(ctx, materialize.GroupRequest{Refs: prof.Groups, Target: target, Depth: cfg.Clone.Depth, Parallel: cfg.Clone.Parallel}); err != nil {
			errs = append(errs, fmt.Errorf("groups: %w", err))
		}
	}
	if len(prof.URLs) > 0 {
		if err := l.urls.Materialize(ctx, materialize.URLRequest{URLs: prof.URLs, Target: target}); err != nil {
			errs = append(errs, fmt.Errorf("urls: %w", err))
		}
	}
	return errors.Join(errs...)
}

func profileNames(cfg ports.Config) string {
	if len(cfg.Profiles) == 0 {
		return "none"
	}
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
