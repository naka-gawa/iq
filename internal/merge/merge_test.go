package merge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	iqerr "iq/internal/errors"
	"iq/internal/merge"
)

func TestMerge_Overwrite_LaterWins(t *testing.T) {
	base := map[string]any{
		"database": map[string]any{"host": "localhost", "port": "5432"},
	}
	prod := map[string]any{
		"database": map[string]any{"host": "prod.example.com"},
	}

	got, err := merge.Merge([]map[string]any{base, prod}, merge.PolicyOverwrite)
	require.NoError(t, err)

	db := got["database"].(map[string]any)
	assert.Equal(t, "prod.example.com", db["host"]) // overwritten
	assert.Equal(t, "5432", db["port"])             // preserved from base
}

func TestMerge_Overwrite_DoesNotMutateInputs(t *testing.T) {
	base := map[string]any{"database": map[string]any{"host": "localhost"}}
	prod := map[string]any{"database": map[string]any{"host": "prod"}}

	_, err := merge.Merge([]map[string]any{base, prod}, merge.PolicyOverwrite)
	require.NoError(t, err)

	assert.Equal(t, "localhost", base["database"].(map[string]any)["host"])
}

func TestMerge_GlobalKeys_TopLevel(t *testing.T) {
	a := map[string]any{"global": "a"}
	b := map[string]any{"other": "b"}

	got, err := merge.Merge([]map[string]any{a, b}, merge.PolicyOverwrite)
	require.NoError(t, err)
	assert.Equal(t, "a", got["global"])
	assert.Equal(t, "b", got["other"])
}

func TestMerge_Append_UnionsScalars(t *testing.T) {
	base := map[string]any{"s": map[string]any{"k": "a"}}
	prod := map[string]any{"s": map[string]any{"k": "b"}}

	got, err := merge.Merge([]map[string]any{base, prod}, merge.PolicyAppend)
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b"}, got["s"].(map[string]any)["k"])
}

func TestMerge_Append_Dedups(t *testing.T) {
	a := map[string]any{"s": map[string]any{"k": "x"}}
	b := map[string]any{"s": map[string]any{"k": "x"}}

	got, err := merge.Merge([]map[string]any{a, b}, merge.PolicyAppend)
	require.NoError(t, err)
	assert.Equal(t, []any{"x"}, got["s"].(map[string]any)["k"])
}

func TestMerge_Append_FlattensExistingArray(t *testing.T) {
	a := map[string]any{"s": map[string]any{"k": []any{"a", "b"}}}
	b := map[string]any{"s": map[string]any{"k": "c"}}

	got, err := merge.Merge([]map[string]any{a, b}, merge.PolicyAppend)
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b", "c"}, got["s"].(map[string]any)["k"])
}

func TestMerge_Strict_ErrorsOnConflict(t *testing.T) {
	base := map[string]any{"s": map[string]any{"k": "a"}}
	prod := map[string]any{"s": map[string]any{"k": "b"}}

	_, err := merge.Merge([]map[string]any{base, prod}, merge.PolicyStrict)
	require.Error(t, err)
	assert.ErrorIs(t, err, iqerr.ErrMergeConflict)
}

func TestMerge_Strict_AllowsIdenticalValues(t *testing.T) {
	base := map[string]any{"s": map[string]any{"k": "same"}}
	prod := map[string]any{"s": map[string]any{"k": "same"}}

	got, err := merge.Merge([]map[string]any{base, prod}, merge.PolicyStrict)
	require.NoError(t, err)
	assert.Equal(t, "same", got["s"].(map[string]any)["k"])
}

func TestMerge_TypeMismatch_OverwriteTakesLater(t *testing.T) {
	a := map[string]any{"x": map[string]any{"nested": "v"}}
	b := map[string]any{"x": "scalar"}

	got, err := merge.Merge([]map[string]any{a, b}, merge.PolicyOverwrite)
	require.NoError(t, err)
	assert.Equal(t, "scalar", got["x"])
}

func TestMerge_TypeMismatch_StrictErrors(t *testing.T) {
	a := map[string]any{"x": map[string]any{"nested": "v"}}
	b := map[string]any{"x": "scalar"}

	_, err := merge.Merge([]map[string]any{a, b}, merge.PolicyStrict)
	require.Error(t, err)
	assert.ErrorIs(t, err, iqerr.ErrMergeConflict)
}

func TestMerge_ThreeFiles_Order(t *testing.T) {
	a := map[string]any{"s": map[string]any{"k": "1"}}
	b := map[string]any{"s": map[string]any{"k": "2"}}
	c := map[string]any{"s": map[string]any{"k": "3"}}

	got, err := merge.Merge([]map[string]any{a, b, c}, merge.PolicyOverwrite)
	require.NoError(t, err)
	assert.Equal(t, "3", got["s"].(map[string]any)["k"])
}
