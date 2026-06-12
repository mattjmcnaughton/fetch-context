package parallel

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimitOneRunsSequentiallyInOrder(t *testing.T) {
	var order []int
	results := Map(context.Background(), 1, []int{10, 20, 30}, func(_ context.Context, item int) int {
		order = append(order, item) // safe: sequential by contract
		return item * 2
	})
	if want := []int{10, 20, 30}; !equal(order, want) {
		t.Errorf("invocation order = %v, want %v", order, want)
	}
	if want := []int{20, 40, 60}; !equal(results, want) {
		t.Errorf("results = %v, want %v", results, want)
	}
}

func TestZeroLimitRunsSequentially(t *testing.T) {
	var order []int
	Map(context.Background(), 0, []int{1, 2}, func(_ context.Context, item int) int {
		order = append(order, item)
		return item
	})
	if want := []int{1, 2}; !equal(order, want) {
		t.Errorf("invocation order = %v, want %v", order, want)
	}
}

// TestResultsLandAtInputIndexes forces item 0 to finish last: it blocks until
// the final item has completed. Results must still come back in input order.
func TestResultsLandAtInputIndexes(t *testing.T) {
	const n = 3
	lastDone := make(chan struct{})
	results := Map(context.Background(), n, []int{0, 1, 2}, func(_ context.Context, item int) int {
		if item == n-1 {
			defer close(lastDone)
		} else {
			<-lastDone
		}
		return item * 10
	})
	if want := []int{0, 10, 20}; !equal(results, want) {
		t.Errorf("results = %v, want %v", results, want)
	}
}

func TestConcurrencyNeverExceedsLimit(t *testing.T) {
	const limit, n = 3, 50
	var inFlight, peak atomic.Int64
	var mu sync.Mutex
	Map(context.Background(), limit, make([]struct{}, n), func(_ context.Context, _ struct{}) struct{} {
		cur := inFlight.Add(1)
		defer inFlight.Add(-1)
		mu.Lock()
		if cur > peak.Load() {
			peak.Store(cur)
		}
		mu.Unlock()
		return struct{}{}
	})
	if got := peak.Load(); got > limit {
		t.Errorf("peak concurrency = %d, want <= %d", got, limit)
	}
}

// TestLimitIsActuallyReached proves Map runs `limit` items at once: every
// item blocks until all of them are in flight together, so an
// under-parallelizing implementation never releases the barrier.
func TestLimitIsActuallyReached(t *testing.T) {
	const limit = 4
	arrived := make(chan struct{}, limit)
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		Map(context.Background(), limit, make([]struct{}, limit), func(_ context.Context, _ struct{}) struct{} {
			arrived <- struct{}{}
			<-release
			return struct{}{}
		})
	}()
	for range limit {
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			t.Fatalf("fewer than %d items in flight concurrently", limit)
		}
	}
	close(release)
	<-done
}

func TestContextIsPassedThrough(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "v")
	for _, limit := range []int{1, 2} {
		results := Map(ctx, limit, []int{0, 1}, func(ctx context.Context, _ int) bool {
			return ctx.Value(key{}) == "v"
		})
		for i, ok := range results {
			if !ok {
				t.Errorf("limit %d: fn %d did not receive the caller's context", limit, i)
			}
		}
	}
}

func TestEmptyItems(t *testing.T) {
	results := Map(context.Background(), 4, nil, func(_ context.Context, _ int) int { return 0 })
	if len(results) != 0 {
		t.Errorf("results = %v, want empty", results)
	}
}

func equal[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
