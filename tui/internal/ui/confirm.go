package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// confirmModel is a yes/no modal guarding destructive actions.
type confirmModel struct {
	a        *App
	title    string
	text     string
	yesLabel string
	onYes    func() tea.Cmd
}

func newConfirmModel(a *App, title, text string, onYes func() tea.Cmd) confirmModel {
	return confirmModel{a: a, title: safeText(title), text: safeText(text), yesLabel: "Remove", onYes: onYes}
}

func (m confirmModel) update(msg tea.KeyMsg) (confirmModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.confirm()
	case "n", "N", "esc", "q", "enter":
		m.a.overlay = overlayNone
	}
	return m, nil
}

func (m confirmModel) confirm() (confirmModel, tea.Cmd) {
	m.a.overlay = overlayNone
	if m.onYes != nil {
		return m, m.onYes()
	}
	return m, nil
}

// mouse handles clicks resolved against the confirm modal's buttons: a click
// runs the same path as pressing the button's key (y/n).
func (m confirmModel) mouse(id string) (confirmModel, tea.Cmd) {
	if kind, arg := splitID(id); kind == "btn" {
		return m.update(dispatchBtn(arg))
	}
	return m, nil
}

func (m confirmModel) buttons() []button {
	return []button{
		{"n", "Cancel", "n", btnGreen}, // green = the safe way out
		{"y", m.yesLabel, "y", btnRed}, // red = the destructive action
	}
}

func (m confirmModel) view() string {
	st := m.a.styles
	buttons := m.buttons()
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render(m.title),
		"",
		st.Text.Render(m.text),
		"",
		m.a.buttonRow(buttons),
	)
	modal := m.a.modalCard(true).Render(body)
	offX, offY := m.a.overlayOffset(modal)
	m.a.registerButtons(offX, offY, modal, buttons)
	return modal
}
