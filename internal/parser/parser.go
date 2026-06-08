// Package parser loads INI files into a *ini.File AST with dialect options applied.
// This is the only package that imports gopkg.in/ini.v1 directly.
package parser

import (
	"fmt"
	"os"

	"gopkg.in/ini.v1"

	"iq/internal/dialect"
	iqerr "iq/internal/errors"
)

// Parse loads an INI file and returns its AST with the given dialect profile applied.
// When path is "" or "-", input is read from os.Stdin.
// All parse errors are wrapped with ErrFileParseFailed.
func Parse(path string, profile dialect.Profile) (*ini.File, error) {
	return ParseWithOptions(path, profile.LoadOptions())
}

// ParseWithOptions loads an INI file using the provided LoadOptions directly.
// Useful when callers need to override specific options (e.g. AllowShadows for duplicate keys).
func ParseWithOptions(path string, opts ini.LoadOptions) (*ini.File, error) {
	var (
		f   *ini.File
		err error
	)

	if path == "" || path == "-" {
		f, err = ini.LoadSources(opts, os.Stdin)
	} else {
		f, err = ini.LoadSources(opts, path)
	}

	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, iqerr.ErrFileParseFailed)
	}

	return f, nil
}
