// Package configstore loads the YAML config at
// <config-home>/.config/fetch-context/config.yaml. It implements
// ports.ConfigStore. Decoding is strict: unknown fields and malformed YAML
// fail loudly with the file position (AC-CONFIG-03); a missing file is not
// an error (AC-CONFIG-04).
package configstore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

type Store struct {
	path string
}

// New builds a store rooted at home (the FETCH_CONTEXT_HOME override or the
// user's home directory): <home>/.config/fetch-context/config.yaml
// (AC-CONFIG-01).
func New(home string) *Store {
	return &Store{path: filepath.Join(home, ".config", "fetch-context", "config.yaml")}
}

func (s *Store) Path() string { return s.path }

// Clone defaults applied when the file omits them (AC-CONFIG-04 spirit:
// absent settings mean defaults, not errors).
const (
	defaultCloneDepth    = 1
	defaultCloneParallel = 4
)

// fileConfig mirrors the on-disk YAML shape.
type fileConfig struct {
	Target   string                 `yaml:"target"`
	Clone    fileClone              `yaml:"clone"`
	Profiles map[string]fileProfile `yaml:"profiles"`
}

type fileClone struct {
	Depth    *int `yaml:"depth"`
	Parallel *int `yaml:"parallel"`
}

type fileProfile struct {
	Target string          `yaml:"target"`
	Repos  []fileRepoEntry `yaml:"repos"`
	Groups []string        `yaml:"groups"`
	URLs   []string        `yaml:"urls"`
}

// fileRepoEntry accepts either a plain ref string or a {ref, depth, branch}
// mapping. A custom unmarshaler bypasses the decoder's KnownFields
// strictness for its subtree, so unknown mapping keys are rejected by hand
// to keep AC-CONFIG-03's loud, positioned errors.
type fileRepoEntry struct {
	Ref    string
	Depth  *int
	Branch string
}

func (e *fileRepoEntry) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Decode(&e.Ref)
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			switch key := node.Content[i]; key.Value {
			case "ref", "depth", "branch":
			default:
				return fmt.Errorf("line %d: field %s not found in repo entry (want ref, depth, branch)", key.Line, key.Value)
			}
		}
		var m struct {
			Ref    string `yaml:"ref"`
			Depth  *int   `yaml:"depth"`
			Branch string `yaml:"branch"`
		}
		if err := node.Decode(&m); err != nil {
			return err
		}
		if m.Ref == "" {
			return fmt.Errorf("line %d: repo entry mapping requires a ref", node.Line)
		}
		*e = fileRepoEntry(m)
		return nil
	default:
		return fmt.Errorf("line %d: repo entry must be a string or a {ref, depth, branch} mapping", node.Line)
	}
}

func (s *Store) Load() (ports.Config, error) {
	defaults := ports.Config{
		Target:   targetpath.DefaultTarget,
		Clone:    ports.CloneDefaults{Depth: defaultCloneDepth, Parallel: defaultCloneParallel},
		Profiles: map[string]ports.Profile{},
	}

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return defaults, nil
	}
	if err != nil {
		return ports.Config{}, fmt.Errorf("reading config %s: %w", s.path, err)
	}

	var fc fileConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&fc); err != nil {
		if errors.Is(err, io.EOF) {
			// An empty file is an empty config, not an error.
			return defaults, nil
		}
		return ports.Config{}, fmt.Errorf("parsing config %s: %w", s.path, err)
	}

	cfg := defaults
	if fc.Target != "" {
		cfg.Target = fc.Target
	}
	if fc.Clone.Depth != nil {
		if *fc.Clone.Depth < 0 {
			return ports.Config{}, fmt.Errorf("parsing config %s: clone.depth must be >= 0 (0 = full history), got %d", s.path, *fc.Clone.Depth)
		}
		cfg.Clone.Depth = *fc.Clone.Depth
	}
	if fc.Clone.Parallel != nil {
		if *fc.Clone.Parallel < 1 {
			return ports.Config{}, fmt.Errorf("parsing config %s: clone.parallel must be >= 1, got %d", s.path, *fc.Clone.Parallel)
		}
		cfg.Clone.Parallel = *fc.Clone.Parallel
	}
	for name, fp := range fc.Profiles {
		if name == "" {
			return ports.Config{}, fmt.Errorf("parsing config %s: profile with empty name", s.path)
		}
		repos := make([]ports.RepoEntry, 0, len(fp.Repos))
		for _, entry := range fp.Repos {
			if entry.Depth != nil && *entry.Depth < 0 {
				return ports.Config{}, fmt.Errorf("parsing config %s: repo %s: depth must be >= 0 (0 = full history), got %d", s.path, entry.Ref, *entry.Depth)
			}
			repos = append(repos, ports.RepoEntry(entry))
		}
		cfg.Profiles[name] = ports.Profile{
			Target: fp.Target,
			Repos:  repos,
			Groups: fp.Groups,
			URLs:   fp.URLs,
		}
	}
	return cfg, nil
}

var _ ports.ConfigStore = (*Store)(nil)
