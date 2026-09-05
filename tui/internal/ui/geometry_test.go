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

// TestListColumnsAlign pins the fix for the reported drift: the right edge of
// each right-aligned numeric column (SIZE/SPEED/CONN/ETA) must coincide between
// the header and an active row, regardless of the row name's width.
func TestListColumnsAlign(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 200, 24
	a.snap = snapshot{Active: []rpc.Status{{
		GID: "g1", Status: "active", TotalLength: "2791728742", CompletedLength: "0",
		DownloadSpeed: "2200000", Connections: "34",
		Files: []rpc.File{{Path: "/dl/Смешарики"}}, // Cyrillic: rune vs byte width
	}}}
	a.list.tab = tabActive
	lines := plainLines(a.list.view())
	// list.view() = status line, tabs, panel(top border, header, row, bottom).
	var h, r string
	for i, ln := range lines {
		if strings.Contains(ln, "PROGRESS") {
			h, r = ln, lines[i+1]
			break
		}
	}
	if h == "" {
		t.Fatal("header row not found")
	}
	rightEdge := func(s, tok string) int {
		i := strings.Index(s, tok)
		if i < 0 {
			t.Fatalf("token %q not found in %q", tok, s)
		}
		return len([]rune(s[:i])) + len([]rune(tok))
	}
	for _, p := range []struct{ head, val string }{
		{"SIZE", "GiB"}, {"SPEED", "MiB/s"}, {"CONN", "34"}, {"ETA", "8s"},
	} {
		if he, ve := rightEdge(h, p.head), rightEdge(r, p.val); he != ve {
			t.Errorf("%s column misaligned: header right edge %d, value right edge %d", p.head, he, ve)
		}
	}
}

// TestListKeybarPinnedToBottom: the key-bar must stay on-screen (never scroll
// off) even on a short terminal or with the checksum strip — the reported
// "hints sometimes disappear". The list is pinned so total == terminal height
// and the key-bar is the last (or second-last, under a flash) visible row.
func TestListKeybarPinnedToBottom(t *testing.T) {
	cases := []struct {
		h, n  int
		strip bool
	}{{6, 1, false}, {8, 3, false}, {12, 3, true}, {10, 30, true}, {24, 30, true}}
	for _, c := range cases {
		a, _ := testApp(t)
		a.width, a.height = 120, c.h
		var rows []rpc.Status
		for i := 0; i < c.n; i++ {
			rows = append(rows, rpc.Status{GID: fmt.Sprintf("g%d", i), Status: "complete",
				TotalLength: "100", CompletedLength: "100", Files: []rpc.File{{Path: "/dl/f"}}})
		}
		a.snap = snapshot{Stopped: rows}
		a.list.tab = tabStopped
		if c.strip {
			a.verify = map[string]*verifyState{"g0": {Expected: "abc", Finished: true, OK: true, Computed: "abc"}}
		}
		lines := plainLines(a.list.view())
		if len(lines) > c.h {
			t.Errorf("h=%d n=%d strip=%v: frame %d lines exceeds terminal", c.h, c.n, c.strip, len(lines))
		}
		kb := -1
		for i, ln := range lines {
			if strings.Contains(ln, "d remove") {
				kb = i
			}
		}
		if kb < 0 || kb >= c.h {
			t.Errorf("h=%d n=%d strip=%v: key-bar off-screen (line %d of %d)", c.h, c.n, c.strip, kb, c.h)
		}
	}
}

// TestListBottomBarTinyHeight exercises the degenerate a.height≤1 guard.
func TestListBottomBarTinyHeight(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 40, 1
	_ = a.list.view() // must not panic
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

// TestAllScreensPinHintBar: every screen (not just the list) must keep its
// hint bar on-screen and never exceed the terminal height, even with tall
// content on a short terminal — the "hints sometimes disappear" report.
func TestAllScreensPinHintBar(t *testing.T) {
	var many []rpc.Status
	var files []rpc.File
	for i := 0; i < 40; i++ {
		many = append(many, rpc.Status{GID: fmt.Sprintf("g%02d", i), Status: "active",
			TotalLength: "1000", CompletedLength: "500", InfoHash: "h", Files: []rpc.File{{Path: fmt.Sprintf("/dl/f%02d", i)}}})
		files = append(files, rpc.File{Index: fmt.Sprintf("%d", i+1), Path: fmt.Sprintf("/dl/f%02d", i), Length: "100", CompletedLength: "50"})
	}
	screens := []struct {
		name  string
		scr   screen
		setup func(a *App)
	}{
		{"detail", screenDetail, func(a *App) {
			a.detail = newDetailModel(a)
			a.detail.gid = "g00"
			a.detail.s = rpc.Status{GID: "g00", Status: "active", InfoHash: "h", Files: files}
		}},
		{"stats", screenStats, func(a *App) {}},
		{"settings", screenSettings, func(a *App) { a.settings = newSettingsModel(a) }},
		{"seeding", screenSeeding, func(a *App) {
			a.seeding = newSeedingModel(a)
			a.seeding.gid = "g00"
			for i := 0; i < 30; i++ {
				a.seeding.trackers = append(a.seeding.trackers, fmt.Sprintf("http://tr%02d", i))
			}
		}},
		{"scheduler", screenScheduler, func(a *App) {}},
	}
	for _, size := range [][2]int{{100, 8}, {100, 12}, {90, 6}} {
		w, h := size[0], size[1]
		for _, sc := range screens {
			a, _ := testApp(t)
			a.width, a.height = w, h
			a.snap.Active = many
			for i := 0; i < 30; i++ {
				a.cfg.Rules = append(a.cfg.Rules, config.Rule{Start: "00:00", End: "01:00", Label: fmt.Sprintf("r%d", i), Down: "1M", Up: "1M"})
			}
			a.screen = sc.scr
			sc.setup(a)
			lines := plainLines(a.View())
			if len(lines) > h {
				t.Errorf("%s %dx%d: frame %d lines exceeds terminal height", sc.name, w, h, len(lines))
			}
			if len(lines) == 0 || !strings.Contains(lines[0], "Terminal too small") {
				t.Errorf("%s %dx%d: missing explicit minimum-size fallback", sc.name, w, h)
			}
		}
	}
}

func TestListRowsDoNotWrapAndRegionsMatch(t *testing.T) {
	for _, w := range []int{80, 100, 120, 160} {
		a, _ := bigApp(t, w, 36)
		lines := plainLines(a.View())
		// Rows must be single lines: the first download's name must sit on
		// the exact line its region claims. (Names truncate hard at narrow
		// widths once every column is shown, so match a short prefix.)
		if got := regionText(t, a, "row:0"); !strings.Contains(got, "debian") {
			t.Errorf("w=%d: row:0 region does not cover the first download; got %q", w, strings.TrimSpace(got))
		}
		if got := regionText(t, a, "row:2"); !strings.Contains(got, "node-v24") {
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
	for i, want := range []string{"All", "Active", "Waiting", "Stopped"} {
		if got := regionText(t, a, fmt.Sprintf("tab:%d", i)); !strings.Contains(got, want) {
			t.Fatalf("tab:%d covers %q, want %q", i, got, want)
		}
	}
}

func TestKeybarRegionsMatchHints(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	if got := regionText(t, a, "key:g"); !strings.Contains(got, "stats") {
		t.Fatalf("key:g covers %q", got)
	}
	// Stopped tab: integrity hints clickable and inside the terminal.
	_, _ = a.Update(key("4"))
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

func TestFilesPickerRegionsMatchRows(t *testing.T) {
	a, _ := bigApp(t, 120, 36)
	a.files = newFilesModel(a)
	a.files.gid = "g0"
	a.files.loading = false
	a.files.root = buildTree([]rpc.File{
		{Index: "1", Path: "/dl/DIR-ALPHA/FILE-ZERO.bin", Length: "10", Selected: "true"},
		{Index: "2", Path: "/dl/DIR-ALPHA/FILE-ONE.bin", Length: "10", Selected: "true"},
		{Index: "3", Path: "/dl/FILE-TWO.bin", Length: "10", Selected: "false"},
	}, "/dl")
	a.files.rows = flatten(a.files.root)
	a.overlay = overlayFiles
	// Visible rows: 0 DIR-ALPHA/, 1 FILE-ZERO, 2 FILE-ONE, 3 FILE-TWO.
	if got := regionText(t, a, "row:0"); !strings.Contains(got, "DIR-ALPHA") {
		t.Fatalf("row:0 covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "row:3"); !strings.Contains(got, "FILE-TWO") {
		t.Fatalf("row:3 covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "check:1"); !strings.Contains(got, "[") {
		t.Fatalf("check:1 covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "btn:enter"); !strings.Contains(got, "Confirm") {
		t.Fatalf("btn:enter covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "btn:esc"); !strings.Contains(got, "Cancel") {
		t.Fatalf("btn:esc covers %q", strings.TrimSpace(got))
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
	for _, w := range []int{80, 100, 120} {
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
	if got := regionText(t, a, "btn:y"); !strings.Contains(got, "Remove") {
		t.Fatalf("btn:y covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "btn:n"); !strings.Contains(got, "Cancel") {
		t.Fatalf("btn:n covers %q", strings.TrimSpace(got))
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
