package cli

import (
	"github.com/spf13/cobra"
)

func newCleanCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "clean [repos|urls|<profile>]",
		Short: "Remove materialized content",
		Long: "Remove everything under the resolved target, one of its repos/urls\n" +
			"subtrees, or a named profile's target. Only ever removes content inside\n" +
			"the tool's own target tree.",
		Args: usageArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := ""
			if len(args) == 1 {
				scope = args[0]
			}
			return deps.Clean.Run(cmd.Context(), scope)
		},
	}
}
