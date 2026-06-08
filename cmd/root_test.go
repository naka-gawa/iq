package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testdataDir = "../testdata/generic/"

func Test_newVersionCommand(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		revision string
		want     string
	}{
		{
			name:     "Show version",
			version:  "1.0.0",
			revision: "abcde",
			want:     "iq version 1.0.0, revision abcde\n",
		},
		{
			name:     "Show version without revision",
			version:  "dev",
			revision: "",
			want:     "iq version dev, revision \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetVersionInfo(tt.version, tt.revision)

			cmd := newVersionCommand()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.Execute()

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestExecute(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name    string
		args    []string
		wantOut string
		wantErr bool
	}{
		{
			name:    "version command",
			args:    []string{"iq", "version"},
			wantOut: "iq version",
			wantErr: false,
		},
		{
			name:    "nonexistent file error",
			args:    []string{"iq", ".section.key", "/nonexistent/path.ini"},
			wantOut: "",
			wantErr: true,
		},
		{
			name:    "help flag",
			args:    []string{"iq", "--help"},
			wantOut: "iq is a fast and flexible CLI tool for parsing INI files.",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			// Redirect stdout and stderr
			r, w, _ := os.Pipe()
			oldOut := os.Stdout
			oldErr := os.Stderr
			defer func() {
				os.Stdout = oldOut
				os.Stderr = oldErr
			}()
			os.Stdout = w
			os.Stderr = w

			err := Execute()

			w.Close()
			out, _ := io.ReadAll(r)
			output := string(out)

			assert.Equal(t, tt.wantErr, err != nil, "Execute() error = %v, wantErr %v", err, tt.wantErr)
			assert.Contains(t, output, tt.wantOut)
		})
	}
}

// --- dispatch ---

func runDispatch(t *testing.T, expr, filePath string, isInPlace bool, outputFmt string, rawStrings bool) (string, error) {
	t.Helper()
	cmd := NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	err := dispatch(cmd, expr, filePath, isInPlace, outputFmt, rawStrings)
	return buf.String(), err
}

func TestDispatch_QueryPath(t *testing.T) {
	out, err := runDispatch(t, ".database.host", testdataDir+"basic.ini", false, "ini", false)
	require.NoError(t, err)
	assert.Equal(t, "localhost\n", out)
}

func TestDispatch_QuerySection(t *testing.T) {
	out, err := runDispatch(t, ".database", testdataDir+"basic.ini", false, "ini", false)
	require.NoError(t, err)
	assert.Contains(t, out, "localhost")
}

func TestDispatch_QueryMissingKey_ReturnsError(t *testing.T) {
	_, err := runDispatch(t, ".database.nosuchkey", testdataDir+"basic.ini", false, "ini", false)
	assert.Error(t, err)
}

func TestDispatch_JSONOutput(t *testing.T) {
	out, err := runDispatch(t, ".", testdataDir+"basic.ini", false, "json", false)
	require.NoError(t, err)
	assert.Contains(t, out, `"database"`)
	assert.Contains(t, out, `"localhost"`)
}

func TestDispatch_JSONOutput_RawStrings(t *testing.T) {
	out, err := runDispatch(t, ".", testdataDir+"basic.ini", false, "json", true)
	require.NoError(t, err)
	assert.Contains(t, out, `"5432"`)
}

func TestDispatch_InPlace_Assignment(t *testing.T) {
	src, err := os.ReadFile(testdataDir + "basic.ini")
	require.NoError(t, err)

	tmp, err := os.CreateTemp(t.TempDir(), "iq-cmd-*.ini")
	require.NoError(t, err)
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	_, err = runDispatch(t, `.database.host = "newhost"`, tmp.Name(), true, "ini", false)
	require.NoError(t, err)

	got, err := os.ReadFile(tmp.Name())
	require.NoError(t, err)
	assert.Contains(t, string(got), "newhost")
}

func TestDispatch_InPlace_Stdin_WritesToStdout(t *testing.T) {
	// When --in-place and filePath is empty, write INI to stdout.
	src, err := os.ReadFile(testdataDir + "basic.ini")
	require.NoError(t, err)

	tmp, err := os.CreateTemp(t.TempDir(), "iq-stdin-*.ini")
	require.NoError(t, err)
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())

	// Redirect stdin to the temp file so Parse("", ...) reads it.
	f, err := os.Open(tmp.Name())
	require.NoError(t, err)
	defer f.Close()
	oldStdin := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = oldStdin }()

	out, err := runDispatch(t, `.database.host = "fromstdin"`, "", true, "ini", false)
	require.NoError(t, err)
	assert.Contains(t, out, "fromstdin")
}

func TestDispatch_NonexistentFile_ReturnsError(t *testing.T) {
	_, err := runDispatch(t, ".key", "/nonexistent/file.ini", false, "ini", false)
	assert.Error(t, err)
}

// --- isMutationExpr ---

func TestIsMutationExpr(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{".a.b = value", true},
		{"del(.a.b)", true},
		{".a.b = v | del(.c)", true},
		{".a.b", false},
		{".", false},
		{".section | keys", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			assert.Equal(t, tt.want, isMutationExpr(tt.expr))
		})
	}
}

// --- parseDotPath ---

func TestParseDotPath(t *testing.T) {
	tests := []struct {
		path        string
		wantSection string
		wantKey     string
		wantErr     bool
	}{
		{".section.key", "section", "key", false},
		{".section", "section", "", false},
		{"section.key", "section", "key", false},
		{".", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			sec, key, err := parseDotPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantSection, sec)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}

// --- resolveValue ---

func TestResolveValue(t *testing.T) {
	t.Run("quoted string", func(t *testing.T) {
		assert.Equal(t, "hello world", resolveValue(`"hello world"`))
	})
	t.Run("bare value", func(t *testing.T) {
		assert.Equal(t, "barevalue", resolveValue("barevalue"))
	})
	t.Run("strenv", func(t *testing.T) {
		t.Setenv("TEST_IQ_RESOLVE", "injected")
		assert.Equal(t, "injected", resolveValue("strenv(TEST_IQ_RESOLVE)"))
	})
	t.Run("strenv missing var", func(t *testing.T) {
		os.Unsetenv("TEST_IQ_MISSING_VAR")
		assert.Equal(t, "", resolveValue("strenv(TEST_IQ_MISSING_VAR)"))
	})
}

// --- parseMutationTargets ---

func TestParseMutationTargets_Assignment(t *testing.T) {
	targets, err := parseMutationTargets(`.section.key = "value"`)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "section", targets[0].Section)
	assert.Equal(t, "key", targets[0].Key)
	assert.Equal(t, "value", targets[0].NewVal)
}

func TestParseMutationTargets_Deletion(t *testing.T) {
	targets, err := parseMutationTargets("del(.section.key)")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "section", targets[0].Section)
	assert.Equal(t, "key", targets[0].Key)
	assert.Nil(t, targets[0].NewVal)
}

func TestParseMutationTargets_Pipe(t *testing.T) {
	targets, err := parseMutationTargets(`.a.b = "x" | del(.c.d)`)
	require.NoError(t, err)
	require.Len(t, targets, 2)
	assert.Equal(t, "a", targets[0].Section)
	assert.Equal(t, "b", targets[0].Key)
	assert.Equal(t, "c", targets[1].Section)
	assert.Nil(t, targets[1].NewVal)
}

func TestParseMutationTargets_InvalidAssignment(t *testing.T) {
	_, err := parseMutationTargets("no-equals-sign")
	assert.Error(t, err)
}

func TestParseMutationTargets_InvalidPath(t *testing.T) {
	_, err := parseMutationTargets(`. = "value"`)
	assert.Error(t, err)
}

func TestParseDeletion_InvalidPath(t *testing.T) {
	_, err := parseDeletion("del(.)")
	assert.Error(t, err)
}

// --- NewRootCommand flags ---

func TestNewRootCommand_NoArgs_PrintsHelp(t *testing.T) {
	cmd := NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "iq is a fast and flexible CLI tool for parsing INI files.")
}

func TestNewRootCommand_InPlaceFlag(t *testing.T) {
	cmd := NewRootCommand()
	f := cmd.Flags().Lookup("in-place")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestNewRootCommand_OutputFlag(t *testing.T) {
	cmd := NewRootCommand()
	f := cmd.Flags().Lookup("output")
	require.NotNil(t, f)
	assert.Equal(t, "ini", f.DefValue)
}

// --- writeINI path (in-place + non-mutation falls back to query) ---

func TestDispatch_InPlace_WithQueryExpr_UsesQueryPath(t *testing.T) {
	// isInPlace=true but expr is not a mutation expr → query path is used.
	out, err := runDispatch(t, ".database.host", testdataDir+"basic.ini", true, "ini", false)
	require.NoError(t, err)
	assert.Equal(t, "localhost\n", out)
}

// helpers

func writeTempINI(t *testing.T, content string) string {
	t.Helper()
	tmp, err := os.CreateTemp(t.TempDir(), "iq-*.ini")
	require.NoError(t, err)
	_, err = tmp.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())
	return tmp.Name()
}

func TestDispatch_InPlace_DeleteKey(t *testing.T) {
	path := writeTempINI(t, "[section]\nkey = value\nother = keep\n")
	_, err := runDispatch(t, "del(.section.key)", path, true, "ini", false)
	require.NoError(t, err)
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(got), "key = value")
	assert.Contains(t, string(got), "other")
}

func TestDispatch_InvalidExpr_ReturnsError(t *testing.T) {
	_, err := runDispatch(t, "{{invalid}}", testdataDir+"basic.ini", false, "ini", false)
	assert.Error(t, err)
}

func TestDispatch_QueryRoot(t *testing.T) {
	out, err := runDispatch(t, ".", testdataDir+"basic.ini", false, "ini", false)
	require.NoError(t, err)
	// Root query returns a JSON-like map representation via FormatValue.
	assert.True(t, strings.Contains(out, "database") || strings.Contains(out, "{"))
}

func TestResolveValue_ShortQuotedString(t *testing.T) {
	// Edge: single char quoted string.
	assert.Equal(t, "x", resolveValue(`"x"`))
}

func TestParseMutationTargets_EmptyParts_Skipped(t *testing.T) {
	// Leading/trailing pipes result in empty parts that should be skipped.
	targets, err := parseMutationTargets(`| .a.b = "v" |`)
	require.NoError(t, err)
	require.Len(t, targets, 1)
}

func TestDispatch_InPlace_AssignSectionOnly(t *testing.T) {
	// Assign at section level (no key) — parseDotPath returns section, key="".
	path := writeTempINI(t, "[section]\nkey = old\n")
	// This is technically a valid parse but mutation with empty key is unusual;
	// just verify no panic and an error or success is returned.
	_, _ = runDispatch(t, `.newsection = "val"`, path, true, "ini", false)
}

func TestParseMutationTargets_AssignmentWithBareValue(t *testing.T) {
	targets, err := parseMutationTargets(".section.key = barevalue")
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, "barevalue", targets[0].NewVal)
}

// Verify the path separator in goldenPath-style resolution is not broken.
func TestFilepathBase(t *testing.T) {
	assert.Equal(t, "file.ini", filepath.Base("some/path/file.ini"))
}
