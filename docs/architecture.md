# Architecture

## Overview

fetch-context is a Go CLI built with strict **hexagonal (ports-and-adapters)**
architecture. The core declares interfaces (ports) for everything outside it;
adapters implement them. The CLI itself is a driving adapter.

Concretely:

- The **core** (`internal/core/`) contains pure domain services and use cases.
  It has **zero** imports from `net/http`, `os/exec`, `os`, third-party SDKs,
  or any other infrastructure package. Its only dependencies are the ports it
  declares.
- **Ports** (`internal/ports/`) are small Go interfaces. The core depends on
  these; nothing else does.
- **Adapters** (`internal/adapters/`) are concrete implementations of ports.
  Each adapter sits behind exactly one port and has one job: speak to its
  external dependency.
- **Wiring** (`cmd/fetch-context/main.go`) is the only place that knows about
  every concrete adapter. It reads env vars, constructs adapters, injects them
  into use cases, and hands the resulting use cases to the cobra root.

The hexagon's outside edge is therefore: git binary, GitHub REST, GitLab REST,
the jina reader proxy, the local filesystem, the user's editor, the host repo's
git metadata, and the YAML config file. Each is reached through exactly one
adapter behind exactly one port.

## Project Structure

```
cmd/fetch-context/
  main.go                       # Wiring: env → adapters → use cases → cobra root
internal/
  core/                         # Pure: zero infrastructure imports
    urlmap/                     # URL → filename mapping (R5)
    repoid/                     # Repo URL normalization (R6)
    targetpath/                 # Target root + spec → on-disk path
    materialize/                # Use cases: MaterializeRepo, MaterializeGroup, MaterializeURL
    profile/                    # Use case: LoadProfile (composes the three above)
    clean/                      # Use case: Clean
    list/                       # Use case: ListProfiles
    editconfig/                 # Use case: EditConfig
  ports/                        # Interfaces the core depends on
    gitrepo.go                  # GitRepo
    forge.go                    # ForgeEnumerator
    pagereader.go               # PageReader
    filestore.go                # FileStore
    hostrepo.go                 # HostRepoLocator
    configstore.go              # ConfigStore
    editor.go                   # Editor
  adapters/
    cli/                        # Driving adapter: cobra commands (one file per subcommand)
    gitrepo/                    # Driven: shells out to git binary
    forge/
      github/                   # Driven: api.github.com REST client
      gitlab/                   # Driven: gitlab.com REST client (subgroup recursion via include_subgroups)
    pagereader/                 # Driven: HTTP client wrapping jina base URL
    filestore/                  # Driven: afero.OsFs behind a narrow interface
    hostrepo/                   # Driven: `git rev-parse --show-toplevel`
    configstore/                # Driven: strict yaml.v3 under $FETCH_CONTEXT_HOME
    editor/                     # Driven: $VISUAL > $EDITOR > vi via os/exec
    envx/                       # Adapter-layer helper: typed env access used by other adapters
  testing/                      # Shared test infrastructure (see docs/testing.md)
    forgemock/                  # GitHub + GitLab mock as httptest.Server
    readermock/                 # jina reader mock as httptest.Server
    gitfixture/                 # Bare-repo server via `git http-backend`
    fakes/                      # In-memory fakes for every port
  version/
    version.go                  # Version string (injectable via ldflags)
tests/
  e2e/                          # Black-box tests against the compiled binary
                                # (build tag `e2e`, one test per AC ID)
```

`internal/cli/` and `internal/config/` from the scaffold are replaced by
`internal/adapters/cli/` and `internal/adapters/configstore/` respectively.

## Layering

```
cmd/fetch-context/main.go        (wiring; the only file that imports every adapter)
  │
  ├── internal/adapters/cli/     (driving adapter: cobra → use case calls)
  │     │
  │     ▼
  ├── internal/core/             (use cases + domain services)
  │     │  (depends only on ↓)
  │     ▼
  ├── internal/ports/            (interfaces)
  │     ▲
  │     │  (implemented by ↑)
  ├── internal/adapters/<rest>/  (driven adapters: git, forge, pagereader, …)
  │
  └── third-party deps           (cobra, afero, yaml.v3, net/http, os/exec, …)
        — only adapters and wiring import these
```

The arrows are strict. A package outside `internal/core/` may import `internal/ports/`;
nothing inside `internal/core/` may import any `internal/adapters/` package.

## Ports

Each port is a small interface declared in `internal/ports/`. The core depends
on these; adapters implement them.

| Port | Purpose | Adapter(s) |
|---|---|---|
| `GitRepo` | shallow clone, fetch + hard-reset, is-managed-clone check | `adapters/gitrepo` (git CLI) |
| `ForgeEnumerator` | given a group slug, return `[]GroupRepo` with pagination + auth | `adapters/forge/github`, `adapters/forge/gitlab` |
| `PageReader` | given URL, return markdown bytes (wraps origin URL with jina base) | `adapters/pagereader` (net/http) |
| `FileStore` | write/mkdir/delete/exists/walk under target | `adapters/filestore` (afero.OsFs) |
| `HostRepoLocator` | `git rev-parse --show-toplevel` from CWD → repo root, or error | `adapters/hostrepo` (git CLI) |
| `ConfigStore` | load/save/validate the YAML config under `$FETCH_CONTEXT_HOME` | `adapters/configstore` (strict yaml.v3) |
| `Editor` | launch `$VISUAL` / `$EDITOR` / `vi` on a path, block until exit | `adapters/editor` (os/exec) |

**One forge port, two adapters.** `ForgeEnumerator` is forge-agnostic; the use
case asks for "the list of repos under this slug" and gets a flat
`[]GroupRepo` (path relative to the group + clone URL). GitLab's subgroup
recursion (`include_subgroups=true`, subgroup path preserved in the result)
and GitHub's flat enumeration are adapter-internal concerns. A future
Codeberg/Gitea adapter slots in without touching the core.

**`HostRepoLocator` is split from `GitRepo`.** Both shell out to `git`, but
they answer questions about different repos — the host workspace vs. upstream
clones. Splitting them lets the core unit tests use a constant-returning
locator and not pretend to be a git binary.

**`FileStore` is a narrow port, not `afero.Fs`.** The interface exposes only
the ~6 methods the core needs (`MkdirAll`, `WriteFile`, `Remove`, `Exists`,
`Walk`, `OpenForRead`). The real adapter wraps `afero.OsFs`; tests use
`fakes.FakeFileStore` (afero `MemMapFs`-backed) or pass a tmpdir-backed real
adapter for integration. The core never sees afero's surface.

## Adapter-layer helper: `envx`

The core has no concept of environment variables. But several adapters
(`forge/github`, `forge/gitlab`, `editor`, `configstore`) need typed env
access, and we want to unit-test their env-driven branches (e.g. `Editor`'s
`$VISUAL > $EDITOR > vi` precedence) without rebuilding the adapter per
scenario.

`internal/adapters/envx/` provides a small `Env` interface
(`Get(key) (string, bool)`) with a real `OsEnv` and a `Fake` for tests. It's
an adapter-layer concern, not a port — the core does not import it.

## Use Cases

One use case per package under `internal/core/`. Each is a struct that holds
its port dependencies and exposes one or a few methods.

| Use case | Subcommand | Ports consumed |
|---|---|---|
| `MaterializeRepo` | `repo` | `GitRepo`, `FileStore`, `HostRepoLocator` |
| `MaterializeGroup` | `group` | `ForgeEnumerator`, `GitRepo`, `FileStore`, `HostRepoLocator` |
| `MaterializeURL` | `url` | `PageReader`, `FileStore`, `HostRepoLocator` |
| `LoadProfile` | `load` | `ConfigStore`, `HostRepoLocator`, and three sub-interfaces (see below) |
| `Clean` | `clean` | `FileStore`, `HostRepoLocator`, `ConfigStore` (for `clean <profile>`) |
| `ListProfiles` | `list` | `ConfigStore`, `FileStore`, `HostRepoLocator` |
| `EditConfig` | `edit` | `Editor`, `ConfigStore`, `FileStore` (creates the config dir before editing) |

### `LoadProfile` composition

`LoadProfile` is the only use case that composes other use cases. It does so
through three small interfaces it declares for itself:

```go
type RepoMaterializer  interface { Materialize(ctx, materialize.RepoRequest)  error }
type GroupMaterializer interface { Materialize(ctx, materialize.GroupRequest) error }
type URLMaterializer   interface { Materialize(ctx, materialize.URLRequest)   error }
```

`*materialize.Repo`, `*materialize.Group`, and `*materialize.URL` satisfy
these directly, so the wiring passes the concrete use cases through. The
shape lets `LoadProfile`'s unit tests pin continue-on-error semantics (R3 in
acceptance.md) with three tiny fakes (instances of the generic
`fakes.FakeMaterializer[Req]`) instead of standing up every port the
sub-use-cases transitively need.

## Pure Domain Services

Three sub-packages under `internal/core/` hold pure functions — no port
dependencies, no I/O, no state. They are exhaustively unit-tested with
table-driven tests.

| Package | Responsibility | Specs |
|---|---|---|
| `core/urlmap` | URL → on-disk filename | R5, AC-URL-01/02/06/07 |
| `core/repoid` | Repo URL/shorthand normalization to `(host, owner, repo)` | R6, AC-REPO-06/11 |
| `core/targetpath` | Target root + repo/url spec → absolute on-disk path | derived from AC-LAYOUT-*, AC-ROOT-01 |

These are the highest-value unit tests in the codebase — the rules are stable,
the inputs are strings, the outputs are strings, and the assertions pin
behavior the acceptance criteria depend on.

## Wiring

`cmd/fetch-context/main.go` is the only file that knows about every concrete
adapter. It:

1. Reads env vars once (via `envx.OsEnv`), including the test seams
   (`GITHUB_API_URL`, `GITLAB_API_URL`, `JINA_BASE_URL`, `FETCH_CONTEXT_HOME`).
2. Constructs each adapter with its env-derived inputs and any sub-adapters.
3. Constructs each use case, injecting the ports it needs.
4. Constructs the cobra root, registering one subcommand per use case.
5. `os.Exit(rootCmd.Execute())`.

Tokens (`$GITHUB_TOKEN`, `$GITLAB_TOKEN`) are read here and passed to the
forge adapters as constructor args. The core never sees them.

## Library choices

| Library | Used by | Notes |
|---|---|---|
| Cobra | `adapters/cli` | Driving adapter. Each subcommand is a thin shim: parse args, call use case, format output. |
| Afero | `adapters/filestore` | Hidden behind the narrow `FileStore` port. The core does not import afero. |
| YAML (gopkg.in/yaml.v3) | `adapters/configstore` | Strict decoding (`KnownFields`) so malformed configs and unknown fields fail loudly (AC-CONFIG-03). Chosen over viper, whose lenient parsing would swallow exactly the errors that AC requires surfaced; `FETCH_CONTEXT_*` env vars are read directly in wiring via `envx`. |

Standard-library choices worth naming:

| Stdlib package | Used by | Notes |
|---|---|---|
| `log/slog` | wiring + adapters | Structured logging (Go 1.21+). Use cases take a `*slog.Logger` as an explicit dependency, not a global. |
| `os/exec` | `adapters/gitrepo`, `adapters/hostrepo`, `adapters/editor` | Wrapped per-adapter; the core does not import it. |
| `net/http` | `adapters/forge/*`, `adapters/pagereader` | Same; core does not import it. |

## Conventions

- The core never imports anything from `internal/adapters/` or third-party
  infrastructure SDKs. Verified mechanically with a lint or `go vet`-style
  check (TBD).
- Commands return errors from cobra's `RunE`. Wiring maps errors to exit codes
  (`1` runtime, `2` usage) per R1.
- Use cases take `context.Context` first parameter. Adapters honor it where
  meaningful (HTTP, exec).
- New external dependencies require a new port. No reaching directly into
  `net/http` or `os/exec` from `internal/core/`.

## See also

- `docs/testing.md` — the four-tier test pyramid, mock packages, and how each
  layer exercises the architecture defined here.
- `docs/acceptance.md` — observable contract every e2e test verifies.
