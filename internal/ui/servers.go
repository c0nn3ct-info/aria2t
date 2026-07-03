package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/config"
)

// serversModel is the multi-server switcher overlay.
type serversModel struct {
	a      *App
	cursor int

	latency map[int]time.Duration
	dead    map[int]bool

	editing bool
	editIdx int                // -1 = adding new
	form    [4]textinput.Model // name, host, port, secret
	formWS  bool
	formFoc int
}

func newServersModel(a *App) serversModel {
	m := serversModel{a: a, cursor: a.cfg.Active, latency: map[int]time.Duration{}, dead: map[int]bool{}}
	for i := range m.form {
		m.form[i] = textinput.New()
		m.form[i].Width = 24
	}
	m.form[0].Placeholder = "seedbox"
	m.form[1].Placeholder = "host"
	m.form[2].Placeholder = "6800"
	m.form[3].Placeholder = "secret"
	m.form[3].EchoMode = textinput.EchoPassword
	return m
}

// probeCmd measures reachability of every configured server. A managed
// server is probed at its daemon's live endpoint, or skipped when the
// daemon isn't running yet (it starts on demand — that's not an error).
func (m serversModel) probeCmd() tea.Cmd {
	dial := m.a.dial
	d := m.a.daemon
	cmds := make([]tea.Cmd, 0, len(m.a.cfg.Servers))
	for i, srv := range m.a.cfg.Servers {
		i, srv := i, srv
		if srv.Managed {
			if d == nil {
				continue
			}
			srv = config.Server{Host: "localhost", Port: d.Port, Secret: d.Secret, Protocol: "ws"}
		}
		cmds = append(cmds, func() tea.Msg {
			start := time.Now()
			c, _, err := dial(srv)
			if err != nil {
				return latencyMsg{index: i, err: err}
			}
			c.Close()
			return latencyMsg{index: i, d: time.Since(start)}
		})
	}
	return tea.Batch(cmds...)
}

func (m *serversModel) absorbLatency(msg latencyMsg) {
	if msg.err != nil {
		m.dead[msg.index] = true
		return
	}
	delete(m.dead, msg.index)
	m.latency[msg.index] = msg.d
}

func (m serversModel) update(msg tea.KeyMsg) (serversModel, tea.Cmd) {
	a := m.a
	key := msg.String()
	if m.editing {
		return m.updateForm(msg)
	}
	switch key {
	case "esc":
		a.overlay = overlayNone
	case "j", "down", "s":
		m.cursor = (m.cursor + 1) % max(1, len(a.cfg.Servers))
	case "k", "up":
		m.cursor = (m.cursor - 1 + max(1, len(a.cfg.Servers))) % max(1, len(a.cfg.Servers))
	case "enter":
		if m.cursor >= 0 && m.cursor < len(a.cfg.Servers) {
			a.cfg.Active = m.cursor
			a.saveConfig()
			a.overlay = overlayNone
			return m, tea.Batch(a.reconnect(), a.flash("switching to "+a.cfg.Servers[m.cursor].Name, false))
		}
	case "+", "a":
		m.editing = true
		m.editIdx = -1
		m.formWS = true
		for i := range m.form {
			m.form[i].SetValue("")
		}
		m.form[2].SetValue("6800")
		m.formFoc = 0
		return m, m.form[0].Focus()
	case "e":
		if m.cursor >= 0 && m.cursor < len(a.cfg.Servers) {
			srv := a.cfg.Servers[m.cursor]
			m.editing = true
			m.editIdx = m.cursor
			m.form[0].SetValue(srv.Name)
			m.form[1].SetValue(srv.Host)
			m.form[2].SetValue(fmt.Sprintf("%d", srv.Port))
			m.form[3].SetValue(srv.Secret)
			m.formWS = srv.Protocol != "http" && srv.Protocol != "https"
			m.formFoc = 0
			return m, m.form[0].Focus()
		}
	case "-", "d":
		if len(a.cfg.Servers) > 1 && m.cursor < len(a.cfg.Servers) {
			a.cfg.Servers = append(a.cfg.Servers[:m.cursor], a.cfg.Servers[m.cursor+1:]...)
			switch {
			case a.cfg.Active == m.cursor:
				a.cfg.Active = 0
			case a.cfg.Active > m.cursor:
				a.cfg.Active-- // keep pointing at the same server
			}
			if m.cursor >= len(a.cfg.Servers) {
				m.cursor = len(a.cfg.Servers) - 1
			}
			// Probe results are keyed by index; indexes just shifted.
			m.latency = map[int]time.Duration{}
			m.dead = map[int]bool{}
			a.saveConfig()
			return m, m.probeCmd()
		}
	}
	return m, nil
}

func (m serversModel) updateForm(msg tea.KeyMsg) (serversModel, tea.Cmd) {
	a := m.a
	switch msg.String() {
	case "esc":
		m.editing = false
		m.form[m.formFoc].Blur()
		return m, nil
	case "tab":
		m.form[m.formFoc].Blur()
		m.formFoc = (m.formFoc + 1) % len(m.form)
		return m, m.form[m.formFoc].Focus()
	case "ctrl+w":
		m.formWS = !m.formWS
		return m, nil
	case "enter":
		port, err := strconv.Atoi(strings.TrimSpace(m.form[2].Value()))
		if err != nil || port <= 0 {
			return m, a.flash("bad port", true)
		}
		proto := "ws"
		if !m.formWS {
			proto = "http"
		}
		srv := config.Server{
			Name:     strings.TrimSpace(m.form[0].Value()),
			Host:     strings.TrimSpace(m.form[1].Value()),
			Port:     port,
			Secret:   m.form[3].Value(),
			Protocol: proto,
		}
		if srv.Name == "" || srv.Host == "" {
			return m, a.flash("name and host required", true)
		}
		if m.editIdx == -1 {
			a.cfg.Servers = append(a.cfg.Servers, srv)
			m.cursor = len(a.cfg.Servers) - 1
		} else {
			a.cfg.Servers[m.editIdx] = srv
		}
		a.saveConfig()
		m.editing = false
		m.form[m.formFoc].Blur()
		return m, m.probeCmd()
	}
	var cmd tea.Cmd
	m.form[m.formFoc], cmd = m.form[m.formFoc].Update(msg)
	return m, cmd
}

// mouse handles clicks inside the server switcher.
func (m serversModel) mouse(id string, double bool) (serversModel, tea.Cmd) {
	kind, arg := splitID(id)
	if kind != "srv" || m.editing {
		return m, nil
	}
	i := argInt(arg)
	if i < 0 || i >= len(m.a.cfg.Servers) {
		return m, nil
	}
	if m.cursor == i && double {
		return m.update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	m.cursor = i
	return m, nil
}

func (m serversModel) view() string {
	a := m.a
	st := a.styles
	if m.editing {
		title := "Add server"
		if m.editIdx >= 0 {
			title = "Edit server"
		}
		proto := st.Badge.Render("ws") + " " + st.Dim.Render("[ http ]")
		if !m.formWS {
			proto = st.Dim.Render("[ ws ]") + " " + st.Badge.Render("http")
		}
		body := lipgloss.JoinVertical(lipgloss.Left,
			st.Title.Render(title),
			"",
			st.Dim.Render("Name")+"\n"+m.form[0].View(),
			st.Dim.Render("Host")+"\n"+m.form[1].View(),
			st.Dim.Render("Port")+"\n"+m.form[2].View(),
			st.Dim.Render("RPC secret")+"\n"+m.form[3].View(),
			"",
			st.Dim.Render("Protocol  ")+proto+st.Dim.Render("  (^w toggles)"),
			"",
			st.Dim.Render("tab next · ")+st.Green.Render("↵ save")+st.Dim.Render(" · esc cancel"),
		)
		return st.Modal.Render(body)
	}

	var rows []string
	for i, srv := range a.cfg.Servers {
		marker := "  "
		nameStyle := st.Text
		if i == m.cursor {
			marker = st.Brand.Render("▸ ")
			nameStyle = st.Title
		}
		state := st.Dim.Render("probing…")
		if srv.Managed && a.daemon == nil {
			state = st.Dim.Render("● starts on demand")
		} else if m.dead[i] {
			state = st.Red.Render("● unreachable")
		} else if d, ok := m.latency[i]; ok {
			state = st.Green.Render(fmt.Sprintf("● %dms", d.Milliseconds()))
		}
		active := ""
		if i == a.cfg.Active {
			active = st.Green.Render(" · connected")
		}
		desc := fmt.Sprintf("%s:%d · %s", srv.Host, srv.Port, srv.Protocol)
		if srv.Managed {
			desc = "built-in · managed"
			if a.daemon != nil {
				desc = fmt.Sprintf("built-in · localhost:%d", a.daemon.Port)
			}
		}
		line := marker + nameStyle.Render(srv.Name) + " " +
			st.Dim.Render(desc) + active +
			"  " + state
		if i == m.cursor {
			line = st.RowSel.Render(line)
		}
		rows = append(rows, line)
	}
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Switch server")+"   "+st.Dim.Render("s to cycle"),
		"",
		strings.Join(rows, "\n"),
		"",
		st.Key.Render("↵")+" "+st.Dim.Render("connect")+"  "+
			st.Key.Render("+")+" "+st.Dim.Render("add")+"  "+
			st.Key.Render("e")+" "+st.Dim.Render("edit")+"  "+
			st.Key.Render("-")+" "+st.Dim.Render("remove")+"  "+
			st.Key.Render("esc")+" "+st.Dim.Render("close"),
	)
	modal := st.Modal.Render(body)
	offX, offY := m.a.overlayOffset(modal)
	for i := range m.a.cfg.Servers {
		y := offY + 4 + i // frame(2) + title + blank
		m.a.hits.add(fmt.Sprintf("srv:%d", i), offX+1, y, offX+lipgloss.Width(modal)-2, y)
	}
	return modal
}
