package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/rpc"
)

func TestListQuitKey(t *testing.T) {
	// With downloads in flight, q asks first (managed daemon pauses them).
	a, _ := testApp(t)
	_, cmd := a.Update(key("q"))
	if a.overlay != overlayConfirm || cmd != nil {
		t.Fatalf("q with active downloads must confirm, overlay=%d", a.overlay)
	}
	_, cmd = a.Update(key("y"))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("confirm-yes must quit")
	}
	// With nothing running, q quits immediately.
	a, _ = testApp(t)
	a.snap = snapshot{}
	_, cmd = a.Update(key("q"))
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q with no active downloads must quit directly")
	}
}

func TestListNavigationKeys(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("3"))
	if a.list.tab != tabWaiting {
		t.Fatalf("tab = %d", a.list.tab)
	}
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
	if a.list.cursor != 1 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyUp})
	if a.list.cursor != 0 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	_, _ = a.Update(key("2"))
	if a.list.tab != tabActive {
		t.Fatalf("tab = %d", a.list.tab)
	}
	_, _ = a.Update(key("1"))
	if a.list.tab != tabAll {
		t.Fatalf("tab = %d", a.list.tab)
	}
}

func TestListScreenAndOverlayKeys(t *testing.T) {
	a, _ := testApp(t)
	if _, cmd := a.Update(key("a")); a.overlay != overlayAdd || cmd == nil {
		t.Fatalf("a: overlay=%d", a.overlay)
	}
	a.overlay = overlayNone
	_, _ = a.Update(key("g"))
	if a.screen != screenStats {
		t.Fatalf("g: screen=%d", a.screen)
	}
	a.screen = screenList
	if _, cmd := a.Update(key(",")); a.screen != screenSettings || cmd == nil {
		t.Fatalf(",: screen=%d", a.screen)
	}
	a.screen = screenList
	_, _ = a.Update(key("s"))
	if a.overlay != overlayServers {
		t.Fatalf("s: overlay=%d", a.overlay)
	}
	a.overlay = overlayNone
	_, _ = a.Update(key("S"))
	if a.screen != screenScheduler {
		t.Fatalf("S: screen=%d", a.screen)
	}
}

func TestThemeToggleViaSettings(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m.fields[4][0].on = true // Light theme
	_, cmd := m.save()
	drain(t, a, cmd)
	if a.cfg.Theme != "light" || a.styles.P.Name != "light" {
		t.Fatalf("settings must switch to light, got %s", a.cfg.Theme)
	}
	m2 := newSettingsModel(a)
	m2.fields[4][0].on = false
	_, cmd = m2.save()
	drain(t, a, cmd)
	if a.cfg.Theme != "dark" {
		t.Fatalf("settings must switch back to dark, got %s", a.cfg.Theme)
	}
}

func TestListResumeAndRemove(t *testing.T) {
	a, fake := testApp(t)
	_, cmd := a.Update(key("r")) // resume a1
	drain(t, a, cmd)
	_, _ = a.Update(key("d")) // remove a1 → confirm modal
	if a.overlay != overlayConfirm {
		t.Fatalf("d must ask first, overlay = %d", a.overlay)
	}
	_, cmd = a.Update(key("y"))
	drain(t, a, cmd)
	if len(fake.removed) != 1 || fake.removed[0] != "a1" {
		t.Fatalf("removed = %v", fake.removed)
	}
	// Active removal also purges the result so --force-save can't resurrect it.
	if len(fake.removedResults) != 1 || fake.removedResults[0] != "a1" {
		t.Fatalf("active remove must purge the result, removedResults = %v", fake.removedResults)
	}
	_, _ = a.Update(key("4")) // stopped tab → RemoveDownloadResult path
	_, _ = a.Update(key("d"))
	_, cmd = a.Update(key("y"))
	drain(t, a, cmd)
	if len(fake.removed) != 1 {
		t.Fatalf("stopped remove must use RemoveDownloadResult, removed = %v", fake.removed)
	}
	// n declines: nothing removed.
	_, _ = a.Update(key("2"))
	_, _ = a.Update(key("d"))
	_, cmd = a.Update(key("n"))
	drain(t, a, cmd)
	if len(fake.removed) != 1 || a.overlay != overlayNone {
		t.Fatalf("decline must not remove: %v", fake.removed)
	}
}

// TestRemoveActiveErrorSkipsPurge: when aria2.remove fails, the follow-up purge
// is skipped and the error surfaces (covers the early-return arm).
// TestSeedingStatusWord: a completed-but-uploading torrent must read "seeding",
// not "active"; a plain active download still reads "active".
func TestSeedingStatusWord(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 120, 20
	a.snap.Active = []rpc.Status{
		{GID: "t1", Status: "active", Seeder: "true", InfoHash: "abc",
			TotalLength: "100", CompletedLength: "100", Files: []rpc.File{{Path: "/dl/torrent.iso"}}},
		{GID: "a2", Status: "active", TotalLength: "100", CompletedLength: "40",
			Files: []rpc.File{{Path: "/dl/plain.iso"}}},
	}
	a.list.tab = tabActive
	v := a.list.view()
	if !strings.Contains(v, "seeding") {
		t.Fatal("a seeding torrent must show STATUS 'seeding'")
	}
	if !strings.Contains(v, "active") {
		t.Fatal("a plain active download must still show 'active'")
	}
}

func TestRemoveActiveErrorSkipsPurge(t *testing.T) {
	a, fake := testApp(t)
	fake.removeErr = errors.New("boom")
	_, _ = a.Update(key("d"))
	_, cmd := a.Update(key("y"))
	drain(t, a, cmd)
	if len(fake.removed) != 1 {
		t.Fatalf("remove must be attempted, removed = %v", fake.removed)
	}
	if len(fake.removedResults) != 0 {
		t.Fatalf("purge must be skipped when remove errors, removedResults = %v", fake.removedResults)
	}
}

func TestListThrottleKey(t *testing.T) {
	a, _ := testApp(t)
	_, cmd := a.Update(key("l"))
	if a.overlay != overlayThrottle || a.throttle.gid != "a1" || cmd == nil {
		t.Fatalf("overlay=%d gid=%q", a.overlay, a.throttle.gid)
	}
}

func TestListTorrentKey(t *testing.T) {
	a, _ := testApp(t)
	_, cmd := a.Update(key("t")) // a1 is not a torrent
	if cmd == nil || a.status != "not a torrent" {
		t.Fatalf("status = %q", a.status)
	}
	a.snap.Active[0].InfoHash = "deadbeef"
	_, cmd = a.Update(key("t"))
	if a.screen != screenSeeding || a.seeding.gid != "a1" || cmd == nil {
		t.Fatalf("screen=%d gid=%q", a.screen, a.seeding.gid)
	}
}

func TestListChecksumPromptFlow(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("4")) // stopped tab, cursor on s1
	_, cmd := a.Update(key("c"))
	if a.overlay != overlayPrompt || cmd == nil {
		t.Fatalf("overlay = %d", a.overlay)
	}
	if a.prompt.onSubmit == nil {
		t.Fatal("prompt must have onSubmit")
	}
	if c := a.prompt.onSubmit("  abcd  "); c == nil {
		t.Fatal("onSubmit must flash")
	}
	v := a.verify["s1"]
	if v == nil || v.Expected != "abcd" || v.Finished {
		t.Fatalf("verify state = %+v", v)
	}
}

func TestListVerifyKeyNoChecksum(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("4"))
	_, cmd := a.Update(key("v"))
	if cmd == nil || !strings.Contains(a.status, "no expected checksum") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestListRedownloadKeyNoURIs(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("4"))
	_, cmd := a.Update(key("R"))
	if cmd == nil || !strings.Contains(a.status, "no source URIs") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestListOpenDirKeyEmpty(t *testing.T) {
	a, _ := testApp(t)
	_, cmd := a.Update(key("o")) // a1 has no Dir
	if cmd == nil || a.status != "no directory" {
		t.Fatalf("status = %q", a.status)
	}
}

func TestReorderMoveGuards(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("3"))
	_, _ = a.Update(key("K")) // grab w1 at top; move(-1) hits the to<0 guard
	if !a.list.reordering || a.list.cursor != 0 {
		t.Fatalf("reordering=%v cursor=%d", a.list.reordering, a.list.cursor)
	}
	_, _ = a.Update(key("g"))
	if !a.list.pendingG {
		t.Fatal("first g must set pendingG")
	}
	_, _ = a.Update(key("g")) // gg at top: move(0) hits the cursor==to guard
	if a.list.cursor != 0 || a.list.pendingG {
		t.Fatalf("cursor=%d pendingG=%v", a.list.cursor, a.list.pendingG)
	}
	_, _ = a.Update(key("G")) // to bottom
	if a.list.cursor != 2 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	_, _ = a.Update(key("J")) // move(3) hits the to>=len guard
	if a.list.cursor != 2 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	_, _ = a.Update(key("g"))
	_, _ = a.Update(key("K")) // K resets pendingG and moves up
	if a.list.pendingG || a.list.cursor != 1 {
		t.Fatalf("pendingG=%v cursor=%d", a.list.pendingG, a.list.cursor)
	}
	_, _ = a.Update(key("esc"))
	if a.list.reordering {
		t.Fatal("esc must exit reorder mode")
	}
}

func TestFrozenWaitingDuringPoll(t *testing.T) {
	a, _ := testApp(t)
	a.list.reordering = true
	a.list.localOrder = []rpc.Status{{GID: "local"}}
	_, _ = a.Update(pollMsg{snap: snapshot{Waiting: []rpc.Status{{GID: "fresh"}}}})
	if len(a.snap.Waiting) != 1 || a.snap.Waiting[0].GID != "local" {
		t.Fatalf("waiting = %+v", a.snap.Waiting)
	}
}

func TestSelectedOutOfRange(t *testing.T) {
	a, _ := testApp(t)
	a.snap = snapshot{}
	if _, ok := a.list.selected(); ok {
		t.Fatal("empty rows must not select")
	}
}

func TestClampCursorBothBounds(t *testing.T) {
	a, _ := testApp(t)
	a.list.tab = tabActive // one active row
	a.list.cursor = 99
	a.list.clampCursor()
	if a.list.cursor != 0 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	a.snap = snapshot{}
	a.list.cursor = 0
	a.list.clampCursor() // n=0 → cursor -1 → clamped to 0
	if a.list.cursor != 0 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
}

func TestPadAndLpad(t *testing.T) {
	if got := pad("abcdef", 3); got != "ab…" {
		t.Errorf("pad trunc = %q", got)
	}
	if got := pad("ab", 1); got != "a" {
		t.Errorf("pad w<=1 = %q", got)
	}
	if got := pad("ab", 4); got != "ab  " {
		t.Errorf("pad fill = %q", got)
	}
	if got := lpad("abc", 2); got != "abc" {
		t.Errorf("lpad long = %q", got)
	}
	if got := lpad("a", 3); got != "  a" {
		t.Errorf("lpad fill = %q", got)
	}
}

func TestStatusCellAll(t *testing.T) {
	a, _ := testApp(t)
	m := a.list
	for _, s := range []string{"complete", "error", "removed", "paused", "waiting", "active"} {
		if got := m.statusCell(rpc.Status{Status: s}); got == "" {
			t.Errorf("statusCell(%s) empty", s)
		}
	}
}

func TestIntegrityCellAll(t *testing.T) {
	a, _ := testApp(t)
	m := a.list
	s := rpc.Status{GID: "s1"}
	cases := []struct {
		v    *verifyState
		want string
	}{
		{nil, "-"},
		{vsWithProgress(5, 10), "verifying"},
		{&verifyState{Running: true}, "verifying"},
		{&verifyState{Finished: true, Err: errors.New("boom")}, "boom"},
		{&verifyState{Finished: true, OK: true}, "verified"},
		{&verifyState{Finished: true}, "MISMATCH"},
		{&verifyState{Expected: "abc"}, "checksum set"},
		{&verifyState{}, "-"},
	}
	for i, c := range cases {
		if c.v == nil {
			delete(a.verify, "s1")
		} else {
			a.verify["s1"] = c.v
		}
		if got := m.integrityCell(s); !strings.Contains(got, c.want) {
			t.Errorf("case %d: %q does not contain %q", i, got, c.want)
		}
	}
}

func TestShortHash(t *testing.T) {
	long := strings.Repeat("ab", 20)
	if got := shortHash(long); got != "abababab…abababab" {
		t.Errorf("long = %q", got)
	}
	if got := shortHash("abc"); got != "abc" {
		t.Errorf("short = %q", got)
	}
}

func TestListViewActiveTabAllStatuses(t *testing.T) {
	a, _ := testApp(t)
	a.snap.Active = []rpc.Status{
		{GID: "c1", Status: "complete", TotalLength: "100", CompletedLength: "100"},
		{GID: "e1", Status: "error", TotalLength: "100", CompletedLength: "10"},
		{GID: "w1", Status: "waiting"},
		{GID: "p1", Status: "paused"},
		{GID: "a1", Status: "active", TotalLength: "100", CompletedLength: "50", DownloadSpeed: "7"},
	}
	a.list.cursor = 0
	v := a.list.view()
	if v == "" || strings.Contains(v, "nothing here") {
		t.Fatal("populated view must render rows")
	}
	for _, want := range []string{"done", "error", "waiting", "paused"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestListViewNarrowWidth(t *testing.T) {
	a, _ := testApp(t)
	a.width = 5 // forces nameW clamp, header gap clamp, keybar gap clamp
	if v := a.list.view(); v == "" {
		t.Fatal("empty view")
	}
}

func TestListViewEmpty(t *testing.T) {
	// Connected + no downloads → the onboarding welcome, not a bare table.
	a, _ := testApp(t)
	a.snap = snapshot{}
	if v := a.list.view(); !strings.Contains(v, "Welcome to Aria2t") {
		t.Fatalf("empty connected view must welcome:\n%s", v)
	}
	// Disconnected with a reason → the reason. It used to be recorded and never
	// shown, so a machine with no aria2c looked like a silently broken app.
	a.connected = false
	a.connErr = errors.New("aria2c not found — install it (brew install aria2)")
	v := a.list.view()
	if !strings.Contains(v, "Not connected") || !strings.Contains(v, "aria2c not found") {
		t.Fatalf("disconnected view must name the failure:\n%s", v)
	}
	// Disconnected with nothing to say → the plain placeholder.
	a.connErr = nil
	if v := a.list.view(); !strings.Contains(v, "nothing here") {
		t.Fatalf("disconnected empty view must show placeholder:\n%s", v)
	}
}

func TestListViewFilterNoMatch(t *testing.T) {
	a, _ := testApp(t)
	a.list.filterInput.SetValue("zzz-no-such-name")
	if v := a.list.view(); !strings.Contains(v, "nothing here") {
		t.Fatalf("filter with no matches must show placeholder:\n%s", v)
	}
}

func TestListViewStoppedChecksumStrip(t *testing.T) {
	a, _ := testApp(t)
	a.snap.Stopped = []rpc.Status{
		{GID: "s1", Status: "complete"},
		{GID: "s2", Status: "error"},
	}
	_, _ = a.Update(key("4"))

	a.verify["s1"] = &verifyState{Expected: strings.Repeat("a", 64)}
	if v := a.list.view(); !strings.Contains(v, "CHECKSUM") || !strings.Contains(v, "expected") {
		t.Fatal("expected strip missing")
	}
	a.verify["s1"] = func() *verifyState { v := vsWithProgress(1, 2); v.Expected = "e"; return v }()
	if v := a.list.view(); !strings.Contains(v, "hashing") {
		t.Fatal("running strip missing")
	}
	a.verify["s1"] = &verifyState{Expected: "e", Finished: true, Computed: strings.Repeat("b", 64)}
	if v := a.list.view(); !strings.Contains(v, "computed") {
		t.Fatal("computed strip missing")
	}
	// Cursor row highlight off: move to s2 (no verify state → no strip).
	_, _ = a.Update(key("j"))
	if v := a.list.view(); strings.Contains(v, "CHECKSUM") {
		t.Fatal("strip must vanish without verify state")
	}
}

func TestListViewReordering(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("3"))
	_, _ = a.Update(key("J")) // grab w1, moved to index 1
	if v := a.list.view(); !strings.Contains(v, "REORDER MODE") || !strings.Contains(v, "grabbed") {
		t.Fatal("reorder view missing markers")
	}
}

func TestKeybarVariants(t *testing.T) {
	a, _ := testApp(t)
	a.width = 200 // wide enough that no hint is width-dropped
	kb := a.list.keybar(10)
	if !strings.Contains(kb, "add") || !strings.Contains(kb, "copy url") || !strings.Contains(kb, "seeding") {
		t.Fatalf("normal keybar must surface add/copy/seeding: %q", kb)
	}
	if strings.Contains(kb, "theme") {
		t.Fatal("theme is no longer a global keybar hint")
	}
	a.list.tab = tabWaiting
	if kb := a.list.keybar(10); !strings.Contains(kb, "reorder") {
		t.Fatal("waiting keybar must offer reorder")
	}
	a.list.tab = tabStopped
	if kb := a.list.keybar(10); !strings.Contains(kb, "re-download") {
		t.Fatal("stopped keybar must offer re-download")
	}
	a.list.reordering = true
	if kb := a.list.keybar(10); !strings.Contains(kb, "drop") {
		t.Fatal("reorder keybar must offer drop")
	}
	a.list.reordering = false
	a.snap = snapshot{} // no rows → no position indicator
	a.list.tab = tabActive
	if kb := a.list.keybar(10); strings.Contains(kb, "1/") {
		t.Fatal("empty list must not show position")
	}
}

// vsWithProgress builds a running verifyState with atomic progress set.
func vsWithProgress(done, total int64) *verifyState {
	v := &verifyState{Running: true}
	v.Done.Store(done)
	v.Total.Store(total)
	return v
}
