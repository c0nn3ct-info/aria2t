package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	addTabURL = iota
	addTabTorrent
	addTabMetalink
	addTabInput
	addTabCount
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

	startNow   bool
	rename     bool
	submitting bool

	focus int // 0 uris/file, 1 dir, 2 split, 3 out (when rename)
}

func newAddModel(a *App) addModel {
	uris := textarea.New()
	uris.Placeholder = "https://…"
	uris.SetHeight(3)
	uris.SetWidth(56)
	file := textinput.New()
	file.Placeholder = "/path/to/file.torrent"
	file.Width = 52 // Width must precede SetValue: overflow windows on set
	// Prefill from the clipboard and open on the matching tab: a URL or
	// magnet lands in the URIs box, a local .torrent/.metalink path in the
	// file box.
	tab := addTabURL
	if clip := strings.TrimSpace(clipboardRead()); clip != "" {
		clip = safeText(clip)
		switch {
		case looksLikeSource(clip):
			uris.SetValue(clip)
		case detectAddTab(clip) != addTabURL:
			tab = detectAddTab(clip)
			file.SetValue(clip)
		}
	}
	dir := textinput.New()
	dir.Width = 22
	dir.SetValue(a.cfg.Dir)
	split := textinput.New()
	split.Width = 6
	split.SetValue(fmt.Sprintf("%d", a.cfg.Split))
	out := textinput.New()
	out.Placeholder = "new-name.iso"
	out.Width = 22
	return addModel{a: a, tab: tab, uris: uris, file: file, dir: dir, split: split, out: out, startNow: true}
}

// detectAddTab guesses the tab that fits a source string: a local
// .torrent/.metalink path opens the matching file tab, everything else
// (URLs, magnets) uses the URL tab.
func detectAddTab(s string) int {
	low := strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasSuffix(low, ".torrent"):
		return addTabTorrent
	case strings.HasSuffix(low, ".metalink"), strings.HasSuffix(low, ".meta4"):
		return addTabMetalink
	default:
		return addTabURL
	}
}

// focusCmd needs a pointer receiver: textinput/textarea Focus mutates the
// model, and focusing a copy leaves the stored overlay deaf to typing.
// applyFocus honours the tab chosen at construction.
func (m *addModel) focusCmd() tea.Cmd { return m.applyFocus() }

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
	if m.submitting {
		return m, nil
	}
	switch key {
	case "esc":
		a.overlay = overlayNone
		return m, nil
	case "ctrl+t":
		m.tab = (m.tab + 1) % addTabCount
		m.focus = 0
		return m, m.applyFocus()
	case "tab":
		limit := 3
		if m.rename {
			limit = 4
		}
		m.focus = (m.focus + 1) % limit
		return m, m.applyFocus()
	case "shift+tab":
		limit := 3
		if m.rename {
			limit = 4
		}
		m.focus = (m.focus + limit - 1) % limit
		return m, m.applyFocus()
	case "ctrl+s":
		m.startNow = !m.startNow
		return m, nil
	case "ctrl+o":
		// Browse the filesystem for a file instead of typing its path.
		if m.tab != addTabURL {
			var exts []string
			switch m.tab {
			case addTabTorrent:
				exts = []string{".torrent"}
			case addTabMetalink:
				exts = []string{".metalink", ".meta4"}
				// addTabInput: nil → show every file
			}
			start := expandHome(strings.TrimSpace(m.dir.Value()))
			a.browse = newBrowseModelAsync(a, start, exts)
			a.overlay = overlayBrowse
			return m, a.browse.loadCmd()
		}
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
		m.uris, cmd = updateTextArea(m.uris, msg)
	case m.focus == 0:
		m.file, cmd = updateInput(m.file, msg)
	case m.focus == 1:
		m.dir, cmd = updateInput(m.dir, msg)
	case m.focus == 2:
		m.split, cmd = updateInput(m.split, msg)
	default:
		m.out, cmd = updateInput(m.out, msg)
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
		// Reject non-URI input (a file path, an aria2 .aria2 control file,
		// pasted junk) with a clear message instead of silently queueing a
		// download that aria2 immediately fails.
		for _, u := range uris {
			if !looksLikeSource(u) {
				return m, a.flash("not a link — use http/https/ftp/sftp or a magnet (for local files use the Torrent/Metalink tabs)", true)
			}
		}
		m.submitting = true
		if len(uris) == 1 && strings.HasPrefix(uris[0], "magnet:") {
			return m, a.addURICmd(uris, opts, m.startNow)
		}
		c := a.client
		if c == nil {
			m.submitting = false
			return m, a.flash("not connected", true)
		}
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := c.AddURI(ctx, uris, opts)
			return addBatchDoneMsg{text: "added", err: err}
		}
	case addTabInput:
		path := expandHome(strings.TrimSpace(m.file.Value()))
		if path == "" {
			return m, a.flash("enter a file path", true)
		}
		base, name := opts, filepath.Base(path)
		c := a.client
		if c == nil {
			return m, a.flash("not connected", true)
		}
		m.submitting = true
		return m, func() tea.Msg {
			raw, err := os.ReadFile(path)
			if err != nil {
				return addBatchDoneMsg{err: err}
			}
			if bytes.IndexByte(raw, 0) >= 0 {
				return addBatchDoneMsg{err: fmt.Errorf("that's a binary file, not an aria2 input list (a .torrent goes on the Torrent tab)")}
			}
			var entries []inputEntry
			for _, e := range parseInputFile(string(raw)) {
				var us []string
				for _, u := range e.uris {
					if looksLikeSource(u) {
						us = append(us, u)
					}
				}
				if len(us) > 0 {
					e.uris = us
					entries = append(entries, e)
				}
			}
			if len(entries) == 0 {
				return addBatchDoneMsg{err: fmt.Errorf("no valid links found in the file")}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			for _, e := range entries {
				if _, err := c.AddURI(ctx, e.uris, mergeOpts(base, e.opts)); err != nil {
					return addBatchDoneMsg{err: err}
				}
			}
			return addBatchDoneMsg{text: fmt.Sprintf("added %d from %s", len(entries), safeText(name))}
		}
	default:
		path := expandHome(strings.TrimSpace(m.file.Value()))
		if path == "" {
			return m, a.flash("enter a file path", true)
		}
		if m.tab == addTabTorrent {
			// Add the torrent paused so the tree picker can show its files
			// before the download starts; unpause after selection if the user
			// wanted it started immediately.
			opts["pause"] = "true"
			unpause := m.startNow
			c := a.client
			if c == nil {
				return m, a.flash("not connected", true)
			}
			m.submitting = true
			return m, func() tea.Msg {
				raw, err := os.ReadFile(path)
				if err != nil {
					return torrentAddedMsg{err: err}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				gid, err := c.AddTorrent(ctx, base64.StdEncoding.EncodeToString(raw), opts)
				return torrentAddedMsg{gid: gid, unpause: unpause, err: err}
			}
		}
		// Metalink: add paused so multi-file metalinks can be pruned before
		// they start downloading.
		opts["pause"] = "true"
		unpause := m.startNow
		name := filepath.Base(path)
		c := a.client
		if c == nil {
			return m, a.flash("not connected", true)
		}
		m.submitting = true
		return m, func() tea.Msg {
			raw, err := os.ReadFile(path)
			if err != nil {
				return metalinkAddedMsg{err: err}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			gids, err := c.AddMetalink(ctx, base64.StdEncoding.EncodeToString(raw), opts)
			return metalinkAddedMsg{gids: gids, name: name, unpause: unpause, err: err}
		}
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

// mouse handles clicks inside the add overlay: tab chips switch tabs, footer
// hints trigger their keys (switch/browse/start/rename/add/close).
func (m addModel) mouse(id string) (addModel, tea.Cmd) {
	kind, arg := splitID(id)
	switch kind {
	case "atab":
		if t := argInt(arg); t >= 0 && t < addTabCount && t != m.tab {
			m.tab = t
			m.focus = 0
			return m, m.applyFocus()
		}
	case "key":
		return m.update(keyFromToken(arg))
	case "btn":
		return m.update(dispatchBtn(arg))
	}
	return m, nil
}

func (m addModel) view() string {
	st := m.a.styles
	names := []string{"URL", "Torrent", "Metalink", "Input file"}
	tabs := make([]string, len(names))
	for i, n := range names {
		tabs[i] = m.a.tab(n, i == m.tab)
	}
	var src string
	label := "File path"
	if m.tab == addTabInput {
		label = "aria2 input file (batch)"
	}
	if m.tab == addTabURL {
		src = st.Dim.Render("URIs — one per line (mirrors)") + "\n" + m.uris.View()
	} else {
		src = st.Dim.Render(label) + "  " + st.Key.Render("^o") + st.Dim.Render(" browse") + "\n" + m.file.View()
	}
	check := func(on bool, label string) string {
		ls := st.Dim
		if on {
			ls = st.Text
		}
		return m.a.checkbox(on) + " " + ls.Render(label)
	}
	navHints := []keyHint{
		{"^t", "^t", "switch"}, {"^o", "^o", "browse"}, {"^s", "^s", "start"}, {"^r", "^r", "rename"},
	}
	navParts := make([]string, len(navHints))
	for i, h := range navHints {
		navParts[i] = st.Key.Render(h.key) + " " + st.Dim.Render(h.label)
	}
	buttons := []button{{"esc", "Cancel", "esc", btnRed}, {"^d", "Add", "^d", btnGreen}}
	body := lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Add download"),
		"",
		strings.Join(tabs, " "),
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
		strings.Join(navParts, "  "),
		m.a.buttonRow(buttons),
	)
	if m.submitting {
		body = lipgloss.JoinVertical(lipgloss.Left, body, st.Yellow.Render("Working: reading and adding…"))
	}
	modal := m.a.modalCard(false).Render(body)

	// Clickable regions: source tabs, the nav-hint line, and the buttons.
	offX, offY := m.a.overlayOffset(modal)
	x := offX + 3 // border + horizontal padding
	tabsY := offY + 4
	for i, tab := range tabs {
		w := lipgloss.Width(tab)
		m.a.hits.add(fmt.Sprintf("atab:%d", i), x, tabsY, x+w-1, tabsY)
		x += w + 1
	}
	navY := offY + lipgloss.Height(modal) - 4
	hx := offX + 3
	for i, h := range navHints {
		w := lipgloss.Width(navParts[i])
		m.a.hits.add("key:"+h.token, hx, navY, hx+w-1, navY)
		hx += w + 2
	}
	m.a.registerButtons(offX, offY, modal, buttons)
	// On the file tabs, make the inline "^o browse" next to the path and the
	// path field itself clickable, so a mouse user opens the picker without the
	// keyboard (the footer hint alone is easy to miss). The header above src —
	// title, blank, tabs, blank — is a fixed four lines, so src's label sits at
	// offY+6 and its input at offY+7.
	if m.tab != addTabURL {
		bx := offX + 3 + lipgloss.Width(label) + 2
		m.a.hits.add("key:^o", bx, offY+6, bx+lipgloss.Width("^o browse")-1, offY+6)
		inW := lipgloss.Width(m.file.View())
		m.a.hits.add("key:^o", offX+3, offY+7, offX+3+inW-1, offY+7)
	}
	return modal
}
