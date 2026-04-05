package main

import (
	"fmt"
	"os"

	"iq/cmd"
)

var (
	Version  = "dev"
	Revision = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, Revision)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
