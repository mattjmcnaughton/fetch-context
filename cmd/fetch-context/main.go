// Package main is the wiring layer: it reads the environment, constructs
// every concrete adapter, injects them into use cases, hands the use cases
// to the cobra root, and maps the resulting error to a process exit code.
package main

import (
	"fmt"
	"os"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/cli"
	"github.com/mattjmcnaughton/fetch-context/internal/core/usageerr"
)

func main() {
	root := cli.NewRoot()
	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch-context:", err)
		if usageerr.IsUsage(err) {
			root.SetOut(os.Stderr)
			_ = root.Usage()
		}
	}
	os.Exit(cli.ExitCode(err))
}
