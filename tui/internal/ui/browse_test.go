package ui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func withReadDir(t *testing.T, fn func(string) ([]os.DirEntry, error)) {
	t.Helper()
	orig := browseReadDir
	browseReadDir = fn
	t.Cleanup(func() { browseReadDir = orig })
}

func sampleDir() []os.DirEntry {
	return []os.DirEntry{
		fakeDirEntry{"sub", true},
		fakeDirEntry{"a.torrent", false},
		fakeDirEntry{"b.metalink", false},
		fakeDirEntry{"c.txt", false},
		fakeDirEntry{".hidden", false},
		fakeDirEntry{"zdir", true},
	}
}

func TestBrowseLoadFilterSort(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	m := newBrowseModel(a, "/x", []string{".torrent"})
	names := make([]string, len(m.entries))
	for i, e := range m.entries {
		names[i] = e.name
	}
	if strings.Join(names, ",") != "..,sub,zdir,a.torrent" {
		t.Fatalf("entries = %v", names)
	}
}

func TestBrowseMatchesAll(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	m := newBrowseModel(a, "/x", nil) // nil exts → every file shows
	files := 0
	for _, e := range m.entries {
		if !e.isDir {
			files++
		}
	}
	if files != 3 { // a.torrent, b.metalink, c.txt
		t.Fatalf("files shown = %d", files)
	}
}

func TestBrowseMetalinkExts(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{fakeDirEntry{"x.meta4", false}, fakeDirEntry{"y.torrent", false}}, nil
	})
	m := newBrowseModel(a, "/x", []string{".metalink", ".meta4"})
	// .meta4 matches, .torrent does not.
	got := 0
	for _, e := range m.entries {
		if e.name == "x.meta4" {
			got++
		}
		if e.name == "y.torrent" {
			t.Fatal(".torrent must be filtered out")
		}
	}
	if got != 1 {
		t.Fatal(".meta4 must be shown")
	}
}

func TestBrowseRootHasNoDotDot(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	m := newBrowseModel(a, "/", nil)
	if len(m.entries) > 0 && m.entries[0].name == ".." {
		t.Fatal("root must not offer ..")
	}
}

func TestBrowseReadError(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return nil, errors.New("permission denied") })
	m := newBrowseModel(a, "/x", nil)
	if m.err == nil {
		t.Fatal("read error must be recorded")
	}
	if out := m.view(); !strings.Contains(out, "permission denied") {
		t.Fatalf("error view: %q", out)
	}
}

func TestBrowseHomeFallback(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return nil, nil })
	orig := browseHome
	browseHome = func() (string, error) { return "/home/tester", nil }
	t.Cleanup(func() { browseHome = orig })
	m := newBrowseModel(a, "", nil)
	if m.dir != "/home/tester" {
		t.Fatalf("dir = %q", m.dir)
	}
	// Home lookup failure → root.
	browseHome = func() (string, error) { return "", errors.New("no home") }
	m = newBrowseModel(a, "", nil)
	if m.dir != "/" {
		t.Fatalf("dir = %q", m.dir)
	}
}

func TestBrowseNavigation(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	m := newBrowseModel(a, "/x", []string{".torrent"}) // .., sub, zdir, a.torrent
	m, _ = m.update(key("j"))
	if m.cursor != 1 {
		t.Fatalf("j cursor = %d", m.cursor)
	}
	// enter on a directory descends.
	m, _ = m.update(key("enter"))
	if m.dir != "/x/sub" {
		t.Fatalf("cd dir = %q", m.dir)
	}
	// h goes back up.
	m, _ = m.update(key("h"))
	if m.dir != "/x" {
		t.Fatalf("up dir = %q", m.dir)
	}
	m, _ = m.update(key("k"))
	if m.cursor != 0 {
		t.Fatalf("k cursor = %d", m.cursor)
	}
	// unknown key inert.
	if m2, cmd := m.update(key("z")); cmd != nil || m2.cursor != m.cursor {
		t.Fatal("unknown key must be inert")
	}
}

func TestBrowseChooseFileFillsAdd(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	a.add = newAddModel(a)
	a.add.tab = addTabTorrent
	a.overlay = overlayBrowse
	m := newBrowseModel(a, "/x", []string{".torrent"})
	m.cursor = 3 // a.torrent
	m, cmd := m.update(key("enter"))
	if a.add.file.Value() != "/x/a.torrent" {
		t.Fatalf("add file = %q", a.add.file.Value())
	}
	if a.overlay != overlayAdd || cmd == nil {
		t.Fatalf("choose must return to add: overlay=%d", a.overlay)
	}
	_ = m
}

func TestBrowseEscReturnsToAdd(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	a.overlay = overlayBrowse
	m := newBrowseModel(a, "/x", nil)
	_, _ = m.update(key("esc"))
	if a.overlay != overlayAdd {
		t.Fatalf("esc overlay = %d", a.overlay)
	}
	// At root, h is a no-op.
	mr := newBrowseModel(a, "/", nil)
	before := mr.dir
	mr, _ = mr.update(key("h"))
	if mr.dir != before {
		t.Fatal("h at root must not move")
	}
}

func TestBrowseMouse(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	a.add = newAddModel(a)
	a.overlay = overlayBrowse
	// Single click on a directory (zdir at index 2) enters it.
	m := newBrowseModel(a, "/x", []string{".torrent"})
	m, _ = m.mouse("row:2")
	if m.dir != "/x/zdir" {
		t.Fatalf("dir click must enter: %q", m.dir)
	}
	// Single click on a file (a.torrent at index 3) chooses it.
	m2 := newBrowseModel(a, "/x", []string{".torrent"})
	a.overlay = overlayBrowse
	m2, cmd := m2.mouse("row:3")
	if a.overlay != overlayAdd || a.add.file.Value() != "/x/a.torrent" || cmd == nil {
		t.Fatalf("file click must choose: overlay=%d file=%q", a.overlay, a.add.file.Value())
	}
	_ = m2
	// Cancel hint returns to the add overlay; out-of-range/foreign inert.
	a.overlay = overlayBrowse
	m3 := newBrowseModel(a, "/x", nil)
	if _, _ = m3.mouse("key:esc"); a.overlay != overlayAdd {
		t.Fatal("cancel hint must return to add")
	}
	for _, id := range []string{"row:99", "row:-1", "zzz:1"} {
		if _, c := m3.mouse(id); c != nil {
			t.Fatalf("%q must be inert", id)
		}
	}
}

func TestBrowseViewAndClamp(t *testing.T) {
	a, _ := testApp(t)
	// Many entries + tight height → scroll + overflow line.
	big := make([]os.DirEntry, 0, 40)
	for i := 0; i < 40; i++ {
		big = append(big, fakeDirEntry{"d" + itoa(i), true})
	}
	withReadDir(t, func(string) ([]os.DirEntry, error) { return big, nil })
	a.height = 14
	m := newBrowseModel(a, "/x", nil)
	if out := m.view(); !strings.Contains(out, "Choose a file") || !strings.Contains(out, "more") {
		t.Fatalf("view: %q", out)
	}
	m.cursor = 30
	m.clamp()
	if m.top == 0 {
		t.Fatal("scroll must move the window")
	}
	m.cursor = 0
	m.clamp()
	if m.top != 0 {
		t.Fatalf("scroll to top must reset window, top=%d", m.top)
	}
	m.top = -1
	m.cursor = 0
	m.clamp()
	if m.top != 0 {
		t.Fatalf("negative top = %d", m.top)
	}
	m.cursor = 999
	m.clamp()
	if m.cursor != len(m.entries)-1 {
		t.Fatalf("clamp high = %d", m.cursor)
	}
	m.cursor = -5
	m.clamp()
	if m.cursor != 0 {
		t.Fatalf("clamp low = %d", m.cursor)
	}
	// Empty directory.
	withReadDir(t, func(string) ([]os.DirEntry, error) { return nil, nil })
	me := newBrowseModel(a, "/", nil)
	if out := me.view(); !strings.Contains(out, "nothing to choose") {
		t.Fatalf("empty view: %q", out)
	}
	a.height = 6
	if v := m.maxVisible(); v != 3 {
		t.Fatalf("maxVisible floor = %d", v)
	}
}

func TestBrowseRegionsMatchRows(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	a.browse = newBrowseModel(a, "/x", []string{".torrent"})
	a.overlay = overlayBrowse
	if got := regionText(t, a, "row:1"); !strings.Contains(got, "sub") {
		t.Fatalf("row:1 covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "row:3"); !strings.Contains(got, "a.torrent") {
		t.Fatalf("row:3 covers %q", strings.TrimSpace(got))
	}
}

func TestAddCtrlOMetalinkExts(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	m := newAddModel(a)
	m.tab = addTabMetalink
	_, _ = m.update(ctrl(tea.KeyCtrlO))
	if a.overlay != overlayBrowse {
		t.Fatalf("overlay = %d", a.overlay)
	}
	if len(a.browse.exts) != 2 || a.browse.exts[0] != ".metalink" {
		t.Fatalf("metalink exts = %v", a.browse.exts)
	}
}

func TestAddCtrlOOpensBrowse(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	_, _ = a.Update(key("a")) // add overlay, URL tab
	_, cmd := a.Update(ctrl(tea.KeyCtrlO))
	if a.overlay != overlayAdd {
		t.Fatal("ctrl+o on the URL tab must be a no-op")
	}
	a.add.tab = addTabTorrent
	_, cmd = a.Update(ctrl(tea.KeyCtrlO))
	if a.overlay != overlayBrowse {
		t.Fatalf("ctrl+o on the torrent tab must open the browser, overlay=%d", a.overlay)
	}
	drain(t, a, cmd)
	// Routed through the app: key + mouse + view + wheel.
	if !a.wheelNavigates() {
		t.Fatal("browser must wheel-scroll")
	}
	_ = a.View()
	_, _ = a.Update(key("j"))
	click(t, a, "row:0")
}
