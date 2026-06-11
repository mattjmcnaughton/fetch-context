package materialize

import (
	"fmt"
	"strings"
)

// ItemError records one failed item of a batch.
type ItemError struct {
	// Ref is the item as the caller named it.
	Ref string
	Err error
}

// BatchError aggregates per-item failures under continue-on-error semantics
// (R3): every item is attempted, failures are collected, and any failure
// makes the whole command fail (exit 1) with a per-item summary.
type BatchError struct {
	Items []ItemError
}

func (b *BatchError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d item(s) failed:", len(b.Items))
	for _, item := range b.Items {
		fmt.Fprintf(&sb, "\n  %s: %v", item.Ref, item.Err)
	}
	return sb.String()
}

// errorOrNil returns nil for an empty batch so callers can return it
// directly.
func errorOrNil(items []ItemError) error {
	if len(items) == 0 {
		return nil
	}
	return &BatchError{Items: items}
}
