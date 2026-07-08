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

// mouse handles clicks inside the server switcher: a row selects; the hint
// bar's connect/add/edit/remove/close all act via mouse.
func (m serversModel) mouse(id string) (serversModel, tea.Cmd) {
	kind, arg := splitID(id)
	if m.editing {
		switch kind {
		case "field":
			if i := argInt(arg); i >= 0 && i < len(m.form) {
				m.form[m.formFoc].Blur()
				m.formFoc = i
				return m, m.form[i].Focus()
			}
		case "proto":
			m.formWS = arg == "ws"
		case "btn":
			return m.updateForm(dispatchBtn(arg))
		}
		return m, nil
	}
	switch kind {
	case "srv":
		if i := argInt(arg); i >= 0 && i < len(m.a.cfg.Servers) {
			m.cursor = i
		}
	case "key":
		return m.update(keyFromToken(arg))
	case "btn":
		return m.update(dispatchBtn(arg))
	}
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
		protoLabel := st.Dim.Render("Protocol  ")
		wsChip := m.a.tab("ws", m.formWS)
		httpChip := m.a.tab("http", !m.formWS)
		buttons := []button{{"esc", "Cancel", "esc", btnRed}, {"enter", "Save", "↵", btnGreen}}
		body := lipgloss.JoinVertical(lipgloss.Left,
			st.Title.Render(title),
			"",
			st.Dim.Render("Name")+"\n"+m.form[0].View(),
			st.Dim.Render("Host")+"\n"+m.form[1].View(),
			st.Dim.Render("Port")+"\n"+m.form[2].View(),
			st.Dim.Render("RPC secret")+"\n"+m.form[3].View(),
			"",
			protoLabel+wsChip+" "+httpChip+st.Dim.Render("  (^w toggles)"),
			"",
			m.a.buttonRow(buttons),
		)
		modal := m.a.modalCard(false).Render(body)
		offX, offY := m.a.overlayOffset(modal)
		// Each field is a label row + input row (label at body row 2+2i, input at
		// 3+2i; content starts at offY+2). Register both rows as one focus target.
		for i := range m.form {
			y0 := offY + 4 + 2*i
			m.a.hits.add(fmt.Sprintf("field:%d", i), offX+3, y0, offX+lipgloss.Width(modal)-4, y0+1)
		}
		protoY := offY + 2 + 11
		px := offX + 3 + lipgloss.Width(protoLabel)
		wsW := lipgloss.Width(wsChip)
		m.a.hits.add("proto:ws", px, protoY, px+wsW-1, protoY)
		hx := px + wsW + 1
		m.a.hits.add("proto:http", hx, protoY, hx+lipgloss.Width(httpChip)-1, protoY)
		m.a.registerButtons(offX, offY, modal, buttons)
		return modal
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
			state = st.Dim.Render("▪ starts on demand")
		} else if m.dead[i] {
			state = st.Red.Render("▪ unreachable")
		} else if d, ok := m.latency[i]; ok {
			state = st.Green.Render(fmt.Sprintf("▪ %dms", d.Milliseconds()))
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
	navHints := []keyHint{{"+", "+", "add"}, {"e", "e", "edit"}, {"-", "-", "remove"}}
	navParts := make([]string, len(navHints))
	for i, h := range navHints {
		navParts[i] = st.Key.Render(h.key) + " " + st.Dim.Render(h.label)
	}
	buttons := []button{{"esc", "Close", "esc", btnRed}, {"enter", "Connect", "↵", btnGreen}}
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Switch server")+"   "+st.Dim.Render("s to cycle"),
		"",
		strings.Join(rows, "\n"),
		"",
		strings.Join(navParts, "  "),
		m.a.buttonRow(buttons),
	)
	modal := m.a.modalCard(false).Render(body)
	offX, offY := m.a.overlayOffset(modal)
	for i := range m.a.cfg.Servers {
		y := offY + 4 + i // frame(2) + title + blank
		m.a.hits.add(fmt.Sprintf("srv:%d", i), offX+1, y, offX+lipgloss.Width(modal)-2, y)
	}
	// Nav-hint line sits one line above the buttons.
	navY := offY + lipgloss.Height(modal) - 4
	hx := offX + 3
	for i, h := range navHints {
		w := lipgloss.Width(navParts[i])
		m.a.hits.add("key:"+h.token, hx, navY, hx+w-1, navY)
		hx += w + 2
	}
	m.a.registerButtons(offX, offY, modal, buttons)
	return modal
}
