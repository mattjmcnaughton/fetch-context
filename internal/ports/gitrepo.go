package ports

import "context"

// CloneOptions controls history depth and branch selection for Clone and
// Refresh. Callers pass a resolved depth (the configured default is 1); a
// Depth of 0 means full history and an empty Branch means the remote's
// default branch.
type CloneOptions struct {
	// Depth is the number of commits of history to fetch; 0 = full history.
	Depth int
	// Branch pins the branch to clone and track; "" = remote default.
	Branch string
}

// GitRepo manages upstream clones beneath the target tree.
type GitRepo interface {
	// Clone clones cloneURL into dest honoring opts. A failed clone must
	// leave no partial directory at dest.
	Clone(ctx context.Context, cloneURL, dest string, opts CloneOptions) error

	// Refresh updates an existing managed clone to the remote's latest and
	// converges it toward opts: fetch at the configured depth (unshallowing
	// or re-trimming as needed), hard reset, branch switch when opts pins a
	// different branch, and removal of untracked files. Local state is
	// discarded by design (AC-REPO-07).
	Refresh(ctx context.Context, dest string, opts CloneOptions) error

	// IsManagedClone reports whether dest is itself the root of a git
	// working tree (not merely a directory inside one — the target lives
	// inside the host repo).
	IsManagedClone(ctx context.Context, dest string) (bool, error)
}
