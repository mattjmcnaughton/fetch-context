package cli

import (
	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
)

func newRepoCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "repo <ref>...",
		Short: "Shallow-clone one or more repos into the target tree",
		Long: "Shallow-clone upstream source into <target>/repos/<host>/<owner>/<repo>/.\n" +
			"Accepts host-qualified paths (github.com/foo/bar) or full clone URLs.\n" +
			"Existing managed clones are fetched and hard-reset to the remote's latest.",
		Args: usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.Repo.Materialize(cmd.Context(), materialize.RepoRequest{
				Refs:   args,
				Target: deps.Target,
			})
		},
	}
}
