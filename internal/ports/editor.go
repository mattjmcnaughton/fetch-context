package ports

import "context"

// Editor launches the user's editor on a path and blocks until it exits.
type Editor interface {
	Edit(ctx context.Context, path string) error
}
