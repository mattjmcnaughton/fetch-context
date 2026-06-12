package cli

import (
	"github.com/spf13/cobra"

	"github.com/mattjmcnaughton/fetch-context/internal/core/materialize"
)

func newURLCmd(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "url <url>...",
		Short: "Fetch one or more pages as markdown into the target tree",
		Long: "Fetch each page through the reader proxy (which strips boilerplate and\n" +
			"returns clean markdown) and write it to <target>/urls/<host>/<path>.md.\n" +
			"URLs are forwarded verbatim to the proxy: never pass a URL containing\n" +
			"secrets (tokens, signed URLs, session IDs).",
		Args: usageArgs(cobra.MinimumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.URL.Materialize(cmd.Context(), materialize.URLRequest{
				URLs:   args,
				Target: deps.Target,
			})
		},
	}
}
