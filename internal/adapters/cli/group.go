package cli

import (
	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
)

func newGroupCmd(deps Deps) *cobra.Command {
	var depth int
	cmd := &cobra.Command{
		Use:   "group <host>/<org-or-group>...",
		Short: "Clone every repo under a GitHub org or GitLab group",
		Long: "Enumerate an org/group via the host's API and clone every repo it\n" +
			"contains. GitHub orgs are flat; GitLab groups are walked recursively and\n" +
			"the subgroup path is preserved in the layout. Private listings require\n" +
			"GITHUB_TOKEN / GITLAB_TOKEN.",
		Args: usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := deps.Config()
			if err != nil {
				return err
			}
			// Precedence: explicit flag > global clone config > default.
			d := cfg.Clone.Depth
			if cmd.Flags().Changed("depth") {
				if depth < 0 {
					return usageerr.Newf("--depth must be >= 0 (0 = full history), got %d", depth)
				}
				d = depth
			}
			return deps.Group.Materialize(cmd.Context(), materialize.GroupRequest{
				Refs:     args,
				Target:   cfg.Target,
				Depth:    d,
				Parallel: cfg.Clone.Parallel,
			})
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 1, "history depth; 0 = full history (overrides config clone.depth)")
	return cmd
}
