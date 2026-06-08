// Package e2e_test runs integration tests against the compiled iq binary.
// The binary is built once in TestMain and reused across all test cases.
package e2e_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	binaryPath  string
	projectRoot string
	testdataDir string
)

func TestMain(m *testing.M) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot = filepath.Join(filepath.Dir(currentFile), "../..")
	testdataDir = filepath.Join(projectRoot, "testdata", "generic")

	tmpDir, err := os.MkdirTemp("", "iq-e2e-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create temp dir:", err)
		os.Exit(1)
	}
	binaryPath = filepath.Join(tmpDir, "iq")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "build failed:", string(out))
		os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// iq runs the binary with the given arguments and returns stdout, stderr, and exit code.
func iq(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return
}

// copyFixture copies a testdata fixture to a temp file and returns the path.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(testdataDir, name))
	require.NoError(t, err)
	tmp, err := os.CreateTemp(t.TempDir(), "iq-e2e-*.ini")
	require.NoError(t, err)
	_, err = tmp.Write(src)
	require.NoError(t, err)
	require.NoError(t, tmp.Close())
	return tmp.Name()
}

// --- Scenario 1: Comments survive read → write (no-change mutation) ---

func TestScenario1_CommentsPreserved_NoMutation(t *testing.T) {
	path := copyFixture(t, "comments.ini")

	// Write a value that is already set — comments must survive the round-trip.
	stdout, _, exitCode := iq(t, "-i", `.database.host = "localhost"`, path)
	require.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(got)

	assert.Contains(t, content, "# Application configuration")
	assert.Contains(t, content, "; This file is managed by iq")
	assert.Contains(t, content, "# Primary database connection")
	assert.Contains(t, content, "; Cache settings")
}

// --- Scenario 2: Comments survive read → mutate one key → write ---

func TestScenario2_CommentsPreserved_AfterMutation(t *testing.T) {
	path := copyFixture(t, "comments.ini")

	stdout, _, exitCode := iq(t, "-i", `.database.host = "db.prod.example.com"`, path)
	require.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(got)

	// Mutated value must appear.
	assert.Contains(t, content, "db.prod.example.com")
	// Comments must survive.
	assert.Contains(t, content, "# Application configuration")
	assert.Contains(t, content, "; This file is managed by iq")
	assert.Contains(t, content, "# Primary database connection")
	assert.Contains(t, content, "; Cache settings")
}

// --- Scenario 3: Global properties (no-section keys) are preserved ---

func TestScenario3_GlobalProperties_Preserved(t *testing.T) {
	path := copyFixture(t, "global_properties.ini")

	// Mutate a sectioned key; global properties must not be lost.
	stdout, _, exitCode := iq(t, "-i", `.app.env = "production"`, path)
	require.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(got)

	// ini.v1 may align values with spaces, so check key presence not exact spacing.
	assert.Contains(t, content, "version")
	assert.Contains(t, content, "1.0.0")
	assert.Contains(t, content, "debug")
	assert.Contains(t, content, "false")
	assert.Contains(t, content, "production")
}

func TestScenario3_GlobalProperties_QueryWorks(t *testing.T) {
	stdout, _, exitCode := iq(t, ".version", filepath.Join(testdataDir, "global_properties.ini"))
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "1.0.0\n", stdout)
}

// --- Scenario 4: Duplicate keys — default parse returns last value ---
//
// AllowShadows is available via parser.ParseWithOptions but is not yet exposed
// as a CLI flag. Default behavior: last value wins for duplicate keys.

func TestScenario4_DuplicateKeys_DefaultBehavior(t *testing.T) {
	path := copyFixture(t, "duplicate_keys.ini")

	stdout, _, exitCode := iq(t, ".service.ExecStart", path)
	assert.Equal(t, 0, exitCode)
	// Without AllowShadows, last value is returned.
	assert.Contains(t, stdout, "/usr/bin/myapp --secondary")
	assert.NotEmpty(t, stdout)
}

// --- Scenario 5: --in-place is effectively atomic (file is fully written) ---

func TestScenario5_InPlace_AtomicWrite(t *testing.T) {
	path := copyFixture(t, "basic.ini")

	info, err := os.Stat(path)
	require.NoError(t, err)
	originalMode := info.Mode().Perm()

	stdout, _, exitCode := iq(t, "-i", `.database.host = "replaced"`, path)
	require.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(got)

	// File must contain the new value.
	assert.Contains(t, content, "replaced")
	// File must not be empty (full write, not truncation).
	assert.Greater(t, len(content), 10)

	// Permissions must be preserved.
	info2, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, originalMode, info2.Mode().Perm())
}

// --- Scenario 6: Missing key → exit code 2 ---

func TestScenario6_MissingKey_ExitCode2(t *testing.T) {
	_, stderr, exitCode := iq(t, ".database.nosuchkey", filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 2, exitCode)
	assert.Contains(t, stderr, "ERROR:")
}

func TestScenario6_MissingSection_ExitCode2(t *testing.T) {
	_, stderr, exitCode := iq(t, ".nosuchsection.key", filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 2, exitCode)
	assert.Contains(t, stderr, "ERROR:")
}

// --- Scenario 7: Invalid syntax → exit code 1 ---

func TestScenario7_InvalidSyntax_ExitCode1(t *testing.T) {
	_, stderr, exitCode := iq(t, "{{invalid}}", filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr, "ERROR:")
}

func TestScenario7_NonexistentFile_ExitCode1(t *testing.T) {
	_, stderr, exitCode := iq(t, ".key", "/nonexistent/file.ini")
	assert.Equal(t, 1, exitCode)
	assert.Contains(t, stderr, "ERROR:")
}

// --- Scenario 8: No ANSI escape codes in non-TTY stdout ---

func TestScenario8_NoANSI_InNonTTYOutput(t *testing.T) {
	// iq is invoked with stdout piped (non-TTY); output must be clean text.
	stdout, _, exitCode := iq(t, ".database.host", filepath.Join(testdataDir, "basic.ini"))
	require.Equal(t, 0, exitCode)
	// ANSI escape sequences start with ESC (\x1b[).
	assert.NotContains(t, stdout, "\x1b[")
}

func TestScenario8_NoANSI_JSONOutput(t *testing.T) {
	stdout, _, exitCode := iq(t, "--output", "json", ".", filepath.Join(testdataDir, "basic.ini"))
	require.Equal(t, 0, exitCode)
	assert.NotContains(t, stdout, "\x1b[")
}

// --- Additional E2E sanity checks ---

func TestE2E_VersionSubcommand(t *testing.T) {
	stdout, _, exitCode := iq(t, "version")
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "iq version")
}

func TestE2E_HelpFlag(t *testing.T) {
	stdout, _, exitCode := iq(t, "--help")
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "iq is a fast and flexible CLI tool")
}

func TestE2E_JSONOutput_ValidJSON(t *testing.T) {
	stdout, _, exitCode := iq(t, "--output", "json", ".", filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 0, exitCode)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(stdout), "{"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(stdout), "}"))
}

func TestE2E_InPlace_DeleteKey(t *testing.T) {
	path := copyFixture(t, "basic.ini")
	stdout, _, exitCode := iq(t, "-i", "del(.database.port)", path)
	require.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(got), "port = 5432")
	assert.Contains(t, string(got), "host = localhost")
}

func TestE2E_InPlace_PipeExpression(t *testing.T) {
	path := copyFixture(t, "basic.ini")
	stdout, _, exitCode := iq(t, "-i", `.database.host = "newhost" | del(.database.port)`, path)
	require.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(got)
	assert.Contains(t, content, "newhost")
	assert.NotContains(t, content, "port = 5432")
}

func TestE2E_Strenv_Resolution(t *testing.T) {
	t.Setenv("IQ_E2E_HOST", "envhost.example.com")
	path := copyFixture(t, "basic.ini")
	stdout, _, exitCode := iq(t, "-i", ".database.host = strenv(IQ_E2E_HOST)", path)
	require.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), "envhost.example.com")
}

func TestE2E_NoArgs_PrintsHelp(t *testing.T) {
	stdout, _, exitCode := iq(t)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "iq")
}

func TestE2E_RawStrings_PortAsString(t *testing.T) {
	stdout, _, exitCode := iq(t, "--output", "json", "--raw-strings", ".", filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, `"5432"`)
}

// --- Issue #47: select() and test() E2E ---

func TestE2E_Filter_SelectTest_Match(t *testing.T) {
	path := copyFixture(t, "filter.ini")
	stdout, _, exitCode := iq(t, `.service.exec | select(test("pre-start"))`, path)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "/usr/bin/pre-start.sh\n", stdout)
}

func TestE2E_Filter_Select_NoMatch_ExitsZero(t *testing.T) {
	path := copyFixture(t, "filter.ini")
	stdout, _, exitCode := iq(t, `.service.name | select(. == "other")`, path)
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)
}

func TestE2E_Filter_Test_Boolean(t *testing.T) {
	path := copyFixture(t, "filter.ini")
	stdout, _, exitCode := iq(t, `.service.name | test("auth")`, path)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "true\n", stdout)
}

func TestE2E_Filter_Test_InvalidRegex_ExitsOne(t *testing.T) {
	path := copyFixture(t, "filter.ini")
	stdout, stderr, exitCode := iq(t, `.service.name | test("[bad")`, path)
	assert.Equal(t, 1, exitCode)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "ERROR:")
}

// --- Issue #48: array iteration ([]) E2E ---
// Tests use keys[] which does not require AllowShadows.
// Duplicate-key ExecStart[] tests are deferred until --profile (#53) is wired.

func TestE2E_ArrayIter_Keys(t *testing.T) {
	stdout, _, exitCode := iq(t, ".database | keys[]", filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout, "host\n")
	assert.Contains(t, stdout, "name\n")
	assert.Contains(t, stdout, "port\n")
}

func TestE2E_ArrayIter_Keys_SelectMatch(t *testing.T) {
	stdout, _, exitCode := iq(t, `.database | keys[] | select(test("ho"))`, filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "host\n", stdout)
}

func TestE2E_ArrayIter_Keys_SelectNoMatch_ExitsZero(t *testing.T) {
	stdout, _, exitCode := iq(t, `.database | keys[] | select(test("zzz"))`, filepath.Join(testdataDir, "basic.ini"))
	assert.Equal(t, 0, exitCode)
	assert.Empty(t, stdout)
}
