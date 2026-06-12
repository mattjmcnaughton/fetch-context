package ports

// Config is the parsed configuration: the profile library plus the global
// install target and clone defaults.
type Config struct {
	// Target is the install target relative to the repo root; the store
	// applies the default (.agentic/sources) when the file omits it.
	Target string
	// Clone holds the global clone defaults; the store applies the
	// defaults (depth 1, parallel 4) when the file omits them.
	Clone CloneDefaults
	// Profiles is the named profile library.
	Profiles map[string]Profile
}

// CloneDefaults are the global clone settings, fully resolved by the store.
type CloneDefaults struct {
	// Depth is the history depth for clones; 0 = full history.
	Depth int
	// Parallel is the maximum number of concurrent clones; 1 = sequential.
	Parallel int
}

// Profile bundles repos, groups, and URLs materialized together by `load`.
// All keys are optional.
type Profile struct {
	// Target optionally overrides the global target for this profile.
	Target string
	Repos  []RepoEntry
	Groups []string
	URLs   []string
}

// RepoEntry is one profile repo: a ref plus optional per-repo clone
// overrides (the YAML accepts a plain string or a {ref, depth, branch}
// mapping).
type RepoEntry struct {
	Ref string
	// Depth overrides the global clone depth when non-nil (0 = full).
	Depth *int
	// Branch pins the branch to clone; "" = remote default.
	Branch string
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
