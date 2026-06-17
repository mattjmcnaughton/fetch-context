# fetch-context — Acceptance Criteria

This document defines the observable behavior `fetch-context` must exhibit to be
considered correct. Each criterion is a Given / When / Then scenario with
concrete commands and checkable assertions, so it can be run by hand or wired
into an integration suite. Criteria are referenced by ID (e.g. `AC-REPO-03`).

A scenario **passes** only if every assertion in its **Then** holds. Unless a
scenario states otherwise, success means exit code `0` and failure means a
non-zero exit code.

---

All criteria run **inside a single container** with no access to the public
internet. There is no install step: the binary under test is **built locally**
from source, and every external dependency (git remotes, the forge API, the
page-reader proxy) is served by a process inside the same container. A scenario
that cannot pass without reaching `github.com`, `gitlab.com`, or `r.jina.ai` is
not a valid acceptance criterion.

### 1.1 Container preconditions

The container provides:

- A Go toolchain and a `git` binary on `PATH`.
- The source tree, from which the suite builds the binary once:
  ```bash
  go build -o "$FCBIN" ./cmd/fetch-context     # $FCBIN is an absolute path
  ```
  Every scenario invokes `"$FCBIN"` (written `fetch-context` below for brevity).
- No outbound network. All remotes resolve to loopback.

### 1.2 Required injection seams

Container-hermetic execution is only possible if the tool lets its three
outbound dependencies be redirected to loopback. These seams are therefore part
of the contract, not just the test rig:

| Seam (env var) | Redirects | Used by |
|---|---|---|
| `GITHUB_API_URL` | GitHub REST base URL → mock forge | `group` (GitHub) |
| `GITLAB_API_URL` | GitLab REST base URL → mock forge | `group` (GitLab) |
| `JINA_BASE_URL` | page-reader proxy base → mock reader | `url` |
| `GITHUB_TOKEN` / `GITLAB_TOKEN` | auth credentials | `group`, private repos |
| `FETCH_CONTEXT_HOME` | config root | all (config) |

Git remotes need no special seam: clone URLs already point wherever the fixture
git server lives (loopback).

### 1.3 In-container fixture topology

Three loopback servers, started once for the suite (e.g. in `TestMain` / a
`setup_suite`), each returning canned data:

1. **Git server** — serves bare fixture repos over `http://127.0.0.1:$GIT_PORT/`.
   Seed repos: `hello` (contains file `MARKER`), `alpha`, `beta`, `gamma`,
   `top`, `nested`. (`git http-backend`, `git daemon`, or a tiny Go handler.)
2. **Mock forge API** — `http://127.0.0.1:$API_PORT/`. Implements only the
   list-repos endpoints the tool calls. GitHub fixture org returns
   `{alpha, beta, gamma}`. GitLab fixture group returns `top` plus a subgroup
   `sub` containing `nested`. Every entry's clone URL points back at the git
   server. Supports a second page to exercise pagination, and returns `401/404`
   for the "private" fixture when no token is presented.
3. **Mock reader proxy** — `http://127.0.0.1:$JINA_PORT/`. Echoes back canned
   markdown containing `ACCEPTANCE-MARKER` for any wrapped URL.

Fixture variables used by the scenarios:

| Variable | Resolves to |
|---|---|
| `$GH_REPO` | `http://127.0.0.1:$GIT_PORT/hello.git` (single-repo happy path) |
| `$GH_ORG` | the org slug the mock forge maps to `{alpha, beta, gamma}` |
| `$GL_GROUP` | the group slug the mock forge maps to `top` + `sub/nested` |
| `$PRIVATE` | a forge slug that requires a token; unauthenticated → error |
| `$URL_PAGE` | a URL with path `/blog/post` (reader returns `ACCEPTANCE-MARKER`) |
| `$URL_ROOT` | a URL at a bare host root, no path |

**Host segment in path assertions.** Because fixture remotes live on loopback,
the host segment the tool derives is the fixture host (e.g. `127.0.0.1` or a
name spoofed via `/etc/hosts`), **not** `github.com`. Scenarios below write
`github.com/foo/bar` for readability; in a run, assert the *derived structure*
`repos/<host>/<owner>/<repo>/` against the fixture's actual host. Choosing
whether to keep loopback hosts or spoof real hostnames via `/etc/hosts` + a
local CA is a harness decision, captured in the e2e mapping doc.

### 1.4 Per-scenario isolation

Every scenario runs against a clean slate; only the three suite-level servers
persist across scenarios.

```bash
setup() {
  WS="$(mktemp -d)"                          # workspace = the "current repo"
  git -C "$WS" init -q
  export FETCH_CONTEXT_HOME="$(mktemp -d)"   # isolated config root
  export GITHUB_API_URL="http://127.0.0.1:$API_PORT"
  export GITLAB_API_URL="http://127.0.0.1:$API_PORT"
  export JINA_BASE_URL="http://127.0.0.1:$JINA_PORT"
  cd "$WS"
}
teardown() {
  rm -rf "$WS" "$FETCH_CONTEXT_HOME"
}
```

`WS` is a real git repo so `git rev-parse --show-toplevel` resolves. The resolved
target for a clean workspace is `"$WS"/.agentic/sources`.

### 1.5 Exit-code convention

These codes are part of the contract (see [Resolved decisions](#17-resolved-decisions), R1).

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Runtime failure (clone failed, auth failed, conflict) |
| `2` | Usage error (unknown command, missing/invalid argument) |

### 1.6 Helper assertions

- `is_git <dir>` → `git -C <dir> rev-parse --git-dir` succeeds.
- `is_shallow <dir>` → `git -C <dir> rev-parse --is-shallow-repository` prints `true`.
- `tree_clean <dir>` → `git -C <dir> status --porcelain` is empty.

---

## 2. CLI smoke

Preconditions: the binary has been built locally to `$FCBIN` (§1.1). There is no
install criterion — building is a harness step, not a behavior under test.

**AC-VERSION-01 — version prints**
- When: `fetch-context version`.
- Then: exit `0`; stdout contains a non-empty version string.

**AC-USAGE-01 — no args prints usage**
- When: `fetch-context` (no subcommand).
- Then: exit `2`; usage text printed to stderr.

**AC-USAGE-02 — unknown subcommand**
- When: `fetch-context frobnicate`.
- Then: exit `2`; stderr names the unknown command.

---

## 3. Repo-root & target resolution

**AC-ROOT-01 — target is resolved against repo root, not CWD**
- Given: workspace `WS`; a nested dir `WS/a/b`; CWD = `WS/a/b`.
- When: `fetch-context repo $GH_REPO`.
- Then: content appears under `WS/.agentic/sources/repos/...` (repo root), **not**
  under `WS/a/b/.agentic/...`.

**AC-ROOT-02 — outside a git repo**
- Given: CWD is a plain directory that is not inside any git repo.
- When: `fetch-context repo $GH_REPO`.
- Then: exit `1`; stderr explains that a repo root could not be resolved; no
  `.agentic/` directory is created.

---

## 4. `repo`

**AC-REPO-01 — single public repo, correct layout**
- Given: clean workspace; no token.
- When: `fetch-context repo $GH_REPO`.
- Then: exit `0`; `is_git .agentic/sources/repos/github.com/<owner>/<repo>`; the
  known `MARKER` file is present in the clone.

**AC-REPO-02 — clone is shallow, default branch**
- When: `fetch-context repo $GH_REPO`.
- Then: `is_shallow` on the clone is true; the checked-out branch is the remote's
  default branch.

**AC-REPO-03 — auto-gitignore written**
- Given: clean workspace with no `.agentic/`.
- When: `fetch-context repo $GH_REPO`.
- Then: `.agentic/sources/.gitignore` exists and its content is exactly `*`.

**AC-REPO-04 — tree is actually ignored by git**
- When: after AC-REPO-03, run `git -C "$WS" status --porcelain`.
- Then: output is empty (nothing under `.agentic/` is shown as untracked).

**AC-REPO-05 — multiple repos in one invocation**
- When: `fetch-context repo $GH_REPO github.com/<other>/<repo>`.
- Then: exit `0`; both clones present at their respective `host/owner/repo` paths.

**AC-REPO-06 — full clone URL accepted**
- When: `fetch-context repo https://github.com/<owner>/<repo>.git`.
- Then: exit `0`; clone lands at `repos/github.com/<owner>/<repo>/` (host/owner/repo,
  `.git` suffix and scheme normalized away).

**AC-REPO-07 — re-run fetches and hard-resets to remote latest**
- Given: `$GH_REPO` already cloned (AC-REPO-01); a local edit and a new untracked
  file injected into the clone.
- When: `fetch-context repo $GH_REPO` again.
- Then: exit `0`; `tree_clean` on the clone is true; the injected edit and
  untracked file are gone; working tree matches remote HEAD.

**AC-REPO-08 — destination exists but is not a git repo → refuse**
- Given: `mkdir -p .agentic/sources/repos/github.com/<owner>/<repo>` and
  `touch …/<repo>/SENTINEL` (a non-git directory at the destination).
- When: `fetch-context repo $GH_REPO`.
- Then: exit `1`; stderr says the destination exists and is not a managed clone;
  `SENTINEL` still present; no clone performed.

**AC-REPO-09 — bad repo reference**
- When: `fetch-context repo github.com/<owner>/does-not-exist-xyz`.
- Then: exit `1`; stderr reports the clone failure; no partial directory left at
  the destination.

**AC-REPO-10 — mixed batch continues on error**
- When: `fetch-context repo $GH_REPO github.com/<owner>/does-not-exist-xyz`.
- Then: exit `1`; the good clone for `$GH_REPO` is fully present at its
  `host/owner/repo` path; stderr lists the failed item with its reason; the
  failed entry left no partial directory.

**AC-REPO-11 — equivalent URL forms collapse to one clone**
- When: `fetch-context repo foo/bar foo/bar/ foo/bar.git` (and/or the full
  `https://<host>/foo/bar.git` form) in a single invocation.
- Then: exit `0`; exactly one clone lands at `repos/<host>/foo/bar/`; no
  sibling or duplicate destination is created.

**AC-REPO-12 — `--depth 0` clones full history**
- Given: a fixture repo with more than one commit.
- When: `fetch-context repo --depth 0 $GH_REPO`.
- Then: exit `0`; `is_shallow` on the clone is false; `git rev-list --count
  HEAD` equals the remote's full commit count.

**AC-REPO-13 — `--branch` clones the named branch**
- Given: a fixture repo with a non-default branch carrying distinct content.
- When: `fetch-context repo --branch <branch> $GH_REPO`.
- Then: exit `0`; the checked-out branch is `<branch>`; the working tree has
  the branch's content.

**AC-REPO-14 — re-run with `--depth 0` keeps full history**
- Given: a full-history clone (AC-REPO-12); the remote advances by one commit.
- When: `fetch-context repo --depth 0 $GH_REPO` again.
- Then: exit `0`; the clone holds the new commit; `is_shallow` is still false
  (the refresh converges to the requested depth instead of re-shallowing).

---

## 5. `group`

**AC-GROUP-01 — GitHub org enumerates flat**
- Given: `GITHUB_TOKEN` set (or org is public).
- When: `fetch-context group $GH_ORG`.
- Then: exit `0`; clones exist for `alpha`, `beta`, `gamma` at
  `repos/github.com/<org>/<name>/`; no extra repos appear.

**AC-GROUP-02 — GitLab group recurses and preserves subgroup path**
- When: `fetch-context group $GL_GROUP`.
- Then: exit `0`; `repos/gitlab.com/<group>/top/` exists **and**
  `repos/gitlab.com/<group>/sub/nested/` exists — the subgroup segment `sub` is
  preserved in the path.

**AC-GROUP-03 — pagination**
- Given: an org/group with more repos than one API page.
- When: `fetch-context group <large>`.
- Then: every repo across all pages is cloned (count matches the API total).

**AC-GROUP-04 — each enumerated repo obeys repo rules**
- When: `fetch-context group $GH_ORG` then mutate one clone and re-run.
- Then: clones are shallow; re-run hard-resets the mutated clone (per AC-REPO-07).

**AC-GROUP-05 — missing token for a private group → reported, not skipped**
- Given: `GITHUB_TOKEN` unset.
- When: `fetch-context group $PRIVATE`.
- Then: exit `1`; stderr names an auth/permission failure; nothing is silently
  skipped and the command does not exit `0`.

**AC-GROUP-06 — one bad repo in a group does not abort the rest**
- Given: mock forge enumerates `{alpha, beta, gamma}` for `$GH_ORG`, but the
  git server is configured to fail clones of `beta` (e.g. 404).
- When: `fetch-context group $GH_ORG`.
- Then: exit `1`; `alpha` and `gamma` are fully cloned at their paths;
  stderr names `beta` as failed with its reason; no partial directory for
  `beta` is left behind.

**AC-GROUP-07 — `--depth 0` applies to every enumerated clone**
- Given: mock forge enumerates `$GH_ORG`, whose repos include one with more
  than one commit.
- When: `fetch-context group --depth 0 $GH_ORG`.
- Then: exit `0`; every cloned repo has full history (`is_shallow` false);
  the multi-commit repo's `git rev-list --count HEAD` equals its remote
  commit count. (Group repos always track the remote default branch — only
  depth is configurable for groups.)

**AC-GROUP-08 — parallel clones keep batch semantics**
- Given: mock forge enumerates `{alpha, beta, gamma}` for `$GH_ORG`; the git
  server fails clones of `beta`.
- When: `fetch-context group --parallel 4 $GH_ORG`.
- Then: exit `1`; `alpha` and `gamma` are fully cloned; stderr names `beta`
  with its reason; no partial directory for `beta`. (Concurrency itself is
  not observable from outside — the criterion is that bounded parallelism
  preserves continue-on-error and the deterministic per-item summary.)

---

## 6. `url`

**AC-URL-01 — fetch to markdown at host/path**
- When: `fetch-context url $URL_PAGE` (path `/blog/post`).
- Then: exit `0`; file `urls/<host>/blog/post.md` exists, is non-empty, and
  contains `ACCEPTANCE-MARKER`.

**AC-URL-02 — root URL → index.md**
- When: `fetch-context url $URL_ROOT` (no path).
- Then: file `urls/<host>/index.md` exists and is non-empty.

**AC-URL-03 — re-fetch overwrites**
- Given: AC-URL-01 has run; the `.md` file is then overwritten with `STALE`.
- When: `fetch-context url $URL_PAGE` again.
- Then: file no longer contains `STALE`; contains `ACCEPTANCE-MARKER` again.

**AC-URL-04 — proxy wrapping**
- When: `fetch-context url http://example.test/x` with request tracing enabled.
- Then: the outbound fetch targets `$JINA_BASE_URL/http://example.test/x`
  (origin URL appended literally to the configured reader base, not
  percent-encoded). In production `$JINA_BASE_URL` is `https://r.jina.ai`; in the
  container it is the mock reader.

**AC-URL-05 — multiple URLs in one invocation**
- When: `fetch-context url $URL_PAGE $URL_ROOT`.
- Then: both markdown files are written at their respective paths; exit `0`.

**AC-URL-06 — query string disambiguates via hash suffix**
- When: `fetch-context url http://example.test/blog/post http://example.test/blog/post?x=1 http://example.test/blog/post?x=2`.
- Then: exit `0`; three distinct files exist:
  - `urls/example.test/blog/post.md` (no query)
  - `urls/example.test/blog/post__<hash1>.md` (for `?x=1`)
  - `urls/example.test/blog/post__<hash2>.md` (for `?x=2`)
  where `<hashN>` is a stable, deterministic short hash of the query string
  (e.g. first 8 hex chars of SHA-256). The two query variants do **not**
  silently overwrite each other or the clean-path file.

**AC-URL-07 — trailing slash collapses to the same file**
- When: `fetch-context url http://example.test/blog http://example.test/blog/`.
- Then: exit `0`; exactly one file exists: `urls/example.test/blog.md`. The
  trailing slash is stripped before mapping; both forms refer to the same
  resource. (Root `/` remains the only case that maps to `index.md`, per
  AC-URL-02 — it is the one path with no filename to derive.)

---

## 7. `load`

**AC-LOAD-01 — materializes all keys of a profile**
- Given: config defining profile `backend` with one `repos` entry, one `groups`
  entry, and one `urls` entry.
- When: `fetch-context load backend`.
- Then: exit `0`; the repo is cloned, the group's repos are cloned, and the URL's
  markdown is written — all under the resolved target.

**AC-LOAD-02 — per-profile target override**
- Given: profile `backend` sets `target: .agentic/backend`.
- When: `fetch-context load backend`.
- Then: content appears under `.agentic/backend/repos|urls/...`, and **not** under
  `.agentic/sources/`.

**AC-LOAD-03 — missing profile name**
- When: `fetch-context load`.
- Then: exit `2`; usage/error printed.

**AC-LOAD-04 — unknown profile**
- When: `fetch-context load no-such-profile`.
- Then: exit `2`; stderr names the unknown profile; nothing materialized.

**AC-LOAD-05 — no implicit profile**
- Given: any config.
- When: `fetch-context load` with no argument.
- Then: it never "guesses" or auto-selects a profile (consistent with AC-LOAD-03).

**AC-LOAD-06 — partial failure in a profile is reported, not fatal-on-first**
- Given: profile `backend` with one good `repos` entry, one bad `repos` entry
  (does-not-exist), one good `urls` entry.
- When: `fetch-context load backend`.
- Then: exit `1`; the good repo is cloned; the good URL's markdown is written;
  stderr names the bad repo with its failure reason; no partial directory is
  left for the bad repo.

**AC-LOAD-08 — `--parallel` accepted and batch semantics kept**
- Given: a profile with several repos entries.
- When: `fetch-context load --parallel 4 <profile>`.
- Then: exit `0`; every entry is materialized (smoke — the bound itself is
  not externally observable). `--parallel 0` is a usage error (exit `2`).

**AC-LOAD-07 — repo entry mapping form honored**
- Given: profile whose `repos` list mixes a plain ref string and a mapping
  `{ref: …, depth: 0, branch: <branch>}` (a fixture repo with several commits
  and a non-default branch).
- When: `fetch-context load <profile>`.
- Then: exit `0`; the plain entry is cloned shallow on the default branch;
  the mapping entry's clone has full history and the named branch checked
  out.

---

## 8. `list`

**AC-LIST-01 — shows configured profiles and contents**
- Given: config with profiles `backend` and `web-stack`.
- When: `fetch-context list`.
- Then: exit `0`; stdout names both profiles and their `repos`/`groups`/`urls`
  entries.

**AC-LIST-02 — reports materialized state**
- Given: `backend` has been `load`ed.
- When: `fetch-context list`.
- Then: stdout indicates which entries are currently materialized on disk under
  the resolved target.

**AC-LIST-03 — empty config**
- Given: no profiles defined.
- When: `fetch-context list`.
- Then: exit `0`; a clear "no profiles" message (not an error).

---

## 9. `clean`

**AC-CLEAN-01 — clean removes the whole target**
- Given: both `repos/` and `urls/` materialized.
- When: `fetch-context clean`.
- Then: exit `0`; `.agentic/sources/repos` and `.agentic/sources/urls` are gone.

**AC-CLEAN-02 — scoped clean (repos)**
- When: `fetch-context clean repos`.
- Then: `repos/` removed; `urls/` untouched.

**AC-CLEAN-03 — scoped clean (urls)**
- When: `fetch-context clean urls`.
- Then: `urls/` removed; `repos/` untouched.

**AC-CLEAN-04 — never deletes outside the target**
- Given: a sentinel file `WS/keep.txt` and content in `WS/.agentic/sources/`.
- When: `fetch-context clean`.
- Then: `WS/keep.txt` still exists; only paths beneath the resolved target were
  removed.

**AC-CLEAN-05 — `clean <profile>` clears that profile's target only**
- Given: profile `backend` sets `target: .agentic/backend` and has been
  `load`ed; profile `web` sets `target: .agentic/web` and has been `load`ed;
  global target `.agentic/sources` also has content from an ad-hoc `repo`
  command.
- When: `fetch-context clean backend`.
- Then: exit `0`; `.agentic/backend/` is removed; `.agentic/web/` is
  untouched; `.agentic/sources/` is untouched. The command never
  auto-discovers other profiles' targets.

---

## 10. `edit`

**AC-EDIT-01 — opens editor and reloads valid edits**
- Given: `EDITOR` set to a script that appends a valid profile to the config.
- When: `fetch-context edit`.
- Then: exit `0`; the new profile is present and a subsequent `list` shows it.

**AC-EDIT-02 — invalid edit is rejected, file preserved**
- Given: `EDITOR` set to a script that writes malformed YAML.
- When: `fetch-context edit`.
- Then: exit non-zero; stderr reports the validation error; the malformed file
  remains on disk (not reverted, not deleted).

**AC-EDIT-03 — editor precedence $VISUAL > $EDITOR > vi**
- Given: both `VISUAL` and `EDITOR` set to distinguishable marker scripts.
- When: `fetch-context edit`.
- Then: the `VISUAL` script ran and the `EDITOR` script did not.

---

## 11. Config parsing & validation

**AC-CONFIG-01 — config path honors FETCH_CONTEXT_HOME**
- Given: `FETCH_CONTEXT_HOME=$CFG`.
- When: any command that reads/writes config (e.g. `edit`).
- Then: the file used is `$CFG/.config/fetch-context/config.yaml`.

**AC-CONFIG-02 — global target override honored**
- Given: config sets top-level `target: .agentic/ctx`.
- When: `fetch-context repo $GH_REPO`.
- Then: clone lands under `.agentic/ctx/repos/...`.

**AC-CONFIG-03 — malformed config errors clearly**
- Given: config file containing invalid YAML.
- When: `fetch-context list`.
- Then: exit non-zero; stderr identifies the parse error; no partial action taken.

**AC-CONFIG-04 — missing config is not fatal for one-offs**
- Given: no config file exists.
- When: `fetch-context repo $GH_REPO`.
- Then: exit `0` (one-off commands need no config); default target is used.

**AC-CONFIG-05 — global clone depth honored**
- Given: config sets `clone: {depth: 0}`; a fixture repo with more than one
  commit.
- When: `fetch-context repo $GH_REPO` (no `--depth` flag).
- Then: exit `0`; the clone has full history (`is_shallow` false).

**AC-CONFIG-06 — unknown field in a repo entry mapping errors clearly**
- Given: config with a profile repo entry `{ref: a/b, brnch: oops}`.
- When: `fetch-context list`.
- Then: exit non-zero; stderr names the unknown field and its line; no
  partial action taken.

**AC-CONFIG-07 — clone.parallel validated and effective**
- Given: config sets `clone: {parallel: 2}`.
- When: `fetch-context group $GH_ORG`.
- Then: exit `0`; every repo is cloned (a smoke check — the bound itself is
  not externally observable). A config with `clone: {parallel: 0}` instead
  fails loudly on any command that loads config.

---

## 12. File layout & auto-gitignore

**AC-LAYOUT-01 — repos/ and urls/ live beneath target**
- When: a `repo` and a `url` are materialized.
- Then: `repos/` and `urls/` are siblings directly under the resolved target.

**AC-LAYOUT-02 — host/owner/repo nesting**
- When: cloning `github.com/foo/bar`.
- Then: path is exactly `repos/github.com/foo/bar/` (host included).

**AC-LAYOUT-03 — gitignore is idempotent**
- Given: `.agentic/sources/.gitignore` already exists with `*`.
- When: another materialization runs.
- Then: file is unchanged (still exactly `*`, not duplicated/appended).

---

## 13. Authentication

**AC-AUTH-01 — public single repo needs no token**
- Given: no `GITHUB_TOKEN`.
- When: `fetch-context repo $GH_REPO` (public).
- Then: exit `0`.

**AC-AUTH-02 — token used for private repo**
- Given: `GITHUB_TOKEN` with access to a private repo.
- When: `fetch-context repo <private-repo>`.
- Then: exit `0`; repo cloned.

**AC-AUTH-03 — auth failure surfaced, not swallowed**
- Given: `GITHUB_TOKEN` unset; a private target.
- When: `fetch-context repo <private-repo>` or `group $PRIVATE`.
- Then: exit `1`; stderr states the auth/permission problem explicitly.

---

## 14. Sandboxing

**AC-SANDBOX-01 — FETCH_CONTEXT_HOME redirects config**
- Covered by AC-CONFIG-01: config resolves under the sandbox root.

**AC-SANDBOX-02 — target stays repo-local regardless of sandbox**
- Given: `FETCH_CONTEXT_HOME=$CFG` (some path unrelated to `WS`).
- When: `fetch-context repo $GH_REPO` inside `WS`.
- Then: clone lands under `WS/.agentic/sources/...`, **not** under `$CFG`.

---

## 15. Conflict & safety behavior

**AC-SAFE-01 — managed clone is refreshed**
- Covered by AC-REPO-07 (fetch + hard reset).

**AC-SAFE-02 — unmanaged directory is never clobbered**
- Covered by AC-REPO-08 (error, sentinel preserved).

**AC-SAFE-03 — url markdown is overwritten on refetch**
- Covered by AC-URL-03.

**AC-SAFE-04 — clean is target-scoped**
- Covered by AC-CLEAN-04.

---

## 16. Scope / negative assertions

**AC-SCOPE-01 — no lockfile is ever written**
- When: any combination of `repo`, `group`, `url`, `load` runs.
- Then: no `*.lock` file appears under the target, the config dir, or `WS`.

**AC-SCOPE-02 — no commit pinning ceremony**
- When: repos are cloned.
- Then: clones are shallow (`is_shallow` true) and track a branch — there is no
  detached-HEAD pin to a recorded SHA.

**AC-SCOPE-03 — nothing committed to the host repo**
- When: any materialization runs in `WS`.
- Then: `git -C "$WS" status --porcelain` is empty; `git -C "$WS" log` shows no
  commits created by the tool.

---

## 17. Resolved decisions

These were "Open questions" in earlier drafts. Each is now pinned; rationale is
kept here so a future reader can understand why scenarios were authored the way
they were.

**R1 — Exit codes.** Pinned as `0` (success) / `1` (runtime failure) / `2`
(usage error), matching §1.5. Standard POSIX shape; lets scripts distinguish
misuse from runtime error.

**R2 — Hermetic forge simulation.** The mock forge in §1.3 is a hand-rolled
HTTP handler running in-process inside the acceptance test binary. It mirrors
GitHub's and GitLab's exact list-repos contracts (URL paths, query parameters,
pagination headers, JSON shape, error codes). Hostnames stay loopback
(`127.0.0.1`); scenarios assert the *derived structure* `repos/<host>/<owner>/<repo>/`
per §1.3, not a literal `github.com` segment. Real-API contract fidelity is
validated separately in dedicated contract/integration tests that live outside
this acceptance suite — they exist to detect drift between the mock and the
real APIs.

**R3 — Partial failure in batches.** All multi-item commands (`repo a b c`,
`group <many-repos>`, `load <profile>`) use **continue-on-error** semantics:
every item is attempted; failures are recorded; the command exits `0` if all
succeeded and `1` if any failed, with a per-item summary on stderr naming
which items failed and why. Rationale: this is a best-effort context loader,
not a transactional operation — one bad repo should not waste the rest of the
batch. See AC-REPO-10, AC-GROUP-06, AC-LOAD-06.

**R4 — Outside a git repo.** Hard error, exit `1`. The repo-local target model
is part of the contract; falling back to CWD would scatter `.agentic/`
directories in unrelated working dirs. See AC-ROOT-02.

**R5 — URL → filename mapping.** Rules:
- Strip scheme. Host becomes the first directory under `urls/`.
- Path segments become directory segments; the final segment + `.md` becomes
  the file. Unsafe characters are percent-decoded then sanitized to `_`.
- A trailing slash is stripped before mapping: `/blog` and `/blog/` resolve
  to the same file (`blog.md`). Rationale: the reader proxy follows
  redirects, so the two forms almost always return identical content; keeping
  them distinct would just produce stale near-duplicates on disk. Root `/`
  is the one exception — it has no filename to derive, so it maps to
  `<host>/index.md` (AC-URL-02).
- A query string, if present, is hashed (first 8 hex chars of
  SHA-256(query-string)) and appended as `__<hash>` before `.md`. Clean URLs
  stay clean; URLs differing only by query string land at distinct files and
  never silently overwrite the clean-path file.

See AC-URL-06 and AC-URL-07.

**R6 — Repo URL normalization.** `foo/bar`, `foo/bar/`, `foo/bar.git`,
`https://<host>/foo/bar.git`, and the SSH forms `git@<host>:foo/bar.git` /
`ssh://git@<host>/foo/bar.git` all normalize to one destination
`repos/<host>/foo/bar/`. The same surface form appearing twice in one
invocation produces one clone, not two. SSH refs clone over SSH (the clone
URL preserves the SSH user and scp-like vs `ssh://` form); HTTP(S) and
host-qualified forms clone over HTTPS. See AC-REPO-11.

**R7 — `clean` and per-profile targets.** `clean` (no argument) and `clean
repos|urls` continue to operate on the resolved global target only. A new
`clean <profile>` form clears the target a profile resolved to (which may
differ from the global target via the profile's `target:` override). `clean`
never auto-discovers other profiles' targets, to keep the blast radius
predictable. See AC-CLEAN-05.

**R8 — URL secret handling.** Advisory only. The README documents that URLs
are forwarded verbatim to the reader proxy (`r.jina.ai` in production), so any
secrets in query strings (`?token=`, `?api_key=`, pre-signed S3 signatures) or
in `user:pass@host` are exposed to that third party. The tool does **not**
attempt heuristic detection — incomplete detection creates a false sense of
safety, and the realistic use case (fetching public docs) does not carry
secrets. No acceptance criterion asserts a guardrail.
