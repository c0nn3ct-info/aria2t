package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpModel is the "?" overlay listing every binding.
type helpModel struct {
	a *App
}

func newHelpModel(a *App) helpModel { return helpModel{a: a} }

func (m helpModel) update(msg tea.KeyMsg) (helpModel, tea.Cmd) {
	m.a.overlay = overlayNone // the footer promises: any key closes
	return m, nil
}

func (m helpModel) view() string {
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
			[2]string{"wheel", "move selection"},
			[2]string{"click", "select · double-click opens"},
			[2]string{"tab / 1-3", "switch tab"},
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
			[2]string{",", "settings"},
			[2]string{"T", "light / dark theme"},
		),
		"",
		section("Stopped tab",
			[2]string{"c", "paste checksum"},
			[2]string{"v", "verify file"},
			[2]string{"R", "re-download"},
			[2]string{"D", "clear stopped list"},
			[2]string{"o", "open folder"},
		),
		"",
		section("Everywhere",
			[2]string{"?", "this help"},
			[2]string{"esc", "back / close"},
			[2]string{"q / ctrl+c", "quit"},
		),
	)
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Keys & mouse"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, left, "      ", right),
		"",
		st.Dim.Render("any key or click closes"),
	)
	modal := st.Modal.Render(body)
	// Whole screen closes on click, matching the footer text.
	m.a.hits.add("help:close", 0, 0, m.a.width-1, m.a.height-1)
	return modal
}
