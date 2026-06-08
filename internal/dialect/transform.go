package dialect

import "strings"

// TransformMap post-processes a toMap result for dialect-specific restructuring.
// For ProfileGitconfig it splits "section \"subsection\"" section names into
// a two-level nested map: result[section][subsection] = keyMap.
func TransformMap(profile Profile, m map[string]any) map[string]any {
	if profile != ProfileGitconfig {
		return m
	}
	result := make(map[string]any)
	for k, v := range m {
		section, sub, ok := splitGitconfigSection(k)
		if !ok {
			result[k] = v
			continue
		}
		if _, exists := result[section]; !exists {
			result[section] = make(map[string]any)
		} else if _, ok := result[section].(map[string]any); !ok {
			// A scalar key already occupies this name (e.g. a global key named
			// "remote" conflicts with a [remote "origin"] subsection). Skip rather
			// than panic; the scalar takes precedence.
			continue
		}
		result[section].(map[string]any)[sub] = v
	}
	return result
}

// splitGitconfigSection parses a section name of the form `section "subsection"`
// into its two components. Returns ok=false when the name does not match the pattern.
func splitGitconfigSection(name string) (section, subsection string, ok bool) {
	i := strings.Index(name, " \"")
	if i < 0 || !strings.HasSuffix(name, "\"") {
		return "", "", false
	}
	return name[:i], name[i+2 : len(name)-1], true
}
