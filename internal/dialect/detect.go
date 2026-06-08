// Package dialect defines INI dialect profiles and their parser options.
package dialect

import (
	"path/filepath"
	"strings"
)

// Detect returns the Profile for a given file path based on extension and basename.
func Detect(path string) Profile {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".service", ".target", ".socket", ".mount", ".timer", ".path", ".scope", ".slice":
		return ProfileSystemd
	}

	base := strings.ToLower(filepath.Base(path))
	if base == ".gitconfig" {
		return ProfileGitconfig
	}
	if base == "config" && filepath.Base(filepath.Dir(filepath.Clean(path))) == ".git" {
		return ProfileGitconfig
	}

	return ProfileGeneric
}
