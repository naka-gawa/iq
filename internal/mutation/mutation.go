// Package mutation applies targeted changes to a live *ini.File AST without re-parsing.
package mutation

import (
	"fmt"

	"gopkg.in/ini.v1"
)

// Target describes a single mutation operation on an INI AST.
type Target struct {
	Section string
	Key     string // empty string means section-level operation
	NewVal  any    // nil means delete
}

// Apply writes all mutation targets into the live AST.
// Sections and keys are auto-created when they do not exist.
// Delete operations on absent targets are no-ops (idempotent).
func Apply(f *ini.File, targets []Target) error {
	for _, t := range targets {
		applyOne(f, t)
	}
	return nil
}

func applyOne(f *ini.File, t Target) {
	if t.Key == "" {
		// Section-level operation.
		if t.NewVal == nil {
			f.DeleteSection(t.Section)
		}
		return
	}

	if t.NewVal == nil {
		// Delete key — no-op if section or key is absent.
		sec, err := f.GetSection(t.Section)
		if err != nil {
			return
		}
		sec.DeleteKey(t.Key)
		return
	}

	// Upsert: create section if missing, then create or update the key.
	val := fmt.Sprintf("%v", t.NewVal)

	if !f.HasSection(t.Section) {
		f.NewSection(t.Section) //nolint:errcheck // fails only when file is in strict mode
	}

	sec := f.Section(t.Section)
	if sec.HasKey(t.Key) {
		sec.Key(t.Key).SetValue(val)
	} else {
		sec.NewKey(t.Key, val) //nolint:errcheck // fails only when file is in strict mode
	}
}
