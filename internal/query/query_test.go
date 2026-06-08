package query_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"iq/internal/dialect"
	iqerr "iq/internal/errors"
	"iq/internal/parser"
	"iq/internal/query"
)

func TestExecute_StringValue(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, ".database.host")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "localhost", results[0])
}

func TestExecute_SectionExtraction(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, ".database")
	require.NoError(t, err)
	require.Len(t, results, 1)

	sec, ok := results[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", sec["host"])
	assert.Equal(t, "5432", sec["port"])
}

func TestExecute_RootExtraction(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, ".")
	require.NoError(t, err)
	require.Len(t, results, 1)

	root, ok := results[0].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, root, "database")
	assert.Contains(t, root, "server")
}

func TestExecute_MissingKey_ReturnsErrKeyNotFound(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	_, err = query.Execute(f, ".database.nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, iqerr.ErrKeyNotFound))
}

func TestExecute_MissingSection_ReturnsErrKeyNotFound(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	_, err = query.Execute(f, ".nosuchsection.key")
	require.Error(t, err)
	assert.True(t, errors.Is(err, iqerr.ErrKeyNotFound))
}

func TestExecute_InvalidExpression_ReturnsErrPathInvalid(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	_, err = query.Execute(f, "{{invalid}}")
	require.Error(t, err)
	assert.True(t, errors.Is(err, iqerr.ErrPathInvalid))
}

func TestExecute_Strenv(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	t.Setenv("TEST_IQ_VAR", "injected_value")

	results, err := query.Execute(f, `strenv("TEST_IQ_VAR")`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "injected_value", results[0])
}

func TestExecute_BracketNotation(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/special_chars.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.["`+`my section`+`"]["`+`host-name`+`"]`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "example.com", results[0])
}

func TestExecute_GlobalProperties(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/global_properties.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, ".version")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "1.0.0", results[0])
}

func TestExecute_PipeKeys(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, ".database | keys")
	require.NoError(t, err)
	require.Len(t, results, 1)

	keys, ok := results[0].([]any)
	require.True(t, ok)
	assert.Contains(t, keys, "host")
	assert.Contains(t, keys, "port")
}

func TestFormatValue_String(t *testing.T) {
	s, err := query.FormatValue("hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", s)
}

func TestFormatValue_Nil(t *testing.T) {
	s, err := query.FormatValue(nil)
	require.NoError(t, err)
	assert.Equal(t, "null", s)
}

func TestFormatValue_Map(t *testing.T) {
	s, err := query.FormatValue(map[string]any{"key": "val"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"key":"val"}`, s)
}

func TestFormatValue_Number(t *testing.T) {
	s, err := query.FormatValue(42)
	require.NoError(t, err)
	assert.Equal(t, "42", s)
}

func TestFormatValue_Slice(t *testing.T) {
	s, err := query.FormatValue([]any{"a", "b"})
	require.NoError(t, err)
	assert.JSONEq(t, `["a","b"]`, s)
}

func TestExecute_Select_MatchingValue(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.database.host | select(. == "localhost")`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "localhost", results[0])
}

func TestExecute_Select_NoMatch_ReturnsNilNil(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.database.host | select(. == "other")`)
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestExecute_MissingKey_ThenSelect_ReturnsNilNil(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	// select filters out the null produced by the missing key — empty set, not not-found.
	results, err := query.Execute(f, `.database.nonexistent | select(. != null)`)
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestExecute_Filter_Select_Match(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/filter.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.name | select(. == "auth-service")`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "auth-service", results[0])
}

func TestExecute_Filter_Select_NoMatch(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/filter.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.name | select(. == "other")`)
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestExecute_Filter_Test_Match(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/filter.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.name | test("auth")`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, true, results[0])
}

func TestExecute_Filter_Test_NoMatch(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/filter.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.name | test("xyz")`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, false, results[0])
}

func TestExecute_Filter_SelectTest_Match(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/filter.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.exec | select(test("pre-start"))`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/usr/bin/pre-start.sh", results[0])
}

func TestExecute_Filter_SelectTest_NoMatch(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/filter.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.exec | select(test("nomatch"))`)
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestExecute_Filter_Test_InvalidRegex(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/filter.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	_, err = query.Execute(f, `.service.name | test("[invalid")`)
	require.Error(t, err)
}

func TestExecute_ArrayIter_DuplicateKeys(t *testing.T) {
	opts := dialect.ProfileGeneric.LoadOptions()
	opts.AllowShadows = true

	f, err := parser.ParseWithOptions("../../testdata/generic/duplicate_keys.ini", opts)
	require.NoError(t, err)

	results, err := query.Execute(f, ".service.ExecStart[]")
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, "/usr/bin/setup.sh", results[0])
	assert.Equal(t, "/usr/bin/myapp --config /etc/myapp.conf", results[1])
	assert.Equal(t, "/usr/bin/myapp --secondary", results[2])
}

func TestExecute_ArrayIter_SelectTest_Match(t *testing.T) {
	opts := dialect.ProfileGeneric.LoadOptions()
	opts.AllowShadows = true

	f, err := parser.ParseWithOptions("../../testdata/generic/duplicate_keys.ini", opts)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.ExecStart[] | select(test("setup"))`)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/usr/bin/setup.sh", results[0])
}

func TestExecute_ArrayIter_SelectTest_NoMatch(t *testing.T) {
	opts := dialect.ProfileGeneric.LoadOptions()
	opts.AllowShadows = true

	f, err := parser.ParseWithOptions("../../testdata/generic/duplicate_keys.ini", opts)
	require.NoError(t, err)

	results, err := query.Execute(f, `.service.ExecStart[] | select(test("nomatch"))`)
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestExecute_ArrayIter_OnScalar_ReturnsError(t *testing.T) {
	f, err := parser.Parse("../../testdata/generic/basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	_, err = query.Execute(f, ".database.host[]")
	require.Error(t, err)
}

func TestExecute_DuplicateKeys(t *testing.T) {
	opts := dialect.ProfileGeneric.LoadOptions()
	opts.AllowShadows = true

	f, err := parser.ParseWithOptions("../../testdata/generic/duplicate_keys.ini", opts)
	require.NoError(t, err)

	results, err := query.Execute(f, ".service.ExecStart")
	require.NoError(t, err)
	require.Len(t, results, 1)

	vals, ok := results[0].([]any)
	require.True(t, ok)
	assert.Len(t, vals, 3)
}
