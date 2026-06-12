# ADR-0002: Bounded-parallel clones via a hand-rolled core runner

Status: **accepted** (2026-06-12)

## Context

`group` (and `repo` / `load` with many items) clones repositories one at a
time; for an org of dozens of repos the wall-clock cost is dominated by
serialized network round trips. We want bounded concurrency (default 4,
configurable as `clone.parallel` / `--parallel`) without giving up the
properties the batch commands already guarantee:

- **Continue-on-error (R3):** every item is attempted; failures are
  collected into one per-item summary (`BatchError`), never short-circuited.
- **Deterministic output:** the failure summary must not depend on
  goroutine scheduling.
- **Core purity:** `internal/core/` admits no third-party imports
  (enforced mechanically by `TestCorePurity`).

## Decision

Concurrency lives in exactly one primitive, `internal/core/parallel.Map`:

```go
func Map[T, R any](ctx context.Context, limit int, items []T,
    fn func(ctx context.Context, item T) R) []R
```

A semaphore channel bounds in-flight invocations at `limit`; results are
written into a slice indexed by input position, so callers fold them back in
input order and their output is deterministic regardless of scheduling.
`limit <= 1` runs strictly sequentially — exactly the pre-parallel behavior.
`fn` runs for every item; `ctx` is passed through for external cancellation
only, never cancelled by Map on item failure.

`golang.org/x/sync/errgroup` was rejected: it is a third-party import the
core purity rule forbids, and its first-error / cancel-siblings semantics are
the opposite of R3's continue-on-error — we would be working around the
library, not using it.

Use cases stay sequential-looking: build the job list, `parallel.Map` over
it, fold results into `failures` by index. Only the clone fan-out is
parallel; forge enumeration and one-time target preparation remain
sequential.

## Consequences

- Per-item `slog` lines may interleave under parallelism (each line is
  atomic; handlers are concurrency-safe). The authoritative report remains
  the deterministic `BatchError`.
- Test fakes invoked through Map must be safe for concurrent use
  (`FakeGitRepo` guards its recordings with a mutex).
- `clone.parallel: 1` (or `--parallel 1`) restores fully sequential
  behavior, byte-for-byte.
