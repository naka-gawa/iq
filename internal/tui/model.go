// Package tui implements an interactive query mode powered by bubbletea.
package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"gopkg.in/ini.v1"

	"iq/internal/query"
)

// Model is the bubbletea model for the interactive TUI.
type Model struct {
	file     *ini.File
	filePath string
	input    textinput.Model
	result   string
	errMsg   string
	chosen   string
}

// New creates a new Model ready to accept query input.
func New(f *ini.File, filePath string) Model {
	ti := textinput.New()
	ti.Placeholder = "type a jq query…"
	ti.Focus()

	m := Model{
		file:     f,
		filePath: filePath,
		input:    ti,
	}
	m.result, m.errMsg = evalQuery(f, "")
	return m
}

// Chosen returns the query string committed by pressing Enter (empty on Esc/Ctrl+C).
func (m Model) Chosen() string { return m.chosen }

// Init satisfies tea.Model; starts the cursor blink.
func (m Model) Init() tea.Cmd { return textinput.Blink }

// Update satisfies tea.Model; handles keyboard input and live query evaluation.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.chosen = m.input.Value()
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.chosen = ""
			return m, tea.Quit
		default:
			m.input, cmd = m.input.Update(msg)
			m.result, m.errMsg = evalQuery(m.file, m.input.Value())
			return m, cmd
		}
	}
	return m, nil
}

// View satisfies tea.Model; renders the two-pane layout.
func (m Model) View() tea.View {
	preview := m.result
	if m.errMsg != "" {
		preview = m.errMsg
	}
	v := tea.NewView(preview + "\n\n> " + m.input.View())
	v.AltScreen = true
	return v
}

// Run starts the bubbletea program and returns the chosen query string.
// Returns an empty string when the user exits without committing (Esc/Ctrl+C).
func Run(f *ini.File, filePath string) (string, error) {
	m := New(f, filePath)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	return final.(Model).chosen, nil
}

// evalQuery runs expr against f and returns (result, errMsg).
// An empty expr renders the full INI content.
func evalQuery(f *ini.File, expr string) (result, errMsg string) {
	if expr == "" {
		var sb strings.Builder
		if _, err := f.WriteTo(&sb); err != nil {
			return "", err.Error()
		}
		return sb.String(), ""
	}

	vals, err := query.Execute(f, expr)
	if err != nil {
		return "", err.Error()
	}

	lines := make([]string, 0, len(vals))
	for _, v := range vals {
		s, fmtErr := query.FormatValue(v)
		if fmtErr != nil {
			return "", fmtErr.Error()
		}
		lines = append(lines, s)
	}
	return strings.Join(lines, "\n"), ""
}
