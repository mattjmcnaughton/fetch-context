# Testing Strategy

## Goals

1. **Unit tests carry most of the confidence.** Pure domain logic and use-case
   orchestration are exhaustively covered with fast, in-process tests.
2. **Integration tests target one adapter against one real local dependency.**
   No mocks of our own ports; the dependency on the other side is real, just
   hermetic.
3. **Contract tests verify our adapters' understanding of third-party APIs**
   still matches reality. Opt-in, credential-required, never in a PR gate.
4. **E2E tests exercise the compiled binary against every acceptance
   criterion in `docs/acceptance.md`**, in a hermetic container.

The architecture in `docs/architecture.md` is what makes (1) cheap and (3)
useful — narrow ports + pure domain services give us a large fast-test surface,
and the same in-process mocks used by e2e are reused as the always-on side of
the contract tests, which keeps the mocks honest.

## The pyramid

```
       /\
      /  \    e2e         tests/e2e/         compiled binary in Docker
     /----\               build tag e2e      ~30-60 tests (1 per AC ID)
    /      \   contract   *_contract_test    real APIs, opt-in
   /        \             build tag contract ~5-10 tests, shape-only
  /----------\  integration *_integration_test  one real adapter + local dep
 /            \            build tag integration ~15-25 tests
/--------------\ unit      *_test.go         pure + use cases with fakes
                                             100+ tests, mostly table-driven
```

Counts are illustrative; the shape — wide unit base, narrow tip — is the goal.

## Unit tests

**Location.** Same package as the code under test, no build tag. Run via
`just test`.

**Two flavors:**

### Pure domain services

The packages `core/urlmap/`, `core/repoid/`, `core/targetpath/` are pure
functions. They get exhaustive table-driven tests pinning every rule in R5/R6
and the AC-LAYOUT-* / AC-URL-* / AC-REPO-* mapping rules.

Example shape:

```go
func TestURLMap(t *testing.T) {
    cases := []struct {
        name string
        in   string
        want string
    }{
        {"path", "http://example.test/blog/post", "example.test/blog/post.md"},
        {"trailing slash", "http://example.test/blog/", "example.test/blog.md"},
        {"root", "http://example.test/", "example.test/index.md"},
        {"query", "http://example.test/p?x=1", "example.test/p__<hash>.md"},
        // ...
    }
    for _, c := range cases { /* ... */ }
}
```

These tests are the highest leverage in the codebase: the rules are stable, the
inputs and outputs are strings, and they pin behavior that every other layer
depends on.

### Use cases with fake adapters

Each use case in `internal/core/` is tested against fakes from
`internal/testing/fakes/`. Tests assert orchestration — "given the fake forge
returns these specs, the fake `GitRepo` recorded clones at these paths in this
order; an error on item 2 did not abort items 1 and 3."

`LoadProfile` tests use the three small materializer fakes
(`FakeRepoMaterializer` etc.) and pin R3 (continue-on-error) at unit speed.

**No redundant unit tests on adapters.** If a behavior is testable only with a
real dependency, write only the integration test. We do not pair every
integration test with a `httptest.NewRecorder`-style adapter unit test —
that's duplicated coverage for adapter-internal translation code.

The exceptions: genuinely tricky pure helpers *inside* adapters get their own
unit tests. Examples: GitLab pagination link-header parsing; the GitLab
subgroup-walk bookkeeping; the URL-wrapping function in `pagereader`. Each is
its own subpackage or unexported function with table tests.

## Integration tests

**Location.** Same package as the adapter under test, file suffix
`*_integration_test.go`, build tag `//go:build integration`. Run via
`just test-integration` (and as part of `just gate`).

**What "integration" means here.** Exactly one of our adapters, against a real
local instance of its external dependency. No mock of the adapter itself.

| Adapter | Real local dependency |
|---|---|
| `adapters/gitrepo` | Real `git` binary; clones from bare repos served by `internal/testing/gitfixture/` |
| `adapters/forge/github` | `httptest.Server` from `internal/testing/forgemock/` returning canned GitHub JSON |
| `adapters/forge/gitlab` | `httptest.Server` from `internal/testing/forgemock/` returning canned GitLab JSON |
| `adapters/pagereader` | `httptest.Server` from `internal/testing/readermock/` returning canned markdown |
| `adapters/filestore` | Real `os.TempDir()` workspace |
| `adapters/hostrepo` | Real `git init` in a `tmpdir` |
| `adapters/configstore` | Real YAML file in a `tmpdir` with `$FETCH_CONTEXT_HOME` pointed at it |
| `adapters/editor` | Real shell script via `EDITOR=/path/to/fake-editor.sh` |

These run on every PR.

## Contract tests

**Location.** Same package as the adapter, file suffix `*_contract_test.go`,
build tag `//go:build contract`. Run only via `just test-contract`; never in
`just gate`.

**Goal.** Confirm that what the real third-party API returns still matches
what the adapter expects to parse. We assert *shape and protocol*, not data.

| Test | Targets | Asserts |
|---|---|---|
| `forge/github` contract | `api.github.com` with `$GITHUB_TOKEN` against an arbitrary public org (configurable, default `spf13`) | Pagination link header present and parseable, JSON fields read by adapter exist, second page differs from first, 401 returned for invalid token |
| `forge/gitlab` contract | `gitlab.com` with `$GITLAB_TOKEN` against an arbitrary public group | Same kind of shape assertions; subgroup recursion endpoint returns expected shape |
| `pagereader` contract | `https://r.jina.ai/<known-stable-URL>` | 200 OK, non-empty markdown, wrapped-URL form (`<base>/<origin>`) accepted literally |

**No maintained fixture org or fixture page.** Contract tests run against
arbitrary public targets and assert protocol, not specific contents. This
costs weaker semantic drift detection in exchange for zero operational burden.

**Twin pattern: contract tests run twice.** Each contract test body is a
parametrized function:

```go
func runGitHubForgeContract(t *testing.T, baseURL string, auth Auth) { /* ... */ }
```

It's invoked in two places:

1. **Always-on, no build tag,** in `forge/github/github_integration_test.go`:
   pointed at `forgemock.New(t)`. This is what guarantees the mock satisfies
   the contract. If the mock can't pass the contract test, the mock is broken.
2. **Opt-in, build tag `contract`,** in `forge/github/github_contract_test.go`:
   pointed at `https://api.github.com` with `$GITHUB_TOKEN`. Catches real-API
   drift from the mock.

Same assertions; different process on the other side of the socket. The
relationship between integration and contract tests is therefore not just
"both test adapters" — for the forge and reader adapters, the *same test body*
runs in both modes.

(For adapters where there is no third-party REST API to drift — `gitrepo`,
`filestore`, `hostrepo`, `configstore`, `editor` — only an integration test
exists. No contract twin is meaningful.)

## E2E tests

**Location.** `tests/e2e/` (top-level, not under `internal/`). Build tag
`//go:build e2e`. Run via `just test-e2e`, which builds and runs them inside
`Dockerfile.dev`.

**Why top-level.** E2E is black-box: it exercises the compiled `$FCBIN` as a
subprocess. It does not import anything from `internal/core/`,
`internal/ports/`, or any `internal/adapters/`. Its only `internal/` imports
are the test infrastructure under `internal/testing/`.

**Layout.**

```
tests/e2e/
  main_test.go              # TestMain: boots forgemock + readermock + gitfixture once
  setup.go                  # per-test workspace helpers (mktemp, git init, FCBIN exec)
  assertions.go             # is_git, is_shallow, tree_clean — mirrors §1.6 of acceptance.md
  cli_smoke_test.go         # AC-VERSION-01, AC-USAGE-01, AC-USAGE-02
  root_resolution_test.go   # AC-ROOT-01, AC-ROOT-02
  repo_test.go              # AC-REPO-01..AC-REPO-11
  group_test.go             # AC-GROUP-01..AC-GROUP-06
  url_test.go               # AC-URL-01..AC-URL-07
  load_test.go              # AC-LOAD-01..AC-LOAD-06
  list_test.go              # AC-LIST-01..AC-LIST-03
  clean_test.go             # AC-CLEAN-01..AC-CLEAN-05
  edit_test.go              # AC-EDIT-01..AC-EDIT-03
  config_test.go            # AC-CONFIG-01..AC-CONFIG-04
  layout_test.go            # AC-LAYOUT-01..AC-LAYOUT-03
  auth_test.go              # AC-AUTH-01..AC-AUTH-03
  sandbox_test.go           # AC-SANDBOX-01, AC-SANDBOX-02
  safe_test.go              # AC-SAFE-01..AC-SAFE-04
  scope_test.go             # AC-SCOPE-01..AC-SCOPE-03
```

**One Go test function per AC ID.** Strict mapping: every `AC-*` identifier in
`docs/acceptance.md` corresponds to one Go test function with a matching name,
and only one. The pattern is `TestAC_<CATEGORY>_<NN>_<ShortName>`:

```go
func TestAC_REPO_07_RerunHardResets(t *testing.T) { /* ... */ }
func TestAC_URL_06_QueryHashSuffix(t *testing.T)  { /* ... */ }
```

This means: any change to `docs/acceptance.md` (adding, renaming, or removing
an AC ID) has a 1:1 effect on `tests/e2e/`. A reviewer can grep the AC ID and
find the test. A test that has no corresponding AC ID is wrong (delete it or
add the AC). An AC ID with no corresponding test is wrong (add the test or
remove the AC). The two documents stay in lock-step.

A small linter or pre-push check (TBD) enumerates AC IDs in `acceptance.md`
and test names in `tests/e2e/` and asserts the sets match.

**Fixture infrastructure (matches `docs/acceptance.md` §1.3).**

| Server | Package | Serves |
|---|---|---|
| Git fixture | `internal/testing/gitfixture/` | Bare repos `hello`, `alpha`, `beta`, `gamma`, `top`, `nested`, etc., over HTTP via `git http-backend` |
| Forge mock | `internal/testing/forgemock/` | GitHub `/orgs/{org}/repos` and GitLab `/groups/{group}/projects` with pagination, auth, the configured fixture org/group → clone URLs pointing back at the git fixture |
| Reader mock | `internal/testing/readermock/` | Returns canned markdown containing `ACCEPTANCE-MARKER` for any wrapped URL |

`TestMain` boots all three on loopback once and exports
`GIT_FIXTURE_URL`, `GITHUB_API_URL`, `GITLAB_API_URL`, `JINA_BASE_URL`. Each
test creates a fresh `WS=mktemp -d`, `git init`s it, exports a fresh
`FETCH_CONTEXT_HOME=mktemp -d`, and exec's `$FCBIN` as a subprocess.

**Building the binary under test.** E2E tests exercise a compiled binary, not
`go run`. The build happens in `TestMain` *unless* `$FCBIN` is already set in
the environment:

```go
func TestMain(m *testing.M) {
    fcbin := os.Getenv("FCBIN")
    if fcbin == "" {
        dir, err := os.MkdirTemp("", "fcbin-")
        // ... handle err ...
        fcbin = filepath.Join(dir, "fetch-context")
        cmd := exec.Command("go", "build", "-o", fcbin, "./cmd/fetch-context")
        cmd.Dir = repoRoot()  // walk up from this test file
        // ... run, handle err, set up cleanup ...
        os.Setenv("FCBIN", fcbin)
    }
    // ... start loopback fixtures, run m.Run(), tear down ...
}
```

This gives the suite two invocation modes that share one code path:

- **Direct (developer loop):** `go test -tags e2e ./tests/e2e/...` builds the
  binary in `TestMain` and runs. Self-contained, no `just` target needed.
- **Via `just test-e2e` (CI / Docker):** the justfile pre-builds the binary
  with `go build -o ./bin/fetch-context ./cmd/fetch-context`, exports
  `FCBIN=$(pwd)/bin/fetch-context`, and then runs `go test -tags e2e
  ./tests/e2e/...`. `TestMain` sees `$FCBIN` already set and skips the build.

The pre-built path is what `Dockerfile.dev` uses: the binary is built into a
layer that can be cached, and the test layer only re-runs when test code
changes.

**Docker.** `just test-e2e` invokes the suite inside `Dockerfile.dev` with the
local source bind-mounted. The container has no outbound network; every
external dependency is the loopback fixture (`forgemock`, `readermock`,
`gitfixture`).

## Shared test infrastructure

All under `internal/testing/`, importable from any test layer.

| Package | Purpose | Used by |
|---|---|---|
| `internal/testing/fakes/` | One in-memory fake per port (`FakeGitRepo`, `FakeForgeEnumerator`, `FakeFileStore`, `FakeHostRepoLocator`, `FakePageReader`, `FakeConfigStore`, `FakeEditor`), plus the three `Fake*Materializer` for `LoadProfile` | Unit tests in `internal/core/` |
| `internal/testing/forgemock/` | `httptest.Server` implementing the GitHub + GitLab endpoints fetch-context calls, with pagination, auth, and a small control surface for fixture seeding | Adapter integration tests, contract-twin tests, e2e |
| `internal/testing/readermock/` | `httptest.Server` returning canned markdown for any wrapped URL | `pagereader` integration tests, contract-twin tests, e2e |
| `internal/testing/gitfixture/` | Bare-repo HTTP server backed by `git http-backend` exec; canned repos seeded into `os.TempDir()` | `gitrepo` integration tests, e2e |

The mocks are not standalone binaries — they are Go packages that expose a
`New(t *testing.T) *Server` (or similar) and clean themselves up via
`t.Cleanup`. This is what lets the same fixture be used from in-process Go
tests at every layer.

## Build tags

| Tag | Meaning | Run by |
|---|---|---|
| _none_ | Pure unit tests; fast; no I/O outside the Go test runtime | `just test`, `just gate` |
| `integration` | Touches a real local dependency (filesystem, exec'd git, httptest.Server) | `just test-integration`, `just gate` |
| `contract` | Touches a real third-party API; requires credentials | `just test-contract` only |
| `e2e` | Exec's the compiled binary against the three loopback fixtures | `just test-e2e` only (run in Docker) |

The contract twin tests are **not** tagged — the always-on side runs as part of
the integration tag, pointed at the mock. The tagged `contract` file is the
opposite parametrization of the same test body.

## `just` targets

| Target | Runs | When |
|---|---|---|
| `just test` | Unit tests only | Every save, every PR |
| `just test-integration` | Integration tests (build tag `integration`) | Every PR |
| `just test-all` | Unit + integration | Every PR |
| `just test-contract` | Contract tests (build tag `contract`); requires `$GITHUB_TOKEN`, `$GITLAB_TOKEN` | Manual / nightly |
| `just test-e2e` | E2E suite in Docker | Manual / before release |
| `just gate` | `fmt` + `vet` + `test` + `test-integration` | Pre-push |
| `just gate-expensive` | `gate` + `test-e2e` | Pre-release |

Contract tests are intentionally excluded from every gate. Real-API flakiness
should not block a PR; the value of contract tests is detecting drift over
time, which a scheduled or manual run captures fine.

## Mapping back to `docs/acceptance.md`

Acceptance criteria are the contract; e2e tests are their executable form.
The relationship is enforced two ways:

1. **One Go test per AC ID,** named to encode the ID, as described above.
2. **A `tests/e2e/coverage_test.go`** (or similar) walks the parsed AC IDs and
   the registered test names and fails if either set has an entry the other
   doesn't. This catches AC drift and forgotten tests at build time.

Anything in `acceptance.md` that the e2e suite cannot run hermetically (e.g.
hypothetical scenarios requiring real `github.com`) is — per §1 of that doc —
not a valid acceptance criterion. The criterion gets rewritten to use the
fixture topology, or moved to the contract suite.

## See also

- `docs/architecture.md` — the hexagonal layout that makes this test pyramid
  cheap to populate.
- `docs/acceptance.md` — the AC IDs every e2e test maps to, and the hermetic
  container topology e2e relies on.
