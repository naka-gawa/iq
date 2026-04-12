package main

import (
	stderrors "errors"
	"fmt"
	"os"

	"iq/cmd"
	iqerr "iq/internal/errors"
)

var (
	Version  = "dev"
	Revision = "unknown"
)

func main() {
	cmd.SetVersionInfo(Version, Revision)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		exitCode := 1
		if stderrors.Is(err, iqerr.ErrKeyNotFound) {
			exitCode = 2
		}
		os.Exit(exitCode)
	}
}
