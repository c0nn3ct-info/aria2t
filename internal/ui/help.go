package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpModel is the "?" overlay listing every binding. It scrolls when the list
// is taller than the terminal, so no binding is ever clipped off a short window.
type helpModel struct {
	a      *App
	offset int
}

func newHelpModel(a *App) helpModel { return helpModel{a: a} }

// availRows is how many content rows fit inside the modal (minus border,
// padding, title, and footer).
func (m helpModel) availRows() int {
	v := m.a.height - 6
	if v < 3 {
		v = 3
	}
	return v
}

func (m helpModel) update(msg tea.KeyMsg) (helpModel, tea.Cmd) {
	maxOff := max(0, len(m.contentLines())-m.availRows())
	switch msg.String() {
	case "up", "k":
		m.offset = max(0, m.offset-1)
		return m, nil
	case "down", "j":
		m.offset = min(maxOff, m.offset+1)
		return m, nil
	}
	m.a.overlay = overlayNone // the footer promises: any other key closes
	return m, nil
}

// contentLines builds the two-column key reference as a flat list of lines.
func (m helpModel) contentLines() []string {
	st := m.a.styles
	section := func(title string, pairs ...[2]string) string {
		lines := []string{st.Dim.Render(strings.ToUpper(title))}
		for _, p := range pairs {
			lines = append(lines, st.Key.Render(pad(p[0], 12))+st.Text.Render(p[1]))
		}
		return strings.Join(lines, "\n")
	}
	left := lipgloss.JoinVertical(lipgloss.Left,
		section("Downloads",
			[2]string{"↑↓ / jk", "move selection"},
			[2]string{"click", "select · every hint is clickable"},
			[2]string{"tab / 1-4", "All / Active / Waiting / Stopped"},
			[2]string{"/", "filter by name"},
			[2]string{"space", "pause / resume"},
			[2]string{"P / U", "pause all / resume all"},
			[2]string{"enter", "details"},
			[2]string{"a", "add download"},
			[2]string{"d", "remove (asks first)"},
			[2]string{"y", "copy source URL"},
			[2]string{"l", "speed limit"},
		),
		"",
		section("Waiting tab",
			[2]string{"J / K", "grab + move in queue"},
			[2]string{"gg / G", "move to top / bottom"},
			[2]string{"enter", "drop · esc cancels"},
		),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		section("Screens",
			[2]string{"g", "global stats"},
			[2]string{"s", "server switcher"},
			[2]string{"S", "bandwidth scheduler"},
			[2]string{"t", "seeding / trackers"},
			[2]string{",", "settings (theme here)"},
		),
		"",
		section("Stopped tab",
			[2]string{"c", "paste checksum"},
			[2]string{"v", "verify file"},
			[2]string{"R", "re-download"},
			[2]string{"X", "clear stopped list"},
			[2]string{"o", "open folder"},
		),
		"",
		section("File picker (f / on add)",
			[2]string{"space", "toggle file / folder"},
			[2]string{"a / n", "select all / none"},
			[2]string{"h / l", "fold / unfold folder"},
			[2]string{"enter", "confirm · esc cancels"},
			[2]string{"^o (in Add)", "browse disk for a file"},
		),
		"",
		section("Everywhere",
			[2]string{"?", "this help"},
			[2]string{"esc", "back / close"},
			[2]string{"q / ctrl+c", "quit"},
		),
	)
	return strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, left, "      ", right), "\n")
}

func (m helpModel) view() string {
	st := m.a.styles
	full := m.contentLines()
	avail := m.availRows()
	footer := "any key or click closes"
	if len(full) > avail {
		off := min(m.offset, len(full)-avail)
		full = full[off : off+avail]
		footer = "↑↓ scroll · any other key closes"
	}
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Keys & mouse"),
		"",
		strings.Join(full, "\n"),
		"",
		st.Dim.Render(footer),
	)
	modal := st.Modal.Render(body)
	m.a.hits.add("help:close", 0, 0, m.a.width-1, m.a.height-1)
	return modal
}
