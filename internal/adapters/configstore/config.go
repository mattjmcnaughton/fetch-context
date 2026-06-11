// Package configstore loads the YAML config under $FETCH_CONTEXT_HOME. The
// full store (profiles, validation) lands with the load/list slice; one-off
// commands only need the resolved target, which must default sanely when no
// config exists (AC-CONFIG-04).
package configstore

import (
	"github.com/mattjmcnaughton/fetch-context/internal/core/targetpath"
)

// Config is the parsed configuration.
type Config struct {
	// Target is the install target relative to the repo root.
	Target string
}

// Default is the configuration used when no config file exists: one-off
// commands work without any config.
func Default() *Config {
	return &Config{Target: targetpath.DefaultTarget}
}
