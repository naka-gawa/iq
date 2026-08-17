package merge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	iqerr "iq/internal/errors"
	"iq/internal/merge"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name    string
		docs    []map[string]any
		policy  merge.Policy
		want    map[string]any
		wantErr bool
	}{
		{
			name: "overwrite: later file wins, unique keys preserved",
			docs: []map[string]any{
				{"database": map[string]any{"host": "localhost", "port": "5432"}},
				{"database": map[string]any{"host": "prod.example.com"}},
			},
			policy: merge.PolicyOverwrite,
			want:   map[string]any{"database": map[string]any{"host": "prod.example.com", "port": "5432"}},
		},
		{
			name: "overwrite: global (top-level) keys merge",
			docs: []map[string]any{
				{"global": "a"},
				{"other": "b"},
			},
			policy: merge.PolicyOverwrite,
			want:   map[string]any{"global": "a", "other": "b"},
		},
		{
			name: "overwrite: three files apply in order",
			docs: []map[string]any{
				{"s": map[string]any{"k": "1"}},
				{"s": map[string]any{"k": "2"}},
				{"s": map[string]any{"k": "3"}},
			},
			policy: merge.PolicyOverwrite,
			want:   map[string]any{"s": map[string]any{"k": "3"}},
		},
		{
			name: "append: unions conflicting scalars",
			docs: []map[string]any{
				{"s": map[string]any{"k": "a"}},
				{"s": map[string]any{"k": "b"}},
			},
			policy: merge.PolicyAppend,
			want:   map[string]any{"s": map[string]any{"k": []any{"a", "b"}}},
		},
		{
			name: "append: dedups identical values",
			docs: []map[string]any{
				{"s": map[string]any{"k": "x"}},
				{"s": map[string]any{"k": "x"}},
			},
			policy: merge.PolicyAppend,
			want:   map[string]any{"s": map[string]any{"k": []any{"x"}}},
		},
		{
			name: "append: flattens an existing array",
			docs: []map[string]any{
				{"s": map[string]any{"k": []any{"a", "b"}}},
				{"s": map[string]any{"k": "c"}},
			},
			policy: merge.PolicyAppend,
			want:   map[string]any{"s": map[string]any{"k": []any{"a", "b", "c"}}},
		},
		{
			name: "strict: identical values are allowed",
			docs: []map[string]any{
				{"s": map[string]any{"k": "same"}},
				{"s": map[string]any{"k": "same"}},
			},
			policy: merge.PolicyStrict,
			want:   map[string]any{"s": map[string]any{"k": "same"}},
		},
		{
			name: "strict: conflicting values error",
			docs: []map[string]any{
				{"s": map[string]any{"k": "a"}},
				{"s": map[string]any{"k": "b"}},
			},
			policy:  merge.PolicyStrict,
			wantErr: true,
		},
		{
			name: "overwrite: section/scalar type mismatch takes later value",
			docs: []map[string]any{
				{"x": map[string]any{"nested": "v"}},
				{"x": "scalar"},
			},
			policy: merge.PolicyOverwrite,
			want:   map[string]any{"x": "scalar"},
		},
		{
			name: "strict: section/scalar type mismatch errors",
			docs: []map[string]any{
				{"x": map[string]any{"nested": "v"}},
				{"x": "scalar"},
			},
			policy:  merge.PolicyStrict,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := merge.Merge(tt.docs, tt.policy)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, iqerr.ErrMergeConflict)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMerge_DoesNotMutateInputs(t *testing.T) {
	// A distinct property test: the exported Merge contract promises the input
	// documents are never mutated, including array leaves during append.
	base := map[string]any{"database": map[string]any{"host": "localhost", "tags": []any{"a"}}}
	prod := map[string]any{"database": map[string]any{"host": "prod", "tags": []any{"b"}}}

	_, err := merge.Merge([]map[string]any{base, prod}, merge.PolicyAppend)
	require.NoError(t, err)

	assert.Equal(t, "localhost", base["database"].(map[string]any)["host"])
	assert.Equal(t, []any{"a"}, base["database"].(map[string]any)["tags"])
	assert.Equal(t, []any{"b"}, prod["database"].(map[string]any)["tags"])
}
