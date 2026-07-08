package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"aria2t/internal/rpc"
)

// These tests exercise the "everything is clickable" pass: each screen and
// overlay key-bar hint triggers its action through the mouse, and no action
// needs a double-click.

func TestKeyFromTokenAll(t *testing.T) {
	cases := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "tab": tea.KeyTab,
		"^s": tea.KeyCtrlS, "^d": tea.KeyCtrlD, "^t": tea.KeyCtrlT,
		"^r": tea.KeyCtrlR, "^o": tea.KeyCtrlO,
	}
	for tok, typ := range cases {
		if keyFromToken(tok).Type != typ {
			t.Fatalf("keyFromToken(%q) type = %v", tok, keyFromToken(tok).Type)
		}
	}
	if r := keyFromToken("x"); r.Type != tea.KeyRunes || string(r.Runes) != "x" {
		t.Fatalf("rune token = %#v", r)
	}
}

func TestDetailKeybarClickable(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "g1"
	a.detail.s = rpc.Status{GID: "g1", Status: "active", Files: []rpc.File{{Index: "1", Path: "/dl/a"}}}
	click(t, a, "key:f") // select-files hint opens the picker — the reported case
	if a.overlay != overlayFiles {
		t.Fatalf("f hint must open the picker, overlay=%d", a.overlay)
	}
	a.overlay = overlayNone
	a.screen = screenDetail
	click(t, a, "key:esc")
	if a.screen != screenList {
		t.Fatalf("esc hint must go back, screen=%d", a.screen)
	}
	if _, c := a.detail.mouse("row:0"); c != nil {
		t.Fatal("non-key detail click must be inert")
	}
}

func TestStatsKeybarClickable(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenStats
	click(t, a, "key:esc")
	if a.screen != screenList {
		t.Fatalf("stats esc hint must go back, screen=%d", a.screen)
	}
}

func TestSeedingKeybarClickable(t *testing.T) {
	a, _ := testApp(t)
	a.seeding = newSeedingModel(a)
	a.seeding.gid = "a1"
	a.screen = screenSeeding
	click(t, a, "key:esc")
	if a.screen != screenList {
		t.Fatalf("seeding esc hint must go back, screen=%d", a.screen)
	}
}

func TestSchedulerKeybarClickable(t *testing.T) {
	a, _ := testApp(t)
	a.scheduler = newSchedulerModel(a)
	a.screen = screenScheduler
	click(t, a, "key:+") // add-rule hint opens the form
	if !a.scheduler.editing {
		t.Fatal("+ hint must open the rule form")
	}
}

func TestSettingsKeybarClickable(t *testing.T) {
	a, _ := testApp(t)
	a.settings = newSettingsModel(a)
	a.screen = screenSettings
	click(t, a, "key:esc")
	if a.screen != screenList {
		t.Fatalf("settings esc hint must go back, screen=%d", a.screen)
	}
}

func TestThrottleCancelHintClickable(t *testing.T) {
	a, _ := testApp(t)
	a.throttle = newThrottleModel(a)
	a.throttle.gid = "a1"
	a.overlay = overlayThrottle
	click(t, a, "btn:esc")
	if a.overlay != overlayNone {
		t.Fatalf("throttle cancel button must cancel, overlay=%d", a.overlay)
	}
}

func TestFilesHintClickable(t *testing.T) {
	a, _ := testApp(t)
	a.files = loadedFiles(a)
	a.overlay = overlayFiles
	click(t, a, "key:n") // select none
	if selectedLeaves(a.files.root) != 0 {
		t.Fatal("n hint must deselect all")
	}
	click(t, a, "key:a") // select all
	if selectedLeaves(a.files.root) != leafCount(a.files.root) {
		t.Fatal("a hint must select all")
	}
}

func TestAddFooterHintClickable(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("a")) // open add overlay
	before := a.add.startNow
	click(t, a, "key:^s") // toggle start-now via the footer hint
	if a.add.startNow == before {
		t.Fatal("^s hint must toggle start-now")
	}
}

// clickAt presses the left mouse button at an exact cell.
func clickAt(a *App, x, y int) {
	_, _ = a.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: x, Y: y})
}

// TestAddInlineBrowseClickable is the reported bug: on the file tabs, the
// "^o browse" hint sitting next to the path field (and the field itself) must
// open the file browser on a real click — not only the footer hint.
// TestDetailFilesPanelClickable: clicking the FILES panel opens the tree picker
// (full mouse parity — not only the f key / footer hint).
func TestDetailFilesPanelClickable(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail = newDetailModel(a)
	a.detail.gid = "g1"
	a.detail.s = rpc.Status{GID: "g1", Status: "active", Dir: "/dl",
		Files: []rpc.File{{Index: "1", Path: "/dl/a"}, {Index: "2", Path: "/dl/b"}}}
	frame := a.View()
	col, y := -1, -1
	for i, ln := range strings.Split(frame, "\n") {
		if c := strings.Index(ansi.Strip(ln), "FILES"); c >= 0 {
			col, y = c, i
			break
		}
	}
	if col < 0 {
		t.Fatal("FILES panel not rendered")
	}
	clickAt(a, col, y)
	if a.overlay != overlayFiles {
		t.Fatalf("clicking the FILES panel must open the picker, overlay=%d", a.overlay)
	}
}

func TestAddInlineBrowseClickable(t *testing.T) {
	find := func(frame, needle string) (col, y int) {
		for i, ln := range strings.Split(frame, "\n") {
			if c := strings.Index(ansi.Strip(ln), needle); c >= 0 {
				return c, i
			}
		}
		return -1, -1
	}

	// Inline "^o browse" next to the label.
	a, _ := testApp(t)
	a.overlay = overlayAdd
	a.add = newAddModel(a)
	a.add.tab = addTabTorrent
	frame := a.View()
	col, y := find(frame, "browse")
	if col < 0 {
		t.Fatal("inline browse hint not rendered")
	}
	clickAt(a, col, y)
	if a.overlay != overlayBrowse {
		t.Fatalf("clicking the inline browse hint must open the browser, overlay=%d", a.overlay)
	}

	// The path field itself.
	a2, _ := testApp(t)
	a2.overlay = overlayAdd
	a2.add = newAddModel(a2)
	a2.add.tab = addTabTorrent
	a2.add.file.SetValue("/some/path.torrent")
	frame = a2.View()
	fcol, fy := find(frame, "path.torrent")
	if fcol < 0 {
		t.Fatal("path field not rendered")
	}
	clickAt(a2, fcol, fy)
	if a2.overlay != overlayBrowse {
		t.Fatalf("clicking the path field must open the browser, overlay=%d", a2.overlay)
	}
}
