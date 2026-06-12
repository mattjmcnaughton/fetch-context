# ADR-0001: Build decisions not pinned by R1–R8

Status: **accepted** (2026-06-12; every decision below was confirmed during
the initial build — decision 4 was revised in flight, see its section)

## Context

`docs/acceptance.md` §17 resolves eight design questions (R1–R8). Implementing
the application surfaces a small number of further decisions that no R-item or
AC pins. They are recorded here so the rationale survives the build.

## Decision 1 — Forge dispatch by host

`group` must pick the GitHub or GitLab adapter from the spec's host.
**Decision:** the wiring constructs a `map[string]ports.ForgeEnumerator` keyed
by host kind (`github` / `gitlab`); `MaterializeGroup` selects by the host of
the parsed group spec. `github.com` maps to the GitHub enumerator,
`gitlab.com` to the GitLab one; any other host is a **usage error** (exit 2) —
the tool cannot know which API dialect an unknown host speaks. The e2e seams
`GITHUB_API_URL` / `GITLAB_API_URL` only move each adapter's base URL; host →
kind mapping is unaffected.

Consequence: a future Codeberg/Gitea adapter is a new map entry in wiring, no
core change.

## Decision 2 — `UsageError` type

R1 pins exit `2` for usage errors, and AC-LOAD-04 shows "usage" extends beyond
cobra flag parsing (unknown profile → 2). **Decision:** the core defines a
`UsageError` wrapper (`internal/core/usageerr`) that use cases return for
caller-mistake failures (unknown profile, malformed spec, unknown forge host).
The CLI adapter maps: cobra parse errors and `UsageError` → exit 2; every
other error → exit 1. The mapping lives in one function
(`adapters/cli.ExitCode`) used by `main.go`.

## Decision 3 — gitfixture auth approach

AC-AUTH-02/03 need a token-gated repo served hermetically. **Decision:** the
fixture git server is a small Go `http.Handler` that fronts
`git http-backend` via CGI (`net/http/cgi`). Token gating is HTTP Basic auth
enforced in the Go handler for repos marked private — no `httpd` config, no
real credential store. The client side passes tokens the same way production
does: an `Authorization` header / URL credential injected by the `gitrepo`
adapter when a token is present for the host.

## Decision 4 — How tokens reach `git clone`

Not pinned by the docs: the mechanics of using `GITHUB_TOKEN` for a private
*clone* (as opposed to forge enumeration). **Decision (revised during
implementation):** the `gitrepo` adapter receives an ordered list of
credentials (kind + token) at construction. Every clone/fetch is attempted
unauthenticated first; on an auth-shaped failure each credential is retried
once via a per-invocation `-c http.extraHeader=Authorization: Basic …` flag
(username `x-access-token` for GitHub-kind, `oauth2` for GitLab-kind).
Rationale for the revision: a per-host map cannot decide which token applies
to a host it has never heard of — including the loopback fixture host the
hermetic e2e suite uses — whereas try-then-retry needs no host knowledge, at
the cost of one extra round trip on private repos. Nothing is written to disk
and the token never appears in the clone's saved remote URL.

## Decision 5 — "Managed clone" detection

AC-REPO-07/08 distinguish a refreshable clone from an unmanaged directory.
**Decision:** a destination is a managed clone iff `git -C <dest> rev-parse
--git-dir` succeeds with `<dest>/.git` as the result (it is itself a working
tree root, not merely inside one). Anything else that exists at the
destination is unmanaged → error, untouched.
