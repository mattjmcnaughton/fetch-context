package cli

import (
	"github.com/spf13/cobra"
)

func newLoadCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "load <profile>",
		Short: "Materialize a named profile from config",
		Long: "Materialize every repos/groups/urls entry the named profile declares,\n" +
			"using the rules of the corresponding one-off command. The profile is\n" +
			"always named explicitly — there is no implicit or auto-loaded profile.",
		Args: usageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Load.Run(cmd.Context(), args[0])
		},
	}
}
