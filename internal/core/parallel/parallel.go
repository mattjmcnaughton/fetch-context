// Package parallel provides the one bounded-concurrency primitive the core
// uses (ADR-0002). Keeping the goroutine choreography here lets the use
// cases stay sequential-looking and deterministic: they fan work out through
// Map and fold the results back in input order.
package parallel

import (
	"context"
	"sync"
)

// Map applies fn to every item, running at most limit invocations
// concurrently, and returns one result per item in input order. A limit of
// one or less runs strictly sequentially in input order. fn is invoked for
// every item regardless of other items' outcomes (continue-on-error, R3);
// ctx is passed through for external cancellation only — Map itself never
// cancels it.
func Map[T, R any](ctx context.Context, limit int, items []T, fn func(ctx context.Context, item T) R) []R {
	results := make([]R, len(items))
	if limit <= 1 {
		for i, item := range items {
			results[i] = fn(ctx, item)
		}
		return results
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = fn(ctx, item)
		}()
	}
	wg.Wait()
	return results
}
