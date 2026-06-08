// Package dialect defines INI dialect profiles and their parser options.
package dialect

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"

	iqerr "iq/internal/errors"
)

// Profile identifies an INI dialect.
type Profile int

const (
	// ProfileGeneric is the default, dialect-agnostic profile.
	ProfileGeneric Profile = iota
	// ProfileSystemd handles systemd unit files.
	ProfileSystemd
	// ProfileGitconfig handles git configuration files.
	ProfileGitconfig
)

// ParseProfile converts a CLI string to a Profile constant.
// Returns an error wrapping ErrDialectDetect for unknown names.
func ParseProfile(s string) (Profile, error) {
	switch strings.ToLower(s) {
	case "generic":
		return ProfileGeneric, nil
	case "systemd":
		return ProfileSystemd, nil
	case "gitconfig":
		return ProfileGitconfig, nil
	default:
		return ProfileGeneric, fmt.Errorf("unknown profile %q: %w", s, iqerr.ErrDialectDetect)
	}
}

// LoadOptions returns the ini.LoadOptions appropriate for this profile.
func (p Profile) LoadOptions() ini.LoadOptions {
	switch p {
	case ProfileSystemd:
		return ini.LoadOptions{
			AllowBooleanKeys:    true,
			AllowShadows:        true,
			IgnoreInlineComment: true,
			IgnoreContinuation:  false,
		}
	case ProfileGitconfig:
		return ini.LoadOptions{
			Insensitive:      true, // section names and key names normalized to lowercase
			AllowBooleanKeys: true,
		}
	default: // ProfileGeneric
		return ini.LoadOptions{
			AllowBooleanKeys:            true,
			AllowShadows:                false,
			IgnoreInlineComment:         false,
			UnescapeValueCommentSymbols: true,
		}
	}
}
