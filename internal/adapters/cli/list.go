package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newListCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show profiles and what is materialized on disk",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := deps.List.Run(cmd.Context())
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), out)
			return err
		},
	}
}
