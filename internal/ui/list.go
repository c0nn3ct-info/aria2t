package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/rpc"
)

const (
	tabActive = iota
	tabWaiting
	tabStopped
)

// listModel is the main download list with its three tabs and reorder mode.
type listModel struct {
	a      *App
	tab    int
	cursor int

	reordering bool
	reorderGID string
	origIndex  int
	localOrder []rpc.Status // waiting list as manipulated locally
	savedOrder []rpc.Status // waiting list as it was when grabbing
	pendingG   bool
}

func newListModel(a *App) listModel { return listModel{a: a} }

// rows returns the downloads of the current tab.
func (m listModel) rows() []rpc.Status {
	switch m.tab {
	case tabWaiting:
		if m.reordering {
			return m.localOrder
		}
		return m.a.snap.Waiting
	case tabStopped:
		return m.a.snap.Stopped
	default:
		return m.a.snap.Active
	}
}

func (m listModel) selected() (rpc.Status, bool) {
	rows := m.rows()
	if m.cursor < 0 || m.cursor >= len(rows) {
		return rpc.Status{}, false
	}
	return rows[m.cursor], true
}

func (m *listModel) clampCursor() {
	if n := len(m.rows()); m.cursor >= n {
		m.cursor = n - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// frozenWaiting keeps the local order visible while the user drags an item.
func (m listModel) frozenWaiting(fresh []rpc.Status) []rpc.Status {
	return m.localOrder
}

func (m listModel) update(msg tea.KeyMsg) (listModel, tea.Cmd) {
	a := m.a
	key := msg.String()

	if m.reordering {
		return m.updateReorder(key)
	}
	m.pendingG = false

	switch key {
	case "q":
		return m, tea.Quit
	case "tab":
		m.tab = (m.tab + 1) % 3
		m.cursor = 0
	case "1", "2", "3":
		m.tab = int(key[0] - '1')
		m.cursor = 0
	case "j", "down":
		m.cursor++
		m.clampCursor()
	case "k", "up":
		m.cursor--
		m.clampCursor()
	case "a":
		a.add = newAddModel(a)
		a.overlay = overlayAdd
		return m, a.add.focusCmd()
	case "g":
		a.screen = screenStats
	case ",":
		a.settings = newSettingsModel(a)
		a.overlay = overlayNone
		a.screen = screenSettings
		return m, a.settings.loadCmd()
	case "s":
		a.servers = newServersModel(a)
		a.overlay = overlayServers
		return m, a.servers.probeCmd()
	case "S":
		a.scheduler = newSchedulerModel(a)
		a.screen = screenScheduler
	case "T":
		if a.cfg.Theme == "light" {
			a.setTheme("dark")
		} else {
			a.setTheme("light")
		}
	case "p":
		if s, ok := m.selected(); ok {
			gid := s.GID
			return m, a.rpcCmd("paused "+s.Name(), func(ctx context.Context, c api) error {
				return c.Pause(ctx, gid)
			})
		}
	case "r":
		if s, ok := m.selected(); ok {
			gid := s.GID
			return m, a.rpcCmd("resumed "+s.Name(), func(ctx context.Context, c api) error {
				return c.Unpause(ctx, gid)
			})
		}
	case "d":
		if s, ok := m.selected(); ok {
			gid, stopped := s.GID, m.tab == tabStopped
			return m, a.rpcCmd("removed "+s.Name(), func(ctx context.Context, c api) error {
				if stopped {
					return c.RemoveDownloadResult(ctx, gid)
				}
				return c.Remove(ctx, gid)
			})
		}
	case "enter":
		if s, ok := m.selected(); ok {
			a.detail = newDetailModel(a)
			a.detail.gid = s.GID
			a.screen = screenDetail
			return m, a.detail.refreshCmd()
		}
	case "l":
		if s, ok := m.selected(); ok {
			if m.tab == tabStopped {
				return m, a.flash("stopped downloads cannot be throttled", true)
			}
			a.throttle = newThrottleModel(a)
			a.throttle.gid = s.GID
			a.throttle.name = s.Name()
			a.throttle.speed = s.DownSpeed()
			a.overlay = overlayThrottle
			return m, a.throttle.loadCmd()
		}
	case "t":
		if s, ok := m.selected(); ok {
			if m.tab == tabStopped {
				return m, a.flash("download already stopped — nothing to seed", true)
			}
			if !s.IsTorrent() {
				return m, a.flash("not a torrent", true)
			}
			a.seeding = newSeedingModel(a)
			a.seeding.gid = s.GID
			a.seeding.name = s.Name()
			a.screen = screenSeeding
			return m, a.seeding.loadCmd()
		}
	case "J", "K":
		if m.tab == tabWaiting {
			if s, ok := m.selected(); ok {
				m.reordering = true
				m.reorderGID = s.GID
				m.origIndex = m.cursor
				m.savedOrder = append([]rpc.Status(nil), a.snap.Waiting...)
				m.localOrder = append([]rpc.Status(nil), a.snap.Waiting...)
				return m.updateReorder(key)
			}
		}
	// Stopped-tab integrity actions.
	case "v":
		if m.tab == tabStopped {
			if s, ok := m.selected(); ok {
				return m, a.startVerify(s)
			}
		}
	case "c":
		if m.tab == tabStopped {
			if s, ok := m.selected(); ok {
				gid := s.GID
				a.prompt = newPromptModel(a, "Expected checksum (sha-256)", "", func(v string) tea.Cmd {
					st := a.verify[gid]
					if st == nil {
						st = &verifyState{}
						a.verify[gid] = st
					}
					st.Expected = strings.TrimSpace(v)
					st.Finished = false
					return a.flash("checksum stored — press v to verify", false)
				})
				a.overlay = overlayPrompt
				return m, a.prompt.focusCmd()
			}
		}
	case "R":
		if m.tab == tabStopped {
			if s, ok := m.selected(); ok {
				return m, a.redownload(s)
			}
		}
	case "o":
		if s, ok := m.selected(); ok {
			return m, a.openDir(s.Dir)
		}
	}
	return m, nil
}

func (m listModel) updateReorder(key string) (listModel, tea.Cmd) {
	a := m.a
	move := func(to int) {
		if to < 0 || to >= len(m.localOrder) || m.cursor == to {
			return
		}
		item := m.localOrder[m.cursor]
		rest := append(append([]rpc.Status(nil), m.localOrder[:m.cursor]...), m.localOrder[m.cursor+1:]...)
		m.localOrder = append(append(append([]rpc.Status(nil), rest[:to]...), item), rest[to:]...)
		m.cursor = to
	}
	switch key {
	case "J":
		m.pendingG = false
		move(m.cursor + 1)
	case "K":
		m.pendingG = false
		move(m.cursor - 1)
	case "G":
		m.pendingG = false
		move(len(m.localOrder) - 1)
	case "g":
		if m.pendingG {
			m.pendingG = false
			move(0)
		} else {
			m.pendingG = true
		}
	case "enter":
		gid, order, cursor := m.reorderGID, m.localOrder, m.cursor
		m.reordering = false
		a.snap.Waiting = m.localOrder
		return m, a.rpcCmd(fmt.Sprintf("moved to #%d", cursor+1), func(ctx context.Context, c api) error {
			// The local order was frozen during the drag; items may have
			// activated or completed meanwhile. Recompute the target rank
			// against the live queue so the drop lands where it looks.
			fresh, err := c.TellWaiting(ctx, 0, 1000)
			if err != nil {
				return err
			}
			still := make(map[string]bool, len(fresh))
			for _, s := range fresh {
				still[s.GID] = true
			}
			if !still[gid] {
				return fmt.Errorf("%s left the waiting queue during reorder", gid)
			}
			pos := 0
			for _, s := range order[:cursor] {
				if still[s.GID] {
					pos++
				}
			}
			_, err = c.ChangePosition(ctx, gid, pos, "POS_SET")
			return err
		})
	case "esc":
		m.reordering = false
		a.snap.Waiting = m.savedOrder
		m.cursor = m.origIndex
		m.clampCursor()
	}
	return m, nil
}

// ---- rendering ----

// pad truncates or right-pads a plain (unstyled) string to w cells.
func pad(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		if w <= 1 {
			return string(r[:w])
		}
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}

// lpad left-pads a plain string to w cells.
func lpad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return s
	}
	return strings.Repeat(" ", w-len(r)) + s
}

func (m listModel) tabsLine() string {
	st := m.a.styles
	names := []string{
		fmt.Sprintf("Active %d", len(m.a.snap.Active)),
		fmt.Sprintf("Waiting %d", len(m.a.snap.Waiting)),
		fmt.Sprintf("Stopped %d", len(m.a.snap.Stopped)),
	}
	parts := make([]string, 3)
	for i, n := range names {
		if i == m.tab {
			parts[i] = st.Badge.Render(n)
		} else {
			parts[i] = st.Dim.Render("[ " + n + " ]")
		}
	}
	return " " + strings.Join(parts, " ")
}

func (m listModel) statusCell(s rpc.Status) string {
	st := m.a.styles
	switch s.Status {
	case "complete":
		return st.Green.Render("✓ done")
	case "error":
		return st.Red.Render("✗ error")
	case "removed":
		return st.Dim.Render("removed")
	case "paused":
		return st.Yellow.Render("⏸ paused")
	case "waiting":
		return st.Yellow.Render("⏸ waiting")
	default:
		return st.Dim.Render(s.Status)
	}
}

// integrityCell renders the stopped-tab INTEGRITY column.
func (m listModel) integrityCell(s rpc.Status) string {
	st := m.a.styles
	v := m.a.verify[s.GID]
	switch {
	case v == nil:
		return st.Dim.Render("—")
	case v.Running:
		frac := 0.0
		if total := v.Total.Load(); total > 0 {
			frac = float64(v.Done.Load()) / float64(total)
		}
		f, e := Bar(frac, 11)
		return st.Cyan.Render("verifying ") + st.Brand.Render(f) + st.Faint.Render(e) + fmt.Sprintf(" %d%%", int(frac*100))
	case v.Finished && v.Err != nil:
		return st.Red.Render("✗ " + v.Err.Error())
	case v.Finished && v.OK:
		return st.Green.Render("✓ sha-256 verified")
	case v.Finished:
		return st.Red.Render("✗ sha-256 MISMATCH — ") + st.Text.Render("R re-download")
	case v.Expected != "":
		return st.Dim.Render("checksum set — v to verify")
	default:
		return st.Dim.Render("—")
	}
}

func (m listModel) view() string {
	a := m.a
	st := a.styles
	var b strings.Builder
	b.WriteString(a.header())
	if m.reordering {
		b.WriteString("  " + st.BadgeWarn.Render("REORDER MODE"))
	}
	b.WriteString("\n" + m.tabsLine() + "\n")

	nameW := a.width - 60
	if nameW < 20 {
		nameW = 20
	}
	rows := m.rows()
	var lines []string

	switch {
	case m.tab == tabStopped:
		head := st.Dim.Render(pad("NAME", nameW+2) + pad("STATUS", 14) + "INTEGRITY")
		lines = append(lines, head)
		for i, s := range rows {
			marker, style := "  ", st.Text
			if i == m.cursor {
				marker, style = st.Brand.Render("▸")+" ", st.Title
			}
			status := m.statusCell(s)
			row := marker + style.Render(pad(s.Name(), nameW)) + " " +
				status + strings.Repeat(" ", max(1, 14-lipgloss.Width(status))) +
				m.integrityCell(s)
			if i == m.cursor {
				row = st.RowSel.Render(row)
			}
			lines = append(lines, row)
		}
	case m.tab == tabWaiting && m.reordering:
		head := st.Dim.Render(pad("POS", 5) + pad("NAME", nameW+2) + lpad("SIZE", 10) + lpad("STATUS", 12))
		lines = append(lines, head)
		for i, s := range rows {
			posCell := st.Dim.Render(pad(fmt.Sprintf("%d", i+1), 5))
			name := pad(s.Name(), nameW)
			row := "  " + posCell + st.Text.Render(name) + "  " +
				st.Dim.Render(lpad(FmtBytes(s.Total()), 10)) + st.Dim.Render(lpad(s.Status, 12))
			if s.GID == m.reorderGID {
				grabbed := st.Magenta.Render(pad(fmt.Sprintf("%d ↕", i+1), 5)) +
					st.Title.Render(name) + st.Magenta.Render(fmt.Sprintf(" ◂ grabbed — was #%d", m.origIndex+1))
				row = st.RowSel.Render("  " + grabbed)
			}
			lines = append(lines, row)
		}
	default:
		head := st.Dim.Render(pad("NAME", nameW+2) + pad("PROGRESS", 28) + lpad("SIZE", 9) + lpad("SPEED", 12) + lpad("ETA", 9))
		lines = append(lines, head)
		for i, s := range rows {
			marker, style := "  ", st.Text
			if i == m.cursor {
				marker, style = st.Brand.Render("▸")+" ", st.Title
			}
			name := s.Name()
			var progress string
			switch s.Status {
			case "complete":
				progress = st.Green.Render(strings.Repeat("━", 20)) + "     "
			case "error":
				f, e := Bar(s.Progress(), 20)
				progress = st.Red.Render(f) + st.Faint.Render(e) + "     "
			case "waiting", "paused":
				progress = st.Faint.Render(strings.Repeat("─", 20)) + "     "
			default:
				f, e := Bar(s.Progress(), 20)
				progress = st.Brand.Render(f) + st.Faint.Render(e) + fmt.Sprintf(" %3d%%", int(s.Progress()*100))
			}
			suffix := ""
			switch s.Status {
			case "complete":
				suffix = " " + st.Green.Render("✓ done")
			case "error":
				suffix = " " + st.Red.Render("✗ error")
			case "waiting":
				suffix = " " + st.Yellow.Render("⏸ waiting")
			case "paused":
				suffix = " " + st.Yellow.Render("⏸ paused")
			}
			nw := max(nameW-lipgloss.Width(suffix), 8)
			row := marker + style.Render(pad(name, nw)) + suffix + "  " + progress +
				st.Dim.Render(lpad(FmtBytes(s.Total()), 9)) +
				st.Cyan.Render(lpad(FmtSpeed(s.DownSpeed()), 12)) +
				st.Dim.Render(lpad(FmtETA(s.Total()-s.Completed(), s.DownSpeed()), 9))
			if i == m.cursor {
				row = st.RowSel.Render(row)
			}
			lines = append(lines, row)
		}
	}
	if len(rows) == 0 {
		lines = append(lines, st.Dim.Render("  nothing here"))
	}
	b.WriteString(st.Panel.Width(a.width - 2).Render(strings.Join(lines, "\n")))
	b.WriteString("\n")

	// Checksum detail strip (design 3d) for the selected stopped download.
	if m.tab == tabStopped {
		if s, ok := m.selected(); ok {
			if v := m.a.verify[s.GID]; v != nil && (v.Expected != "" || v.Running || v.Finished) {
				var det []string
				det = append(det, st.Dim.Render("CHECKSUM · "+s.Name()))
				det = append(det, st.Dim.Render("expected ")+st.Text.Render("sha-256:"+shortHash(v.Expected)))
				switch {
				case v.Running:
					det = append(det, st.Dim.Render("computed ")+st.Cyan.Render(
						fmt.Sprintf("hashing %s / %s…", FmtBytes(v.Done.Load()), FmtBytes(v.Total.Load()))))
				case v.Finished && v.Computed != "":
					det = append(det, st.Dim.Render("computed ")+st.Text.Render("sha-256:"+shortHash(v.Computed)))
				}
				b.WriteString(st.Panel.Width(a.width - 2).Render(strings.Join(det, "\n")))
				b.WriteString("\n")
			}
		}
	}

	b.WriteString(m.keybar())
	b.WriteString(a.statusLine())
	return b.String()
}

func shortHash(h string) string {
	if len(h) > 16 {
		return h[:8] + "…" + h[len(h)-8:]
	}
	return h
}

func (m listModel) keybar() string {
	st := m.a.styles
	key := func(k, label string) string { return st.Key.Render(k) + " " + st.Dim.Render(label) }
	var parts []string
	if m.reordering {
		parts = []string{
			st.Magenta.Bold(true).Render("J/K") + " " + st.Dim.Render("move down/up"),
			st.Magenta.Bold(true).Render("gg/G") + " " + st.Dim.Render("to top/bottom"),
			st.Magenta.Bold(true).Render("↵") + " " + st.Dim.Render("drop"),
			st.Magenta.Bold(true).Render("esc") + " " + st.Dim.Render("cancel"),
		}
	} else {
		parts = []string{
			key("a", "add"), key("p", "pause"), key("r", "resume"), key("d", "remove"),
			key("↵", "details"), key("g", "stats"), key("l", "limit"), key("s", "servers"),
			key("S", "sched"), key(",", "settings"), key("q", "quit"),
		}
		if m.tab == tabWaiting {
			parts = append(parts, key("J/K", "reorder"))
		}
		if m.tab == tabStopped {
			parts = append(parts, key("v", "verify"), key("R", "re-download"), key("c", "checksum"), key("o", "open"))
		}
	}
	pos := ""
	if n := len(m.rows()); n > 0 {
		pos = st.Dim.Render(fmt.Sprintf("%d/%d", m.cursor+1, n))
	}
	line := " " + strings.Join(parts, "  ")
	gap := m.a.width - lipgloss.Width(line) - lipgloss.Width(pos) - 2
	if gap < 1 {
		gap = 1
	}
	return line + strings.Repeat(" ", gap) + pos
}
