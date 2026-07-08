package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/rpc"
)

const (
	tabAll = iota
	tabActive
	tabWaiting
	tabStopped
)

// isStopped reports whether a status is a finished/failed result rather than
// a live download — the rows that live on the Stopped tab.
func isStopped(status string) bool {
	return status == "complete" || status == "error" || status == "removed"
}

// listModel is the main download list with its three tabs and reorder mode.
type listModel struct {
	a      *App
	tab    int
	cursor int
	offset int // first visible row (scrolling)

	reordering bool
	reorderGID string
	origIndex  int
	localOrder []rpc.Status // waiting list as manipulated locally
	savedOrder []rpc.Status // waiting list as it was when grabbing
	pendingG   bool

	// filtering is true while the user types into the filter; a non-empty
	// filterInput value keeps the filter applied after enter commits it.
	filtering   bool
	filterInput textinput.Model
}

func newListModel(a *App) listModel {
	in := textinput.New()
	in.Placeholder = "type to filter"
	in.CharLimit = 64
	in.Width = 28
	return listModel{a: a, filterInput: in}
}

// filterQuery is the active filter text, "" when none.
func (m listModel) filterQuery() string { return strings.TrimSpace(m.filterInput.Value()) }

// filtered narrows a status slice by the active name filter.
func (m listModel) filtered(base []rpc.Status) []rpc.Status {
	q := strings.ToLower(m.filterQuery())
	if q == "" {
		return base
	}
	out := make([]rpc.Status, 0, len(base))
	for _, s := range base {
		if strings.Contains(strings.ToLower(s.Name()), q) {
			out = append(out, s)
		}
	}
	return out
}

// rows returns the downloads of the current tab, narrowed by the filter. The
// All tab concatenates active, waiting and stopped so a download stays on
// screen (and just changes badge) as it moves between states.
func (m listModel) rows() []rpc.Status {
	switch m.tab {
	case tabWaiting:
		if m.reordering {
			return m.localOrder // reorder and filter are mutually exclusive
		}
		return m.filtered(m.a.snap.Waiting)
	case tabStopped:
		return m.filtered(m.a.snap.Stopped)
	case tabActive:
		return m.filtered(m.a.snap.Active)
	default: // tabAll — always a fresh slice so append never scribbles on snap
		act := m.filtered(m.a.snap.Active)
		wait := m.filtered(m.a.snap.Waiting)
		stop := m.filtered(m.a.snap.Stopped)
		out := make([]rpc.Status, 0, len(act)+len(wait)+len(stop))
		out = append(out, act...)
		out = append(out, wait...)
		out = append(out, stop...)
		return out
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
	vis := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// visibleRows is how many download rows fit between the chrome lines.
func (m listModel) visibleRows() int {
	v := m.a.height - 8 - m.stripHeight()
	if v < 3 {
		v = 3
	}
	return v
}

// stripHeight is the height of the checksum detail strip when it renders.
func (m listModel) stripHeight() int {
	if m.tab != tabStopped {
		return 0
	}
	s, ok := m.selected()
	if !ok {
		return 0
	}
	v := m.a.verify[s.GID]
	if v == nil || (v.Expected == "" && !v.Running && !v.Finished) {
		return 0
	}
	h := 4 // borders + 2 lines
	if v.Running || (v.Finished && v.Computed != "") {
		h++
	}
	return h
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
	if m.filtering {
		switch key {
		case "esc":
			m.filtering = false
			m.filterInput.SetValue("")
			m.filterInput.Blur()
		case "enter":
			m.filtering = false
			m.filterInput.Blur()
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.clampCursor()
			return m, cmd
		}
		m.clampCursor()
		return m, nil
	}
	m.pendingG = false

	switch key {
	case "q":
		// Reassure before quitting mid-transfer: the managed daemon pauses
		// and resumes the session, but a bare quit looks like data loss.
		if n := len(a.snap.Active) + len(a.snap.Waiting); n > 0 {
			a.confirm = newConfirmModel(a, "Quit aria2t?",
				fmt.Sprintf("%d download(s) still going — they'll pause now and resume next launch.", n),
				func() tea.Cmd { return tea.Quit })
			a.confirm.yesLabel = "Quit"
			a.overlay = overlayConfirm
			return m, nil
		}
		return m, tea.Quit
	case "/":
		m.filtering = true
		return m, m.filterInput.Focus()
	case "esc":
		if m.filterQuery() != "" {
			m.filterInput.SetValue("")
			m.clampCursor()
		}
	case "tab":
		m.tab = (m.tab + 1) % 4
		m.cursor, m.offset = 0, 0
		m.filterInput.SetValue("")
	case "1", "2", "3", "4":
		m.tab = int(key[0] - '1')
		m.cursor, m.offset = 0, 0
		m.filterInput.SetValue("")
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
	case "p":
		if s, ok := m.selected(); ok {
			gid := s.GID
			return m, a.rpcCmd("paused "+s.Name(), func(ctx context.Context, c api) error {
				return c.Pause(ctx, gid)
			})
		}
	case "P":
		return m, a.rpcCmd("paused all", func(ctx context.Context, c api) error {
			return c.PauseAll(ctx)
		})
	case "U":
		return m, a.rpcCmd("resumed all", func(ctx context.Context, c api) error {
			return c.UnpauseAll(ctx)
		})
	case "y":
		if s, ok := m.selected(); ok {
			return m, a.yank(s)
		}
	case "X":
		if m.tab == tabStopped || m.tab == tabAll {
			a.confirm = newConfirmModel(a, "Clear stopped list?",
				fmt.Sprintf("%d download results will be forgotten", len(a.snap.Stopped)),
				func() tea.Cmd {
					return a.rpcCmd("stopped list cleared", func(ctx context.Context, c api) error {
						return c.PurgeDownloadResult(ctx)
					})
				})
			a.confirm.yesLabel = "Clear"
			a.overlay = overlayConfirm
		}
	case " ":
		// Smart toggle keyed off the row's own state, so it works on any tab.
		if s, ok := m.selected(); ok {
			if isStopped(s.Status) {
				return m, a.flash("download already finished", true)
			}
			gid, name := s.GID, s.Name()
			if s.Status == "paused" {
				return m, a.rpcCmd("resumed "+name, func(ctx context.Context, c api) error {
					return c.Unpause(ctx, gid)
				})
			}
			return m, a.rpcCmd("paused "+name, func(ctx context.Context, c api) error {
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
			gid, stopped, name := s.GID, isStopped(s.Status), s.Name()
			return m, a.confirmRemove(name, func() tea.Cmd {
				return a.rpcCmd("removed "+name, func(ctx context.Context, c api) error {
					if stopped {
						return c.RemoveDownloadResult(ctx, gid)
					}
					if err := c.Remove(ctx, gid); err != nil {
						return err
					}
					_ = c.RemoveDownloadResult(ctx, gid) // purge so --force-save can't resurrect it
					return nil
				})
			})
		}
	case "enter":
		if s, ok := m.selected(); ok {
			a.detail = newDetailModel(a)
			a.detail.gid = s.GID
			a.screen = screenDetail
			return m, a.detail.refreshCmd()
		}
		// Empty list: offer a one-tap add from the clipboard (magnets still go
		// through the pick-before-start flow).
		if src := strings.TrimSpace(clipboardRead()); looksLikeSource(src) {
			return m, a.addURICmd([]string{src}, nil, true)
		}
		return m, a.flash("no link on the clipboard — press a to add", true)
	case "l":
		if s, ok := m.selected(); ok {
			if isStopped(s.Status) {
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
			if isStopped(s.Status) {
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
			if m.filterQuery() != "" {
				return m, a.flash("clear the filter (esc) before reordering", true)
			}
			if s, ok := m.selected(); ok {
				m.reordering = true
				m.reorderGID = s.GID
				m.origIndex = m.cursor
				m.savedOrder = append([]rpc.Status(nil), a.snap.Waiting...)
				m.localOrder = append([]rpc.Status(nil), a.snap.Waiting...)
				return m.updateReorder(key)
			}
		} else if m.tab == tabAll {
			return m, a.flash("switch to the Waiting tab to reorder", true)
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
	total := len(m.a.snap.Active) + len(m.a.snap.Waiting) + len(m.a.snap.Stopped)
	names := []string{
		fmt.Sprintf("All %d", total),
		fmt.Sprintf("Active %d", len(m.a.snap.Active)),
		fmt.Sprintf("Waiting %d", len(m.a.snap.Waiting)),
		fmt.Sprintf("Stopped %d", len(m.a.snap.Stopped)),
	}
	parts := make([]string, len(names))
	x := 1 // leading space
	for i, n := range names {
		if i == m.tab {
			parts[i] = st.Badge.Render(n)
		} else {
			parts[i] = st.Dim.Render("[ " + n + " ]")
		}
		w := lipgloss.Width(parts[i])
		m.a.hits.add(fmt.Sprintf("tab:%d", i), x, 1, x+w-1, 1)
		x += w + 1
	}
	line := " " + strings.Join(parts, " ")
	if q := m.filterQuery(); q != "" {
		line += "  " + st.Cyan.Render("⌕ "+q)
	}
	return line
}

// connCellStyled renders the connections column (and seeds for torrents, as
// "conns:seeds"), left-padded to a fixed width so rows stay aligned. The
// separator is ":" not "·": the middle dot is East-Asian-ambiguous (2 cells on
// terminals in ambiguous-wide mode), which would break column alignment.
func connCellStyled(st Styles, s rpc.Status) string {
	var cell string
	switch {
	case s.IsTorrent():
		cell = fmt.Sprintf("%d:%d", s.Conns(), s.Seeds())
	case s.Conns() > 0:
		cell = fmt.Sprintf("%d", s.Conns())
	default:
		cell = "-"
	}
	return st.Dim.Render(lpad(trunc(cell, 7), 7))
}

func (m listModel) statusCell(s rpc.Status) string {
	st := m.a.styles
	switch s.Status {
	case "complete":
		return st.Green.Render("done")
	case "error":
		return st.Red.Render("error")
	case "removed":
		return st.Dim.Render("removed")
	case "paused":
		return st.Yellow.Render("paused")
	case "waiting":
		return st.Yellow.Render("waiting")
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
		if s.Status == "error" {
			return st.Red.Render(pad("✗ "+friendlyError(s.ErrorCode, s.ErrorMessage), 44))
		}
		return st.Dim.Render("-")
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
		return st.Dim.Render("-")
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

	// Empty, connected and unfiltered → a friendly welcome instead of a bare
	// table, so a first-time user knows what to do.
	if a.connected && !m.filtering && m.filterQuery() == "" &&
		len(a.snap.Active)+len(a.snap.Waiting)+len(a.snap.Stopped) == 0 {
		welcome := []string{
			st.Title.Render("Welcome to aria2t"),
			"",
			st.Text.Render("Your downloads will show up here."),
			st.Text.Render("Press ") + st.Key.Render("a") + st.Text.Render(" to add one — a URL, a magnet link, or a .torrent file."),
			st.Text.Render("Or press ") + st.Key.Render("↵") + st.Text.Render(" to add a link from your clipboard."),
		}
		b.WriteString(st.Panel.Width(a.width - 2).Render(strings.Join(welcome, "\n")))
		return m.bottomBar(b.String())
	}

	// Panel content wraps at a.width-4 (Width minus padding); rows are
	// nameW+barW+cols cells, so on narrow terminals the deficit comes out
	// of the progress bar to keep every row on a single line.
	nameW := a.width - 72
	barW := 20
	if nameW < 20 {
		barW -= 20 - nameW
		if barW < 8 {
			barW = 8
		}
		nameW = 20
	}
	rows := m.rows()
	start := m.offset
	if start > len(rows) {
		start = len(rows)
	}
	end := start + m.visibleRows()
	if end > len(rows) {
		end = len(rows)
	}
	win := rows[start:end]
	rowRegion := func(wi int) {
		a.hits.add(fmt.Sprintf("row:%d", start+wi), 1, 4+wi, a.width-2, 4+wi)
	}
	var lines []string

	switch {
	case m.tab == tabStopped:
		head := st.Dim.Render(pad("NAME", nameW+2) + pad("STATUS", 14) + "INTEGRITY")
		lines = append(lines, head)
		for wi, s := range win {
			i := start + wi
			rowRegion(wi)
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
		for wi, s := range win {
			i := start + wi
			rowRegion(wi)
			posCell := st.Dim.Render(pad(fmt.Sprintf("%d", i+1), 5))
			name := pad(s.Name(), nameW)
			row := "  " + posCell + st.Text.Render(name) + "  " +
				st.Dim.Render(lpad(FmtBytes(s.Total()), 10)) + st.Dim.Render(lpad(s.Status, 12))
			if s.GID == m.reorderGID {
				grabbed := st.Magenta.Render(pad(fmt.Sprintf("%d", i+1), 5)) +
					st.Title.Render(name) + st.Magenta.Render(fmt.Sprintf(" ◂ grabbed — was #%d", m.origIndex+1))
				row = st.RowSel.Render("  " + grabbed)
			}
			lines = append(lines, row)
		}
	default:
		// Columns: [marker+NAME] STATUS PROGRESS SIZE SPEED CONN ETA. STATUS is
		// its own fixed-width column (a coloured word) so it never rags against
		// the name or shifts PROGRESS; PROGRESS is bar(barW)+5-cell %/gap. The
		// name shrinks by statusW+gap to keep the total budget (a.width-72).
		statusW := 9
		nameCol := max(nameW-statusW-1, 8)
		head := st.Dim.Render(pad("NAME", nameCol+2) + " " + pad("STATUS", statusW) + " " +
			pad("PROGRESS", barW+5) + lpad("SIZE", 9) + lpad("SPEED", 12) + lpad("CONN", 7) + lpad("ETA", 9))
		lines = append(lines, head)
		for wi, s := range win {
			i := start + wi
			rowRegion(wi)
			marker, style := "  ", st.Text
			if i == m.cursor {
				marker, style = st.Brand.Render("▸")+" ", st.Title
			}
			var progress string
			switch s.Status {
			case "complete":
				progress = st.Green.Render(strings.Repeat("━", barW)) + "     "
			case "error":
				f, e := Bar(s.Progress(), barW)
				progress = st.Red.Render(f) + st.Faint.Render(e) + "     "
			case "waiting", "paused":
				progress = st.Faint.Render(strings.Repeat("─", barW)) + "     "
			default:
				f, e := Bar(s.Progress(), barW)
				progress = st.Brand.Render(f) + st.Faint.Render(e) + fmt.Sprintf(" %3d%%", int(s.Progress()*100))
			}
			// Status is a coloured word, no leading icon: the glyphs (● ⏸ ✓) are
			// East-Asian-ambiguous or emoji and render 2 cells on some terminals,
			// which would shift every column right of here.
			word, wstyle := s.Status, st.Dim
			switch s.Status {
			case "complete":
				word, wstyle = "done", st.Green
			case "error":
				word, wstyle = "error", st.Red
			case "waiting":
				word, wstyle = "waiting", st.Yellow
			case "paused":
				word, wstyle = "paused", st.Yellow
			case "active":
				word, wstyle = "active", st.Green
			}
			// A torrent whose download finished but is still uploading stays
			// "active" in aria2; show it as "seeding" (upload-coloured) so the
			// status visibly changes on completion.
			if s.IsSeeding() {
				word, wstyle = "seeding", st.Magenta
			}
			row := marker + style.Render(pad(s.Name(), nameCol)) + " " +
				wstyle.Render(pad(word, statusW)) + " " + progress +
				st.Dim.Render(lpad(FmtBytes(s.Total()), 9)) +
				st.Cyan.Render(lpad(FmtSpeed(s.DownSpeed()), 12)) +
				connCellStyled(st, s) +
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

	return m.bottomBar(b.String())
}

// bottomBar pins the key-bar (and the transient status line, when a flash is
// showing) to the bottom of the terminal: it pads the middle content down so
// the key-bar is always the last visible row and never scrolls off a short
// terminal — the reason the hints "sometimes disappeared". Overlong content is
// clipped (windowing normally prevents that; this is only a tiny-terminal
// safety net that keeps the key-bar rather than the panel's bottom border).
func (m listModel) bottomBar(mid string) string {
	a := m.a
	status := a.statusLine() // "" or "\n <flash>"
	top := a.height - 1 - strings.Count(status, "\n")
	if top < 1 {
		top = 1
	}
	lines := strings.Split(strings.TrimRight(mid, "\n"), "\n")
	if len(lines) > top {
		lines = lines[:top]
	}
	for len(lines) < top {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n" + m.keybar(top) + status
}

func shortHash(h string) string {
	if len(h) > 16 {
		return h[:8] + "…" + h[len(h)-8:]
	}
	return h
}

// keybar renders the hint line at row y via the shared hintbarEx renderer;
// every tokened hint is clickable. While the filter is being typed it shows the
// filter input instead.
func (m listModel) keybar(y int) string {
	st := m.a.styles
	if m.filtering {
		return " " + st.Key.Render("/") + " " + m.filterInput.View() +
			"  " + st.Dim.Render("↵ keep · esc clear")
	}
	var hints []keyHint
	keyStyle := st.Key
	if m.reordering {
		keyStyle = st.Magenta.Bold(true)
		hints = []keyHint{
			{"J", "J", "down"}, {"K", "K", "up"},
			{"", "gg/G", "to top/bottom"}, // a chord — no single-click semantics
			{"enter", "↵", "drop"}, {"esc", "esc", "cancel"},
		}
	} else if m.tab == tabStopped {
		// Stopped tab leads with integrity actions; ordered most-used-first,
		// hintbarEx drops the rightmost (?, q) first on a narrow terminal.
		hints = []keyHint{
			{"a", "a", "add"}, {"enter", "↵", "details"}, {"o", "o", "open"},
			{"v", "v", "verify"}, {"c", "c", "checksum"}, {"R", "R", "re-download"},
			{"d", "d", "remove"}, {"X", "X", "clear"}, {"/", "/", "filter"},
			{"y", "y", "copy url"}, {"s", "s", "servers"}, {",", ",", "settings"},
			{"?", "?", "help"}, {"q", "q", "quit"},
		}
	} else {
		// Active/All/Waiting, ordered most-used-first. The two newly-surfaced
		// screen switches (S/t) sit at the tail so they are the first the
		// width-adaptive bar drops on a narrow terminal — the essentials
		// (settings/help/quit) survive; S/t are also in the ? reference.
		hints = []keyHint{
			{"a", "a", "add"}, {" ", "space", "pause"}, {"enter", "↵", "details"},
			{"d", "d", "remove"}, {"l", "l", "limit"}, {"/", "/", "filter"},
			{"y", "y", "copy url"}, {"g", "g", "stats"}, {"s", "s", "servers"},
			{",", ",", "settings"}, {"?", "?", "help"}, {"q", "q", "quit"}, // q guarded by Quit confirm
			{"S", "S", "scheduler"}, {"t", "t", "seeding"},
		}
		if m.tab == tabWaiting {
			hints = append(hints, keyHint{"", "J/K", "reorder"}) // teaser: no row to grab yet
		}
	}
	pos := ""
	if n := len(m.rows()); n > 0 {
		pos = st.Dim.Render(fmt.Sprintf("%d/%d", m.cursor+1, n))
	}
	return m.a.hintbarEx(y, hints, keyStyle, pos)
}

// mouse handles clicks on the list screen. Single click selects a row; every
// action is reached by clicking its key-bar hint (including ↵ details).
func (m listModel) mouse(id string) (listModel, tea.Cmd) {
	kind, arg := splitID(id)
	switch kind {
	case "tab":
		if t := argInt(arg); t >= 0 && t <= 3 && !m.reordering {
			m.tab = t
			m.cursor, m.offset = 0, 0
		}
	case "row":
		i := argInt(arg)
		if i < 0 || i >= len(m.rows()) || m.reordering {
			return m, nil // dragging is keyboard-driven
		}
		m.cursor = i
		m.clampCursor()
		// A click opens the row's details — the expected primary action, and
		// the only mouse path to them if the key-bar is scrolled off a short
		// terminal. enter/keyboard still select without leaving the list.
		return m.update(key_("enter"))
	case "key":
		return m.update(keyFromToken(arg))
	}
	return m, nil
}

// keyFromToken converts a keybar token back into a key message, so a click on
// a hint triggers exactly the same handler as the key.
func keyFromToken(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "^s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	case "^d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "^t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	case "^r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case "^o":
		return tea.KeyMsg{Type: tea.KeyCtrlO}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}
