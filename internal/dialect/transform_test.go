package dialect_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"iq/internal/dialect"
)

func TestTransformMap_NonGitconfig_ReturnsUnchanged(t *testing.T) {
	t.Parallel()
	m := map[string]any{"section": map[string]any{"key": "val"}}
	got := dialect.TransformMap(dialect.ProfileGeneric, m)
	assert.Equal(t, m, got)
}

func TestTransformMap_Gitconfig_PlainSection(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"core": map[string]any{"bare": "false", "filemode": "true"},
	}
	got := dialect.TransformMap(dialect.ProfileGitconfig, m)
	core, ok := got["core"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "false", core["bare"])
}

func TestTransformMap_Gitconfig_SubsectionExpanded(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		`remote "origin"`: map[string]any{"url": "https://example.com", "fetch": "+refs/*:refs/*"},
	}
	got := dialect.TransformMap(dialect.ProfileGitconfig, m)

	remote, ok := got["remote"].(map[string]any)
	require.True(t, ok, "expected remote to be a map")

	origin, ok := remote["origin"].(map[string]any)
	require.True(t, ok, "expected origin to be a map")
	assert.Equal(t, "https://example.com", origin["url"])
}

func TestTransformMap_Gitconfig_MultipleSubsections(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		`remote "origin"`:   map[string]any{"url": "https://origin.example.com"},
		`remote "upstream"`: map[string]any{"url": "https://upstream.example.com"},
	}
	got := dialect.TransformMap(dialect.ProfileGitconfig, m)

	remote, ok := got["remote"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, remote, "origin")
	assert.Contains(t, remote, "upstream")
}

func TestTransformMap_Gitconfig_ScalarConflictDoesNotPanic(t *testing.T) {
	t.Parallel()
	// A scalar key named "remote" conflicts with a [remote "origin"] subsection.
	// The scalar takes precedence; no panic should occur.
	m := map[string]any{
		"remote":          "scalar_value",
		`remote "origin"`: map[string]any{"url": "https://example.com"},
	}
	assert.NotPanics(t, func() {
		got := dialect.TransformMap(dialect.ProfileGitconfig, m)
		assert.Equal(t, "scalar_value", got["remote"])
	})
}
