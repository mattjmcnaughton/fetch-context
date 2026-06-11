# Development

## Prerequisites

- Go 1.25.0+
- [just](https://just.systems/)

## Setup

```sh
# Install dependencies
go mod tidy

# Copy environment file
cp .env.example .env
```

## Common Tasks

```sh
# Format and fix
just fmt-fix

# Run vet
just vet

# Run tests
just test
just test-all

# Build binary
just build

# Run directly
just run example

# Full pre-push check (fmt + vet + unit + integration)
just gate

# Full check including the Docker-bound e2e suite
just gate-expensive
```

## Testing

Tests use the stdlib `testing` package:

- Unit tests live alongside the code they test (e.g.
  `internal/core/repoid/repoid_test.go`)
- Integration tests use the `//go:build integration` build tag
- Run integration tests with `just test-integration`
- Contract tests (`//go:build contract`) and e2e tests (`//go:build e2e`)
  are opt-in; see `docs/testing.md` for the full pyramid

## Building with a Version

```sh
go build -ldflags "-X github.com/mattjmcnaughton/fetch-context/internal/version.Version=1.0.0" \
  -o bin/fetch-context ./cmd/fetch-context
```

## Adding a New Command

1. Create `internal/adapters/cli/<name>.go` with a `new<Name>Cmd(...)` function
   that takes the use case(s) it drives as arguments.
2. Register it in `internal/adapters/cli/root.go` via
   `root.AddCommand(new<Name>Cmd(...))`, threading the use case through from
   the wiring in `cmd/fetch-context/main.go`.
3. Keep the command a thin shim — parse args, call the use case, format
   output. Errors return from cobra's `RunE`; the wiring maps them to exit
   codes (`2` usage, `1` runtime).
4. Business logic lives in a use case under `internal/core/`, depending only
   on ports from `internal/ports/`. A new external dependency means a new
   port plus an adapter under `internal/adapters/`. Read
   `docs/architecture.md` before adding either.
