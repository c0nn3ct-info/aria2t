package ui

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// addRecAPI records add-method arguments on top of fakeAPI.
type addRecAPI struct {
	*fakeAPI
	uris        []string
	uriOpts     map[string]string
	torrentB64  string
	torrentOpts map[string]string
	metaB64     string
	metaOpts    map[string]string
}

func (f *addRecAPI) AddURI(_ context.Context, uris []string, opts map[string]string) (string, error) {
	f.uris, f.uriOpts = uris, opts
	return "gid", nil
}

func (f *addRecAPI) AddTorrent(_ context.Context, b64 string, opts map[string]string) (string, error) {
	f.torrentB64, f.torrentOpts = b64, opts
	return "gid", nil
}

func (f *addRecAPI) AddMetalink(_ context.Context, b64 string, opts map[string]string) ([]string, error) {
	f.metaB64, f.metaOpts = b64, opts
	return nil, nil
}

func addTestApp(t *testing.T) (*App, *addRecAPI) {
	t.Helper()
	a, fake := testApp(t)
	rec := &addRecAPI{fakeAPI: fake}
	a.client = rec
	return a, rec
}

func ctrl(k tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: k} }

func TestAddFocusCmdAndApplyFocus(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	if m.focusCmd() == nil {
		t.Fatal("focusCmd must focus the textarea")
	}
	m.focus = 0
	_ = m.applyFocus()
	if !m.uris.Focused() {
		t.Fatal("focus 0 on URL tab must focus uris")
	}
	m.tab = addTabTorrent
	_ = m.applyFocus()
	if !m.file.Focused() || m.uris.Focused() {
		t.Fatal("focus 0 on torrent tab must focus file")
	}
	m.focus = 1
	_ = m.applyFocus()
	if !m.dir.Focused() {
		t.Fatal("focus 1 must focus dir")
	}
	m.focus = 2
	_ = m.applyFocus()
	if !m.split.Focused() {
		t.Fatal("focus 2 must focus split")
	}
	m.focus = 3
	_ = m.applyFocus()
	if !m.out.Focused() {
		t.Fatal("focus 3 must focus out")
	}
}

func TestAddEscCloses(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayAdd
	m := newAddModel(a)
	m, cmd := m.update(key("esc"))
	if a.overlay != overlayNone || cmd != nil {
		t.Fatalf("overlay = %d", a.overlay)
	}
	_ = m
}

func TestAddCtrlTCyclesTabs(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	m.focus = 2
	for i, want := range []int{addTabTorrent, addTabMetalink, addTabURL} {
		var cmd tea.Cmd
		m, cmd = m.update(ctrl(tea.KeyCtrlT))
		if m.tab != want || m.focus != 0 || cmd == nil {
			t.Fatalf("press %d: tab=%d focus=%d", i, m.tab, m.focus)
		}
	}
}

func TestAddTabFocusCycle(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	for _, want := range []int{1, 2, 0} { // limit 3 without rename
		m, _ = m.update(key("tab"))
		if m.focus != want {
			t.Fatalf("focus = %d, want %d", m.focus, want)
		}
	}
	m.rename = true
	for _, want := range []int{1, 2, 3, 0} { // limit 4 with rename
		m, _ = m.update(key("tab"))
		if m.focus != want {
			t.Fatalf("focus = %d, want %d", m.focus, want)
		}
	}
}

func TestAddCtrlSToggle(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	m, _ = m.update(ctrl(tea.KeyCtrlS))
	if m.startNow {
		t.Fatal("ctrl+s must toggle startNow off")
	}
	m, _ = m.update(ctrl(tea.KeyCtrlS))
	if !m.startNow {
		t.Fatal("ctrl+s must toggle startNow back on")
	}
}

func TestAddCtrlRToggleAndFocusReset(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	m, _ = m.update(ctrl(tea.KeyCtrlR))
	if !m.rename {
		t.Fatal("ctrl+r must enable rename")
	}
	m.focus = 3
	m, _ = m.update(ctrl(tea.KeyCtrlR))
	if m.rename || m.focus != 0 {
		t.Fatalf("rename=%v focus=%d", m.rename, m.focus)
	}
}

func TestAddEnterOnURLTabInsertsNewline(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	_ = m.applyFocus()
	m, _ = m.update(key("enter"))
	if !strings.Contains(m.uris.Value(), "\n") {
		t.Fatalf("enter must reach the textarea, value = %q", m.uris.Value())
	}
	if a.status != "" {
		t.Fatalf("enter on textarea must not submit, status = %q", a.status)
	}
}

func TestAddEnterSubmitsFromOtherFocus(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	m.focus = 1
	_, _ = m.update(key("enter"))
	if a.status != "enter at least one URI" || !a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
}

func TestAddCtrlDSubmitsURIs(t *testing.T) {
	a, rec := addTestApp(t)
	a.overlay = overlayAdd
	m := newAddModel(a)
	m.uris.SetValue("http://x\n \nhttp://y")
	m.startNow = false
	_, cmd := m.update(ctrl(tea.KeyCtrlD))
	if a.overlay != overlayNone {
		t.Fatalf("overlay = %d", a.overlay)
	}
	drain(t, a, cmd)
	if len(rec.uris) != 2 || rec.uris[0] != "http://x" || rec.uris[1] != "http://y" {
		t.Fatalf("uris = %v", rec.uris)
	}
	if rec.uriOpts["pause"] != "true" {
		t.Fatalf("opts = %v", rec.uriOpts)
	}
}

func TestAddTypingRoutesToFocusedInput(t *testing.T) {
	a, _ := testApp(t)

	m := newAddModel(a)
	_ = m.applyFocus()
	m, _ = m.update(key("h"))
	if !strings.Contains(m.uris.Value(), "h") {
		t.Fatalf("uris = %q", m.uris.Value())
	}

	m = newAddModel(a)
	m.tab = addTabTorrent
	_ = m.applyFocus()
	m, _ = m.update(key("q"))
	if !strings.Contains(m.file.Value(), "q") {
		t.Fatalf("file = %q", m.file.Value())
	}

	m = newAddModel(a)
	m.focus = 1
	_ = m.applyFocus()
	m, _ = m.update(key("z"))
	if !strings.Contains(m.dir.Value(), "z") {
		t.Fatalf("dir = %q", m.dir.Value())
	}

	m = newAddModel(a)
	m.focus = 2
	_ = m.applyFocus()
	m, _ = m.update(key("9"))
	if !strings.Contains(m.split.Value(), "9") {
		t.Fatalf("split = %q", m.split.Value())
	}

	m = newAddModel(a)
	m.rename = true
	m.focus = 3
	_ = m.applyFocus()
	m, _ = m.update(key("n"))
	if !strings.Contains(m.out.Value(), "n") {
		t.Fatalf("out = %q", m.out.Value())
	}
}

func TestAddOptions(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	a, _ := testApp(t)

	m := newAddModel(a)
	m.dir.SetValue("~/dl")
	m.split.SetValue("8")
	m.rename = true
	m.out.SetValue("x.iso")
	m.startNow = false
	opts := m.options()
	if opts["dir"] != "/home/tester/dl" {
		t.Fatalf("dir = %q", opts["dir"])
	}
	if opts["split"] != "8" || opts["max-connection-per-server"] != "8" {
		t.Fatalf("split opts = %v", opts)
	}
	if opts["out"] != "x.iso" || opts["pause"] != "true" {
		t.Fatalf("opts = %v", opts)
	}

	m = newAddModel(a)
	m.dir.SetValue(" ")
	m.split.SetValue("")
	m.rename = true
	m.out.SetValue(" ")
	if opts := m.options(); len(opts) != 0 {
		t.Fatalf("opts = %v", opts)
	}
}

func TestAddSubmitTorrent(t *testing.T) {
	a, rec := addTestApp(t)
	m := newAddModel(a)
	m.tab = addTabTorrent

	m.file.SetValue("  ")
	_, _ = m.submit()
	if a.status != "enter a file path" || !a.statusErr {
		t.Fatalf("status = %q", a.status)
	}

	m.file.SetValue(filepath.Join(t.TempDir(), "missing.torrent"))
	_, _ = m.submit()
	if !a.statusErr || !strings.Contains(a.status, "missing.torrent") {
		t.Fatalf("status = %q", a.status)
	}

	p := filepath.Join(t.TempDir(), "x.torrent")
	raw := []byte("d8:announce4:teste")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	a.overlay = overlayAdd
	m.file.SetValue(p)
	_, cmd := m.submit()
	if a.overlay != overlayNone {
		t.Fatalf("overlay = %d", a.overlay)
	}
	drain(t, a, cmd)
	if rec.torrentB64 != base64.StdEncoding.EncodeToString(raw) {
		t.Fatalf("torrentB64 = %q", rec.torrentB64)
	}
	if a.status != "added x.torrent" {
		t.Fatalf("status = %q", a.status)
	}
}

func TestAddSubmitMetalink(t *testing.T) {
	a, rec := addTestApp(t)
	m := newAddModel(a)
	m.tab = addTabMetalink
	p := filepath.Join(t.TempDir(), "m.meta4")
	raw := []byte("<metalink/>")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	m.file.SetValue(p)
	_, cmd := m.submit()
	drain(t, a, cmd)
	if rec.metaB64 != base64.StdEncoding.EncodeToString(raw) {
		t.Fatalf("metaB64 = %q", rec.metaB64)
	}
	if rec.torrentB64 != "" {
		t.Fatal("metalink submit must not call AddTorrent")
	}
}

func TestExpandHome(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	if got := expandHome("~/x"); got != "/home/tester/x" {
		t.Fatalf("got %q", got)
	}
	if got := expandHome("/abs/x"); got != "/abs/x" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("HOME", "")
	if got := expandHome("~/x"); got != "~/x" {
		t.Fatalf("got %q", got)
	}
}

func TestAddViewVariants(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	out := m.view()
	for _, want := range []string{"Add download", "URIs — one per line", "[x] Start immediately"} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q", want)
		}
	}
	if strings.Contains(out, "Save as") {
		t.Fatal("rename section must be hidden by default")
	}

	m.tab = addTabTorrent
	m.rename = true
	m.startNow = false
	out = m.view()
	for _, want := range []string{"File path", "Save as", "[ ] Start immediately", "[x] Rename file"} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q", want)
		}
	}
}
