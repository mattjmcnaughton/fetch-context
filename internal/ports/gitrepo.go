package ports

import "context"

// GitRepo manages upstream clones beneath the target tree.
type GitRepo interface {
	// Clone shallow-clones (depth 1, default branch) cloneURL into dest.
	// A failed clone must leave no partial directory at dest.
	Clone(ctx context.Context, cloneURL, dest string) error

	// Refresh updates an existing managed clone to the remote's latest:
	// fetch, hard reset, and removal of untracked files. Local state is
	// discarded by design (AC-REPO-07).
	Refresh(ctx context.Context, dest string) error

	// IsManagedClone reports whether dest is itself the root of a git
	// working tree (not merely a directory inside one — the target lives
	// inside the host repo).
	IsManagedClone(ctx context.Context, dest string) (bool, error)
}
