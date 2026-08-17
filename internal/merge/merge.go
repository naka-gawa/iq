// Package merge deep-merges multiple INI documents (as map[string]any) under a
// configurable conflict policy. It operates purely on maps produced by
// query.ToMap, so it is independent of the ini.v1 AST and easy to unit test.
package merge

import (
	"fmt"
	"reflect"

	iqerr "iq/internal/errors"
)

// Policy selects how conflicting keys are reconciled during a merge.
type Policy int

const (
	// PolicyOverwrite makes later documents win on conflict (default).
	PolicyOverwrite Policy = iota
	// PolicyAppend unions conflicting scalar values into an array.
	PolicyAppend
	// PolicyStrict errors when two documents disagree on a key's value.
	PolicyStrict
)

// Merge deep-merges docs in order and returns the combined document.
// Nested maps (sections) are merged recursively; scalar/array conflicts are
// resolved by policy. The input docs are not mutated.
func Merge(docs []map[string]any, policy Policy) (map[string]any, error) {
	acc := make(map[string]any)
	for _, doc := range docs {
		if err := mergeInto(acc, doc, policy); err != nil {
			return nil, err
		}
	}
	return acc, nil
}

// mergeInto merges src into dst (mutating dst) under the given policy.
func mergeInto(dst, src map[string]any, policy Policy) error {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = deepCopy(sv)
			continue
		}

		dm, dIsMap := dv.(map[string]any)
		sm, sIsMap := sv.(map[string]any)
		switch {
		case dIsMap && sIsMap:
			if err := mergeInto(dm, sm, policy); err != nil {
				return err
			}
		case dIsMap != sIsMap:
			// One side is a section (map), the other a scalar/array.
			if err := resolveTypeMismatch(dst, k, sv, policy); err != nil {
				return err
			}
		default:
			if err := resolveScalar(dst, k, dv, sv, policy); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveTypeMismatch handles a conflict where one value is a map and the other
// is not. Overwrite takes the incoming value; strict and append both error,
// because unioning a section with a scalar has no sensible meaning.
func resolveTypeMismatch(dst map[string]any, key string, sv any, policy Policy) error {
	if policy == PolicyOverwrite {
		dst[key] = deepCopy(sv)
		return nil
	}
	return fmt.Errorf("key %q has incompatible types across files (section vs value): %w", key, iqerr.ErrMergeConflict)
}

// resolveScalar handles a conflict between two non-map values under the policy.
func resolveScalar(dst map[string]any, key string, dv, sv any, policy Policy) error {
	switch policy {
	case PolicyOverwrite:
		dst[key] = deepCopy(sv)
	case PolicyAppend:
		dst[key] = union(dv, sv)
	case PolicyStrict:
		if !reflect.DeepEqual(dv, sv) {
			return fmt.Errorf("key %q differs across files (%v != %v): %w", key, dv, sv, iqerr.ErrMergeConflict)
		}
	}
	return nil
}

// union combines two scalar-or-array values into a deduplicated []any,
// preserving first-seen order. It copies values into a freshly owned slice and
// never appends onto an input slice, so the Merge no-mutation contract holds
// even for inputs whose array leaves carry spare capacity.
func union(a, b any) []any {
	out := make([]any, 0)
	seen := make(map[string]struct{})
	for _, src := range [][]any{toSlice(a), toSlice(b)} {
		for _, v := range src {
			key := fmt.Sprintf("%v", v)
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, deepCopy(v))
		}
	}
	return out
}

// toSlice normalizes a value into a []any (a non-array becomes a single element).
func toSlice(v any) []any {
	if arr, ok := v.([]any); ok {
		return arr
	}
	return []any{v}
}

// deepCopy clones nested maps and slices so the merged result never aliases the
// input documents.
func deepCopy(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, e := range val {
			cp[k] = deepCopy(e)
		}
		return cp
	case []any:
		cp := make([]any, len(val))
		for i, e := range val {
			cp[i] = deepCopy(e)
		}
		return cp
	default:
		return v
	}
}
