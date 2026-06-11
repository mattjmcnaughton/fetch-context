// Package main is the wiring layer: it reads the environment, constructs
// every concrete adapter, injects them into use cases, hands the use cases
// to the cobra root, and maps the resulting error to a process exit code.
// This is the only file that imports every concrete adapter.
package main

import (
	"fmt"
	"os"

	"log/slog"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/cli"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/configstore"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/filestore"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/gitrepo"
	"github.com/mattjmcnaughton/fetch-context/internal/adapters/hostrepo"
	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
)

func main() {
	log := newLogger()

	// Tokens are read here and only here; the core never sees them.
	var creds []gitrepo.Credential
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		creds = append(creds, gitrepo.Credential{Kind: gitrepo.KindGitHub, Token: t})
	}
	if t := os.Getenv("GITLAB_TOKEN"); t != "" {
		creds = append(creds, gitrepo.Credential{Kind: gitrepo.KindGitLab, Token: t})
	}

	git := gitrepo.New(log, creds...)
	fs := filestore.New()
	locator := hostrepo.New()
	cfg := configstore.Default()

	deps := cli.Deps{
		Repo:   materialize.NewRepo(git, fs, locator, log),
		Target: cfg.Target,
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

// newLogger builds the logger injected into adapters and use cases. The
// level comes from FETCH_CONTEXT_LOG_LEVEL (the --log-level flag tunes the
// process-global default logger separately, in the CLI adapter).
func newLogger() *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv("FETCH_CONTEXT_LOG_LEVEL"))); err != nil {
		level = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}
