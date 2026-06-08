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

func TestUpdate_QueryEvaluation(t *testing.T) {
	tests := []struct {
		name            string
		inputRunes      string
		expectInView    []string
		expectNotInView []string
	}{
		{
			name:         "empty query shows full INI",
			inputRunes:   "",
			expectInView: []string{"database", "> "},
		},
		{
			name:         "valid query shows result",
			inputRunes:   ".database.host",
			expectInView: []string{"localhost"},
		},
		{
			name:            "invalid expression shows no ANSI and no valid result",
			inputRunes:      "{{bad}}",
			expectNotInView: []string{"\x1b[", "localhost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := loadFixture(t, "basic.ini")
			m := tui.New(f, "basic.ini")

			for _, r := range tt.inputRunes {
				updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
				m = updated.(tui.Model)
			}

			view := m.View()
			for _, want := range tt.expectInView {
				assert.Contains(t, view, want)
			}
			for _, notWant := range tt.expectNotInView {
				assert.NotContains(t, view, notWant)
			}
		})
	}
}

// --- #51: Enter / Esc integration ---

func TestUpdate_KeyboardControls(t *testing.T) {
	tests := []struct {
		name           string
		inputRunes     string
		terminationKey tea.KeyType
		expectChosen   string
	}{
		{
			name:           "Enter commits query",
			inputRunes:     ".database.host",
			terminationKey: tea.KeyEnter,
			expectChosen:   ".database.host",
		},
		{
			name:           "Esc clears chosen",
			inputRunes:     ".database.host",
			terminationKey: tea.KeyEsc,
			expectChosen:   "",
		},
		{
			name:           "Ctrl+C clears chosen",
			inputRunes:     "",
			terminationKey: tea.KeyCtrlC,
			expectChosen:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := loadFixture(t, "basic.ini")
			m := tui.New(f, "basic.ini")

			for _, r := range tt.inputRunes {
				updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
				m = updated.(tui.Model)
			}

			updated, _ := m.Update(tea.KeyMsg{Type: tt.terminationKey})
			final := updated.(tui.Model)

			assert.Equal(t, tt.expectChosen, final.Chosen())
		})
	}
}
