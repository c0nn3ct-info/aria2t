package ui

// Frame-consistency tests: the review found that hitmap regions can drift
// from what is actually rendered (wrapping, off-by-one panel math). These
// tests strip ANSI from the real frame and assert that each region's
// coordinates land on the text the user sees — the click helper in
// mouse_test.go cannot catch that, because it trusts the hitmap.

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"aria2t/internal/config"
	"aria2t/internal/rpc"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plainLines(s string) []string {
	return strings.Split(ansiRE.ReplaceAllString(s, ""), "\n")
}

// regionText returns the plain frame content under a region.
func regionText(t *testing.T, a *App, id string) string {
	t.Helper()
	lines := plainLines(a.View())
	for i := len(a.hits.regions) - 1; i >= 0; i-- {
		r := a.hits.regions[i]
		if r.id != id {
			continue
		}
		var b strings.Builder
		for y := r.y0; y <= r.y1 && y < len(lines); y++ {
			row := []rune(lines[y])
			x1 := r.x1
			if x1 >= len(row) {
				x1 = len(row) - 1
			}
			if r.x0 <= x1 {
				b.WriteString(string(row[r.x0 : x1+1]))
			}
			b.WriteString("\n")
		}
		return b.String()
	}
	t.Fatalf("region %q not registered", id)
	return ""
}

// bigApp builds an app with realistic long names at a given size.
func bigApp(t *testing.T, w, h int) (*App, *fakeAPI) {
	t.Helper()
	a, fake := testApp(t)
	a.width, a.height = w, h
	a.snap.Active = []rpc.Status{
		{GID: "g0", Status: "active", TotalLength: "658505728", CompletedLength: "408273485",
			DownloadSpeed: "13002342", Files: []rpc.File{{Path: "/dl/debian-13.1.0-amd64-netinst.iso"}}},
		{GID: "g1", Status: "active", TotalLength: "6227702579", CompletedLength: "1806083748",
			DownloadSpeed: "10276044", Files: []rpc.File{{Path: "/dl/ubuntu-26.04-desktop-amd64.iso"}}},
		{GID: "g2", Status: "complete", TotalLength: "50331648", CompletedLength: "50331648",
			Files: []rpc.File{{Path: "/dl/node-v24.3.0-linux-x64.tar.xz"}}},
	}
	return a, fake
}

func TestFrameNeverExceedsTerminalHeight(t *testing.T) {
	for _, size := range [][2]int{{80, 12}, {100, 24}, {120, 36}, {90, 10}} {
		a, _ := bigApp(t, size[0], size[1])
		var many []rpc.Status
		for i := 0; i < 40; i++ {
			many = append(many, rpc.Status{GID: fmt.Sprintf("g%02d", i), Status: "active",
				TotalLength: "1000", Files: []rpc.File{{Path: fmt.Sprintf("/dl/file-%02d.iso", i)}}})
		}
		a.snap.Active = many
		if lines := plainLines(a.View()); len(lines) > size[1] {
			t.Errorf("%dx%d: frame is %d lines — renderer would clip and shift all regions",
				size[0], size[1], len(lines))
		}
	}
}

func TestListRowsDoNotWrapAndRegionsMatch(t *testing.T) {
	for _, w := range []int{80, 100, 120, 160} {
		a, _ := bigApp(t, w, 36)
		lines := plainLines(a.View())
		// Rows must be single lines: the first download's name must sit on
		// the exact line its region claims.
		if got := regionText(t, a, "row:0"); !strings.Contains(got, "debian-13.1.0") {
			t.Errorf("w=%d: row:0 region does not cover the first download; got %q", w, strings.TrimSpace(got))
		}
		if got := regionText(t, a, "row:2"); !strings.Contains(got, "node-v24.3.0") {
			t.Errorf("w=%d: row:2 region does not cover the third download; got %q", w, strings.TrimSpace(got))
		}
		// The column header must not spill (ETA on its own line was the bug).
		for _, l := range lines {
			if strings.TrimSpace(l) == "ETA" || strings.TrimSpace(l) == "ETA │" {
				t.Errorf("w=%d: header wrapped, ETA spilled to its own line", w)
			}
		}
	}
}

func TestTabRegionsMatchRenderedChips(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	if got := regionText(t, a, "tab:1"); !strings.Contains(got, "Waiting") {
		t.Fatalf("tab:1 covers %q", got)
	}
	if got := regionText(t, a, "tab:2"); !strings.Contains(got, "Stopped") {
		t.Fatalf("tab:2 covers %q", got)
	}
}

func TestKeybarRegionsMatchHints(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	if got := regionText(t, a, "key:g"); !strings.Contains(got, "stats") {
		t.Fatalf("key:g covers %q", got)
	}
	// Stopped tab: integrity hints clickable and inside the terminal.
	_, _ = a.Update(key("3"))
	if got := regionText(t, a, "key:R"); !strings.Contains(got, "re-download") {
		t.Fatalf("key:R covers %q", got)
	}
	_ = a.View()
	for _, r := range a.hits.regions {
		if r.x1 >= a.width {
			t.Fatalf("region %q exceeds terminal width", r.id)
		}
	}
}

func TestDetailFileRegionsMatchRenderedRows(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	a.detail = newDetailModel(a)
	a.detail.gid = "g0"
	a.detail.s = rpc.Status{GID: "g0", Status: "active", Files: []rpc.File{
		{Index: "1", Path: "/dl/FILE-ZERO.bin", Selected: "true"},
		{Index: "2", Path: "/dl/FILE-ONE.bin", Selected: "true"},
		{Index: "3", Path: "/dl/FILE-TWO.bin", Selected: "true"},
	}}
	a.screen = screenDetail
	for i, name := range []string{"FILE-ZERO", "FILE-ONE", "FILE-TWO"} {
		if got := regionText(t, a, fmt.Sprintf("file:%d", i)); !strings.Contains(got, name) {
			t.Fatalf("file:%d region covers %q, want %s", i, strings.TrimSpace(got), name)
		}
	}
}

func TestThrottleChipRegionsMatchRenderedChips(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	a.throttle = newThrottleModel(a)
	a.throttle.gid = "g0"
	a.overlay = overlayThrottle
	// downSel=0: the selected chip renders narrower — the historic drift.
	for i, want := range []string{"∞", "1M", "5M", "10M", "custom"} {
		got := regionText(t, a, fmt.Sprintf("chip:0:%d", i))
		if !strings.Contains(got, want) {
			t.Fatalf("chip:0:%d covers %q, want %q", i, strings.TrimSpace(got), want)
		}
	}
	// And with a custom value set (chip width changes).
	a.throttle.downSel = len(downPresets)
	a.throttle.custom[0].SetValue("3M")
	if got := regionText(t, a, "chip:0:4"); !strings.Contains(got, "3M") {
		t.Fatalf("custom chip covers %q", strings.TrimSpace(got))
	}
}

func TestSeedingTrackerRegionsMatchRows(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	a.snap.Active[0].BitTorrent = &rpc.BTInfo{AnnounceList: [][]string{{"https://embedded.example/announce"}}}
	a.seeding = newSeedingModel(a)
	a.seeding.gid = "g0"
	a.seeding.embedded = []string{"https://embedded.example/announce"}
	a.seeding.trackers = []string{"https://extra-one.example", "https://extra-two.example"}
	a.screen = screenSeeding
	if got := regionText(t, a, "trk:0"); !strings.Contains(got, "extra-one") {
		t.Fatalf("trk:0 covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "trk:1"); !strings.Contains(got, "extra-two") {
		t.Fatalf("trk:1 covers %q", strings.TrimSpace(got))
	}
	// Without embedded trackers the offset math must still hold.
	a.seeding.embedded = nil
	if got := regionText(t, a, "trk:0"); !strings.Contains(got, "extra-one") {
		t.Fatalf("no-embedded trk:0 covers %q", strings.TrimSpace(got))
	}
}

func TestSchedulerRuleRegionsMatchRowsAndNoWrap(t *testing.T) {
	for _, w := range []int{78, 100, 120} {
		a, _ := bigApp(t, w, 36)
		a.cfg.Rules = []config.Rule{
			{Start: "09:00", End: "18:00", Label: "RULE-ALPHA working hours", Down: "5M", Up: "1M"},
			{Start: "21:00", End: "06:00", Label: "RULE-BETA night", Down: "0", Up: "0"},
		}
		a.scheduler = newSchedulerModel(a)
		a.screen = screenScheduler
		if got := regionText(t, a, "rule:1"); !strings.Contains(got, "RULE-BETA") {
			t.Fatalf("w=%d: rule:1 covers %q", w, strings.TrimSpace(got))
		}
	}
}

func TestServersRowRegionsMatchRows(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	a.cfg.Servers = append(a.cfg.Servers, a.cfg.Servers[0])
	a.cfg.Servers[1].Name = "SRV-SECOND"
	a.cfg.Servers[1].Managed = false
	a.servers = newServersModel(a)
	a.overlay = overlayServers
	if got := regionText(t, a, "srv:1"); !strings.Contains(got, "SRV-SECOND") {
		t.Fatalf("srv:1 covers %q", strings.TrimSpace(got))
	}
}

func TestConfirmButtonRegionsMatchButtons(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	a.confirm = newConfirmModel(a, "Remove download?", "x", nil)
	a.overlay = overlayConfirm
	if got := regionText(t, a, "btn:yes"); !strings.Contains(got, "Remove") {
		t.Fatalf("btn:yes covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "btn:no"); !strings.Contains(got, "Cancel") {
		t.Fatalf("btn:no covers %q", strings.TrimSpace(got))
	}
}

func TestWheelNeverTypesIntoInputs(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	// Settings: an input focused — wheel must be inert.
	a.settings = newSettingsModel(a)
	a.screen = screenSettings
	a.settings.inSide = false
	before := a.settings.fields[0][0].input.Value()
	wheel(a, false)
	if got := a.settings.fields[0][0].input.Value(); got != before {
		t.Fatalf("wheel typed into settings input: %q", got)
	}
	// Add overlay textarea.
	a.screen = screenList
	_, _ = a.Update(key("a"))
	wheel(a, false)
	if got := a.add.uris.Value(); got != "" {
		t.Fatalf("wheel typed into add textarea: %q", got)
	}
	_, _ = a.Update(key("esc"))
	// Seeding with ratio input focused.
	a.seeding = newSeedingModel(a)
	a.screen = screenSeeding
	a.seeding.focus = 0
	wheel(a, false)
	if got := a.seeding.ratio.Value(); got != "" {
		t.Fatalf("wheel typed into seeding ratio: %q", got)
	}
	// Seeding tracker section: wheel navigates.
	a.seeding.trackers = []string{"a", "b"}
	a.seeding.focus = a.seeding.trackersStart()
	wheel(a, false)
	if a.seeding.tCursor != 1 {
		t.Fatalf("wheel must move the tracker cursor, got %d", a.seeding.tCursor)
	}
}

func TestWheelNavigatesBranches(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	// Servers overlay: navigates unless the form is open.
	a.overlay = overlayServers
	if !a.wheelNavigates() {
		t.Fatal("servers list must wheel")
	}
	a.servers.editing = true
	if a.wheelNavigates() {
		t.Fatal("servers form must not wheel")
	}
	a.servers.editing = false
	// Other overlays never wheel.
	a.overlay = overlayAdd
	if a.wheelNavigates() {
		t.Fatal("add overlay must not wheel")
	}
	a.overlay = overlayNone
	// Detail wheels (moves file cursor at most).
	a.screen = screenDetail
	if !a.wheelNavigates() {
		t.Fatal("detail must wheel")
	}
	// Scheduler: only outside the rule form.
	a.screen = screenScheduler
	a.scheduler.editing = true
	if a.wheelNavigates() {
		t.Fatal("scheduler form must not wheel")
	}
	a.scheduler.editing = false
	if !a.wheelNavigates() {
		t.Fatal("scheduler list must wheel")
	}
	// Stats: nothing to scroll.
	a.screen = screenStats
	if a.wheelNavigates() {
		t.Fatal("stats must not wheel")
	}
}

func TestHelpClickViaOverlayHandler(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	_, _ = a.Update(key("?"))
	cmd := click(t, a, "help:close") // routed through the overlayHelp mouse case
	if a.overlay != overlayNone || cmd != nil {
		t.Fatalf("overlay = %d", a.overlay)
	}
}

func TestStripHeightVariants(t *testing.T) {
	a, _ := testApp(t)
	a.list.tab = tabStopped
	// No verify state → 0.
	if h := a.list.stripHeight(); h != 0 {
		t.Fatalf("h = %d", h)
	}
	// Expected only → 4 lines.
	a.verify["s1"] = &verifyState{Expected: "abc"}
	if h := a.list.stripHeight(); h != 4 {
		t.Fatalf("h = %d", h)
	}
	// Running → extra computed line.
	a.verify["s1"].Running = true
	if h := a.list.stripHeight(); h != 5 {
		t.Fatalf("h = %d", h)
	}
	// Finished with a computed hash → extra line too.
	a.verify["s1"].Running = false
	a.verify["s1"].Finished = true
	a.verify["s1"].Computed = "beef"
	if h := a.list.stripHeight(); h != 5 {
		t.Fatalf("h = %d", h)
	}
	// Off the stopped tab → always 0.
	a.list.tab = tabActive
	if h := a.list.stripHeight(); h != 0 {
		t.Fatalf("h = %d", h)
	}
}
