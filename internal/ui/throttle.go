package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	downPresets = []string{"0", "1M", "5M", "10M"}
	upPresets   = []string{"0", "256K", "1M"}
)

// throttleModel is the per-download speed-limit popup.
type throttleModel struct {
	a     *App
	gid   string
	name  string
	speed int64

	row     int // 0 download, 1 upload
	downSel int // index into downPresets; len == custom
	upSel   int
	custom  [2]textinput.Model
	editing bool
}

func newThrottleModel(a *App) throttleModel {
	var custom [2]textinput.Model
	for i := range custom {
		custom[i] = textinput.New()
		custom[i].Placeholder = "8M"
		custom[i].Width = 8
		custom[i].CharLimit = 10
	}
	return throttleModel{a: a, custom: custom}
}

// loadCmd reads current limits so the popup opens on the live values.
func (m throttleModel) loadCmd() tea.Cmd {
	c := m.a.client
	gid := m.gid
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		opts, err := c.GetOption(ctx, gid)
		return gidOptionsMsg{gid: gid, opts: opts, err: err}
	}
}

// absorbOptions selects the chips matching the download's current limits.
func (m *throttleModel) absorbOptions(msg gidOptionsMsg) {
	if msg.gid != m.gid {
		return
	}
	sel := func(presets []string, value string, input *textinput.Model) int {
		if value == "" {
			value = "0"
		}
		value = NormalizeLimit(value) // aria2 reports limits as plain bytes
		for i, p := range presets {
			if p == value {
				return i
			}
		}
		input.SetValue(value)
		return len(presets)
	}
	m.downSel = sel(downPresets, msg.opts["max-download-limit"], &m.custom[0])
	m.upSel = sel(upPresets, msg.opts["max-upload-limit"], &m.custom[1])
}

func (m throttleModel) presets(row int) []string {
	if row == 0 {
		return downPresets
	}
	return upPresets
}

func (m throttleModel) selection(row int) int {
	if row == 0 {
		return m.downSel
	}
	return m.upSel
}

func (m *throttleModel) setSelection(row, v int) {
	if row == 0 {
		m.downSel = v
	} else {
		m.upSel = v
	}
}

// values resolves both rows into aria2 option values.
func (m throttleModel) values() (map[string]string, error) {
	out := map[string]string{}
	for row, key := range []string{"max-download-limit", "max-upload-limit"} {
		presets := m.presets(row)
		sel := m.selection(row)
		if sel < len(presets) {
			out[key] = presets[sel]
			continue
		}
		v, err := ParseLimit(m.custom[row].Value())
		if err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, nil
}

func (m throttleModel) update(msg tea.KeyMsg) (throttleModel, tea.Cmd) {
	a := m.a
	key := msg.String()
	if m.editing {
		switch key {
		case "esc", "enter", "tab":
			m.editing = false
			m.custom[m.row].Blur()
			if key == "enter" {
				return m.apply()
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.custom[m.row], cmd = m.custom[m.row].Update(msg)
		return m, cmd
	}
	switch key {
	case "esc":
		a.overlay = overlayNone
	case "tab", "j", "k", "down", "up":
		m.row = 1 - m.row
	case "h", "left":
		if s := m.selection(m.row); s > 0 {
			m.setSelection(m.row, s-1)
		}
	case "l", "right":
		if s := m.selection(m.row); s < len(m.presets(m.row)) {
			m.setSelection(m.row, s+1)
		}
	case "enter":
		if m.selection(m.row) == len(m.presets(m.row)) && !m.editing {
			m.editing = true
			return m, m.custom[m.row].Focus()
		}
		return m.apply()
	}
	return m, nil
}

func (m throttleModel) apply() (throttleModel, tea.Cmd) {
	a := m.a
	opts, err := m.values()
	if err != nil {
		return m, a.flash(err.Error(), true)
	}
	gid := m.gid
	a.overlay = overlayNone
	return m, a.rpcCmd("limits applied", func(ctx context.Context, c api) error {
		return c.ChangeOption(ctx, gid, opts)
	})
}

func (m throttleModel) chipRow(row int, label string) string {
	st := m.a.styles
	presets := m.presets(row)
	sel := m.selection(row)
	chips := make([]string, 0, len(presets)+1)
	for i, p := range presets {
		text := FmtLimit(p)
		if i == sel {
			chips = append(chips, st.Yellow.Bold(true).Reverse(true).Padding(0, 1).Render(text))
		} else {
			chips = append(chips, st.Dim.Render("[ "+text+" ]"))
		}
	}
	customText := "custom…"
	if v := m.custom[row].Value(); v != "" {
		customText = v
	}
	if sel == len(presets) {
		if m.editing && m.row == row {
			chips = append(chips, m.custom[row].View())
		} else {
			chips = append(chips, st.Yellow.Bold(true).Reverse(true).Padding(0, 1).Render(customText))
		}
	} else {
		chips = append(chips, st.Dim.Render("[ "+customText+" ]"))
	}
	cursor := "  "
	if m.row == row {
		cursor = st.Yellow.Render("▸ ")
	}
	return cursor + st.Dim.Render(label) + "\n  " + lipgloss.JoinHorizontal(lipgloss.Center, chips...)
}

func (m throttleModel) view() string {
	st := m.a.styles
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Throttle")+"  "+st.Dim.Render(m.name),
		"",
		m.chipRow(0, "Download limit"),
		"",
		m.chipRow(1, "Upload limit"),
		"",
		st.Dim.Render("Currently ")+st.Cyan.Render("▼ "+FmtSpeed(m.speed))+st.Dim.Render(" · limit takes effect immediately"),
		"",
		st.Dim.Render("h/l select · tab row · ")+st.Yellow.Render("↵ apply")+st.Dim.Render(" · esc cancel"),
	)
	return st.Modal.BorderForeground(m.a.styles.P.Yellow).Render(body)
}
