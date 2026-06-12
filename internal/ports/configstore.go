package ports

// Config is the parsed configuration: the profile library plus the global
// install target.
type Config struct {
	// Target is the install target relative to the repo root; the store
	// applies the default (.agentic/sources) when the file omits it.
	Target string
	// Profiles is the named profile library.
	Profiles map[string]Profile
}

// Profile bundles repos, groups, and URLs materialized together by `load`.
// All keys are optional.
type Profile struct {
	// Target optionally overrides the global target for this profile.
	Target string
	Repos  []string
	Groups []string
	URLs   []string
}

// ConfigStore loads the YAML config under the config home.
type ConfigStore interface {
	// Load parses the config file. A missing file yields an empty config
	// with defaults (AC-CONFIG-04); a malformed file errors loudly and
	// precisely (AC-CONFIG-03).
	Load() (Config, error)
	// Path is the config file location (used by `edit` and messages).
	Path() string
}
