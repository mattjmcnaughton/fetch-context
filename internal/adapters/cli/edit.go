package cli

import (
	"github.com/spf13/cobra"
)

func newEditCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the config in $VISUAL/$EDITOR/vi and validate the result",
		Args:  usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Edit.Run(cmd.Context())
		},
	}
}
