package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/config"
)

// setField is one editable entry: a text input or a toggle. A readonly
// field only displays daemon state (options aria2 fixes at startup).
type setField struct {
	label    string
	optKey   string // global option key; "" = config-backed field
	toggle   bool
	on       bool
	readonly bool
	input    textinput.Model
}

// settingsModel is the sidebar-driven settings screen.
type settingsModel struct {
	a        *App
	section  int
	focus    int
	inSide   bool // true: sidebar focused, false: fields focused
	dirty    bool
	sections []string
	fields   [][]setField
	loaded   map[string]string // global options as fetched
}

func newSettingsModel(a *App) settingsModel {
	mk := func(label, optKey, value string, width int) setField {
		in := textinput.New()
		in.Width = width // Width must precede SetValue: overflow windows on set
		in.SetValue(value)
		return setField{label: label, optKey: optKey, input: in}
	}
	tg := func(label, optKey string) setField {
		return setField{label: label, optKey: optKey, toggle: true}
	}
	tgRO := func(label, optKey string) setField {
		return setField{label: label, optKey: optKey, toggle: true, readonly: true}
	}
	srv := a.cfg.ActiveServer()
	secret := mk("RPC secret", "", srv.Secret, 20)
	secret.input.EchoMode = textinput.EchoPassword
	m := settingsModel{
		a:        a,
		inSide:   true,
		sections: []string{"Connection", "Limits", "Directories", "BitTorrent", "Interface"},
		fields: [][]setField{
			{ // Connection — config-backed
				mk("Host", "", srv.Host, 20),
				mk("Port", "", fmt.Sprintf("%d", srv.Port), 8),
				secret,
				tg("Use websocket (off = http)", ""),
			},
			{ // Limits
				mk("Max concurrent downloads", "max-concurrent-downloads", "", 8),
				mk("Global download limit", "max-overall-download-limit", "", 10),
				mk("Global upload limit", "max-overall-upload-limit", "", 10),
			},
			{ // Directories
				mk("Default directory", "dir", "", 32),
			},
			{ // BitTorrent — the four switches are fixed at daemon startup;
				// changeGlobalOption silently ignores them, so they are
				// shown read-only.
				tgRO("DHT", "enable-dht"),
				tgRO("Peer exchange", "enable-peer-exchange"),
				tgRO("Local peer discovery", "bt-enable-lpd"),
				tgRO("Require encryption", "bt-require-crypto"),
				mk("Default seed ratio", "seed-ratio", "", 8),
				mk("Default seed time (min)", "seed-time", "", 8),
			},
			{ // Interface
				tg("Light theme", ""),
			},
		},
	}
	m.fields[0][3].on = srv.Protocol != "http" && srv.Protocol != "https"
	m.fields[4][0].on = a.cfg.Theme == "light"
	return m
}

// loadCmd fetches global options to fill the option-backed fields.
func (m settingsModel) loadCmd() tea.Cmd {
	c := m.a.client
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		opts, err := c.GetGlobalOption(ctx)
		return globalOptionsMsg{opts: opts, err: err}
	}
}

func (m *settingsModel) absorbGlobal(opts map[string]string) {
	m.loaded = opts
	for si := range m.fields {
		for fi := range m.fields[si] {
			f := &m.fields[si][fi]
			if f.optKey == "" {
				continue
			}
			v, ok := opts[f.optKey]
			if !ok {
				continue
			}
			if f.toggle {
				f.on = v == "true"
			} else if !m.dirty {
				f.input.SetValue(v)
			}
		}
	}
}

func (m *settingsModel) blurAll() {
	for si := range m.fields {
		for fi := range m.fields[si] {
			m.fields[si][fi].input.Blur()
		}
	}
}

func (m settingsModel) update(msg tea.KeyMsg) (settingsModel, tea.Cmd) {
	a := m.a
	key := msg.String()
	switch key {
	case "esc":
		a.screen = screenList
		return m, nil
	case "ctrl+s":
		return m.save()
	}
	if m.inSide {
		switch key {
		case "q":
			a.screen = screenList
		case "j", "down":
			if m.section < len(m.sections)-1 {
				m.section++
			}
		case "k", "up":
			if m.section > 0 {
				m.section--
			}
		case "tab", "enter", "l", "right":
			m.inSide = false
			m.focus = 0
			f := &m.fields[m.section][0]
			if !f.toggle {
				return m, f.input.Focus()
			}
		}
		return m, nil
	}
	// Fields focused.
	switch key {
	case "tab":
		m.blurAll()
		m.focus++
		if m.focus >= len(m.fields[m.section]) {
			m.inSide = true
			m.focus = 0
			return m, nil
		}
		f := &m.fields[m.section][m.focus]
		if !f.toggle {
			return m, f.input.Focus()
		}
		return m, nil
	case "shift+tab", "left":
		m.blurAll()
		m.inSide = true
		return m, nil
	case " ":
		f := &m.fields[m.section][m.focus]
		if f.toggle {
			if f.readonly {
				return m, a.flash(f.label+" is set at aria2 startup — not changeable at runtime", true)
			}
			f.on = !f.on
			m.dirty = true
			return m, nil
		}
	}
	f := &m.fields[m.section][m.focus]
	if !f.toggle {
		before := f.input.Value()
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		if f.input.Value() != before {
			m.dirty = true
		}
		return m, cmd
	}
	return m, nil
}

// storeActiveServer writes srv back into the active config slot, recovering
// from an empty server list or a stale active index.
func (a *App) storeActiveServer(srv config.Server) {
	if len(a.cfg.Servers) == 0 {
		a.cfg.Servers = append(a.cfg.Servers, srv)
		a.cfg.Active = 0
		return
	}
	if a.cfg.Active < 0 || a.cfg.Active >= len(a.cfg.Servers) {
		a.cfg.Active = 0
	}
	a.cfg.Servers[a.cfg.Active] = srv
}

// save persists connection changes to config and pushes global options.
func (m settingsModel) save() (settingsModel, tea.Cmd) {
	a := m.a

	// Connection section → config. Skipped for the managed built-in daemon,
	// whose endpoint is decided at spawn time.
	changedConn := false
	if srv := a.cfg.ActiveServer(); !srv.Managed {
		conn := m.fields[0]
		port, err := strconv.Atoi(strings.TrimSpace(conn[1].input.Value()))
		if err != nil || port <= 0 {
			return m, a.flash("bad port", true)
		}
		proto := "http"
		if conn[3].on {
			proto = "ws"
		}
		changedConn = srv.Host != strings.TrimSpace(conn[0].input.Value()) ||
			srv.Port != port || srv.Secret != conn[2].input.Value() || srv.Protocol != proto
		srv.Host = strings.TrimSpace(conn[0].input.Value())
		srv.Port = port
		srv.Secret = conn[2].input.Value()
		srv.Protocol = proto
		a.storeActiveServer(srv)
	}

	// Interface section → config.
	theme := "dark"
	if m.fields[4][0].on {
		theme = "light"
	}
	if err := a.setTheme(theme); err != nil { // also saves config
		return m, a.flash("config save failed: "+err.Error(), true)
	}

	// Option-backed sections → changeGlobalOption (only changed keys).
	opts := map[string]string{}
	for si := 1; si <= 3; si++ {
		for _, f := range m.fields[si] {
			// Every option-backed toggle today is a startup option
			// (readonly); only text inputs reach changeGlobalOption.
			if f.optKey == "" || f.toggle {
				continue
			}
			v := strings.TrimSpace(f.input.Value())
			if v != "" && v != m.loaded[f.optKey] {
				opts[f.optKey] = v
			}
		}
	}
	m.dirty = false
	cmds := []tea.Cmd{}
	if len(opts) > 0 {
		cmds = append(cmds, a.rpcCmd("settings saved", func(ctx context.Context, c api) error {
			return c.ChangeGlobalOption(ctx, opts)
		}))
	} else {
		cmds = append(cmds, a.flash("settings saved", false))
	}
	if changedConn {
		cmds = append(cmds, a.reconnect())
	}
	return m, tea.Batch(cmds...)
}

// mouse handles clicks: sidebar sections and fields.
func (m settingsModel) mouse(id string) (settingsModel, tea.Cmd) {
	kind, arg := splitID(id)
	switch kind {
	case "side":
		if i := argInt(arg); i >= 0 && i < len(m.sections) {
			m.section = i
			m.inSide = true
			m.blurAll()
		}
	case "field":
		i := argInt(arg)
		if i < 0 || i >= len(m.fields[m.section]) {
			return m, nil
		}
		m.inSide = false
		m.focus = i
		m.blurAll()
		f := &m.fields[m.section][i]
		if f.toggle {
			return m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
		}
		return m, f.input.Focus()
	}
	return m, nil
}

func (m settingsModel) view() string {
	a := m.a
	st := a.styles
	var b strings.Builder
	marker := ""
	if m.dirty {
		marker = "   " + st.Yellow.Render("● unsaved changes")
	}
	b.WriteString(" " + st.Dim.Render("← esc") + st.Faint.Render(" │ ") + st.Title.Render("Settings") + marker + "\n")

	a.hits.line("back", 0, a.width)
	var side []string
	for i, name := range m.sections {
		a.hits.add(fmt.Sprintf("side:%d", i), 1, 2+i, 18, 2+i)
		if i == m.section {
			line := st.Brand.Render("▸ ") + st.Title.Render(name)
			if m.inSide {
				line = st.RowSel.Render(line)
			}
			side = append(side, line)
		} else {
			side = append(side, st.Dim.Render("  "+name))
		}
	}
	sidebar := st.Panel.Render(strings.Join(side, "\n"))

	var rows []string
	fieldY := 3 // panel border + section caption
	rows = append(rows, st.Dim.Render(strings.ToUpper(m.sections[m.section])))
	if m.section == 0 && a.cfg.ActiveServer().Managed {
		rows = append(rows,
			st.Text.Render("Built-in daemon — aria2t spawns and manages aria2c itself."),
			st.Dim.Render("Endpoint and secret are chosen at launch; nothing to configure."),
			st.Dim.Render("Use the server switcher (s → +) to add an external server."))
		form := st.Panel.Width(a.width - lipgloss.Width(sidebar) - 4).Render(strings.Join(rows, "\n"))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", form) + "\n")
		key := func(k, label string) string { return st.Key.Render(k) + " " + st.Dim.Render(label) }
		b.WriteString(" " + strings.Join([]string{
			key("↑↓", "section"), key("^s", "save"), key("esc", "back"),
		}, "  "))
		b.WriteString(a.statusLine())
		return b.String()
	}
	fx0, fx1 := 20, a.width-4 // form panel content x-range (after sidebar)
	for i, f := range m.fields[m.section] {
		focused := !m.inSide && i == m.focus
		h := 1 // toggle rows are one line
		if !f.toggle {
			h = 4 // label + bordered input box
		}
		a.hits.add(fmt.Sprintf("field:%d", i), fx0, fieldY, fx1, fieldY+h-1)
		fieldY += h
		if f.toggle {
			box := st.Dim.Render("[ ]")
			if f.on {
				box = st.Green.Render("[x]")
			}
			label := " " + f.label
			switch {
			case f.readonly && focused:
				label = st.Title.Render(" "+f.label) + st.Dim.Render(" · startup option, read-only")
			case f.readonly:
				label = st.Dim.Render(" " + f.label + " · startup")
			case focused:
				label = st.Title.Render(" " + f.label + " ◂ space toggles")
			}
			rows = append(rows, box+label)
		} else {
			boxStyle := st.Input
			if focused {
				boxStyle = st.InputHot
			}
			rows = append(rows, st.Dim.Render(f.label)+"\n"+boxStyle.Render(f.input.View()))
		}
	}
	form := st.Panel.Width(a.width - lipgloss.Width(sidebar) - 4).Render(strings.Join(rows, "\n"))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", form) + "\n")

	key := func(k, label string) string { return st.Key.Render(k) + " " + st.Dim.Render(label) }
	b.WriteString(" " + strings.Join([]string{
		key("↑↓", "section"), key("tab", "next field"), key("space", "toggle"),
		key("^s", "save"), key("esc", "back"),
	}, "  "))
	b.WriteString(a.statusLine())
	return b.String()
}
