// Package dialect defines INI dialect profiles and their parser options.
package dialect

import "gopkg.in/ini.v1"

// Profile identifies an INI dialect.
type Profile int

const (
	// ProfileGeneric is the default, dialect-agnostic profile.
	ProfileGeneric Profile = iota
)

// LoadOptions returns the ini.LoadOptions appropriate for this profile.
func (p Profile) LoadOptions() ini.LoadOptions {
	switch p {
	default: // ProfileGeneric
		return ini.LoadOptions{
			AllowBooleanKeys:            true,
			AllowShadows:                false,
			IgnoreInlineComment:         false,
			UnescapeValueCommentSymbols: true,
		}
	}
}
