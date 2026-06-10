# fetch-context

Pull external context into the current repo: clone upstream source repos and render web pages to markdown, so an agent can Read and Grep them locally.

`fetch-context` materializes context **into the working repo** under `.agentic/sources/`. It clones git repositories (single repos, or every repo under a GitHub org / GitLab group) into `sources/repos/`, and fetches URLs as clean markdown into `sources/urls/`. Named **profiles** in a config file bundle a set of repos, groups, and URLs together so a whole context set comes down with one command. Everything is repo-local and gitignored — nothing is committed, nothing leaks into a global cache.

There is no SHA pinning and no lockfile: clones track the latest commit on their default branch, and re-running refreshes them. If you need reproducibility, vendor the content into your repo yourself — `fetch-context` only manages the transient `.agentic/sources/` tree.

## Install

```
go install github.com/mattjmcnaughton/fetch-context/cmd/fetch-context@latest
```

Or, from a checkout:
```
just install
```

## Commands

```
fetch-context repo <url>...              # clone one or more repos (no profile needed)
fetch-context group <host>/<org-or-group>...   # clone every repo under an org / group
fetch-context url <url>...               # fetch one or more pages to markdown
fetch-context load <profile>             # materialize a named profile from config
fetch-context list                       # show profiles, and what's materialized on disk
fetch-context clean [repos|urls]         # remove materialized content
fetch-context edit                       # open config in $VISUAL/$EDITOR/vi
fetch-context version
```

The first three commands are the one-off path — they take explicit targets and need no config. `load` runs a saved bundle. The one-off commands and the profile keys (`repos`, `groups`, `urls`) are the same three concepts under two names.

### `repo`

Shallow-clone upstream source into `sources/repos/<host>/<owner>/<repo>/`.

```
fetch-context repo github.com/redis/redis
fetch-context repo github.com/foo/bar gitlab.com/acme/lib
```

- Clones shallow (`--depth=1`) against the default branch.
- If the destination already exists as a clone, it is `git fetch`ed and **hard-reset to the remote's latest** — local state in the clone is discarded by design.
- Accepts a host-qualified path (`github.com/foo/bar`) or a full clone URL.

### `group`

Enumerate an org/group via the host's API and clone every repo it contains.

```
fetch-context group github.com/my-org
fetch-context group gitlab.com/acme/platform
```

- **GitHub** orgs are flat: every repo in the org is cloned.
- **GitLab** groups are recursive: the group and all of its subgroups are walked, and the **subgroup path is preserved** in the layout — `gitlab.com/acme/platform/team/utils` clones to `sources/repos/gitlab.com/acme/platform/team/utils/`.
- Enumeration hits the GitHub/GitLab REST APIs and follows pagination. Private repos and most group listings require a token — see [Authentication](#authentication).
- Each resolved repo is then cloned with the same rules as `repo` (shallow, fetch-and-reset on re-run).

### `url`

Fetch a page through `https://r.jina.ai/` — which strips boilerplate and returns clean markdown — and write it to `sources/urls/<host>/<path>.md`.

```
fetch-context url https://example.com/blog/some-post
```

- The original URL is wrapped literally: `https://r.jina.ai/https://example.com/blog/some-post`. The page is sent to a third-party proxy, so **never pass a URL containing secrets** (tokens, signed URLs, session IDs).
- Re-fetching overwrites the existing markdown file.
- A root URL with no path is written to `<host>/index.md`.

### `load`

Materialize a named profile: every `repos`, `groups`, and `urls` entry it declares, using the rules of the corresponding one-off command.

```
fetch-context load backend
```

You always name the profile — there is no implicit or auto-loaded profile.

### `list`

Shows every profile defined in config with its `repos` / `groups` / `urls` contents, and reports what is currently materialized under the resolved target on disk.

### `clean`

```
fetch-context clean          # remove everything under the resolved target
fetch-context clean repos    # remove only sources/repos/
fetch-context clean urls     # remove only sources/urls/
```

`fetch-context` only ever removes content inside its own target tree.

### `edit`

Opens the config in `$VISUAL`, then `$EDITOR`, then `vi`. After the editor exits, the config is reloaded and validated; an invalid edit prints an error and leaves the broken file on disk for you to fix.

## File layout

`fetch-context` writes into the **current repo**, located via `git rev-parse --show-toplevel`.

```
<repo-root>/.agentic/sources/
  .gitignore                       # `*` — the whole tree is ignored, written automatically
  repos/
    github.com/redis/redis/        # <host>/<owner>/<repo>/
    gitlab.com/acme/platform/team/utils/   # subgroup path preserved
  urls/
    example.com/blog/some-post.md  # <host>/<path>.md
```

The target defaults to `.agentic/sources` (relative to the repo root) and is configurable globally or per profile. `repos/` and `urls/` are always created beneath it. The tree is gitignored automatically, so cloned source and fetched pages never enter your repo's history.

### Config format

Config lives at `~/.config/fetch-context/config.yaml` and holds the profile library, defined once and reused across repos.

```yaml
# Optional. Install target relative to the repo root. Defaults to .agentic/sources.
target: .agentic/sources

profiles:
  backend:
    # Optional per-profile target override.
    target: .agentic/backend
    repos:
      - github.com/redis/redis
      - gitlab.com/acme/lib
    groups:
      - github.com/my-org           # every repo in the org
      - gitlab.com/acme/platform    # group + all subgroups, recursively
    urls:
      - https://example.com/blog/some-post
      - https://docs.example.com/changelog

  web-stack:
    repos:
      - github.com/foo/bar
      - github.com/foo/baz
```

All three keys are optional; a profile may declare any combination of `repos`, `groups`, and `urls`.

## Authentication

`group` enumeration and any private repo require a token, read from the environment — there is no config-based credential storage.

| Host | Variable |
|---|---|
| GitHub | `GITHUB_TOKEN` |
| GitLab | `GITLAB_TOKEN` |

Public single-repo `repo` clones need no token. If enumeration or a clone fails with an auth error, `fetch-context` reports it rather than silently skipping.

## Sandboxing

Set `FETCH_CONTEXT_HOME` to redirect the config directory (and any `~` expansion) under a custom root. Useful for tests and parallel runs.

```
FETCH_CONTEXT_HOME=/tmp/sandbox fetch-context load backend
```

The materialized target is always resolved against the current repo root, independent of `FETCH_CONTEXT_HOME`.

## Conflict behavior

`fetch-context` only touches content inside its own target tree, and refuses to clobber anything it didn't create:

- A destination under `repos/` that **is** a `fetch-context` clone → `git fetch` + hard reset to remote latest.
- A destination under `repos/` that exists but is **not** a git repo → error, no changes.
- A markdown file under `urls/` → overwritten on re-fetch.
- `clean` removes only paths beneath the resolved target.

## Scope

`fetch-context` deliberately does **not**:

- Pin commits or maintain a lockfile — clones track default-branch latest.
- Query documentation indexes — it fetches source and pages, nothing else.
- Manage content outside `.agentic/sources/` or commit anything to your repo.
