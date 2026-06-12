package cli

import (
	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/core/profile"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
)

func newLoadCmd(deps Deps) *cobra.Command {
	var parallel int
	cmd := &cobra.Command{
		Use:   "load <profile>",
		Short: "Materialize a named profile from config",
		Long: "Materialize every repos/groups/urls entry the named profile declares,\n" +
			"using the rules of the corresponding one-off command. The profile is\n" +
			"always named explicitly — there is no implicit or auto-loaded profile.",
		Args: usageArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts profile.Options
			if cmd.Flags().Changed("parallel") {
				if parallel < 1 {
					return usageerr.Newf("--parallel must be >= 1, got %d", parallel)
				}
				opts.Parallel = parallel
			}
			return deps.Load.Run(cmd.Context(), args[0], opts)
		},
	}
	cmd.Flags().IntVar(&parallel, "parallel", 4, "max concurrent clones (overrides config clone.parallel)")
	return cmd
}
