package parser_test

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"iq/internal/dialect"
	iqerr "iq/internal/errors"
	"iq/internal/parser"
)

const testdataDir = "../../testdata/generic/"

func TestParse_BasicFile(t *testing.T) {
	f, err := parser.Parse(testdataDir+"basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)
	require.NotNil(t, f)

	assert.Equal(t, "localhost", f.Section("database").Key("host").Value())
	assert.Equal(t, "5432", f.Section("database").Key("port").Value())
	assert.Equal(t, "8080", f.Section("server").Key("port").Value())
}

func TestParse_CommentsFile(t *testing.T) {
	f, err := parser.Parse(testdataDir+"comments.ini", dialect.ProfileGeneric)
	require.NoError(t, err)
	require.NotNil(t, f)

	assert.Equal(t, "localhost", f.Section("database").Key("host").Value())
	assert.Equal(t, "127.0.0.1", f.Section("cache").Key("host").Value())
}

func TestParse_GlobalPropertiesFile(t *testing.T) {
	f, err := parser.Parse(testdataDir+"global_properties.ini", dialect.ProfileGeneric)
	require.NoError(t, err)
	require.NotNil(t, f)

	// Global properties (before first section) are stored in the DEFAULT section ("").
	assert.Equal(t, "1.0.0", f.Section("").Key("version").Value())
	assert.Equal(t, "false", f.Section("").Key("debug").Value())
	assert.Equal(t, "myapp", f.Section("app").Key("name").Value())
}

func TestParse_SpecialCharsFile(t *testing.T) {
	f, err := parser.Parse(testdataDir+"special_chars.ini", dialect.ProfileGeneric)
	require.NoError(t, err)
	require.NotNil(t, f)

	assert.Equal(t, "example.com", f.Section("my section").Key("host-name").Value())
	assert.Equal(t, "info", f.Section("my section").Key("log-level").Value())
}

func TestParse_DuplicateKeysFile(t *testing.T) {
	// duplicate_keys.ini requires AllowShadows to parse without error.
	opts := dialect.ProfileGeneric.LoadOptions()
	opts.AllowShadows = true

	f, err := parser.ParseWithOptions(testdataDir+"duplicate_keys.ini", opts)
	require.NoError(t, err)
	require.NotNil(t, f)

	vals := f.Section("service").Key("ExecStart").ValueWithShadows()
	assert.Len(t, vals, 3)
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := parser.Parse("nonexistent.ini", dialect.ProfileGeneric)
	require.Error(t, err)
	assert.True(t, errors.Is(err, iqerr.ErrFileParseFailed))
}

func TestParse_Stdin(t *testing.T) {
	content := "[section]\nkey = value\n"

	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = io.WriteString(w, content)
	require.NoError(t, err)
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		r.Close()
	})

	f, err := parser.Parse("-", dialect.ProfileGeneric)
	require.NoError(t, err)
	assert.Equal(t, "value", f.Section("section").Key("key").Value())
}

func TestParse_EmptyPathUsesStdin(t *testing.T) {
	content := "[s]\nk = v\n"

	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = io.WriteString(w, content)
	require.NoError(t, err)
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		r.Close()
	})

	f, err := parser.Parse("", dialect.ProfileGeneric)
	require.NoError(t, err)
	assert.Equal(t, "v", f.Section("s").Key("k").Value())
}
