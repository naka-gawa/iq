package tui_test

import (
	"path/filepath"
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"

	"iq/internal/tui"
)

func loadFixture(t *testing.T, name string) *ini.File {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(currentFile), "../../testdata/generic", name)
	f, err := ini.Load(path)
	require.NoError(t, err)
	return f
}

// --- #49: scaffold tests ---

func TestNew_InitialState(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	assert.Empty(t, m.Chosen(), "chosen must be empty on creation")
}

func TestNew_InitReturnsCmd(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	cmd := m.Init()
	assert.NotNil(t, cmd, "Init() must return a non-nil Cmd (Blink)")
}

func TestNew_SatisfiesTeaModel(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	var _ tea.Model = tui.New(f, "basic.ini")
}

// --- #50: live preview + keyboard controls ---

func TestUpdate_EmptyQuery_ShowsFullINI(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	view := m.View()
	assert.Contains(t, view, "database", "empty query must render full INI in preview")
	assert.Contains(t, view, "> ", "input prompt must be present")
}

func TestUpdate_Keystroke_LiveEval(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	for _, r := range ".database.host" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(tui.Model)
	}

	view := m.View()
	assert.Contains(t, view, "localhost", "live eval must show query result")
}

func TestUpdate_InvalidExpr_ShowsError(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	for _, r := range "{{bad}}" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(tui.Model)
	}

	view := m.View()
	assert.NotContains(t, view, "\x1b[", "no ANSI codes in error view")
	// Error message should appear in the preview pane (not on stderr).
	// The view must not show a valid result.
	assert.NotContains(t, view, "localhost")
}

// --- #51: Enter / Esc integration ---

func TestUpdate_Enter_SetsChosen(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	for _, r := range ".database.host" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(tui.Model)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(tui.Model)

	assert.Equal(t, ".database.host", final.Chosen())
}

func TestUpdate_Esc_ChosenEmpty(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	for _, r := range ".database.host" {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(tui.Model)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	final := updated.(tui.Model)

	assert.Empty(t, final.Chosen())
}

func TestUpdate_CtrlC_ChosenEmpty(t *testing.T) {
	f := loadFixture(t, "basic.ini")
	m := tui.New(f, "basic.ini")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	final := updated.(tui.Model)

	assert.Empty(t, final.Chosen())
}
