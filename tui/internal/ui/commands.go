package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type commandEntry struct {
	name, key string
}

type commandModel struct {
	a      *App
	query  textinput.Model
	cursor int
}

func newCommandModel(a *App) commandModel {
	in := textinput.New()
	in.Placeholder = "Type a command…"
	in.Width = 48
	return commandModel{a: a, query: in}
}

func (m *commandModel) focusCmd() tea.Cmd { return m.query.Focus() }

func (m commandModel) entries() []commandEntry {
	all := []commandEntry{
		{"Add download", "a"},
		{"Global stats", "g"},
		{"Open settings", ","},
		{"Bandwidth scheduler", "S"},
		{"Server switcher", "s"},
		{"Pause all downloads", "P"},
		{"Resume all downloads", "U"},
		{"Keyboard help", "?"},
	}
	q := strings.ToLower(strings.TrimSpace(m.query.Value()))
	if q == "" {
		return all
	}
	out := make([]commandEntry, 0, len(all))
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.name), q) {
			out = append(out, e)
		}
	}
	return out
}

func (m *commandModel) clamp() {
	n := len(m.entries())
	if n == 0 {
		m.cursor = 0
	} else if m.cursor >= n {
		m.cursor = n - 1
	} else if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m commandModel) choose() (commandModel, tea.Cmd) {
	entries := m.entries()
	if len(entries) == 0 {
		return m, nil
	}
	e := entries[m.cursor]
	m.a.overlay = overlayNone
	_, cmd := m.a.handleKey(key_(e.key))
	return m, cmd
}

func (m commandModel) update(msg tea.KeyMsg) (commandModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+p":
		m.a.overlay = overlayNone
		return m, nil
	case "up":
		m.cursor--
		m.clamp()
		return m, nil
	case "down":
		m.cursor++
		m.clamp()
		return m, nil
	case "enter":
		return m.choose()
	}
	var cmd tea.Cmd
	m.query, cmd = updateInput(m.query, msg)
	m.cursor = 0
	return m, cmd
}

func (m commandModel) mouse(id string) (commandModel, tea.Cmd) {
	kind, arg := splitID(id)
	switch kind {
	case "cmd":
		m.cursor = argInt(arg)
		m.clamp()
		return m.choose()
	case "btn":
		return m.update(dispatchBtn(arg))
	}
	return m, nil
}

func (m commandModel) view() string {
	st := m.a.styles
	entries := m.entries()
	lines := []string{st.Title.Render("Commands"), m.query.View(), ""}
	rowStart := len(lines)
	if len(entries) == 0 {
		lines = append(lines, st.Dim.Render("No matching commands"))
	} else {
		for i, e := range entries {
			marker, style := "  ", st.Text
			if i == m.cursor {
				marker, style = st.Brand.Render("▸ "), st.Title
			}
			line := marker + style.Render(pad(e.name, 36)) + st.Key.Render(e.key)
			if i == m.cursor {
				line = st.RowSel.Render(line)
			}
			lines = append(lines, line)
		}
	}
	buttons := []button{{"esc", "Close", "esc", btnRed}, {"enter", "Run", "↵", btnGreen}}
	lines = append(lines, "", m.a.buttonRow(buttons))
	modal := m.a.modalCard(false).Render(strings.Join(lines, "\n"))
	offX, offY := m.a.overlayOffset(modal)
	for i := range entries {
		y := offY + 2 + rowStart + i
		m.a.hits.add(fmt.Sprintf("cmd:%d", i), offX+3, y, offX+lipgloss.Width(modal)-4, y)
	}
	m.a.registerButtons(offX, offY, modal, buttons)
	return modal
}
