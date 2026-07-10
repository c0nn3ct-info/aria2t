package ui

import (
	"strings"
	"testing"
)

func TestThrottleLoadCmd(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)
	m.gid = "g1"
	cmd := m.loadCmd()
	if cmd == nil {
		t.Fatal("loadCmd with client must return a command")
	}
	msg, ok := cmd().(gidOptionsMsg)
	if !ok || msg.gid != "g1" || msg.err != nil || msg.opts == nil {
		t.Fatalf("msg = %#v", msg)
	}

	a.client = nil
	if m.loadCmd() != nil {
		t.Fatal("nil client must yield nil cmd")
	}
}

func TestThrottleAbsorbMismatchAndEmptyDefault(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)
	m.gid = "g1"
	m.downSel = 3
	m.absorbOptions(gidOptionsMsg{gid: "other", opts: map[string]string{"max-download-limit": "1M"}})
	if m.downSel != 3 {
		t.Fatal("gid mismatch must be ignored")
	}
	// Empty values default to "0" → first preset on both rows.
	m.absorbOptions(gidOptionsMsg{gid: "g1", opts: map[string]string{}})
	if m.downSel != 0 || m.upSel != 0 {
		t.Fatalf("sel = %d/%d", m.downSel, m.upSel)
	}
}

func TestThrottlePresetsSelectionSetSelection(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)
	if len(m.presets(0)) != len(downPresets) || len(m.presets(1)) != len(upPresets) {
		t.Fatal("presets rows mixed up")
	}
	m.setSelection(0, 2)
	m.setSelection(1, 1)
	if m.selection(0) != 2 || m.selection(1) != 1 {
		t.Fatalf("selection = %d/%d", m.selection(0), m.selection(1))
	}
}

func TestThrottleUpdateEditingSubmodes(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)
	m.gid = "g1"

	// Typing routes into the custom input.
	m.editing = true
	m.row = 0
	m.custom[0].Focus()
	m, _ = m.update(key("5"))
	if !strings.Contains(m.custom[0].Value(), "5") {
		t.Fatalf("custom = %q", m.custom[0].Value())
	}
	// esc leaves editing without applying.
	m, cmd := m.update(key("esc"))
	if m.editing || cmd != nil {
		t.Fatal("esc must leave editing")
	}
	// tab leaves editing too.
	m.editing = true
	m, cmd = m.update(key("tab"))
	if m.editing || cmd != nil {
		t.Fatal("tab must leave editing")
	}
	// enter leaves editing and applies.
	m.editing = true
	m.downSel = len(downPresets)
	m.custom[0].SetValue("8M")
	m, cmd = m.update(key("enter"))
	if m.editing || cmd == nil || a.overlay != overlayNone {
		t.Fatal("enter must apply")
	}
}

func TestThrottleUpdateNavigation(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)

	a.overlay = overlayThrottle
	m, _ = m.update(key("esc"))
	if a.overlay != overlayNone {
		t.Fatal("esc must close overlay")
	}

	m, _ = m.update(key("tab"))
	if m.row != 1 {
		t.Fatalf("row = %d", m.row)
	}
	m, _ = m.update(key("j"))
	if m.row != 0 {
		t.Fatalf("row = %d", m.row)
	}
	m, _ = m.update(key("k"))
	if m.row != 1 {
		t.Fatalf("row = %d", m.row)
	}
	m.row = 0

	// h guard at the left end.
	m.downSel = 0
	m, _ = m.update(key("h"))
	if m.downSel != 0 {
		t.Fatal("h at 0 must not move")
	}
	// l moves right, h moves back.
	m, _ = m.update(key("l"))
	if m.downSel != 1 {
		t.Fatalf("downSel = %d", m.downSel)
	}
	m, _ = m.update(key("h"))
	if m.downSel != 0 {
		t.Fatalf("downSel = %d", m.downSel)
	}
	// l guard at the right end (custom chip).
	m.downSel = len(downPresets)
	m, _ = m.update(key("l"))
	if m.downSel != len(downPresets) {
		t.Fatal("l at end must not move")
	}
	// Second row uses upSel.
	m.row = 1
	m.upSel = 0
	m, _ = m.update(key("l"))
	if m.upSel != 1 {
		t.Fatalf("upSel = %d", m.upSel)
	}
	m, _ = m.update(key("h"))
	if m.upSel != 0 {
		t.Fatalf("upSel = %d", m.upSel)
	}
	// Unknown key falls through.
	m, cmd := m.update(key("x"))
	if cmd != nil {
		t.Fatal("unknown key must be inert")
	}
}

func TestThrottleEnterOnCustomEntersEditing(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)
	m.row = 0
	m.downSel = len(downPresets)
	m, cmd := m.update(key("enter"))
	if !m.editing || cmd == nil {
		t.Fatal("enter on custom chip must enter editing")
	}
}

func TestThrottleEnterOnPresetApplies(t *testing.T) {
	a, fake := testApp(t)
	m := newThrottleModel(a)
	m.gid = "g1"
	m.downSel = 2 // "5M"
	m.upSel = 1   // "256K"
	a.overlay = overlayThrottle
	m, cmd := m.update(key("enter"))
	if cmd == nil || a.overlay != overlayNone {
		t.Fatal("enter on preset must apply and close")
	}
	msg := cmd()
	if dm, ok := msg.(actionDoneMsg); !ok || dm.err != nil || dm.text != "limits applied" {
		t.Fatalf("msg = %#v", msg)
	}
	got := fake.changedOptions["g1"]
	if got["max-download-limit"] != "5M" || got["max-upload-limit"] != "256K" {
		t.Fatalf("opts = %v", got)
	}
}

func TestThrottleApplyErrorFlashes(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)
	m.downSel = len(downPresets)
	m.custom[0].SetValue("warp9")
	m.row = 1 // enter on the preset row applies; values() fails on the junk custom
	m.upSel = 0
	m, cmd := m.update(key("enter"))
	_ = m
	if cmd == nil || !a.statusErr || a.status == "" {
		t.Fatalf("junk custom must flash an error, status = %q", a.status)
	}
}

func TestThrottleChipRowCombinations(t *testing.T) {
	a, _ := testApp(t)
	m := newThrottleModel(a)

	// Selected preset, empty custom, cursor on this row.
	m.row = 0
	m.downSel = 1
	if out := m.chipRow(0, "Download limit"); !strings.Contains(out, "Download limit") {
		t.Fatalf("out = %q", out)
	}
	// Selected custom, not editing, with a value; cursor elsewhere.
	m.downSel = len(downPresets)
	m.custom[0].SetValue("8M")
	m.row = 1
	if out := m.chipRow(0, "dl"); !strings.Contains(out, "8M") {
		t.Fatalf("out = %q", out)
	}
	// Selected custom while editing on this row.
	m.editing = true
	m.row = 0
	if out := m.chipRow(0, "dl"); out == "" {
		t.Fatal("editing chip row empty")
	}
	// Custom not selected but holding a value.
	m.editing = false
	m.downSel = 0
	if out := m.chipRow(0, "dl"); !strings.Contains(out, "8M") {
		t.Fatalf("out = %q", out)
	}
	// View renders both rows.
	if m.view() == "" {
		t.Fatal("view empty")
	}
}
