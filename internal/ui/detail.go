package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/rpc"
)

// detailModel shows one download: summary, pieces, peers, files, announce.
type detailModel struct {
	a     *App
	gid   string
	s     rpc.Status
	peers []rpc.Peer
	err   error

	filesFocused bool
	fileCursor   int
}

func newDetailModel(a *App) detailModel { return detailModel{a: a} }

// refreshCmd fetches status and peers for the shown gid.
func (m detailModel) refreshCmd() tea.Cmd {
	c := m.a.client
	gid := m.gid
	if c == nil || gid == "" {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s, err := c.TellStatus(ctx, gid)
		if err != nil {
			return detailDataMsg{err: err}
		}
		var peers []rpc.Peer
		if s.IsTorrent() {
			peers, _ = c.GetPeers(ctx, gid) // peers are decoration; ignore failures
		}
		return detailDataMsg{status: s, peers: peers}
	}
}

func (m *detailModel) absorb(msg detailDataMsg) {
	if msg.err != nil {
		m.err = msg.err
		return
	}
	m.err = nil
	m.s = msg.status
	m.peers = msg.peers
	if n := len(m.s.Files); m.fileCursor >= n && n > 0 {
		m.fileCursor = n - 1
	}
}

func (m detailModel) update(msg tea.KeyMsg) (detailModel, tea.Cmd) {
	a := m.a
	switch msg.String() {
	case "esc", "q":
		if m.filesFocused {
			m.filesFocused = false
			return m, nil
		}
		a.screen = screenList
		return m, nil
	case "f":
		m.filesFocused = !m.filesFocused
		return m, nil
	case "j", "down":
		if m.filesFocused && m.fileCursor < len(m.s.Files)-1 {
			m.fileCursor++
		}
		return m, nil
	case "k", "up":
		if m.filesFocused && m.fileCursor > 0 {
			m.fileCursor--
		}
		return m, nil
	case " ":
		if m.filesFocused && len(m.s.Files) > 0 {
			cmd := m.toggleFileCmd()
			// Flip locally too, so a second toggle before the next refresh
			// builds its select-file list from what the user sees.
			f := &m.s.Files[m.fileCursor]
			if f.IsSelected() {
				f.Selected = "false"
			} else {
				f.Selected = "true"
			}
			return m, cmd
		}
		return m, nil
	case "p":
		gid, paused := m.gid, m.s.Status == "paused"
		return m, a.rpcCmd("", func(ctx context.Context, c api) error {
			if paused {
				return c.Unpause(ctx, gid)
			}
			return c.Pause(ctx, gid)
		})
	case "d":
		gid, name := m.gid, m.s.Name()
		// Stopped downloads live in the result list; aria2.remove only
		// works on active/waiting ones.
		stopped := m.s.Status == "complete" || m.s.Status == "error" || m.s.Status == "removed"
		return m, a.confirmRemove(name, func() tea.Cmd {
			a.screen = screenList
			return a.rpcCmd("removed "+name, func(ctx context.Context, c api) error {
				if stopped {
					return c.RemoveDownloadResult(ctx, gid)
				}
				return c.Remove(ctx, gid)
			})
		})
	case "t":
		if m.s.IsTorrent() {
			a.seeding = newSeedingModel(a)
			a.seeding.gid = m.gid
			a.seeding.name = m.s.Name()
			a.screen = screenSeeding
			return m, a.seeding.loadCmd()
		}
		return m, a.flash("not a torrent", true)
	case "o":
		return m, a.openDir(m.s.Dir)
	}
	return m, nil
}

// toggleFileCmd flips selection of the highlighted file via select-file.
func (m detailModel) toggleFileCmd() tea.Cmd {
	sel := make([]string, 0, len(m.s.Files))
	for i, f := range m.s.Files {
		selected := f.IsSelected()
		if i == m.fileCursor {
			selected = !selected
		}
		if selected {
			sel = append(sel, f.Index)
		}
	}
	gid, value := m.gid, strings.Join(sel, ",")
	return m.a.rpcCmd("file selection updated", func(ctx context.Context, c api) error {
		return c.ChangeOption(ctx, gid, map[string]string{"select-file": value})
	})
}

// mouse handles clicks on the detail screen: file rows toggle selection.
func (m detailModel) mouse(id string) (detailModel, tea.Cmd) {
	kind, arg := splitID(id)
	if kind != "file" {
		return m, nil
	}
	i := argInt(arg)
	if i < 0 || i >= len(m.s.Files) {
		return m, nil
	}
	m.filesFocused = true
	m.fileCursor = i
	return m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
}

func (m detailModel) view() string {
	a := m.a
	st := a.styles
	s := m.s
	var b strings.Builder

	badge := st.Badge.Render(s.Status)
	b.WriteString(" " + st.Dim.Render("← esc") + st.Faint.Render(" │ ") + st.Title.Render(s.Name()) + "  " + badge + "\n")
	if m.err != nil {
		b.WriteString(st.Red.Render(" " + m.err.Error()))
		return b.String()
	}

	// Summary panel.
	f, e := Bar(s.Progress(), 20)
	sum := []string{
		st.Brand.Render(f) + st.Faint.Render(e) + " " +
			st.Title.Render(fmt.Sprintf("%d%%", int(s.Progress()*100))) + " " +
			st.Dim.Render(FmtBytes(s.Completed())+" / "+FmtBytes(s.Total())) + "   " +
			st.Cyan.Render("▼ "+FmtSpeed(s.DownSpeed())) + " " +
			st.Magenta.Render("▲ "+FmtSpeed(s.UpSpeed())) + " " +
			st.Dim.Render("eta "+FmtETA(s.Total()-s.Completed(), s.DownSpeed())),
		st.Dim.Render("gid ") + st.Text.Render(s.GID) +
			st.Dim.Render("  seeds ") + st.Green.Render(fmt.Sprintf("%d", s.Seeds())) +
			st.Dim.Render("  conns ") + st.Text.Render(fmt.Sprintf("%d", s.Conns())) +
			st.Dim.Render("  ratio ") + st.Text.Render(fmt.Sprintf("%.2f", s.Ratio())) +
			st.Dim.Render("  dir ") + st.Text.Render(s.Dir),
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(sum, "\n")) + "\n")

	// Pieces panel.
	width := a.width - 8
	if width < 20 {
		width = 20
	}
	pieces := []string{
		st.Dim.Render(fmt.Sprintf("PIECES — %d · %s each", s.Pieces(), FmtBytes(s.PieceLen()))),
		st.Brand.Render(Pieces(s.Bitfield, s.Pieces(), width)),
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(pieces, "\n")) + "\n")

	// Peers and files, side by side.
	var peerLines []string
	peerLines = append(peerLines, st.Dim.Render(pad("PEER", 24)+pad("CLIENT", 10)+lpad("▼ DOWN", 12)+lpad("▲ UP", 12)))
	shown := m.peers
	if len(shown) > 6 {
		shown = shown[:6]
	}
	for _, p := range shown {
		peerLines = append(peerLines,
			st.Text.Render(pad(p.IP+":"+p.Port, 24))+
				st.Dim.Render(pad(clientName(p.PeerID), 10))+
				st.Cyan.Render(lpad(FmtSpeed(p.DownSpeed()), 12))+
				st.Dim.Render(lpad(FmtSpeed(p.UpSpeed()), 12)))
	}
	if extra := len(m.peers) - len(shown); extra > 0 {
		peerLines = append(peerLines, st.Dim.Render(fmt.Sprintf("… %d more", extra)))
	}
	if len(m.peers) == 0 {
		peerLines = append(peerLines, st.Dim.Render("no peers"))
	}

	selCount := 0
	for _, f := range s.Files {
		if f.IsSelected() {
			selCount++
		}
	}
	focusMark := ""
	if m.filesFocused {
		focusMark = st.Brand.Render(" ● j/k + space")
	}
	fileLines := []string{st.Dim.Render(fmt.Sprintf("FILES — %d selected of %d", selCount, len(s.Files))) + focusMark}
	for i, f := range s.Files {
		box := st.Dim.Render("[ ]")
		if f.IsSelected() {
			box = st.Green.Render("[x]")
		}
		name := pad(trimPathPrefix(f.Path, s.Dir), 30)
		line := box + " " + st.Text.Render(name) + " " + st.Dim.Render(FmtBytes(f.Len()))
		if m.filesFocused && i == m.fileCursor {
			line = st.RowSel.Render(line)
		}
		fileLines = append(fileLines, line)
	}
	if s.BitTorrent != nil && len(s.BitTorrent.AnnounceList) > 0 {
		fileLines = append(fileLines, st.Dim.Render("ANNOUNCE"))
		n := 0
		for _, tier := range s.BitTorrent.AnnounceList {
			for _, u := range tier {
				if n >= 3 {
					break
				}
				fileLines = append(fileLines, st.Text.Render(pad(u, 40)))
				n++
			}
		}
	}

	peersPanel := st.Panel.Render(strings.Join(peerLines, "\n"))
	filesPanel := st.Panel.Render(strings.Join(fileLines, "\n"))
	// Clickable regions: full header line goes back, file rows toggle.
	a.hits.line("back", 0, a.width)
	panelsY := lipgloss.Height(b.String()) // lines rendered above these panels
	fx := lipgloss.Width(peersPanel) + 2   // files panel content x
	for i := range s.Files {
		y := panelsY + 2 + i // top border + FILES header line
		a.hits.add(fmt.Sprintf("file:%d", i), fx, y, fx+lipgloss.Width(filesPanel)-3, y)
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, peersPanel, filesPanel) + "\n")

	key := func(k, label string) string { return st.Key.Render(k) + " " + st.Dim.Render(label) }
	b.WriteString(" " + strings.Join([]string{
		key("p", "pause/resume"), key("d", "remove"), key("f", "select files"),
		key("t", "trackers"), key("o", "open dir"), key("esc", "back"),
	}, "  "))
	b.WriteString(a.statusLine())
	return b.String()
}

// clientName maps a peer id prefix to a short client label.
func clientName(peerID string) string {
	// Azureus-style ids begin with -XX1234-; peerID arrives percent-encoded.
	s := strings.TrimPrefix(peerID, "%2D")
	if len(s) < 2 {
		return "?"
	}
	switch s[:2] {
	case "qB":
		return "qB"
	case "TR":
		return "TR"
	case "lt", "LT":
		return "lt"
	case "AZ":
		return "AZ"
	case "DE":
		return "DE"
	case "UT":
		return "µT"
	case "A2", "ar":
		return "aria2"
	default:
		return s[:2]
	}
}

// trimPathPrefix shows the file path relative to the download dir.
func trimPathPrefix(path, dir string) string {
	if dir != "" && strings.HasPrefix(path, dir) {
		return strings.TrimPrefix(strings.TrimPrefix(path, dir), "/")
	}
	return path
}
