// Package query translates a *ini.File AST into a map and executes jq expressions.
package query

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/itchyny/gojq"
	"gopkg.in/ini.v1"

	iqerr "iq/internal/errors"
)

// Execute runs a jq expression against the INI AST and returns matched values.
// Returns ErrKeyNotFound (wrapped) when the expression yields no results.
// Returns ErrPathInvalid (wrapped) when the expression cannot be parsed or compiled.
// If transform is non-nil it is applied to the intermediate map before jq evaluation,
// enabling dialect-specific restructuring (e.g. gitconfig subsection expansion).
func Execute(f *ini.File, expr string, transform func(map[string]any) map[string]any) ([]any, error) {
	q, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid expression %q: %w", expr, iqerr.ErrPathInvalid)
	}

	code, err := gojq.Compile(q,
		gojq.WithFunction("strenv", 1, 1, func(_ any, args []any) any {
			key, ok := args[0].(string)
			if !ok {
				return fmt.Errorf("strenv: argument must be a string")
			}
			return os.Getenv(key)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("compiling expression %q: %w", expr, iqerr.ErrPathInvalid)
	}

	m := toMap(f)
	if transform != nil {
		m = transform(m)
	}
	iter := code.Run(m)

	var results []any
	var sawNull bool
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if e, ok := v.(error); ok {
			return nil, fmt.Errorf("query error: %w", e)
		}
		// jq yields null when a path does not exist. Track this separately from a
		// filter expression that matched nothing (which yields no values at all).
		if v == nil {
			sawNull = true
			continue
		}
		results = append(results, v)
	}

	// null yielded → path was evaluated and resolved to null → key not found.
	// no values and no null → filter (select/test/[]) returned empty set → exit 0.
	if len(results) == 0 && sawNull {
		return nil, fmt.Errorf("path not found: %w", iqerr.ErrKeyNotFound)
	}

	return results, nil
}

// FormatValue renders a query result value as a string suitable for stdout.
// Objects and arrays are JSON-encoded; scalars are printed as-is.
func FormatValue(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case nil:
		return "null", nil
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return "", fmt.Errorf("formatting result: %w", err)
		}
		return string(b), nil
	}
}

// ToMap converts a *ini.File into a map[string]any using the same shape as the
// query evaluator (sections as nested maps, global keys flattened to the top
// level, duplicate keys as arrays). It is exported for reuse by the merge path.
func ToMap(f *ini.File) map[string]any {
	return toMap(f)
}

// toMap converts a *ini.File into a map[string]any for gojq evaluation.
// All values are kept as strings; type coercion is the serializer's responsibility.
// Keys before the first section header are placed at the top level.
func toMap(f *ini.File) map[string]any {
	result := make(map[string]any)

	for _, sec := range f.Sections() {
		name := sec.Name()

		// DEFAULT section holds global properties (keys before the first section header).
		// Flatten them to the top level.
		if name == ini.DefaultSection {
			for _, k := range sec.Keys() {
				result[k.Name()] = k.Value()
			}
			continue
		}

		secMap := make(map[string]any)
		for _, k := range sec.Keys() {
			shadows := k.ValueWithShadows()
			if len(shadows) <= 1 {
				secMap[k.Name()] = k.Value()
				continue
			}
			vals := make([]any, len(shadows))
			for i, s := range shadows {
				vals[i] = s
			}
			secMap[k.Name()] = vals
		}
		result[name] = secMap
	}

	return result
}
