// Package cli is the driving adapter: cobra commands as thin shims over the
// use cases in internal/core/. One file per subcommand.
package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/core/clean"
	"github.com/mattjmcnaughton/fetch-context/internal/core/editconfig"
	"github.com/mattjmcnaughton/fetch-context/internal/core/list"
	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
	"github.com/mattjmcnaughton/fetch-context/internal/core/profile"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
	"github.com/mattjmcnaughton/fetch-context/internal/ports"
)

// Deps holds the use cases the wiring injects into the CLI, plus a lazy
// config accessor: commands that need the resolved target load the config
// on demand, so a malformed config fails exactly the commands that depend
// on it (and `version`/usage never touch it).
type Deps struct {
	Repo   *materialize.Repo
	Group  *materialize.Group
	URL    *materialize.URL
	Load   *profile.Load
	List   *list.List
	Clean  *clean.Clean
	Edit   *editconfig.Edit
	Config func() (ports.Config, error)
}

// target resolves the global target from config.
func (d Deps) target() (string, error) {
	cfg, err := d.Config()
	if err != nil {
		return "", err
	}
	return cfg.Target, nil
}

func NewRoot(deps Deps) *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:           "fetch-context",
		Short:         "Pull external context into the current repo: clone upstream source repos and render web pages to markdown, so an agent can Read and Grep them locally.",
		SilenceUsage:  true,
		SilenceErrors: true,
		// ArbitraryArgs + a RunE that always errors: with a Run defined,
		// cobra routes unmatched first arguments here instead of failing
		// internally, which lets the usage-error mapping live in one place.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return usageerr.New("a subcommand is required")
			}
			return usageerr.Newf("unknown command %q for %q", args[0], cmd.Name())
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setupLogging(logLevel)
		},
	}

	// Flag-parse failures are the caller's mistake → exit 2.
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return usageerr.Wrap(err)
	})

	root.PersistentFlags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"Log level (debug, info, warn, error)",
	)

	root.AddCommand(
		newVersionCmd(),
		newRepoCmd(deps),
		newGroupCmd(deps),
		newURLCmd(deps),
		newLoadCmd(deps),
		newListCmd(deps),
		newCleanCmd(deps),
		newEditCmd(deps),
	)

	return root
}

// usageArgs adapts a cobra positional-args validator so its failures are
// usage errors (exit 2) rather than runtime errors.
func usageArgs(v cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := v(cmd, args); err != nil {
			return usageerr.Wrap(err)
		}
		return nil
	}
}

func setupLogging(level string) error {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	slog.SetDefault(slog.New(handler))
	return nil
}
