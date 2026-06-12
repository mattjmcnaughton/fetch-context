// Package clean removes materialized content. It only ever touches paths
// beneath a resolved target inside the host repo (AC-CLEAN-04, AC-SAFE-04).
package clean

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

type Clean struct {
	config  ports.ConfigStore
	fs      ports.FileStore
	locator ports.HostRepoLocator
	log     *slog.Logger
}

func New(config ports.ConfigStore, fs ports.FileStore, locator ports.HostRepoLocator, log *slog.Logger) *Clean {
	return &Clean{config: config, fs: fs, locator: locator, log: log}
}

// Run removes materialized content. scope is "" (whole global target),
// "repos"/"urls" (one subtree of the global target), or a profile name
// (that profile's resolved target only — never auto-discovering other
// profiles' targets, R7).
func (c *Clean) Run(ctx context.Context, scope string) error {
	cfg, err := c.config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	root, err := c.locator.RepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("resolving repo root: %w", err)
	}

	globalTarget := targetpath.Resolve(root, cfg.Target)
	var remove string
	switch scope {
	case "":
		remove = globalTarget
	case "repos":
		remove = targetpath.ReposRoot(globalTarget)
	case "urls":
		remove = targetpath.URLsRoot(globalTarget)
	default:
		prof, ok := cfg.Profiles[scope]
		if !ok {
			return usageerr.Newf("unknown clean scope %q: want repos, urls, or a profile name", scope)
		}
		profTarget := prof.Target
		if profTarget == "" {
			profTarget = cfg.Target
		}
		remove = targetpath.Resolve(root, profTarget)
	}

	if err := containedIn(root, remove); err != nil {
		return err
	}
	c.log.Debug("removing", "path", remove)
	return c.fs.Remove(remove)
}

// containedIn refuses any removal that is not a proper subdirectory of the
// repo root: clean only ever deletes inside its own target tree.
func containedIn(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove %s: it is not strictly inside the repo root %s", path, root)
	}
	return nil
}
