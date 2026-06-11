package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the fetch-context version",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			// cobra's Println falls back to stderr; version belongs on stdout.
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version.Version)
			return err
		},
	}
}
