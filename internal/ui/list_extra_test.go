package ui

import (
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

func TestPauseGidCmd(t *testing.T) {
	a, fake := testApp(t)
	drain(t, a, a.pauseGidCmd("z1"))
	if len(fake.paused) != 1 || fake.paused[0] != "z1" {
		t.Fatalf("pauseGidCmd must pause: %v", fake.paused)
	}
	a.client = nil
	if a.pauseGidCmd("z") != nil {
		t.Fatal("nil client → nil cmd")
	}
}

func TestConnCellStyled(t *testing.T) {
	st := NewStyles(TokyoNight)
	if got := connCellStyled(st, rpc.Status{InfoHash: "h", Connections: "12", NumSeeders: "3"}); !strings.Contains(got, "12:3") {
		t.Fatalf("torrent cell = %q", got)
	}
	if got := connCellStyled(st, rpc.Status{Connections: "5"}); !strings.Contains(got, "5") {
		t.Fatalf("http cell = %q", got)
	}
	if got := connCellStyled(st, rpc.Status{}); !strings.Contains(got, "-") {
		t.Fatalf("idle cell = %q", got)
	}
}

func TestEnterAddsClipboardOnEmptyList(t *testing.T) {
	a, fake := testApp(t)
	a.snap = snapshot{} // empty list → enter offers a clipboard add
	withClipboardText(t, "https://mirror.example/x.iso")
	_, cmd := a.Update(key("enter"))
	drain(t, a, cmd)
	if len(fake.addedURIs) != 1 || fake.addedURIs[0][0] != "https://mirror.example/x.iso" {
		t.Fatalf("addedURIs = %v", fake.addedURIs)
	}
}

func TestEnterEmptyNoClipboardFlashes(t *testing.T) {
	a, _ := testApp(t)
	a.snap = snapshot{}
	_, cmd := a.Update(key("enter")) // TestMain clipboard stub returns ""
	if cmd == nil || !a.statusErr || !strings.Contains(a.status, "no link") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestReorderFlashOnAllTab(t *testing.T) {
	a, _ := testApp(t) // default tab is All
	_, cmd := a.Update(key("J"))
	if cmd == nil || !a.statusErr || !strings.Contains(a.status, "Waiting tab") {
		t.Fatalf("status = %q", a.status)
	}
	if a.list.reordering {
		t.Fatal("reorder must not start on the All tab")
	}
}

func TestPurgeOnAllTab(t *testing.T) {
	a, _ := testApp(t) // All tab
	_, _ = a.Update(key("D"))
	if a.overlay != overlayConfirm || a.confirm.yesLabel != "Clear (y)" {
		t.Fatalf("D on All must confirm: overlay=%d label=%q", a.overlay, a.confirm.yesLabel)
	}
}

func TestQuitConfirmLabel(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("q"))
	if a.overlay != overlayConfirm || a.confirm.yesLabel != "Quit (y)" {
		t.Fatalf("quit confirm label = %q", a.confirm.yesLabel)
	}
	if v := a.View(); !strings.Contains(v, "Quit (y)") {
		t.Fatal("quit confirm must render its label")
	}
}

func TestAllTabConcatenatesGroups(t *testing.T) {
	a, _ := testApp(t) // active 1 + waiting 3 + stopped 1
	if n := len(a.list.rows()); n != 5 {
		t.Fatalf("All rows = %d, want 5", n)
	}
	if v := a.list.view(); !strings.Contains(v, "All 5") {
		t.Fatalf("tab bar must show All count:\n%s", v)
	}
}
