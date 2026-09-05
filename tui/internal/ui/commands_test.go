package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandPaletteKeyboardAndFiltering(t *testing.T) {
	a, _ := testApp(t)
	_, cmd := a.Update(ctrl(tea.KeyCtrlP))
	if a.overlay != overlayCommands || cmd == nil {
		t.Fatal("ctrl+p did not open and focus the command palette")
	}
	_, _ = a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if a.overlay != overlayNone {
		t.Fatal("root key routing did not close the palette")
	}
	a.overlay = overlayCommands
	m := a.commands
	if len(m.entries()) != 8 {
		t.Fatalf("commands = %d", len(m.entries()))
	}
	m.query.SetValue("stats")
	if got := m.entries(); len(got) != 1 || got[0].name != "Global stats" {
		t.Fatalf("filtered commands = %#v", got)
	}
	m.cursor = 4
	m.clamp()
	if m.cursor != 0 {
		t.Fatalf("clamped cursor = %d", m.cursor)
	}
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyEnter})
	if a.overlay != overlayNone || a.screen != screenStats {
		t.Fatal("palette did not dispatch the existing semantic action")
	}

	a.overlay = overlayCommands
	m = newCommandModel(a)
	_ = m.focusCmd()
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyUp})
	m.cursor = -1
	m.clamp()
	if m.cursor != 0 {
		t.Fatal("negative command cursor was not clamped")
	}
	m, _ = m.update(key("sett"))
	if !strings.Contains(m.query.Value(), "sett") || m.cursor != 0 {
		t.Fatal("command query did not accept text")
	}
	m.query.SetValue("no such command")
	m.cursor = 9
	m.clamp()
	if m.cursor != 0 {
		t.Fatal("empty command list cursor was not clamped")
	}
	if next, cmd := m.choose(); cmd != nil || next.cursor != m.cursor {
		t.Fatal("empty result should be inert")
	}
	m, _ = m.update(ctrl(tea.KeyCtrlP))
	if a.overlay != overlayNone {
		t.Fatal("ctrl+p did not close the palette")
	}
}

func TestCommandPaletteMouseAndViews(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayCommands
	m := newCommandModel(a)
	a.commands = m
	out := a.View()
	if !strings.Contains(out, "Commands") || !strings.Contains(out, "Add download") {
		t.Fatalf("palette view:\n%s", out)
	}
	if _, ok := commandRegion(a, "cmd:0"); !ok {
		t.Fatal("command row has no mouse target")
	}
	m.query.SetValue("nothing")
	if out = m.view(); !strings.Contains(out, "No matching commands") {
		t.Fatal("empty search state missing")
	}
	m.query.SetValue("stats")
	m, _ = m.mouse("cmd:0")
	if a.screen != screenStats {
		t.Fatal("command-row click did not execute")
	}
	a.overlay = overlayCommands
	m, _ = m.mouse("cmd:999")
	m, _ = m.mouse("btn:esc")
	if a.overlay != overlayNone {
		t.Fatal("close button did not close palette")
	}
	if next, cmd := m.mouse("other:x"); cmd != nil || next.cursor != m.cursor {
		t.Fatal("unknown palette mouse target changed state")
	}
}

func TestCommandPaletteRoutesThroughAppMouse(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayCommands
	a.commands = newCommandModel(a)
	_ = a.View()
	r, ok := commandRegion(a, "cmd:1")
	if !ok {
		t.Fatal("missing command region")
	}
	_, _ = a.Update(tea.MouseMsg{X: r.x0, Y: r.y0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if a.screen != screenStats {
		t.Fatal("root mouse routing did not execute command")
	}
}

func commandRegion(a *App, id string) (region, bool) {
	for _, r := range a.hits.regions {
		if r.id == id {
			return r, true
		}
	}
	return region{}, false
}
