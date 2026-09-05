package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

func loadedFiles(a *App) filesModel {
	m := newFilesModel(a)
	m.gid = "g1"
	m.loading = false
	m.root = buildTree(sampleFiles(), "/d")
	m.rows = flatten(m.root)
	return m
}

func TestFilesLoadCmd(t *testing.T) {
	a, fake := testApp(t)
	fake.status = rpc.Status{Dir: "/d", Files: sampleFiles()}
	m := newFilesModel(a)
	m.gid = "g1"
	msg, ok := m.loadCmd()().(filesDataMsg)
	if !ok || msg.gid != "g1" || len(msg.files) != 3 || msg.dir != "/d" {
		t.Fatalf("msg = %#v", msg)
	}
	a.client = nil
	if m.loadCmd() != nil {
		t.Fatal("nil client must yield nil cmd")
	}
}

func TestFilesRetryCmd(t *testing.T) {
	a, _ := testApp(t)
	m := newFilesModel(a)
	m.gid = "g1"
	if msg, ok := m.retryCmd()().(filesRetryMsg); !ok || msg.gid != "g1" {
		t.Fatalf("retry msg = %#v", msg)
	}
}

func TestFilesAbsorb(t *testing.T) {
	a, _ := testApp(t)

	// Wrong gid → ignored.
	m := newFilesModel(a)
	m.gid = "g1"
	m2, _ := m.absorb(filesDataMsg{gid: "other", files: sampleFiles()})
	if m2.root != nil {
		t.Fatal("mismatched gid must be ignored")
	}

	// Error.
	m2, _ = m.absorb(filesDataMsg{gid: "g1", err: errors.New("boom")})
	if m2.err == nil || m2.loading {
		t.Fatalf("error absorb: %#v", m2)
	}

	// Empty, still retrying.
	m2, cmd := m.absorb(filesDataMsg{gid: "g1"})
	if m2.tries != 1 || cmd == nil {
		t.Fatalf("empty absorb must retry: tries=%d", m2.tries)
	}

	// Empty, exhausted.
	m.tries = 5
	m2, cmd = m.absorb(filesDataMsg{gid: "g1"})
	if m2.loading || cmd != nil {
		t.Fatalf("exhausted absorb: loading=%v cmd=%v", m2.loading, cmd)
	}

	// Files present (detail flow).
	m = newFilesModel(a)
	m.gid = "g1"
	m2, _ = m.absorb(filesDataMsg{gid: "g1", dir: "/d", files: sampleFiles()})
	if m2.root == nil || len(m2.rows) != 4 || m2.loading {
		t.Fatalf("loaded absorb: %#v", m2)
	}
}

func TestFilesAbsorbSingleFileAddFlow(t *testing.T) {
	a, fake := testApp(t)
	a.overlay = overlayFiles
	m := newFilesModel(a)
	m.gid = "g1"
	m.fromAdd = true
	m.unpauseAfter = true
	one := []rpc.File{{Index: "1", Path: "/d/only.iso", Length: "1"}}
	_, cmd := m.absorb(filesDataMsg{gid: "g1", dir: "/d", files: one})
	if a.overlay != overlayNone {
		t.Fatal("single-file add must close the picker")
	}
	drain(t, a, cmd)
	if len(fake.unpaused) != 1 || fake.unpaused[0] != "g1" {
		t.Fatalf("single-file start must unpause: %v", fake.unpaused)
	}

	// Same, but start-paused: no unpause, just a flash.
	a.overlay = overlayFiles
	m.unpauseAfter = false
	_, cmd = m.absorb(filesDataMsg{gid: "g1", dir: "/d", files: one})
	if a.overlay != overlayNone || cmd == nil {
		t.Fatal("single-file paused must close with a flash")
	}
}

func TestFilesUpdateNavigation(t *testing.T) {
	a, _ := testApp(t)
	m := loadedFiles(a)
	m, _ = m.update(key("j"))
	if m.cursor != 1 {
		t.Fatalf("j cursor = %d", m.cursor)
	}
	m, _ = m.update(key("k"))
	if m.cursor != 0 {
		t.Fatalf("k cursor = %d", m.cursor)
	}
	// space toggles a leaf.
	m.cursor = 1 // x.bin, selected
	m, _ = m.update(key(" "))
	if m.rows[1].selected {
		t.Fatal("space must toggle the leaf")
	}
	// a selects all, n selects none.
	m, _ = m.update(key("a"))
	if selectedLeaves(m.root) != 3 {
		t.Fatal("a must select all")
	}
	m, _ = m.update(key("n"))
	if selectedLeaves(m.root) != 0 {
		t.Fatal("n must select none")
	}
	// unknown key inert.
	if m2, cmd := m.update(key("z")); cmd != nil || m2.cursor != m.cursor {
		t.Fatal("unknown key must be inert")
	}
}

func TestFilesUpdateFolding(t *testing.T) {
	a, _ := testApp(t)
	m := loadedFiles(a)
	// Collapse the A directory (cursor 0).
	m, _ = m.update(key("h"))
	if !m.rows[0].collapsed || len(m.rows) != 2 {
		t.Fatalf("h must collapse: rows=%d", len(m.rows))
	}
	// Expand it again.
	m, _ = m.update(key("l"))
	if m.rows[0].collapsed || len(m.rows) != 4 {
		t.Fatalf("l must expand: rows=%d", len(m.rows))
	}
	// h on a leaf jumps to its parent.
	m.cursor = 1 // x.bin under A
	m, _ = m.update(key("h"))
	if m.cursor != 0 {
		t.Fatalf("h on leaf must jump to parent, cursor=%d", m.cursor)
	}
	// current() nil guard: empty model.
	empty := newFilesModel(a)
	if empty.current() != nil {
		t.Fatal("empty current must be nil")
	}
	empty, _ = empty.update(key(" ")) // no panic
	empty, _ = empty.update(key("h"))
}

func TestFilesConfirmCmd(t *testing.T) {
	// Selection present → select-file + unpause.
	a, fake := testApp(t)
	m := loadedFiles(a)
	m.unpauseAfter = true
	a.overlay = overlayFiles
	_, cmd := m.update(key("enter"))
	if a.overlay != overlayNone {
		t.Fatal("confirm must close")
	}
	drain(t, a, cmd)
	if got := fake.changedOptions["g1"]["select-file"]; got != "1,3" {
		t.Fatalf("select-file = %q", got)
	}
	if len(fake.unpaused) != 1 {
		t.Fatalf("confirm must unpause: %v", fake.unpaused)
	}

	// Nothing selected → flash, stays open.
	a2, _ := testApp(t)
	m2 := loadedFiles(a2)
	setSelected(m2.root, false)
	a2.overlay = overlayFiles
	_, cmd = m2.update(key("enter"))
	if a2.overlay != overlayFiles || !a2.statusErr {
		t.Fatalf("empty selection must warn: overlay=%d status=%q", a2.overlay, a2.status)
	}

	// Nothing loaded, add flow → close + unpause.
	a3, fake3 := testApp(t)
	m3 := newFilesModel(a3)
	m3.gid = "g1"
	m3.fromAdd = true
	m3.unpauseAfter = true
	a3.overlay = overlayFiles
	_, cmd = m3.update(key("enter"))
	drain(t, a3, cmd)
	if a3.overlay != overlayNone || len(fake3.unpaused) != 1 {
		t.Fatalf("nil-root confirm must unpause: %v", fake3.unpaused)
	}

	// Nothing loaded, not add flow → just close.
	a4, _ := testApp(t)
	m4 := newFilesModel(a4)
	m4.gid = "g1"
	a4.overlay = overlayFiles
	if _, cmd = m4.update(key("enter")); cmd != nil || a4.overlay != overlayNone {
		t.Fatal("nil-root non-add confirm must just close")
	}
}

func TestFilesCancelCmd(t *testing.T) {
	// Cancel in the add flow removes the provisional paused download.
	a, fake := testApp(t)
	m := loadedFiles(a)
	m.fromAdd = true
	m.unpauseAfter = true
	a.overlay = overlayFiles
	_, cmd := m.update(key("esc"))
	if a.overlay != overlayNone {
		t.Fatal("esc must close")
	}
	drain(t, a, cmd)
	if len(fake.removed) != 1 || len(fake.unpaused) != 0 {
		t.Fatalf("add-flow cancel must remove, not start: removed=%v unpaused=%v", fake.removed, fake.unpaused)
	}

	// Add flow + start-paused has the same cancel semantics.
	a2, fake2 := testApp(t)
	m2 := loadedFiles(a2)
	m2.fromAdd = true
	a2.overlay = overlayFiles
	_, cmd = m2.update(key("esc"))
	drain(t, a2, cmd)
	if len(fake2.unpaused) != 0 || len(fake2.removed) != 1 {
		t.Fatalf("paused cancel must remove without unpausing: removed=%v unpaused=%v", fake2.removed, fake2.unpaused)
	}

	// Detail flow → close, no side effects.
	a3, _ := testApp(t)
	m3 := loadedFiles(a3)
	a3.overlay = overlayFiles
	if _, cmd = m3.update(key("esc")); cmd != nil {
		t.Fatal("detail cancel must have no command")
	}
}

func TestFilesMouse(t *testing.T) {
	a, fake := testApp(t)
	m := loadedFiles(a)
	a.overlay = overlayFiles

	// Row click moves the cursor.
	m, _ = m.mouse("row:2")
	if m.cursor != 2 {
		t.Fatalf("row click cursor = %d", m.cursor)
	}
	// Check click toggles.
	before := m.rows[3].selected
	m, _ = m.mouse("check:3")
	if m.rows[3].selected == before {
		t.Fatal("check click must toggle")
	}
	// Tri click folds a directory.
	m, _ = m.mouse("tri:0")
	if !m.rows[0].collapsed {
		t.Fatal("tri click must collapse the dir")
	}
	// Tri on a leaf is inert (re-expand first).
	m, _ = m.mouse("tri:0")
	m.cursor = 1
	m, _ = m.mouse("tri:1")
	// Out-of-range and foreign ids inert.
	for _, id := range []string{"row:99", "check:-1", "tri:99", "zzz:1"} {
		if _, cmd := m.mouse(id); cmd != nil {
			t.Fatalf("%q must be inert", id)
		}
	}
	// Buttons.
	if _, cmd := m.mouse("btn:esc"); cmd != nil {
		t.Fatal("cancel button yields no cmd")
	}
	a.overlay = overlayFiles
	_, cmd := m.mouse("btn:enter")
	drain(t, a, cmd)
	if fake.changedOptions["g1"] == nil {
		t.Fatal("confirm button must apply selection")
	}
}

func TestFilesView(t *testing.T) {
	a, _ := testApp(t)

	// Loading.
	m := newFilesModel(a)
	m.name = "big.torrent"
	if out := m.view(); !strings.Contains(out, "loading") || !strings.Contains(out, "big.torrent") {
		t.Fatalf("loading view: %q", out)
	}
	// Error.
	m.loading = false
	m.err = errors.New("nope")
	if out := m.view(); !strings.Contains(out, "nope") {
		t.Fatalf("error view: %q", out)
	}
	// Empty (loaded, no rows).
	m.err = nil
	if out := m.view(); !strings.Contains(out, "no files") {
		t.Fatalf("empty view: %q", out)
	}
	// Loaded.
	lm := loadedFiles(a)
	out := lm.view()
	for _, want := range []string{"Pick files", "x.bin", "z.bin", "Confirm", "Cancel", "2/3 selected"} {
		if !strings.Contains(out, want) {
			t.Fatalf("loaded view missing %q:\n%s", want, out)
		}
	}
}

func TestFilesViewMoreQueued(t *testing.T) {
	if plural(1) != "" || plural(2) != "s" {
		t.Fatal("plural")
	}
	a, _ := testApp(t)
	m := loadedFiles(a)
	m.moreQueued = 2
	if out := m.view(); !strings.Contains(out, "+2 more magnets") {
		t.Fatalf("queued-count badge missing:\n%s", out)
	}
	m.moreQueued = 1
	if out := m.view(); !strings.Contains(out, "+1 more magnet)") {
		t.Fatalf("singular badge missing:\n%s", out)
	}
}

func TestFilesViewOverflow(t *testing.T) {
	a, _ := testApp(t)
	a.height = 14 // tight modal forces the tree to scroll
	var many []rpc.File
	for i := 0; i < 40; i++ {
		many = append(many, rpc.File{Index: itoa(i + 1), Path: "/d/f" + itoa(i), Length: "1", Selected: "true"})
	}
	m := newFilesModel(a)
	m.gid = "g1"
	m.loading = false
	m.root = buildTree(many, "/d")
	m.rows = flatten(m.root)
	if out := m.view(); !strings.Contains(out, "more") {
		t.Fatalf("overflowing tree must show a more line:\n%s", out)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

type coErrAPI struct{ *fakeAPI }

func (coErrAPI) ChangeOption(context.Context, string, map[string]string) error {
	return errors.New("option rejected")
}

func TestFilesConfirmChangeOptionError(t *testing.T) {
	a, fake := testApp(t)
	a.client = coErrAPI{fake}
	m := loadedFiles(a)
	a.overlay = overlayFiles
	_, cmd := m.update(key("enter"))
	drain(t, a, cmd)
	if !a.statusErr || !strings.Contains(a.status, "option rejected") {
		t.Fatalf("ChangeOption error must flash: %q", a.status)
	}
}

func TestFilesConfirmDetailNoUnpause(t *testing.T) {
	a, fake := testApp(t)
	m := loadedFiles(a) // detail flow: fromAdd/unpauseAfter false
	a.overlay = overlayFiles
	_, cmd := m.update(key("enter"))
	drain(t, a, cmd)
	if fake.changedOptions["g1"]["select-file"] != "1,3" {
		t.Fatalf("select-file = %q", fake.changedOptions["g1"]["select-file"])
	}
	if len(fake.unpaused) != 0 {
		t.Fatalf("detail confirm must not unpause: %v", fake.unpaused)
	}
}

func TestFilesViewCollapsedDir(t *testing.T) {
	a, _ := testApp(t)
	m := loadedFiles(a)
	m.rows[0].collapsed = true // A directory folded
	m.rows = flatten(m.root)
	if out := m.view(); !strings.Contains(out, "▸") {
		t.Fatalf("collapsed dir must render a ▸ marker:\n%s", out)
	}
}

func TestFilesClampScroll(t *testing.T) {
	a, _ := testApp(t)
	a.height = 14
	m := newFilesModel(a)
	m.gid = "g1"
	m.loading = false
	var many []rpc.File
	for i := 0; i < 40; i++ {
		many = append(many, rpc.File{Index: itoa(i + 1), Path: "/d/f" + itoa(i), Length: "1"})
	}
	m.root = buildTree(many, "/d")
	m.rows = flatten(m.root)
	m.cursor = 30
	m.clamp()
	if m.top == 0 {
		t.Fatal("scrolling down must move the window")
	}
	m.cursor = 0
	m.clamp()
	if m.top != 0 {
		t.Fatalf("scrolling to the top must reset the window, top=%d", m.top)
	}
	// Negative top is clamped up.
	m.top = -1
	m.cursor = 0
	m.clamp()
	if m.top != 0 {
		t.Fatalf("negative top = %d", m.top)
	}
}

func TestFilesMaxVisibleAndClamp(t *testing.T) {
	a, _ := testApp(t)
	m := loadedFiles(a)
	a.height = 8
	if v := m.maxVisible(); v != 3 {
		t.Fatalf("floor = %d", v)
	}
	a.height = 40
	if v := m.maxVisible(); v <= 3 {
		t.Fatalf("tall = %d", v)
	}
	m.cursor = 99
	m.clamp()
	if m.cursor != len(m.rows)-1 {
		t.Fatalf("clamp high = %d", m.cursor)
	}
	m.cursor = -5
	m.clamp()
	if m.cursor != 0 {
		t.Fatalf("clamp low = %d", m.cursor)
	}
}
