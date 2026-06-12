// Package main is the wiring layer: it reads the environment, constructs
// every concrete adapter, injects them into use cases, hands the use cases
// to the cobra root, and maps the resulting error to a process exit code.
// This is the only file that imports every concrete adapter.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/cli"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/configstore"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/editor"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/envx"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/filestore"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/forge/github"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/forge/gitlab"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/gitrepo"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/hostrepo"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/pagereader"
	"github.com/mattjmcnaughton/fetch-context/internal/core/clean"
	"github.com/mattjmcnaughton/fetch-context/internal/core/editconfig"
	"github.com/mattjmcnaughton/fetch-context/internal/core/list"
	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
	"github.com/mattjmcnaughton/fetch-context/internal/core/profile"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

func main() {
	env := envx.OsEnv{}
	log := newLogger(env)

	// Tokens are read here and only here; the core never sees them.
	githubToken, _ := env.Get("GITHUB_TOKEN")
	gitlabToken, _ := env.Get("GITLAB_TOKEN")
	var creds []gitrepo.Credential
	if githubToken != "" {
		creds = append(creds, gitrepo.Credential{Kind: gitrepo.KindGitHub, Token: githubToken})
	}
	if gitlabToken != "" {
		creds = append(creds, gitrepo.Credential{Kind: gitrepo.KindGitLab, Token: gitlabToken})
	}

	git := gitrepo.New(log, creds...)
	fs := filestore.New()
	locator := hostrepo.New()
	store := configstore.New(configHome(env))
	ed := editor.New(env)

	// JINA_BASE_URL / GITHUB_API_URL / GITLAB_API_URL are contract seams
	// (acceptance.md §1.2): they redirect the outbound dependencies to
	// loopback mocks in hermetic runs.
	reader := pagereader.New(getenv(env, "JINA_BASE_URL"), log)

	// Forge dispatch is keyed by host (ADR-0001 decision 1); a future forge
	// is a new entry here, no core change.
	enumerators := map[string]ports.ForgeEnumerator{
		"github.com": github.New(getenv(env, "GITHUB_API_URL"), githubToken, log),
		"gitlab.com": gitlab.New(getenv(env, "GITLAB_API_URL"), gitlabToken, log),
	}

	repoUC := materialize.NewRepo(git, fs, locator, log)
	groupUC := materialize.NewGroup(enumerators, git, fs, locator, log)
	urlUC := materialize.NewURL(reader, fs, locator, log)

	deps := cli.Deps{
		Repo:   repoUC,
		Group:  groupUC,
		URL:    urlUC,
		Load:   profile.NewLoad(store, repoUC, groupUC, urlUC, log),
		List:   list.New(store, fs, locator, log),
		Clean:  clean.New(store, fs, locator, log),
		Edit:   editconfig.New(ed, store, fs, log),
		Config: sync.OnceValues(store.Load),
	}

	root := cli.NewRoot(deps)
	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch-context:", err)
		if usageerr.IsUsage(err) {
			root.SetOut(os.Stderr)
			_ = root.Usage()
		}
	}
	os.Exit(cli.ExitCode(err))
}

// configHome resolves the config root: $FETCH_CONTEXT_HOME redirects it for
// sandboxing (AC-CONFIG-01); otherwise the user's home directory.
func configHome(env envx.Env) string {
	if home, ok := env.Get("FETCH_CONTEXT_HOME"); ok && home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// newLogger builds the logger injected into adapters and use cases. The
// level comes from FETCH_CONTEXT_LOG_LEVEL (the --log-level flag tunes the
// process-global default logger separately, in the CLI adapter).
func newLogger(env envx.Env) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(getenv(env, "FETCH_CONTEXT_LOG_LEVEL"))); err != nil {
		level = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func getenv(env envx.Env, key string) string {
	v, _ := env.Get(key)
	return v
}
