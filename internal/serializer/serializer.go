// Package serializer writes *ini.File ASTs to disk or output streams.
package serializer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"gopkg.in/ini.v1"
)

// WriteInPlace atomically replaces path with the serialized AST.
// It writes to a temp file in the same directory, preserves the original
// file permissions, then calls os.Rename for an atomic replace.
func WriteInPlace(f *ini.File, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".iq-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Best-effort cleanup of the temp file on error.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if _, err := f.WriteTo(tmp); err != nil {
		tmp.Close()
		return fmt.Errorf("write to temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpName, info.Mode()); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	success = true
	return nil
}

// WriteINI renders the AST as INI text to w.
func WriteINI(f *ini.File, w io.Writer) error {
	_, err := f.WriteTo(w)
	return err
}

// WriteJSON renders the AST as a JSON object to w.
// When rawStrings is false, numeric and boolean string values are type-coerced.
// If transform is non-nil it is applied to the map before JSON encoding,
// enabling dialect-specific restructuring (e.g. gitconfig subsection expansion).
func WriteJSON(f *ini.File, w io.Writer, rawStrings bool, transform func(map[string]any) map[string]any) error {
	m := toJSONMap(f, rawStrings)
	if transform != nil {
		m = transform(m)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
}

// WriteMergedINI renders a merged document (as produced by internal/merge) to w
// as INI text. Top-level scalar keys become global properties; nested maps
// become sections; two-level nested maps become gitconfig-style subsections
// (`[section "subsection"]`); array values are expanded to repeated keys.
func WriteMergedINI(m map[string]any, w io.Writer) error {
	f := ini.Empty(ini.LoadOptions{AllowShadows: true})

	// Keys are visited in sorted order so identical input always serializes
	// identically (Go map iteration order is otherwise randomized).
	def := f.Section(ini.DefaultSection)
	for _, name := range sortedKeys(m) {
		if sub, ok := m[name].(map[string]any); ok {
			if err := writeSection(f, name, sub); err != nil {
				return err
			}
			continue
		}
		if err := writeKey(def, name, m[name]); err != nil {
			return err
		}
	}

	_, err := f.WriteTo(w)
	return err
}

// writeSection materializes a section and its keys in deterministic order.
// Nested maps are emitted as gitconfig-style subsections. The parent section is
// created lazily on its first scalar key, so a section that holds only
// subsections does not emit an empty `[parent]` header before them.
func writeSection(f *ini.File, name string, body map[string]any) error {
	keys := sortedKeys(body)

	// First pass: scalar keys (creating the parent section lazily).
	var sec *ini.Section
	for _, k := range keys {
		if _, isMap := body[k].(map[string]any); isMap {
			continue
		}
		if sec == nil {
			s, err := f.NewSection(name)
			if err != nil {
				return fmt.Errorf("create section %q: %w", name, err)
			}
			sec = s
		}
		if err := writeKey(sec, k, body[k]); err != nil {
			return err
		}
	}

	// Second pass: nested maps become subsections.
	for _, k := range keys {
		sub, isMap := body[k].(map[string]any)
		if !isMap {
			continue
		}
		if err := writeSection(f, fmt.Sprintf("%s %q", name, k), sub); err != nil {
			return err
		}
	}
	return nil
}

// sortedKeys returns the map's keys in ascending order for deterministic output.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeKey writes a single key, expanding []any into repeated (shadow) keys.
func writeKey(sec *ini.Section, name string, v any) error {
	vals := toStrings(v)
	if len(vals) == 0 {
		return nil
	}
	key, err := sec.NewKey(name, vals[0])
	if err != nil {
		return fmt.Errorf("create key %q: %w", name, err)
	}
	for _, extra := range vals[1:] {
		if err := key.AddShadow(extra); err != nil {
			return fmt.Errorf("add shadow for key %q: %w", name, err)
		}
	}
	return nil
}

// toStrings normalizes a scalar or []any value into a slice of strings.
func toStrings(v any) []string {
	if arr, ok := v.([]any); ok {
		out := make([]string, len(arr))
		for i, e := range arr {
			out[i] = fmt.Sprintf("%v", e)
		}
		return out
	}
	return []string{fmt.Sprintf("%v", v)}
}

// WriteMergedJSON renders a merged document to w as JSON, applying the same type
// coercion as INI-to-JSON conversion (disabled when rawStrings is true).
func WriteMergedJSON(m map[string]any, w io.Writer, rawStrings bool) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(coerceValue(m, rawStrings))
}

// coerceValue recursively coerces string leaves in a merged document.
func coerceValue(v any, rawStrings bool) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, e := range val {
			out[k] = coerceValue(e, rawStrings)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = coerceValue(e, rawStrings)
		}
		return out
	case string:
		return coerce(val, rawStrings)
	default:
		return val
	}
}

// toJSONMap converts the INI AST to a nested map suitable for JSON encoding.
func toJSONMap(f *ini.File, rawStrings bool) map[string]any {
	result := make(map[string]any)

	for _, sec := range f.Sections() {
		name := sec.Name()

		if name == ini.DefaultSection {
			for _, k := range sec.Keys() {
				result[k.Name()] = coerce(k.Value(), rawStrings)
			}
			continue
		}

		secMap := make(map[string]any)
		for _, k := range sec.Keys() {
			secMap[k.Name()] = coerce(k.Value(), rawStrings)
		}
		result[name] = secMap
	}

	return result
}

// coerce converts an INI string value to its most natural Go type for JSON output.
// When rawStrings is true the value is always returned as a string.
func coerce(s string, rawStrings bool) any {
	if rawStrings {
		return s
	}

	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
