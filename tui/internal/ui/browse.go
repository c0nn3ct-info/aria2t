package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// browseReadDir and browseHome are the filesystem seams, swapped in tests.
var (
	browseReadDir = os.ReadDir
	browseHome    = os.UserHomeDir
)

// browseEntry is one row in the file browser.
type browseEntry struct {
	name  string
	isDir bool
}

// browseModel is the file browser overlay: it navigates the filesystem and
// hands a chosen aria2 file (.torrent / .metalink) back to the add overlay.
type browseModel struct {
	a           *App
	dir         string
	exts        []string // shown file extensions (dirs always shown); nil = all
	entries     []browseEntry
	cursor, top int
	err         error
}

func newBrowseModel(a *App, dir string, exts []string) browseModel {
	if dir == "" {
		if h, err := browseHome(); err == nil {
			dir = h
		} else {
			dir = "/"
		}
	}
	m := browseModel{a: a, dir: dir, exts: exts}
	m.load()
	return m
}

// load reads the current directory into a dirs-first, name-sorted list.
func (m *browseModel) load() {
	m.cursor, m.top = 0, 0
	m.entries = nil
	ents, err := browseReadDir(m.dir)
	if err != nil {
		m.err = err
		return
	}
	m.err = nil
	if m.dir != "/" {
		m.entries = append(m.entries, browseEntry{name: "..", isDir: true})
	}
	var dirs, files []browseEntry
	for _, e := range ents {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip dotfiles
		}
		if e.IsDir() {
			dirs = append(dirs, browseEntry{name: name, isDir: true})
		} else if m.matches(name) {
			files = append(files, browseEntry{name: name})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	m.entries = append(m.entries, dirs...)
	m.entries = append(m.entries, files...)
}

// matches reports whether a filename passes the extension filter.
func (m browseModel) matches(name string) bool {
	if len(m.exts) == 0 {
		return true
	}
	low := strings.ToLower(name)
	for _, e := range m.exts {
		if strings.HasSuffix(low, e) {
			return true
		}
	}
	return false
}

func (m browseModel) maxVisible() int {
	v := m.a.height - 10
	if v < 3 {
		v = 3
	}
	return v
}

func (m *browseModel) clamp() {
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
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

// cd changes directory (sub may be ".." to go up) and reloads.
func (m *browseModel) cd(sub string) {
	if sub == ".." {
		m.dir = filepath.Dir(m.dir)
	} else {
		m.dir = filepath.Join(m.dir, sub)
	}
	m.load()
}

// choose hands the selected file back to the add overlay and closes.
func (m browseModel) choose(name string) tea.Cmd {
	a := m.a
	a.add.file.SetValue(filepath.Join(m.dir, name))
	a.add.focus = 0
	a.overlay = overlayAdd
	return a.add.applyFocus()
}

func (m browseModel) update(msg tea.KeyMsg) (browseModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.a.overlay = overlayAdd
		return m, nil
	case "j", "down":
		m.cursor++
		m.clamp()
	case "k", "up":
		m.cursor--
		m.clamp()
	case "h", "left", "backspace":
		if m.dir != "/" {
			m.cd("..")
		}
	case "enter", "l", "right":
		if m.cursor >= 0 && m.cursor < len(m.entries) {
			e := m.entries[m.cursor]
			if e.isDir {
				m.cd(e.name)
				return m, nil
			}
			return m, m.choose(e.name)
		}
	}
	return m, nil
}

// mouse handles clicks: a single click activates a row (enter a directory or
// choose a file); the cancel hint returns to the add overlay.
func (m browseModel) mouse(id string) (browseModel, tea.Cmd) {
	kind, arg := splitID(id)
	switch kind {
	case "key":
		return m.update(keyFromToken(arg))
	case "btn":
		return m.update(dispatchBtn(arg))
	case "row":
		i := argInt(arg)
		if i < 0 || i >= len(m.entries) {
			return m, nil
		}
		m.cursor = i
		m.clamp()
		return m.update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	return m, nil
}

func (m browseModel) view() string {
	st := m.a.styles
	title := st.Title.Render("Choose a file") + "   " + st.Dim.Render(trunc(m.dir, 60))
	lines := []string{title, ""}
	rowStart := len(lines)

	var window []browseEntry
	start := 0
	if m.err != nil {
		lines = append(lines, st.Red.Render("✗ "+m.err.Error()))
	} else if len(m.entries) == 0 {
		lines = append(lines, st.Dim.Render("empty — nothing to choose here"))
	} else {
		start = m.top
		end := m.top + m.maxVisible()
		if end > len(m.entries) {
			end = len(m.entries)
		}
		window = m.entries[start:end]
		for wi, e := range window {
			icon, name := "  ", e.name
			if e.isDir {
				icon, name = st.Brand.Render("▸ "), e.name+"/"
			}
			style := st.Text
			if e.isDir {
				style = st.Title
			}
			row := icon + style.Render(name)
			if start+wi == m.cursor {
				row = st.RowSel.Render(row)
			}
			lines = append(lines, row)
		}
		if extra := len(m.entries) - end; extra > 0 {
			lines = append(lines, st.Dim.Render(fmt.Sprintf("… %d more", extra)))
		}
	}
	navHints := []keyHint{{"h", "h", "up"}}
	navParts := make([]string, len(navHints))
	for i, h := range navHints {
		navParts[i] = st.Key.Render(h.key) + " " + st.Dim.Render(h.label)
	}
	navPrefix := st.Dim.Render("↑↓ move   ")
	buttons := []button{{"esc", "Cancel", "esc", btnRed}, {"enter", "Open", "↵", btnGreen}}
	lines = append(lines, "", navPrefix+strings.Join(navParts, "  "), m.a.buttonRow(buttons))
	modal := m.a.modalCard(false).Render(strings.Join(lines, "\n"))

	offX, offY := m.a.overlayOffset(modal)
	for wi := range window {
		y := offY + 2 + rowStart + wi
		m.a.hits.add(fmt.Sprintf("row:%d", start+wi), offX+3, y, offX+lipgloss.Width(modal)-4, y)
	}
	// Nav-hint line (h up clickable) sits one line above the buttons.
	navY := offY + 2 + len(lines) - 2
	hx := offX + 3 + lipgloss.Width(navPrefix)
	for i, h := range navHints {
		w := lipgloss.Width(navParts[i])
		m.a.hits.add("key:"+h.token, hx, navY, hx+w-1, navY)
		hx += w + 2
	}
	m.a.registerButtons(offX, offY, modal, buttons)
	return modal
}
