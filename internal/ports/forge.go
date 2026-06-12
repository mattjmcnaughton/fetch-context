package ports

import "context"

// GroupRepo is one repo discovered under an org/group.
type GroupRepo struct {
	// Path is the repo path relative to the group root, with subgroup
	// segments preserved (e.g. "alpha" or "sub/nested").
	Path string
	// CloneURL is the URL the forge reports for cloning.
	CloneURL string
}

// ForgeEnumerator lists every repo under an org/group slug, following
// pagination. Implementations are forge-specific (GitHub flat, GitLab
// recursive); the core only sees the flat result.
type ForgeEnumerator interface {
	Enumerate(ctx context.Context, slug string) ([]GroupRepo, error)
}
