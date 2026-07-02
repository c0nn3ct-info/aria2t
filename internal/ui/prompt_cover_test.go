package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPromptEscCloses(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayPrompt
	m := newPromptModel(a, "title", "seed", nil)
	m, cmd := m.update(key("esc"))
	if a.overlay != overlayNone || cmd != nil {
		t.Fatalf("esc must close overlay, overlay=%d", a.overlay)
	}
	if m.input.Value() != "seed" {
		t.Fatalf("initial value = %q", m.input.Value())
	}
}

func TestPromptEnterSubmits(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayPrompt
	var got string
	m := newPromptModel(a, "title", "", func(v string) tea.Cmd {
		got = v
		return a.flash("stored", false)
	})
	if cmd := m.focusCmd(); cmd == nil {
		t.Fatal("focusCmd must not be nil")
	}
	m.input.Focus() // focusCmd works on a copy; focus this instance for typing
	m, _ = m.update(key("x"))
	m, cmd := m.update(key("enter"))
	if a.overlay != overlayNone {
		t.Fatal("enter must close overlay")
	}
	if got != "x" {
		t.Fatalf("submitted value = %q", got)
	}
	if cmd == nil {
		t.Fatal("onSubmit cmd must propagate")
	}
}

func TestPromptEnterWithoutOnSubmit(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayPrompt
	m := newPromptModel(a, "title", "", nil)
	_, cmd := m.update(key("enter"))
	if a.overlay != overlayNone || cmd != nil {
		t.Fatalf("bare enter: overlay=%d cmd=%v", a.overlay, cmd)
	}
}

func TestPromptView(t *testing.T) {
	a, _ := testApp(t)
	m := newPromptModel(a, "Checksum", "", nil)
	if v := m.view(); v == "" {
		t.Fatal("empty prompt view")
	}
}
