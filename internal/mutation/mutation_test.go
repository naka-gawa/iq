package mutation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"iq/internal/dialect"
	"iq/internal/mutation"
	"iq/internal/parser"
)

func TestApply_UpdateExistingKey(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	targets := []mutation.Target{
		{Section: "database", Key: "host", NewVal: "prod.example.com"},
	}
	require.NoError(t, mutation.Apply(f, targets))

	assert.Equal(t, "prod.example.com", f.Section("database").Key("host").Value())
	// Other keys in the section must be untouched.
	assert.Equal(t, "5432", f.Section("database").Key("port").Value())
}

func TestApply_CreateNewKey(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	targets := []mutation.Target{
		{Section: "database", Key: "sslmode", NewVal: "require"},
	}
	require.NoError(t, mutation.Apply(f, targets))

	assert.Equal(t, "require", f.Section("database").Key("sslmode").Value())
}

func TestApply_CreateNewSection(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	targets := []mutation.Target{
		{Section: "logging", Key: "level", NewVal: "debug"},
	}
	require.NoError(t, mutation.Apply(f, targets))

	assert.True(t, f.HasSection("logging"))
	assert.Equal(t, "debug", f.Section("logging").Key("level").Value())
}

func TestApply_DeleteKey(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	require.True(t, f.Section("database").HasKey("name"))

	targets := []mutation.Target{
		{Section: "database", Key: "name", NewVal: nil},
	}
	require.NoError(t, mutation.Apply(f, targets))

	assert.False(t, f.Section("database").HasKey("name"))
	// Other keys survive.
	assert.True(t, f.Section("database").HasKey("host"))
}

func TestApply_DeleteSection(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	require.True(t, f.HasSection("server"))

	targets := []mutation.Target{
		{Section: "server", Key: "", NewVal: nil},
	}
	require.NoError(t, mutation.Apply(f, targets))

	assert.False(t, f.HasSection("server"))
	// Other sections survive.
	assert.True(t, f.HasSection("database"))
}

func TestApply_DeleteAbsentKey_IsNoOp(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	targets := []mutation.Target{
		{Section: "database", Key: "nonexistent", NewVal: nil},
	}
	require.NoError(t, mutation.Apply(f, targets))
}

func TestApply_DeleteAbsentSection_IsNoOp(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	targets := []mutation.Target{
		{Section: "nosuchsection", Key: "", NewVal: nil},
	}
	require.NoError(t, mutation.Apply(f, targets))
}

func TestApply_MultipleTargets(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	targets := []mutation.Target{
		{Section: "database", Key: "host", NewVal: "newhost"},
		{Section: "server", Key: "port", NewVal: "9090"},
		{Section: "cache", Key: "ttl", NewVal: "60"},
	}
	require.NoError(t, mutation.Apply(f, targets))

	assert.Equal(t, "newhost", f.Section("database").Key("host").Value())
	assert.Equal(t, "9090", f.Section("server").Key("port").Value())
	assert.Equal(t, "60", f.Section("cache").Key("ttl").Value())
}

func TestApply_CommentsPreserved(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/comments.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	comment := f.Section("database").Comment

	targets := []mutation.Target{
		{Section: "database", Key: "host", NewVal: "newhost"},
	}
	require.NoError(t, mutation.Apply(f, targets))

	// Comment must survive the mutation.
	assert.Equal(t, comment, f.Section("database").Comment)
	assert.Equal(t, "newhost", f.Section("database").Key("host").Value())
}

func TestApply_NumericValue(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	targets := []mutation.Target{
		{Section: "server", Key: "port", NewVal: 9443},
	}
	require.NoError(t, mutation.Apply(f, targets))

	assert.Equal(t, "9443", f.Section("server").Key("port").Value())
}
