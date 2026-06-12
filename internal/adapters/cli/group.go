package cli

import (
	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
)

func newGroupCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "group <host>/<org-or-group>...",
		Short: "Clone every repo under a GitHub org or GitLab group",
		Long: "Enumerate an org/group via the host's API and clone every repo it\n" +
			"contains. GitHub orgs are flat; GitLab groups are walked recursively and\n" +
			"the subgroup path is preserved in the layout. Private listings require\n" +
			"GITHUB_TOKEN / GITLAB_TOKEN.",
		Args: usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Group.Materialize(cmd.Context(), materialize.GroupRequest{
				Refs:   args,
				Target: deps.Target,
			})
		},
	}
}
