package serializer_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"iq/internal/dialect"
	"iq/internal/parser"
	"iq/internal/serializer"
)

var update = flag.Bool("update", false, "regenerate golden files")

const testdataDir = "../../testdata/generic/"

// goldenPath returns the path of the golden file for a given fixture.
func goldenPath(fixture string) string {
	return testdataDir + filepath.Base(fixture) + ".golden"
}

// checkGolden compares got against the golden file, regenerating it when -update is set.
func checkGolden(t *testing.T, fixture string, got []byte) {
	t.Helper()
	gp := goldenPath(fixture)
	if *update {
		require.NoError(t, os.WriteFile(gp, got, 0644))
		return
	}
	want, err := os.ReadFile(gp)
	require.NoError(t, err, "golden file missing; run: go test ./internal/serializer -update")
	if !bytes.Equal(got, want) {
		t.Errorf("output differs from golden file %s\ngot:\n%s\nwant:\n%s", gp, got, want)
	}
}

// --- WriteINI ---

func TestWriteINI_RoundTrip_Basic(t *testing.T) {
	f, err := parser.Parse(testdataDir+"basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteINI(f, &buf))
	checkGolden(t, "basic.ini", buf.Bytes())
}

func TestWriteINI_RoundTrip_Comments(t *testing.T) {
	f, err := parser.Parse(testdataDir+"comments.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteINI(f, &buf))
	checkGolden(t, "comments.ini", buf.Bytes())
}

func TestWriteINI_RoundTrip_GlobalProperties(t *testing.T) {
	f, err := parser.Parse(testdataDir+"global_properties.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteINI(f, &buf))
	checkGolden(t, "global_properties.ini", buf.Bytes())
}

// --- WriteInPlace ---

func TestWriteInPlace_Atomic(t *testing.T) {
	// Copy fixture to a temp file so we don't modify the original.
	src, err := os.ReadFile(testdataDir + "basic.ini")
	require.NoError(t, err)

	tmp, err := os.CreateTemp(t.TempDir(), "iq-test-*.ini")
	require.NoError(t, err)
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	f, err := parser.Parse(tmp.Name(), dialect.ProfileGeneric)
	require.NoError(t, err)

	require.NoError(t, serializer.WriteInPlace(f, tmp.Name()))

	got, err := os.ReadFile(tmp.Name())
	require.NoError(t, err)
	assert.Contains(t, string(got), "localhost")
}

func TestWriteInPlace_PreservesPermissions(t *testing.T) {
	src, err := os.ReadFile(testdataDir + "basic.ini")
	require.NoError(t, err)

	tmp, err := os.CreateTemp(t.TempDir(), "iq-perm-*.ini")
	require.NoError(t, err)
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	// Set a specific permission.
	require.NoError(t, os.Chmod(tmp.Name(), 0640))

	f, err := parser.Parse(tmp.Name(), dialect.ProfileGeneric)
	require.NoError(t, err)
	require.NoError(t, serializer.WriteInPlace(f, tmp.Name()))

	info, err := os.Stat(tmp.Name())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0640), info.Mode().Perm())
}

func TestWriteInPlace_NonExistentFile_ReturnsError(t *testing.T) {
	f, err := parser.Parse(testdataDir+"basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	err = serializer.WriteInPlace(f, "/nonexistent/path/file.ini")
	assert.Error(t, err)
}

// --- WriteJSON ---

func TestWriteJSON_TypeCoercion(t *testing.T) {
	f, err := parser.Parse(testdataDir+"basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteJSON(f, &buf, false, nil))

	out := buf.String()
	// Numeric port values must be coerced to numbers in JSON output.
	assert.Contains(t, out, `5432`)
	assert.Contains(t, out, `8080`)
	// String host must remain a JSON string.
	assert.Contains(t, out, `"localhost"`)
}

func TestWriteJSON_RawStrings_NoCoercion(t *testing.T) {
	f, err := parser.Parse(testdataDir+"basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteJSON(f, &buf, true, nil))

	out := buf.String()
	// With rawStrings=true, port must be a JSON string, not a number.
	assert.Contains(t, out, `"5432"`)
	assert.Contains(t, out, `"8080"`)
}

func TestWriteJSON_BoolCoercion(t *testing.T) {
	f, err := parser.Parse(testdataDir+"global_properties.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteJSON(f, &buf, false, nil))

	out := buf.String()
	// "false" must be coerced to JSON false (not "false").
	assert.Contains(t, out, `: false`)
}

func TestWriteJSON_FloatCoercion(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "iq-float-*.ini")
	require.NoError(t, err)
	_, err = tmp.WriteString("[section]\npi = 3.14\n")
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	f, err := parser.Parse(tmp.Name(), dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteJSON(f, &buf, false, nil))

	out := buf.String()
	assert.Contains(t, out, `3.14`)
	// Must be a number (not a string) — no surrounding quotes around 3.14.
	assert.NotContains(t, out, `"3.14"`)
}

func TestWriteJSON_IsValidJSON(t *testing.T) {
	f, err := parser.Parse(testdataDir+"basic.ini", dialect.ProfileGeneric)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, serializer.WriteJSON(f, &buf, false, nil))

	// Output must be valid JSON.
	var result map[string]any
	assert.NoError(t, unmarshalJSON(buf.Bytes(), &result))
	assert.Contains(t, result, "database")
	assert.Contains(t, result, "server")
}

func unmarshalJSON(data []byte, v any) error {
	dec := bytes.NewReader(data)
	return json.NewDecoder(dec).Decode(v)
}

func TestWriteMergedINI_SectionsAndGlobals(t *testing.T) {
	m := map[string]any{
		"global": "g",
		"database": map[string]any{
			"host": "prod.example.com",
			"port": "5432",
		},
	}
	var buf bytes.Buffer
	require.NoError(t, serializer.WriteMergedINI(m, &buf))
	out := buf.String()
	assert.Contains(t, out, "global = g")
	assert.Contains(t, out, "[database]")
	assert.Contains(t, out, "host = prod.example.com")
	assert.Contains(t, out, "port = 5432")
}

func TestWriteMergedINI_ArrayExpandsToRepeatedKeys(t *testing.T) {
	m := map[string]any{
		"Service": map[string]any{
			"ExecStart": []any{"/bin/a", "/bin/b"},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, serializer.WriteMergedINI(m, &buf))
	out := buf.String()
	assert.Contains(t, out, "ExecStart = /bin/a")
	assert.Contains(t, out, "ExecStart = /bin/b")
}

func TestWriteMergedINI_NestedMapBecomesSubsection(t *testing.T) {
	m := map[string]any{
		"remote": map[string]any{
			"origin": map[string]any{"url": "https://example.com"},
		},
	}
	var buf bytes.Buffer
	require.NoError(t, serializer.WriteMergedINI(m, &buf))
	assert.Contains(t, buf.String(), `[remote "origin"]`)
}

func TestWriteMergedJSON_CoercesLeaves(t *testing.T) {
	m := map[string]any{
		"database": map[string]any{"port": "5432", "host": "localhost"},
	}
	var buf bytes.Buffer
	require.NoError(t, serializer.WriteMergedJSON(m, &buf, false))
	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	db := got["database"].(map[string]any)
	assert.Equal(t, float64(5432), db["port"]) // coerced to number
	assert.Equal(t, "localhost", db["host"])
}

func TestWriteMergedJSON_RawStrings(t *testing.T) {
	m := map[string]any{"s": map[string]any{"port": "5432"}}
	var buf bytes.Buffer
	require.NoError(t, serializer.WriteMergedJSON(m, &buf, true))
	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, "5432", got["s"].(map[string]any)["port"])
}
