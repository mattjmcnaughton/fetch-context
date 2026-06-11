# CLAUDE.md

Pull external context into the current repo: clone upstream source repos and render web pages to markdown, so an agent can Read and Grep them locally.

Go CLI built with **strict hexagonal (ports-and-adapters) architecture**. Cobra for the driving CLI, Viper for config, slog for logging, afero behind a narrow file-store port.

## Quick Reference

| Command | Purpose |
| ------- | ------- |
| `just fmt` / `just fmt-fix` | Check / fix formatting |
| `just vet` | Run go vet |
| `just test` | Unit tests (no build tag) |
| `just test-integration` | Integration tests (`//go:build integration`) |
| `just test-all` | Unit + integration |
| `just test-contract` | Contract tests (`//go:build contract`); requires `$GITHUB_TOKEN`, `$GITLAB_TOKEN`. Opt-in, never gated. |
| `just test-e2e` | E2E suite (`//go:build e2e`) inside `Dockerfile.dev`, no outbound network |
| `just build` | Build binary to `bin/` |
| `just run [args]` | Run via `go run` |
| `just tidy` | Tidy dependencies |
| `just gate` | Pre-push: fmt + vet + test + test-integration |
| `just gate-expensive` | gate + test-e2e |

## Architecture in one paragraph

The **core** (`internal/core/`) contains pure use cases and domain services with zero infrastructure imports. **Ports** (`internal/ports/`) are small interfaces the core depends on. **Adapters** (`internal/adapters/`) implement those ports — git CLI, GitHub/GitLab REST, the jina reader proxy, filesystem, editor, config store. **Wiring** in `cmd/fetch-context/main.go` is the only place that imports every concrete adapter; it constructs them, injects them into use cases, and hands those to the cobra root. The core never imports `net/http`, `os/exec`, `os`, or any third-party SDK. **Read `docs/architecture.md` before adding a new module, port, or adapter.**

## Project Structure (high level)

```
cmd/fetch-context/main.go     # Wiring: env → adapters → use cases → cobra root
internal/
  core/                       # Pure: use cases + domain services (urlmap, repoid, targetpath, …)
  ports/                      # Interfaces the core depends on
  adapters/                   # Concrete implementations (cli, gitrepo, forge/*, pagereader, filestore, …)
  testing/                    # Shared fakes, forgemock, readermock, gitfixture
  version/
tests/e2e/                    # Black-box tests against the compiled binary (build tag e2e)
docs/                         # acceptance, architecture, testing, development, adrs/
```

Full layout, port table, and use-case table live in `docs/architecture.md`.

## Key Conventions

- **One port per external dependency.** No reaching directly into `net/http` or `os/exec` from `internal/core/`. New external dep → new port.
- **One file per cobra subcommand** under `internal/adapters/cli/`. Each is a thin shim: parse args, call use case, format output. Errors return from cobra's `RunE`.
- **Use cases take `context.Context` first** and an explicit `*slog.Logger` — no globals.
- **Tokens (`GITHUB_TOKEN`, `GITLAB_TOKEN`) are read in wiring** and passed to forge adapters as constructor args. The core never sees them.
- **Config** loads from `~/.config/fetch-context/config.yaml` via viper; `FETCH_CONTEXT_HOME` redirects the config dir for tests/sandboxing. Env vars are `FETCH_CONTEXT_`-prefixed.
- **Test build tags:** unit (none) / `integration` / `contract` / `e2e`. See `docs/testing.md` for the pyramid, mock packages, and the AC-ID ↔ e2e-test 1:1 rule.
- **Version** defaults to `"dev"`, overridden at build time with `-ldflags "-X github.com/mattjmcnaughton/fetch-context/internal/version.Version=x.y.z"`.

## More Information

Progressive disclosure — pull in the relevant doc when the task touches it:

- `README.md` — user-facing command surface, file layout, config format, auth.
- `docs/architecture.md` — hexagonal layout, port/adapter tables, use-case wiring. **Read before changing structure, adding a port, or adding a third-party dep.**
- `docs/testing.md` — four-tier pyramid (unit / integration / contract / e2e), mock packages (`forgemock`, `readermock`, `gitfixture`, `fakes`), build-tag conventions, the AC-ID ↔ e2e-test mapping. **Read before writing tests beyond a plain unit test.**
- `docs/acceptance.md` — observable contract; every `AC-*` ID maps 1:1 to an e2e test. **Read before changing user-visible behavior.**
- `docs/development.md` — environment setup, debugging, common tasks.
- `docs/adrs/` — Architecture Decision Records.

<!-- managed:setup-permissions -->
## Commands

Use these commands as the canonical entry points. The agent is preauthorised to run them; ad-hoc shell invocations may require a prompt.

| Intent   | Command                       | Notes                              |
|----------|-------------------------------|------------------------------------|
| lint     | `just vet`                    | static checks; non-mutating        |
| fmt      | `just fmt`                    | format-only; no logic changes      |
| test     | `just test`                   | full suite                         |
| test-one | `just test-one PATTERN [PATH]`| single test by regexp; PATH=`./...`|
| lint-fix | `just fmt-fix`                | auto-applies safe fixes            |
| gate     | `just gate`                   | full pre-commit gate (lint + test) |

Filesystem scope: agent reads/writes the working tree, including `.agentic/` (agentic-coding workspace), `.worktrees/` (worktree provisioning), and `.sandcastle/worktrees/` (sandcastle worktrees). Committed `.env.example` / `.env.sample` / `.env.template` stay readable; live `.env` and the common `.env.local` / `.env.production` / etc. variants are denied. Other `.gitignore` entries are denied.
<!-- /managed:setup-permissions -->
