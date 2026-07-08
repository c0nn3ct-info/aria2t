package ui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/rpc"
)

// TestButtonHelpers covers the shared button/checkbox/tab/modalCard helpers.
func TestButtonHelpers(t *testing.T) {
	a, _ := testApp(t)
	st := a.styles
	// Exercise every style arm; the test color profile strips colour, so assert
	// on the label/key text rather than the (stripped) background.
	for _, v := range []btnVariant{btnPrimary, btnDanger, btnNeutral} {
		if got := (button{"y", "Go", "y", v}).render(st); !strings.Contains(got, "Go") || !strings.Contains(got, "y") {
			t.Fatalf("button render missing label/key: %q", got)
		}
	}
	if !strings.Contains(a.checkbox(true), "x") || strings.Contains(a.checkbox(false), "x") {
		t.Fatalf("checkbox glyphs wrong: on=%q off=%q", a.checkbox(true), a.checkbox(false))
	}
	if !strings.Contains(a.tab("Zed", true), "Zed") || !strings.Contains(a.tab("Zed", false), "Zed") {
		t.Fatal("tab must contain its label in both states")
	}
	if !strings.Contains(a.modalCard(true).Render("x"), "x") || !strings.Contains(a.modalCard(false).Render("x"), "x") {
		t.Fatal("modalCard must render its content")
	}
}

// TestHintbarExTrailer covers the right-aligned trailer path (list reorder uses
// it) and a too-narrow terminal where the gap clamps.
func TestHintbarExTrailer(t *testing.T) {
	a, _ := testApp(t)
	a.width = 80
	line := a.hintbarEx(3, []keyHint{{"a", "a", "add"}}, a.styles.Key, a.styles.Dim.Render("1/9"))
	if !strings.Contains(line, "1/9") {
		t.Fatalf("trailer missing: %q", line)
	}
	a.width = 4 // gap clamps to 1
	_ = a.hintbarEx(3, []keyHint{{"a", "a", "add"}}, a.styles.Key, a.styles.Dim.Render("1/9"))
}

// TestPromptButtonClicks: the checksum prompt's Confirm/Cancel buttons work by
// mouse (the overlay was fully non-clickable before).
func TestPromptButtonClicks(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 100, 30
	got := ""
	a.prompt = newPromptModel(a, "Checksum", "abc", func(v string) tea.Cmd {
		got = v
		return nil
	})
	a.overlay = overlayPrompt
	click(t, a, "btn:enter")
	if a.overlay != overlayNone || got != "abc" {
		t.Fatalf("Confirm button must submit + close: overlay=%d got=%q", a.overlay, got)
	}
	a.prompt = newPromptModel(a, "Checksum", "", nil)
	a.overlay = overlayPrompt
	click(t, a, "btn:esc")
	if a.overlay != overlayNone {
		t.Fatal("Cancel button must close the prompt")
	}
	if _, cmd := a.prompt.mouse("bogus:1"); cmd != nil {
		t.Fatal("non-button prompt click must be inert")
	}
}

// TestAddThrottleBrowseButtons covers the new "btn" mouse cases on those overlays.
func TestAddThrottleBrowseButtons(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 100, 40

	a.add = newAddModel(a)
	a.overlay = overlayAdd
	click(t, a, "btn:esc") // Cancel closes
	if a.overlay != overlayNone {
		t.Fatalf("add Cancel button must close, overlay=%d", a.overlay)
	}

	a.throttle = newThrottleModel(a)
	a.throttle.gid = "a1"
	a.overlay = overlayThrottle
	click(t, a, "key:tab")          // nav-hint row switch (key mouse path)
	cmd := click(t, a, "btn:enter") // Apply
	drain(t, a, cmd)
	if a.overlay != overlayNone {
		t.Fatalf("throttle Apply button must close, overlay=%d", a.overlay)
	}

	withReadDir(t, func(string) ([]os.DirEntry, error) { return nil, nil })
	a.browse = newBrowseModel(a, "/x", nil)
	a.overlay = overlayBrowse
	click(t, a, "btn:esc") // Cancel returns to add
	if a.overlay != overlayAdd {
		t.Fatalf("browse Cancel must return to add, overlay=%d", a.overlay)
	}
}

// TestServersEditFormClicks covers the servers edit-form mouse branch (field
// focus, protocol toggle, save/cancel) — keyboard-only before.
func TestServersEditFormClicks(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 100, 40
	a.servers = newServersModel(a)
	a.overlay = overlayServers
	click(t, a, "key:+") // open the add-server form via the nav hint (key mouse path)
	if !a.servers.editing {
		t.Fatal("+ must open the edit form")
	}
	// Drift guard: the field region must actually sit on the Host label/input.
	if got := regionText(t, a, "field:1"); !strings.Contains(got, "Host") {
		t.Fatalf("field:1 covers %q, want Host", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "proto:ws"); !strings.Contains(got, "ws") {
		t.Fatalf("proto:ws covers %q", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "btn:enter"); !strings.Contains(got, "Save") {
		t.Fatalf("servers save button covers %q", strings.TrimSpace(got))
	}
	click(t, a, "field:1") // focus Host
	if a.servers.formFoc != 1 {
		t.Fatalf("field click must focus host, formFoc=%d", a.servers.formFoc)
	}
	click(t, a, "proto:http")
	if a.servers.formWS {
		t.Fatal("proto:http click must select http")
	}
	click(t, a, "proto:ws")
	if !a.servers.formWS {
		t.Fatal("proto:ws click must select ws")
	}
	click(t, a, "btn:esc") // Cancel
	if a.servers.editing {
		t.Fatal("Cancel button must leave the edit form")
	}
}

// TestSchedulerEditFormClicks covers the scheduler edit-form mouse branch.
func TestSchedulerEditFormClicks(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 120, 40
	a.scheduler = newSchedulerModel(a)
	a.screen = screenScheduler
	_, _ = a.Update(key("+")) // open the rule form
	if !a.scheduler.editing {
		t.Fatal("+ must open the rule form")
	}
	if got := regionText(t, a, "field:0"); !strings.Contains(got, "Window start") {
		t.Fatalf("field:0 covers %q, want Window start", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "day:0"); !strings.Contains(got, "Su") {
		t.Fatalf("day:0 covers %q, want Su", strings.TrimSpace(got))
	}
	click(t, a, "field:0") // focus first field
	if a.scheduler.formFoc != 0 {
		t.Fatalf("field click must focus, formFoc=%d", a.scheduler.formFoc)
	}
	before := a.scheduler.days
	click(t, a, "day:0") // toggle Sunday (index 0)
	if a.scheduler.days == before {
		t.Fatal("day click must toggle a day")
	}
	click(t, a, "btn:esc") // Cancel
	if a.scheduler.editing {
		t.Fatal("Cancel button must leave the rule form")
	}
}

// TestSeedingInputClicks covers the seeding screen's clickable inputs + toggles.
func TestSeedingInputClicks(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 120, 40
	a.snap.Active = []rpc.Status{{GID: "a1", Status: "active", BitTorrent: &rpc.BTInfo{}}}
	a.seeding = newSeedingModel(a)
	a.seeding.gid = "a1"
	a.screen = screenSeeding
	// Drift guards: regions must sit on the labelled inputs and the first toggle.
	if got := regionText(t, a, "field:0"); !strings.Contains(got, "Stop at ratio") {
		t.Fatalf("field:0 covers %q, want Stop at ratio", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "field:1"); !strings.Contains(got, "Or after seeding") {
		t.Fatalf("field:1 covers %q, want Or after seeding", strings.TrimSpace(got))
	}
	if got := regionText(t, a, "tog:0"); !strings.Contains(got, "DHT") {
		t.Fatalf("tog:0 covers %q, want DHT", strings.TrimSpace(got))
	}
	click(t, a, "field:1") // focus seed-time
	if a.seeding.focus != 1 {
		t.Fatalf("field:1 must focus stime, focus=%d", a.seeding.focus)
	}
	click(t, a, "field:0") // focus ratio
	if a.seeding.focus != 0 {
		t.Fatalf("field:0 must focus ratio, focus=%d", a.seeding.focus)
	}
	cmd := click(t, a, "tog:0") // read-only toggle → flash
	if cmd == nil {
		t.Fatal("toggle click must flash the read-only message")
	}
}

// TestListQuitAndReorderClickable: q and the reorder J/K are now clickable.
func TestListQuitAndReorderClickable(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 120, 30
	click(t, a, "key:q") // active downloads → quit confirm
	if a.overlay != overlayConfirm {
		t.Fatalf("q click must open the quit confirm, overlay=%d", a.overlay)
	}
	a.overlay = overlayNone
	_, _ = a.Update(key("3")) // waiting tab
	_, _ = a.Update(key("J")) // enter reorder mode
	if !a.list.reordering {
		t.Fatal("J must enter reorder mode")
	}
	click(t, a, "key:J") // clickable move-down
	click(t, a, "key:K") // clickable move-up
}
