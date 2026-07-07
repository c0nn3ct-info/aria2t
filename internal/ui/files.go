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

// filesModel is the tree+checkbox file picker. It is opened both from the add
// flow (after a torrent is added paused) and from the detail screen (f). It
// loads the download's files via tellStatus, presents a collapsible tree, and
// applies the chosen subset through the select-file option.
type filesModel struct {
	a    *App
	gid  string   // single-download picker (torrent / magnet / detail f)
	gids []string // multi-download picker (metalink)
	name string

	root   *treeNode
	rows   []*treeNode // flattened visible nodes
	cursor int
	top    int // first visible row (scrolling)

	loading bool
	tries   int
	err     error

	// add-flow follow-up: unpause once the user confirms/cancels.
	fromAdd      bool
	unpauseAfter bool
	moreQueued   int    // more magnets waiting behind this picker
	pickKey      string // persisted-pick key to clear when answered ("" = none)
}

func newFilesModel(a *App) filesModel { return filesModel{a: a, loading: true} }

// loadCmd fetches the download's file list (single) or every download's status
// (multi-download metalink).
func (m filesModel) loadCmd() tea.Cmd {
	c := m.a.client
	if c == nil {
		return nil
	}
	if len(m.gids) > 0 {
		gids := m.gids
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var sts []rpc.Status
			for _, g := range gids {
				s, err := c.TellStatus(ctx, g)
				if err != nil {
					return filesMultiMsg{gids: gids, err: err}
				}
				sts = append(sts, s)
			}
			return filesMultiMsg{gids: gids, statuses: sts}
		}
	}
	gid := m.gid
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s, err := c.TellStatus(ctx, gid)
		return filesDataMsg{gid: gid, dir: s.Dir, files: s.Files, err: err}
	}
}

// absorbMulti builds the metalink picker from the downloads' statuses.
func (m filesModel) absorbMulti(msg filesMultiMsg) (filesModel, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.loading = false
		return m, nil
	}
	m.root = buildForest(msg.statuses)
	m.rows = flatten(m.root)
	m.loading = false
	m.err = nil
	m.clamp()
	return m, nil
}

// retryCmd re-loads after a short delay: a just-added torrent may not have
// been parsed into its file list yet.
func (m filesModel) retryCmd() tea.Cmd {
	gid := m.gid
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
		return filesRetryMsg{gid: gid}
	})
}

// absorb applies a file list into the tree, handling the not-yet-parsed and
// add-flow-single-file cases.
func (m filesModel) absorb(msg filesDataMsg) (filesModel, tea.Cmd) {
	if msg.gid != m.gid {
		return m, nil
	}
	if msg.err != nil {
		m.err = msg.err
		m.loading = false
		return m, nil
	}
	if len(msg.files) == 0 {
		if m.tries < 5 {
			m.tries++
			return m, m.retryCmd()
		}
		m.loading = false
		return m, nil
	}
	m.root = buildTree(msg.files, msg.dir)
	m.rows = flatten(m.root)
	m.loading = false
	m.err = nil
	// A single-file torrent added for selection needs no picker.
	if m.fromAdd && leafCount(m.root) <= 1 {
		return m, m.finishSingle()
	}
	m.clamp()
	return m, nil
}

// finishSingle closes the add-flow picker for a single-file torrent.
func (m filesModel) finishSingle() tea.Cmd {
	a := m.a
	a.overlay = overlayNone
	a.clearPick(m.pickKey)
	if m.unpauseAfter {
		gid := m.gid
		return a.rpcCmd("added", func(ctx context.Context, c api) error {
			return c.Unpause(ctx, gid)
		})
	}
	return a.flash("added (paused)", false)
}

func (m *filesModel) clamp() {
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	vis := m.maxVisible()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+vis {
		m.top = m.cursor - vis + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

// maxVisible is how many tree rows fit inside the modal.
func (m filesModel) maxVisible() int {
	v := m.a.height - 12 // title, blanks, hint, buttons, modal chrome
	if v < 3 {
		v = 3
	}
	return v
}

func (m filesModel) current() *treeNode {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	return m.rows[m.cursor]
}

func (m filesModel) update(msg tea.KeyMsg) (filesModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, m.cancelCmd()
	case "enter", "ctrl+s":
		return m, m.confirmCmd()
	case "j", "down":
		m.cursor++
		m.clamp()
	case "k", "up":
		m.cursor--
		m.clamp()
	case " ":
		if n := m.current(); n != nil {
			toggleNode(n)
		}
	case "a":
		if m.root != nil {
			setSelected(m.root, true)
		}
	case "n":
		if m.root != nil {
			setSelected(m.root, false)
		}
	case "l", "right":
		if n := m.current(); n != nil && !n.isLeaf() && n.collapsed {
			n.collapsed = false
			m.rows = flatten(m.root)
			m.clamp()
		}
	case "h", "left":
		n := m.current()
		if n == nil {
			return m, nil
		}
		if !n.isLeaf() && !n.collapsed {
			n.collapsed = true
			m.rows = flatten(m.root)
			m.clamp()
			return m, nil
		}
		// Otherwise jump to the parent, if it is visible.
		if n.parent != nil {
			for i, r := range m.rows {
				if r == n.parent {
					m.cursor = i
					m.clamp()
					break
				}
			}
		}
	}
	return m, nil
}

// confirmMultiCmd keeps the chosen metalink downloads and removes the rest.
func (m filesModel) confirmMultiCmd() tea.Cmd {
	a := m.a
	keep := selectedGids(m.root)
	if len(keep) == 0 {
		return a.flash("select at least one file", true)
	}
	keepSet := map[string]bool{}
	for _, g := range keep {
		keepSet[g] = true
	}
	var drop []string
	for _, g := range m.gids {
		if !keepSet[g] {
			drop = append(drop, g)
		}
	}
	unpause := m.unpauseAfter
	a.overlay = overlayNone
	a.clearPick(m.pickKey)
	return a.rpcCmd("files selected", func(ctx context.Context, c api) error {
		for _, g := range drop {
			if err := c.Remove(ctx, g); err != nil {
				return err
			}
		}
		if unpause {
			for _, g := range keep {
				if err := c.Unpause(ctx, g); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// confirmCmd applies the selection (and unpauses in the add flow).
func (m filesModel) confirmCmd() tea.Cmd {
	if len(m.gids) > 0 {
		return m.confirmMultiCmd()
	}
	a := m.a
	gid, unpause := m.gid, m.unpauseAfter
	if m.root == nil { // nothing loaded: keep all files, just proceed
		a.overlay = overlayNone
		a.clearPick(m.pickKey)
		if unpause {
			return a.rpcCmd("added", func(ctx context.Context, c api) error {
				return c.Unpause(ctx, gid)
			})
		}
		return nil
	}
	idx := selectedIndices(m.root)
	if len(idx) == 0 {
		return a.flash("select at least one file", true)
	}
	value := strings.Join(idx, ",")
	a.overlay = overlayNone
	a.clearPick(m.pickKey)
	return a.rpcCmd("files selected", func(ctx context.Context, c api) error {
		if err := c.ChangeOption(ctx, gid, map[string]string{"select-file": value}); err != nil {
			return err
		}
		if unpause {
			return c.Unpause(ctx, gid)
		}
		return nil
	})
}

// cancelCmd closes the picker. From the add flow it still honours start-now
// (adds all files) so the download is not left in limbo unintentionally.
func (m filesModel) cancelCmd() tea.Cmd {
	a := m.a
	a.overlay = overlayNone
	a.clearPick(m.pickKey)
	if m.fromAdd && m.unpauseAfter {
		ids := m.gids
		if len(ids) == 0 {
			ids = []string{m.gid}
		}
		return a.rpcCmd("added", func(ctx context.Context, c api) error {
			for _, g := range ids {
				if err := c.Unpause(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return nil
}

// mouse handles clicks inside the picker.
func (m filesModel) mouse(id string) (filesModel, tea.Cmd) {
	kind, arg := splitID(id)
	switch kind {
	case "btn":
		if arg == "ok" {
			return m, m.confirmCmd()
		}
		if arg == "cancel" {
			return m, m.cancelCmd()
		}
	case "key":
		return m.update(keyFromToken(arg))
	case "row":
		if i := argInt(arg); i >= 0 && i < len(m.rows) {
			m.cursor = i
			m.clamp()
		}
	case "check":
		if i := argInt(arg); i >= 0 && i < len(m.rows) {
			m.cursor = i
			toggleNode(m.rows[i])
			m.clamp()
		}
	case "tri":
		if i := argInt(arg); i >= 0 && i < len(m.rows) {
			n := m.rows[i]
			if !n.isLeaf() {
				n.collapsed = !n.collapsed
				m.rows = flatten(m.root)
				m.cursor = i
				m.clamp()
			}
		}
	}
	return m, nil
}

func (m filesModel) view() string {
	st := m.a.styles
	title := st.Title.Render("Pick files")
	if m.name != "" {
		title += st.Dim.Render(" — " + m.name)
	}
	if m.moreQueued > 0 {
		title += st.Yellow.Render(fmt.Sprintf("  (+%d more magnet%s)", m.moreQueued, plural(m.moreQueued)))
	}
	if m.root != nil {
		title += "   " + st.Dim.Render(fmt.Sprintf("%d/%d selected · %s",
			selectedLeaves(m.root), leafCount(m.root), FmtBytes(nodeSize(m.root))))
	}

	lines := []string{title, ""}
	treeStart := len(lines) // body-line index where tree rows begin

	var window []*treeNode
	start, end := 0, 0
	switch {
	case m.loading:
		lines = append(lines, st.Dim.Render("loading files…"))
	case m.err != nil:
		lines = append(lines, st.Red.Render("✗ "+m.err.Error()))
	case len(m.rows) == 0:
		lines = append(lines, st.Dim.Render("no files"))
	default:
		start = m.top
		end = m.top + m.maxVisible()
		if end > len(m.rows) {
			end = len(m.rows)
		}
		window = m.rows[start:end]
		for wi, n := range window {
			lines = append(lines, m.rowString(n, start+wi == m.cursor))
		}
		if extra := len(m.rows) - end; extra > 0 {
			lines = append(lines, st.Dim.Render(fmt.Sprintf("  … %d more", extra)))
		}
	}

	fhints := []keyHint{{" ", "space", "toggle"}, {"a", "a", "all"}, {"n", "n", "none"}, {"", "h/l", "fold/unfold"}}
	fhintParts := make([]string, len(fhints))
	for i, h := range fhints {
		fhintParts[i] = st.Key.Render(h.key) + " " + st.Dim.Render(h.label)
	}
	confirmBtn := st.Green.Reverse(true).Bold(true).Padding(0, 2).Render("Confirm ↵")
	cancelBtn := lipgloss.NewStyle().Foreground(st.P.FgDim).Padding(0, 2).Render("Cancel esc")
	lines = append(lines,
		"",
		strings.Join(fhintParts, "  "),
		confirmBtn+"   "+cancelBtn,
	)
	modal := st.Modal.Render(strings.Join(lines, "\n"))

	// Regions. Content sits at offX+3 (border+padding), offY+2 (border+padding).
	offX, offY := m.a.overlayOffset(modal)
	hintY := offY + 2 + len(lines) - 2 // hint line sits above the buttons
	hx := offX + 3
	for i, h := range fhints {
		w := lipgloss.Width(fhintParts[i])
		if h.token != "" {
			m.a.hits.add("key:"+h.token, hx, hintY, hx+w-1, hintY)
		}
		hx += w + 2
	}
	for wi, n := range window {
		gi := start + wi
		y := offY + 2 + treeStart + wi
		row := m.rowString(n, false)
		// Whole row selects; register first so check/tri shadow it.
		m.a.hits.add(fmt.Sprintf("row:%d", gi), offX+3, y, offX+3+lipgloss.Width(row)-1, y)
		triX := offX + 3 + 2*n.depth
		if !n.isLeaf() {
			m.a.hits.add(fmt.Sprintf("tri:%d", gi), triX, y, triX, y)
		}
		checkX := triX + 2
		m.a.hits.add(fmt.Sprintf("check:%d", gi), checkX, y, checkX+2, y)
	}
	btnY := offY + lipgloss.Height(modal) - 3
	x0 := offX + 3
	m.a.hits.add("btn:ok", x0, btnY, x0+lipgloss.Width(confirmBtn)-1, btnY)
	cx := x0 + lipgloss.Width(confirmBtn) + 3
	m.a.hits.add("btn:cancel", cx, btnY, cx+lipgloss.Width(cancelBtn)-1, btnY)
	return modal
}

// rowString renders one tree row. The plain layout is:
// [indent][tri][space][check][space][name][gap][pct][size].
func (m filesModel) rowString(n *treeNode, selected bool) string {
	st := m.a.styles
	indent := strings.Repeat("  ", n.depth)
	tri := " "
	if !n.isLeaf() {
		if n.collapsed {
			tri = "▸"
		} else {
			tri = "▾"
		}
	}
	box := checkGlyph(n)
	var boxStyled string
	switch box {
	case "[x]":
		boxStyled = st.Green.Render(box)
	case "[~]":
		boxStyled = st.Yellow.Render(box)
	default:
		boxStyled = st.Dim.Render(box)
	}
	name := n.name
	if !n.isLeaf() {
		name += "/"
	}
	name = pad(name, 34)
	size := lpad(FmtBytes(nodeSize(n)), 10)
	pct := "    "
	if total := nodeSize(n); total > 0 {
		pct = lpad(fmt.Sprintf("%d%%", int(float64(nodeDone(n))/float64(total)*100)), 4)
	}
	nameStyle := st.Text
	if !n.isLeaf() {
		nameStyle = st.Title
	}
	row := indent + st.Dim.Render(tri) + " " + boxStyled + " " +
		nameStyle.Render(name) + " " + st.Dim.Render(pct) + " " + st.Dim.Render(size)
	if selected {
		row = st.RowSel.Render(row)
	}
	return row
}
