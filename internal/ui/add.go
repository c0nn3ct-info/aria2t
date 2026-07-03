package ui

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	addTabURL = iota
	addTabTorrent
	addTabMetalink
)

// addModel is the "Add download" overlay.
type addModel struct {
	a   *App
	tab int

	uris  textarea.Model  // URL tab: one URI per line (mirrors of one file)
	file  textinput.Model // torrent/metalink path
	dir   textinput.Model
	split textinput.Model
	out   textinput.Model

	startNow bool
	rename   bool

	focus int // 0 uris/file, 1 dir, 2 split, 3 out (when rename)
}

func newAddModel(a *App) addModel {
	uris := textarea.New()
	uris.Placeholder = "https://…"
	uris.SetHeight(3)
	uris.SetWidth(56)
	file := textinput.New()
	file.Placeholder = "/path/to/file.torrent"
	file.Width = 52
	dir := textinput.New()
	dir.SetValue(a.cfg.Dir)
	dir.Width = 22
	split := textinput.New()
	split.SetValue(fmt.Sprintf("%d", a.cfg.Split))
	split.Width = 6
	out := textinput.New()
	out.Placeholder = "new-name.iso"
	out.Width = 22
	return addModel{a: a, uris: uris, file: file, dir: dir, split: split, out: out, startNow: true}
}

func (m addModel) focusCmd() tea.Cmd { return m.uris.Focus() }

// fields returns the focusable inputs for the current tab.
func (m *addModel) applyFocus() tea.Cmd {
	m.uris.Blur()
	m.file.Blur()
	m.dir.Blur()
	m.split.Blur()
	m.out.Blur()
	switch m.focus {
	case 0:
		if m.tab == addTabURL {
			return m.uris.Focus()
		}
		return m.file.Focus()
	case 1:
		return m.dir.Focus()
	case 2:
		return m.split.Focus()
	default:
		return m.out.Focus()
	}
}

func (m addModel) update(msg tea.KeyMsg) (addModel, tea.Cmd) {
	a := m.a
	key := msg.String()
	switch key {
	case "esc":
		a.overlay = overlayNone
		return m, nil
	case "ctrl+t":
		m.tab = (m.tab + 1) % 3
		m.focus = 0
		return m, m.applyFocus()
	case "tab":
		limit := 3
		if m.rename {
			limit = 4
		}
		m.focus = (m.focus + 1) % limit
		return m, m.applyFocus()
	case "ctrl+s":
		m.startNow = !m.startNow
		return m, nil
	case "ctrl+r":
		m.rename = !m.rename
		if !m.rename && m.focus == 3 {
			m.focus = 0
		}
		return m, m.applyFocus()
	case "enter":
		// Textarea consumes enter for new lines; submit is ctrl+d there.
		if m.tab == addTabURL && m.focus == 0 {
			break
		}
		return m.submit()
	case "ctrl+d":
		return m.submit()
	}
	var cmd tea.Cmd
	switch {
	case m.focus == 0 && m.tab == addTabURL:
		m.uris, cmd = m.uris.Update(msg)
	case m.focus == 0:
		m.file, cmd = m.file.Update(msg)
	case m.focus == 1:
		m.dir, cmd = m.dir.Update(msg)
	case m.focus == 2:
		m.split, cmd = m.split.Update(msg)
	default:
		m.out, cmd = m.out.Update(msg)
	}
	return m, cmd
}

func (m addModel) options() map[string]string {
	opts := map[string]string{}
	if v := strings.TrimSpace(m.dir.Value()); v != "" {
		opts["dir"] = expandHome(v)
	}
	if v := strings.TrimSpace(m.split.Value()); v != "" {
		opts["split"] = v
		// aria2 rejects max-connection-per-server above 16.
		if n, err := strconv.Atoi(v); err == nil && n > 16 {
			opts["max-connection-per-server"] = "16"
		} else {
			opts["max-connection-per-server"] = v
		}
	}
	if m.rename {
		if v := strings.TrimSpace(m.out.Value()); v != "" {
			opts["out"] = v
		}
	}
	if !m.startNow {
		opts["pause"] = "true"
	}
	return opts
}

func (m addModel) submit() (addModel, tea.Cmd) {
	a := m.a
	opts := m.options()
	switch m.tab {
	case addTabURL:
		var uris []string
		for _, line := range strings.Split(m.uris.Value(), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				uris = append(uris, line)
			}
		}
		if len(uris) == 0 {
			return m, a.flash("enter at least one URI", true)
		}
		a.overlay = overlayNone
		return m, a.rpcCmd("added", func(ctx context.Context, c api) error {
			_, err := c.AddURI(ctx, uris, opts)
			return err
		})
	default:
		path := expandHome(strings.TrimSpace(m.file.Value()))
		if path == "" {
			return m, a.flash("enter a file path", true)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return m, a.flash(err.Error(), true)
		}
		b64 := base64.StdEncoding.EncodeToString(raw)
		isTorrent := m.tab == addTabTorrent
		a.overlay = overlayNone
		return m, a.rpcCmd("added "+filepath.Base(path), func(ctx context.Context, c api) error {
			if isTorrent {
				_, err := c.AddTorrent(ctx, b64, opts)
				return err
			}
			_, err := c.AddMetalink(ctx, b64, opts)
			return err
		})
	}
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func (m addModel) view() string {
	st := m.a.styles
	tabs := make([]string, 3)
	for i, n := range []string{"URL", "Torrent", "Metalink"} {
		if i == m.tab {
			tabs[i] = st.Badge.Render(n)
		} else {
			tabs[i] = st.Dim.Render("[ " + n + " ]")
		}
	}
	var src string
	if m.tab == addTabURL {
		src = st.Dim.Render("URIs — one per line (mirrors)") + "\n" + m.uris.View()
	} else {
		src = st.Dim.Render("File path") + "\n" + m.file.View()
	}
	check := func(on bool, label string) string {
		if on {
			return st.Green.Render("[x] ") + st.Text.Render(label)
		}
		return st.Dim.Render("[ ] " + label)
	}
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Add download")+"   "+st.Dim.Render("esc to close"),
		"",
		strings.Join(tabs, " ")+"  "+st.Dim.Render("^t switch"),
		"",
		src,
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			st.Dim.Render("Save to")+"\n"+m.dir.View(),
			"   ",
			st.Dim.Render("Connections")+"\n"+m.split.View(),
		),
		"",
		check(m.startNow, "Start immediately (^s)")+"   "+check(m.rename, "Rename file (^r)"),
		func() string {
			if m.rename {
				return "\n" + st.Dim.Render("Save as") + "\n" + m.out.View()
			}
			return ""
		}(),
		"",
		st.Dim.Render("tab next field · ")+st.Green.Render("↵/^d add"),
	)
	return st.Modal.Render(body)
}
