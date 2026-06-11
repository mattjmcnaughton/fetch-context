package main

import (
	"fmt"
	"os"

	"github.com/mattjmcnaughton/fetch-context/internal/adapters/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "fetch-context:", err)
		os.Exit(1)
	}
}
