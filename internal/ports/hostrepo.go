package ports

import "context"

// HostRepoLocator resolves the root of the repo the user is working in (the
// repo fetch-context materializes into), or errors when the CWD is not
// inside any git repo (R4).
type HostRepoLocator interface {
	RepoRoot(ctx context.Context) (string, error)
}
