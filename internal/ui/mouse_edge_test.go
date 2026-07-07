package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseHandlerGuards(t *testing.T) {
	a, _ := testApp(t)

	// list: invalid ids and out-of-range rows are inert.
	l := a.list
	for _, id := range []string{"junk", "tab:9", "tab:x", "row:99", "row:-1", "nope:1"} {
		if _, cmd := l.mouse(id); cmd != nil {
			t.Fatalf("%q must be inert", id)
		}
	}
	// list: clicks ignored while reordering.
	l.reordering = true
	l.localOrder = a.snap.Waiting
	if m2, _ := l.mouse("tab:2"); m2.tab != l.tab {
		t.Fatal("tab click during reorder must be ignored")
	}
	if m2, _ := l.mouse("row:0"); m2.cursor != l.cursor {
		t.Fatal("row click during reorder must be ignored")
	}

	// scheduler guards: editing swallows clicks, bad index inert.
	s := newSchedulerModel(a)
	s.editing = true
	if m2, _ := s.mouse("rule:0"); m2.cursor != s.cursor {
		t.Fatal("editing must swallow rule clicks")
	}
	s.editing = false
	if _, cmd := s.mouse("rule:42"); cmd != nil {
		t.Fatal("bad rule index must be inert")
	}
	if _, cmd := s.mouse("other:1"); cmd != nil {
		t.Fatal("foreign id must be inert")
	}

	// seeding guards.
	se := newSeedingModel(a)
	se.trackers = []string{"t"}
	for _, id := range []string{"trk:5", "trk:x", "other"} {
		if _, cmd := se.mouse(id); cmd != nil {
			t.Fatalf("%q must be inert", id)
		}
	}

	// servers guards.
	sv := newServersModel(a)
	sv.editing = true
	if m2, _ := sv.mouse("srv:0"); m2.cursor != sv.cursor {
		t.Fatal("form editing must swallow row clicks")
	}
	sv.editing = false
	if _, cmd := sv.mouse("srv:99"); cmd != nil {
		t.Fatal("bad server index must be inert")
	}
	if _, cmd := sv.mouse("blah"); cmd != nil {
		t.Fatal("foreign id must be inert")
	}

	// settings guards.
	st := newSettingsModel(a)
	if _, cmd := st.mouse("side:99"); cmd != nil {
		t.Fatal("bad section must be inert")
	}
	if _, cmd := st.mouse("field:99"); cmd != nil {
		t.Fatal("bad field must be inert")
	}
	if _, cmd := st.mouse("zzz:1"); cmd != nil {
		t.Fatal("foreign id must be inert")
	}
	// non-toggle field click focuses the input.
	st.section = 0
	m2, cmd := st.mouse("field:0")
	if cmd == nil || m2.inSide || m2.focus != 0 {
		t.Fatal("input field click must focus it")
	}

	// throttle guards.
	th := newThrottleModel(a)
	for _, id := range []string{"nope", "chip:bad", "chip:5:0", "chip:0:99", "chip:x:y"} {
		if _, cmd := th.mouse(id); cmd != nil {
			t.Fatalf("%q must be inert", id)
		}
	}

	// add guards.
	ad := newAddModel(a)
	if _, cmd := ad.mouse("zzz:0"); cmd != nil {
		t.Fatal("foreign id must be inert")
	}
	if m2, _ := ad.mouse("atab:0"); m2.tab != ad.tab {
		t.Fatal("same tab click must be a no-op")
	}
	if _, cmd := ad.mouse("btn:nothing"); cmd != nil {
		t.Fatal("unknown button must be inert")
	}
}

func TestAddSubmitClickHint(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	m.uris.SetValue("") // empty → flash error path via the ^d add hint
	m2, cmd := m.mouse("key:^d")
	_ = m2
	if cmd == nil || a.status == "" {
		t.Fatalf("empty submit must flash, status = %q", a.status)
	}
}

func TestOverlayOffsetClamps(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 4, 2 // tiny terminal: modal larger than screen
	x, y := a.overlayOffset("wide modal line\n2\n3\n4\n5\n6")
	if x != 0 || y != 0 {
		t.Fatalf("offsets must clamp to zero: %d %d", x, y)
	}
}

func TestConfirmWithoutCallback(t *testing.T) {
	a, _ := testApp(t)
	a.confirm = newConfirmModel(a, "t", "x", nil)
	a.overlay = overlayConfirm
	m, cmd := a.confirm.update(key("y"))
	_ = m
	if cmd != nil || a.overlay != overlayNone {
		t.Fatal("nil callback confirm must just close")
	}
	// mouse variant with foreign id is inert.
	a.overlay = overlayConfirm
	if _, cmd := a.confirm.mouse("zzz"); cmd != nil || a.overlay != overlayConfirm {
		t.Fatal("foreign id must be inert")
	}
}

func TestVisibleRowsFloorAndOffsetClamp(t *testing.T) {
	a, _ := testApp(t)
	a.height = 5
	if v := a.list.visibleRows(); v != 3 {
		t.Fatalf("visibleRows floor = %d", v)
	}
	a.list.offset = -4
	a.list.cursor = 0
	a.list.clampCursor()
	if a.list.offset != 0 {
		t.Fatalf("offset = %d", a.list.offset)
	}
}

func TestListViewOffsetPastEnd(t *testing.T) {
	a, _ := testApp(t)
	a.list.offset = 50 // stale offset beyond the row count
	if v := a.View(); v == "" {
		t.Fatal("view must survive a stale offset")
	}
}

func TestKeyFromTokenEsc(t *testing.T) {
	if keyFromToken("esc").Type != tea.KeyEsc {
		t.Fatal("esc token")
	}
	if keyFromToken("enter").Type != tea.KeyEnter {
		t.Fatal("enter token")
	}
}
