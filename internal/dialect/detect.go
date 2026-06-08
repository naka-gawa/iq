// Package dialect defines INI dialect profiles and their parser options.
package dialect

// Detect returns the Profile for a given file path.
// MVP: always returns ProfileGeneric regardless of input.
func Detect(path string) Profile {
	return ProfileGeneric
}
