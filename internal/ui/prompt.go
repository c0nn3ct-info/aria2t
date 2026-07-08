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
	in.CharLimit = 256
	in.Width = 68 // Width must precede SetValue: overflow windows on set
	in.SetValue(initial)
	return promptModel{a: a, title: title, input: in, onSubmit: onSubmit}
}

// focusCmd needs a pointer receiver: Focus mutates the model, and focusing
// a copy leaves the stored overlay deaf to typing.
func (m *promptModel) focusCmd() tea.Cmd { return m.input.Focus() }

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

// mouse routes a button click to the same path as its key.
func (m promptModel) mouse(id string) (promptModel, tea.Cmd) {
	if kind, arg := splitID(id); kind == "btn" {
		return m.update(dispatchBtn(arg))
	}
	return m, nil
}

func (m promptModel) view() string {
	st := m.a.styles
	buttons := []button{{"esc", "Cancel", "esc", btnNeutral}, {"enter", "Confirm", "↵", btnPrimary}}
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render(m.title),
		"",
		m.input.View(),
		"",
		m.a.buttonRow(buttons),
	)
	modal := m.a.modalCard(false).Render(body)
	offX, offY := m.a.overlayOffset(modal)
	m.a.registerButtons(offX, offY, modal, buttons)
	return modal
}
