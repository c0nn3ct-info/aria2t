package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// promptModel is a one-line input modal (used for checksum entry).
type promptModel struct {
	a        *App
	title    string
	input    textinput.Model
	onSubmit func(value string) tea.Cmd
}

func newPromptModel(a *App, title, initial string, onSubmit func(string) tea.Cmd) promptModel {
	in := textinput.New()
	in.SetValue(initial)
	in.CharLimit = 256
	in.Width = 68
	return promptModel{a: a, title: title, input: in, onSubmit: onSubmit}
}

func (m promptModel) focusCmd() tea.Cmd { return m.input.Focus() }

func (m promptModel) update(msg tea.KeyMsg) (promptModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.a.overlay = overlayNone
		return m, nil
	case "enter":
		m.a.overlay = overlayNone
		if m.onSubmit != nil {
			return m, m.onSubmit(m.input.Value())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m promptModel) view() string {
	st := m.a.styles
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render(m.title),
		"",
		m.input.View(),
		"",
		st.Dim.Render("↵ confirm · esc cancel"),
	)
	return st.Modal.Render(body)
}
