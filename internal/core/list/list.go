// Package list renders the profile library and what is currently
// materialized on disk under each resolved target.
package list

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattjmcnaughton/fetch-context/internal/core/repoid"
	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
	"github.com/mattjmcnaughton/fetch-context/internal/core/urlmap"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

type List struct {
	config  ports.ConfigStore
	fs      ports.FileStore
	locator ports.HostRepoLocator
	log     *slog.Logger
}

func New(config ports.ConfigStore, fs ports.FileStore, locator ports.HostRepoLocator, log *slog.Logger) *List {
	return &List{config: config, fs: fs, locator: locator, log: log}
}

// Run renders the configured profiles with their entries and per-entry
// materialized state under the resolved target.
func (l *List) Run(ctx context.Context) (string, error) {
	cfg, err := l.config.Load()
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	if len(cfg.Profiles) == 0 {
		return fmt.Sprintf("no profiles defined (config: %s)\n", l.config.Path()), nil
	}
	root, err := l.locator.RepoRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("resolving repo root: %w", err)
	}

	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	var sb strings.Builder
	fmt.Fprintf(&sb, "profiles (config: %s)\n", l.config.Path())
	for _, name := range names {
		prof := cfg.Profiles[name]
		target := prof.Target
		if target == "" {
			target = cfg.Target
		}
		targetAbs := targetpath.Resolve(root, target)
		fmt.Fprintf(&sb, "\n%s (target: %s)\n", name, target)
		repoRefs := make([]string, 0, len(prof.Repos))
		for _, entry := range prof.Repos {
			repoRefs = append(repoRefs, entry.Ref)
		}
		l.renderEntries(&sb, "repos", repoRefs, func(ref string) string {
			spec, err := repoid.Parse(ref)
			if err != nil {
				return ""
			}
			return targetpath.RepoDir(targetAbs, spec)
		})
		l.renderEntries(&sb, "groups", prof.Groups, func(ref string) string {
			spec, err := repoid.ParseGroup(ref)
			if err != nil {
				return ""
			}
			return filepath.Join(targetpath.ReposRoot(targetAbs), spec.Host, spec.Slug)
		})
		l.renderEntries(&sb, "urls", prof.URLs, func(raw string) string {
			mapped, err := urlmap.Map(raw)
			if err != nil {
				return ""
			}
			return targetpath.URLFile(targetAbs, mapped)
		})
	}
	return sb.String(), nil
}

// renderEntries prints one profile key with per-entry materialized state
// (AC-LIST-02). pathFor returns "" for entries that cannot be mapped.
func (l *List) renderEntries(sb *strings.Builder, key string, entries []string, pathFor func(string) string) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(sb, "  %s:\n", key)
	for _, entry := range entries {
		state := "[absent]"
		if path := pathFor(entry); path != "" {
			if exists, err := l.fs.Exists(path); err == nil && exists {
				state = "[materialized]"
			}
		} else {
			state = "[invalid entry]"
		}
		fmt.Fprintf(sb, "    %s  %s\n", entry, state)
	}
}
