package cli

import (
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
)

// ExitCode maps an error returned by the root command to the process exit
// code pinned by R1: 0 success, 1 runtime failure, 2 usage error.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case usageerr.IsUsage(err):
		return 2
	default:
		return 1
	}
}
