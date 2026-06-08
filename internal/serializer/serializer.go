// Package serializer writes *ini.File ASTs to disk or output streams.
package serializer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
func WriteJSON(f *ini.File, w io.Writer, rawStrings bool) error {
	m := toJSONMap(f, rawStrings)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(m)
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
