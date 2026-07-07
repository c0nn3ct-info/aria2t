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
	return confirmModel{a: a, title: title, text: text, yesLabel: "Remove (y)", onYes: onYes}
}

func (m confirmModel) update(msg tea.KeyMsg) (confirmModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		return m.confirm()
	case "n", "N", "esc", "q":
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

// mouse handles clicks resolved against the confirm modal's regions.
func (m confirmModel) mouse(id string) (confirmModel, tea.Cmd) {
	switch id {
	case "btn:yes":
		return m.confirm()
	case "btn:no":
		m.a.overlay = overlayNone
	}
	return m, nil
}

func (m confirmModel) view() string {
	st := m.a.styles
	yes := st.Red.Reverse(true).Bold(true).Padding(0, 2).Render(m.yesLabel)
	no := lipgloss.NewStyle().Foreground(st.P.FgDim).Padding(0, 2).Render("Cancel (n)")
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render(m.title),
		"",
		st.Text.Render(m.text),
		"",
		yes+"   "+no,
	)
	modal := st.Modal.BorderForeground(m.a.styles.P.Red).Render(body)

	// Register button regions relative to the modal placement.
	offX, offY := m.a.overlayOffset(modal)
	btnY := offY + lipgloss.Height(modal) - 3 // inside bottom border + padding
	yesW := lipgloss.Width(yes)
	x0 := offX + 3 // border + padding
	m.a.hits.add("btn:yes", x0, btnY, x0+yesW-1, btnY)
	noX := x0 + yesW + 3
	m.a.hits.add("btn:no", noX, btnY, noX+lipgloss.Width(no)-1, btnY)
	return modal
}
