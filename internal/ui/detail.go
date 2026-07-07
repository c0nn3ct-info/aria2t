package ui

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/rpc"
)

// detailModel shows one download: summary, pieces, peers or servers, files.
type detailModel struct {
	a       *App
	gid     string
	s       rpc.Status
	peers   []rpc.Peer
	servers []rpc.ServerStat
	err     error
}

func newDetailModel(a *App) detailModel { return detailModel{a: a} }

// refreshCmd fetches status plus peers (torrents) or servers (HTTP/FTP).
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
		var servers []rpc.ServerStat
		if s.IsTorrent() {
			peers, _ = c.GetPeers(ctx, gid) // decoration; ignore failures
		} else {
			servers, _ = c.GetServers(ctx, gid)
		}
		return detailDataMsg{status: s, peers: peers, servers: servers}
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
	m.servers = msg.servers
}

func (m detailModel) update(msg tea.KeyMsg) (detailModel, tea.Cmd) {
	a := m.a
	switch msg.String() {
	case "esc", "q":
		a.screen = screenList
		return m, nil
	case "f":
		if len(m.s.Files) == 0 {
			return m, a.flash("no files to select", true)
		}
		a.files = newFilesModel(a)
		a.files.gid = m.gid
		a.files.name = m.s.Name()
		a.overlay = overlayFiles
		return m, a.files.loadCmd()
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
		stopped := isStopped(m.s.Status)
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

func (m detailModel) view() string {
	a := m.a
	st := a.styles
	s := m.s
	var b strings.Builder

	badge := st.Badge.Render(s.Status)
	seed := ""
	if s.Seeder == "true" {
		seed = " " + st.Green.Render("seeding")
	}
	b.WriteString(" " + st.Dim.Render("← esc") + st.Faint.Render(" │ ") + st.Title.Render(s.Name()) + "  " + badge + seed + "\n")
	if m.err != nil {
		b.WriteString(st.Red.Render(" " + m.err.Error()))
		return b.String()
	}
	if s.Status == "error" {
		b.WriteString(" " + st.Red.Render("✗ "+friendlyError(s.ErrorCode, s.ErrorMessage)) + "\n")
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
		m.summaryStats(s),
	}
	if s.BitTorrent != nil {
		meta := st.Dim.Render("torrent ") + st.Text.Render(shortHash(s.InfoHash))
		if s.BitTorrent.Mode != "" {
			meta += st.Dim.Render("  mode ") + st.Text.Render(s.BitTorrent.Mode)
		}
		if c := strings.TrimSpace(s.BitTorrent.Comment); c != "" {
			meta += st.Dim.Render("  “") + st.Text.Render(trunc(c, 40)) + st.Dim.Render("”")
		}
		sum = append(sum, meta)
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(sum, "\n")) + "\n")

	// Pieces panel — only when aria2 reports pieces (torrents, split HTTP).
	if s.Pieces() > 0 {
		width := a.width - 8
		if width < 20 {
			width = 20
		}
		pieces := []string{
			st.Dim.Render(fmt.Sprintf("PIECES — %d · %s each", s.Pieces(), FmtBytes(s.PieceLen()))),
			st.Brand.Render(Pieces(s.Bitfield, s.Pieces(), width)),
		}
		b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(pieces, "\n")) + "\n")
	}

	// Remaining height for the two side-by-side panels (each: 2 border + 1
	// header + data; plus a keybar and a status line below).
	rowCap := a.height - strings.Count(b.String(), "\n") - 5
	if rowCap < 2 {
		rowCap = 2
	}

	leftLines := m.peersOrServers(s, rowCap)
	fileLines := m.filesPanel(s, rowCap)

	leftPanel := st.Panel.Render(strings.Join(leftLines, "\n"))
	filesPanel := st.Panel.Render(strings.Join(fileLines, "\n"))
	a.hits.line("back", 0, a.width)
	// Clicking the FILES panel opens the tree picker (same as f) — full mouse
	// parity, so the picker is reachable without the keyboard.
	yP := strings.Count(b.String(), "\n")
	a.hits.add("key:f", lipgloss.Width(leftPanel), yP, a.width-1, yP+lipgloss.Height(filesPanel)-1)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, filesPanel) + "\n")

	b.WriteString(a.hintbar(strings.Count(b.String(), "\n"), []keyHint{
		{"p", "p", "pause/resume"}, {"d", "d", "remove"}, {"f", "f", "select files"},
		{"t", "t", "trackers"}, {"o", "o", "open dir"}, {"esc", "esc", "back"},
	}))
	b.WriteString(a.statusLine())
	return b.String()
}

// mouse routes key-bar clicks on the detail screen (the back region is handled
// globally); every action is reachable without the keyboard.
func (m detailModel) mouse(id string) (detailModel, tea.Cmd) {
	if kind, arg := splitID(id); kind == "key" {
		return m.update(keyFromToken(arg))
	}
	return m, nil
}

// summaryStats builds the second summary line: id, upload, ratio, conns,
// seeds (torrents), and the download directory.
func (m detailModel) summaryStats(s rpc.Status) string {
	st := m.a.styles
	line := st.Dim.Render("gid ") + st.Text.Render(s.GID) +
		st.Dim.Render("  ↑ ") + st.Text.Render(FmtBytes(s.Uploaded())) +
		st.Dim.Render("  ratio ") + st.Text.Render(fmt.Sprintf("%.2f", s.Ratio())) +
		st.Dim.Render("  conns ") + st.Text.Render(fmt.Sprintf("%d", s.Conns()))
	if s.IsTorrent() {
		line += st.Dim.Render("  seeds ") + st.Green.Render(fmt.Sprintf("%d", s.Seeds()))
	}
	return line + st.Dim.Render("  dir ") + st.Text.Render(s.Dir)
}

// peersOrServers renders the left panel: bittorrent peers for torrents, the
// connected HTTP/FTP mirrors otherwise.
func (m detailModel) peersOrServers(s rpc.Status, rowCap int) []string {
	st := m.a.styles
	if !s.IsTorrent() {
		header := st.Dim.Render(pad("SERVER", 44) + lpad("DOWN", 12))
		var data []string
		for _, fs := range m.servers {
			for _, sv := range fs.Servers {
				uri := sv.CurrentURI
				if uri == "" {
					uri = sv.URI
				}
				data = append(data,
					st.Text.Render(pad(serverHost(uri), 44))+
						st.Cyan.Render(lpad(FmtSpeed(sv.DownSpeed()), 12)))
			}
		}
		return append([]string{header}, capRows(data, rowCap, st.Dim, "no active servers")...)
	}
	seeds := 0
	for _, p := range m.peers {
		if p.Seeder == "true" {
			seeds++
		}
	}
	header := st.Dim.Render(fmt.Sprintf("PEERS — %d · %d seeding", len(m.peers), seeds))
	cols := st.Dim.Render(pad("PEER", 22) + pad("CLIENT", 8) + lpad("DONE", 6) + lpad("DOWN", 11) + lpad("UP", 11) + "  FLG")
	var data []string
	for _, p := range m.peers {
		prog := int(bitfieldProgress(p.Bitfield, s.Pieces()) * 100)
		data = append(data,
			st.Text.Render(pad(p.IP+":"+p.Port, 22))+
				st.Dim.Render(pad(clientName(p.PeerID), 8))+
				st.Text.Render(lpad(fmt.Sprintf("%d%%", prog), 6))+
				st.Cyan.Render(lpad(FmtSpeed(p.DownSpeed()), 11))+
				st.Dim.Render(lpad(FmtSpeed(p.UpSpeed()), 11))+
				"  "+st.Yellow.Render(peerFlags(p)))
	}
	return append([]string{header, cols}, capRows(data, rowCap-1, st.Dim, "no peers")...)
}

// filesPanel renders the read-only file list with per-file progress; editing
// selection happens in the tree picker (f).
func (m detailModel) filesPanel(s rpc.Status, rowCap int) []string {
	st := m.a.styles
	sel := 0
	for _, f := range s.Files {
		if f.IsSelected() {
			sel++
		}
	}
	header := st.Dim.Render(fmt.Sprintf("FILES — %d of %d selected  (f to change)", sel, len(s.Files)))
	var data []string
	for _, f := range s.Files {
		box := st.Dim.Render("[ ]")
		if f.IsSelected() {
			box = st.Green.Render("[x]")
		}
		pct := "   -"
		if f.Len() > 0 {
			pct = lpad(fmt.Sprintf("%d%%", int(float64(f.Completed())/float64(f.Len())*100)), 4)
		}
		data = append(data, box+" "+st.Text.Render(pad(trimPathPrefix(f.Path, s.Dir), 28))+
			" "+st.Dim.Render(pct)+" "+st.Dim.Render(lpad(FmtBytes(f.Len()), 10)))
	}
	return append([]string{header}, capRows(data, rowCap, st.Dim, "no files")...)
}

// capRows returns up to cap rendered rows, collapsing the overflow into a
// trailing "… N more" line; an empty list shows the empty message.
func capRows(rows []string, cap int, dim lipgloss.Style, empty string) []string {
	if len(rows) == 0 {
		return []string{dim.Render(empty)}
	}
	if cap < 1 {
		cap = 1
	}
	if len(rows) <= cap {
		return rows
	}
	out := append([]string{}, rows[:cap-1]...)
	return append(out, dim.Render(fmt.Sprintf("… %d more", len(rows)-(cap-1))))
}

// peerFlags renders a compact three-flag state: S seed, d can-download
// (peer not choking us), u can-upload (we are not choking them).
func peerFlags(p rpc.Peer) string {
	flag := func(on bool, c string) string {
		if on {
			return c
		}
		return "-"
	}
	return flag(p.Seeder == "true", "S") +
		flag(p.PeerChoking == "false", "d") +
		flag(p.AmChoking == "false", "u")
}

// serverHost shortens a mirror URI to scheme://host for display.
func serverHost(uri string) string {
	if u, err := url.Parse(uri); err == nil && u.Host != "" {
		return u.Scheme + "://" + u.Host
	}
	return uri
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
