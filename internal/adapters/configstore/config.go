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

// fileConfig mirrors the on-disk YAML shape.
type fileConfig struct {
	Target   string                 `yaml:"target"`
	Profiles map[string]fileProfile `yaml:"profiles"`
}

type fileProfile struct {
	Target string   `yaml:"target"`
	Repos  []string `yaml:"repos"`
	Groups []string `yaml:"groups"`
	URLs   []string `yaml:"urls"`
}

func (s *Store) Load() (ports.Config, error) {
	defaults := ports.Config{Target: targetpath.DefaultTarget, Profiles: map[string]ports.Profile{}}

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
	for name, fp := range fc.Profiles {
		if name == "" {
			return ports.Config{}, fmt.Errorf("parsing config %s: profile with empty name", s.path)
		}
		cfg.Profiles[name] = ports.Profile{
			Target: fp.Target,
			Repos:  fp.Repos,
			Groups: fp.Groups,
			URLs:   fp.URLs,
		}
	}
	return cfg, nil
}

var _ ports.ConfigStore = (*Store)(nil)
