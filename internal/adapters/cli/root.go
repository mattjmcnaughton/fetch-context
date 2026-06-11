package cli

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mattjmcnaughton/fetch-context/internal/version"
)

func NewRoot() *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:           "fetch-context",
		Short:         "Pull external context into the current repo: clone upstream source repos and render web pages to markdown, so an agent can Read and Grep them locally.",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setupLogging(logLevel)
		},
	}

	root.PersistentFlags().StringVar(
		&logLevel,
		"log-level",
		"info",
		"Log level (debug, info, warn, error)",
	)

	viper.SetEnvPrefix("FETCH_CONTEXT")
	viper.AutomaticEnv()

	return root
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
