package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
	"aria2t/internal/rpc"
)

func TestHitmapBasics(t *testing.T) {
	var h hitmap
	h.add("a", 0, 0, 9, 0)
	h.add("b", 5, 0, 9, 0) // overlaps; later wins
	h.line("c", 2, 40)
	if id, ok := h.hit(7, 0); !ok || id != "b" {
		t.Fatalf("hit = %q %v", id, ok)
	}
	if id, ok := h.hit(2, 0); !ok || id != "a" {
		t.Fatalf("hit = %q %v", id, ok)
	}
	if id, ok := h.hit(39, 2); !ok || id != "c" {
		t.Fatalf("hit = %q %v", id, ok)
	}
	if _, ok := h.hit(50, 50); ok {
		t.Fatal("miss expected")
	}
	h.reset()
	if _, ok := h.hit(7, 0); ok {
		t.Fatal("reset must clear regions")
	}
	if k, a := splitID("row:4"); k != "row" || a != "4" {
		t.Fatalf("splitID = %q %q", k, a)
	}
	if argInt("junk") != -1 || argInt("7") != 7 {
		t.Fatal("argInt")
	}
}

// click renders the current view (rebuilding the hitmap) and clicks the
// center of the region with the given id.
func click(t *testing.T, a *App, id string) tea.Cmd {
	t.Helper()
	_ = a.View()
	for i := len(a.hits.regions) - 1; i >= 0; i-- {
		r := a.hits.regions[i]
		if r.id == id {
			_, cmd := a.Update(tea.MouseMsg{
				Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
				X: (r.x0 + r.x1) / 2, Y: (r.y0 + r.y1) / 2,
			})
			return cmd
		}
	}
	t.Fatalf("no region %q in hitmap: %v", id, regionIDs(a))
	return nil
}

func regionIDs(a *App) []string {
	ids := make([]string, 0, len(a.hits.regions))
	for _, r := range a.hits.regions {
		ids = append(ids, r.id)
	}
	return ids
}

func wheel(a *App, up bool) {
	b := tea.MouseButtonWheelDown
	if up {
		b = tea.MouseButtonWheelUp
	}
	_, _ = a.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: b})
}

func TestMouseTabAndRowSelection(t *testing.T) {
	a, _ := testApp(t)
	click(t, a, "tab:2")
	if a.list.tab != tabWaiting {
		t.Fatalf("tab = %d", a.list.tab)
	}
	click(t, a, "row:2")
	if a.list.cursor != 2 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
}

func TestMouseRowClickOpensDetails(t *testing.T) {
	a, _ := testApp(t)
	// A single click selects the row AND opens its details — the expected
	// primary action, and the only mouse path to details if the key-bar is
	// scrolled off a short terminal.
	click(t, a, "row:0")
	if a.list.cursor != 0 || a.screen != screenDetail {
		t.Fatalf("row click must select and open details: cursor=%d screen=%d", a.list.cursor, a.screen)
	}
	// The clickable "↵ details" hint also opens it (keyboard/hint path).
	a.screen = screenList
	click(t, a, "key:enter")
	if a.screen != screenDetail {
		t.Fatalf("details hint must open detail, screen=%d", a.screen)
	}
}

func TestMouseWheelMovesSelection(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("3")) // waiting tab, 3 rows
	wheel(a, false)
	if a.list.cursor != 1 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	wheel(a, true)
	if a.list.cursor != 0 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
}

func TestMouseKeybarClick(t *testing.T) {
	a, _ := testApp(t)
	click(t, a, "key:g")
	if a.screen != screenStats {
		t.Fatalf("screen = %d", a.screen)
	}
}

func TestMouseMissAndNonLeftIgnored(t *testing.T) {
	a, _ := testApp(t)
	_ = a.View()
	_, cmd := a.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: 500, Y: 500})
	if cmd != nil {
		t.Fatal("miss must be inert")
	}
	_, _ = a.Update(tea.MouseMsg{Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	_, _ = a.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight})
}

func TestConfirmModalFlow(t *testing.T) {
	a, fake := testApp(t)
	_, _ = a.Update(key("d"))
	if a.overlay != overlayConfirm {
		t.Fatalf("overlay = %d", a.overlay)
	}
	// Click Cancel: nothing removed.
	cmd := click(t, a, "btn:n")
	drain(t, a, cmd)
	if a.overlay != overlayNone || len(fake.removed) != 0 {
		t.Fatalf("cancel failed: overlay=%d removed=%v", a.overlay, fake.removed)
	}
	// Again, click Remove.
	_, _ = a.Update(key("d"))
	cmd = click(t, a, "btn:y")
	drain(t, a, cmd)
	if len(fake.removed) != 1 {
		t.Fatalf("removed = %v", fake.removed)
	}
}

func TestHelpOverlay(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("?"))
	if a.overlay != overlayHelp {
		t.Fatalf("overlay = %d", a.overlay)
	}
	if v := a.View(); v == "" {
		t.Fatal("help view empty")
	}
	_, _ = a.Update(key("x")) // any key closes? only listed keys close
	_, _ = a.Update(key("esc"))
	if a.overlay != overlayNone {
		t.Fatalf("overlay = %d", a.overlay)
	}
	// Click closes too.
	_, _ = a.Update(key("?"))
	click(t, a, "help:close")
	if a.overlay != overlayNone {
		t.Fatal("click must close help")
	}
}

func TestSmartSpaceToggle(t *testing.T) {
	a, fake := testApp(t)
	_, cmd := a.Update(key(" ")) // a1 is active → pause
	drain(t, a, cmd)
	if len(fake.paused) != 1 || fake.paused[0] != "a1" {
		t.Fatalf("paused = %v", fake.paused)
	}
	// Paused item resumes.
	a.snap.Active[0].Status = "paused"
	_, cmd = a.Update(key(" "))
	drain(t, a, cmd)
	if len(fake.paused) != 1 { // no second pause
		t.Fatalf("paused = %v", fake.paused)
	}
	// Stopped tab: space on a finished download flashes rather than acting.
	_, _ = a.Update(key("4"))
	_, cmd = a.Update(key(" "))
	drain(t, a, cmd)
	if !a.statusErr || len(fake.paused) != 1 {
		t.Fatalf("space on stopped must flash, status=%q paused=%v", a.status, fake.paused)
	}
}

func TestListScrollFollowsCursor(t *testing.T) {
	a, _ := testApp(t)
	a.height = 24 // minimum supported viewport
	var many []rpc.Status
	for i := 0; i < 30; i++ {
		many = append(many, rpc.Status{GID: fmt.Sprintf("g%02d", i), Status: "active"})
	}
	a.snap.Active = many
	for i := 0; i < 20; i++ {
		_, _ = a.Update(key("j"))
	}
	if a.list.cursor != 20 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	vis := a.list.visibleRows()
	if a.list.offset != a.list.cursor-vis+1 {
		t.Fatalf("offset = %d, vis = %d", a.list.offset, vis)
	}
	// Cursor back above the window pulls offset up.
	a.list.cursor = 2
	a.list.clampCursor()
	if a.list.offset != 2 {
		t.Fatalf("offset = %d", a.list.offset)
	}
	if v := a.View(); v == "" {
		t.Fatal("scrolled view empty")
	}
}

func TestThrottleChipClick(t *testing.T) {
	a, fake := testApp(t)
	_, _ = a.Update(key("l")) // open throttle for a1
	cmd := click(t, a, "chip:0:2")
	drain(t, a, cmd)
	if a.overlay != overlayNone {
		t.Fatal("chip click must apply and close")
	}
	if got := fake.changedOptions["a1"]["max-download-limit"]; got != "5M" {
		t.Fatalf("limit = %q", got)
	}
	// Custom chip enters editing instead of applying.
	_, _ = a.Update(key("l"))
	_ = click(t, a, "chip:0:4")
	if !a.throttle.editing {
		t.Fatal("custom chip must start editing")
	}
}

func TestServersRowSelectAndConnectHint(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.Servers = append(a.cfg.Servers, a.cfg.Servers[0])
	a.cfg.Servers[1].Name = "second"
	a.cfg.Servers[1].Managed = false
	_, _ = a.Update(key("s"))
	// Single click selects; connect is a separate clickable hint (no double).
	click(t, a, "srv:1")
	if a.servers.cursor != 1 {
		t.Fatalf("cursor = %d", a.servers.cursor)
	}
	cmd := click(t, a, "btn:enter") // the "Connect ↵" button
	if a.cfg.Active != 1 || a.overlay != overlayNone || cmd == nil {
		t.Fatalf("connect button must switch: active = %d overlay = %d", a.cfg.Active, a.overlay)
	}
}

func TestSettingsSidebarAndFieldClick(t *testing.T) {
	a, _ := testApp(t)
	a.settings = newSettingsModel(a)
	a.screen = screenSettings
	click(t, a, "side:3")
	if a.settings.section != 3 {
		t.Fatalf("section = %d", a.settings.section)
	}
	click(t, a, "field:0") // readonly DHT toggle → flash
	if a.settings.inSide || !a.statusErr {
		t.Fatalf("inSide=%v status=%q", a.settings.inSide, a.status)
	}
	// Clicking header goes back.
	click(t, a, "back")
	if a.screen != screenList {
		t.Fatalf("screen = %d", a.screen)
	}
}

func TestSchedulerRuleClick(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.Rules = []config.Rule{
		{Start: "09:00", End: "18:00", Label: "work", Down: "5M", Up: "1M"},
		{Start: "21:00", End: "06:00", Label: "night", Down: "0", Up: "0"},
	}
	a.scheduler = newSchedulerModel(a)
	a.screen = screenScheduler
	click(t, a, "rule:1")
	if a.scheduler.cursor != 1 {
		t.Fatalf("cursor = %d", a.scheduler.cursor)
	}
}

func TestSeedingTrackerClick(t *testing.T) {
	a, _ := testApp(t)
	a.seeding = newSeedingModel(a)
	a.seeding.gid = "a1"
	a.seeding.trackers = []string{"t1", "t2"}
	a.screen = screenSeeding
	click(t, a, "trk:1")
	if a.seeding.tCursor != 1 || a.seeding.focus != a.seeding.trackersStart() {
		t.Fatalf("tCursor = %d focus = %d", a.seeding.tCursor, a.seeding.focus)
	}
}

func TestAddOverlayTabClick(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("a"))
	click(t, a, "atab:1")
	if a.add.tab != addTabTorrent {
		t.Fatalf("tab = %d", a.add.tab)
	}
}
