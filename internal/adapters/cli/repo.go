package cli

import (
	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
)

func newRepoCmd(deps Deps) *cobra.Command {
	var depth int
	var branch string
	cmd := &cobra.Command{
		Use:   "repo <ref>...",
		Short: "Clone one or more repos into the target tree",
		Long: "Clone upstream source into <target>/repos/<host>/<owner>/<repo>/.\n" +
			"Accepts host-qualified paths (github.com/foo/bar) or full clone URLs.\n" +
			"Clones are shallow (depth 1) unless --depth or the config's clone\n" +
			"section says otherwise; existing managed clones are fetched and\n" +
			"hard-reset to the remote's latest, converging to the requested\n" +
			"depth and branch.",
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
			return deps.Repo.Materialize(cmd.Context(), materialize.RepoRequest{
				Items:    materialize.ItemsFromRefs(args, d, branch),
				Target:   cfg.Target,
				Parallel: cfg.Clone.Parallel,
			})
		},
	}
	cmd.Flags().IntVar(&depth, "depth", 1, "history depth; 0 = full history (overrides config clone.depth)")
	cmd.Flags().StringVar(&branch, "branch", "", "branch to clone and track (default: the remote's default branch)")
	return cmd
}
