package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHelpScrollAndClose(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 100, 8 // short: availRows floors at 3, content windows
	a.help = newHelpModel(a)
	a.overlay = overlayHelp

	a.help, _ = a.help.update(key("j")) // scroll down
	if a.help.offset != 1 {
		t.Fatalf("j must scroll down, offset=%d", a.help.offset)
	}
	a.help, _ = a.help.update(key("k")) // scroll up
	if a.help.offset != 0 {
		t.Fatalf("k must scroll up, offset=%d", a.help.offset)
	}
	// k at the top clamps to 0.
	a.help, _ = a.help.update(key("k"))
	if a.help.offset != 0 {
		t.Fatalf("k at top must stay 0, offset=%d", a.help.offset)
	}
	// Many downs clamp at the max offset (never past the content).
	for i := 0; i < 200; i++ {
		a.help, _ = a.help.update(key("j"))
	}
	maxOff := len(a.help.contentLines()) - a.help.availRows()
	if a.help.offset != maxOff {
		t.Fatalf("offset must clamp to maxOff %d, got %d", maxOff, a.help.offset)
	}
	// The windowed view shows the scroll hint.
	if !strings.Contains(ansi.Strip(a.help.view()), "scroll") {
		t.Fatal("windowed help must show a scroll hint")
	}

	// A tall terminal shows everything (no scroll hint) and any key closes.
	a.height = 40
	if strings.Contains(ansi.Strip(a.help.view()), "↑↓ scroll") {
		t.Fatal("full-height help must not show a scroll hint")
	}
	a.help, _ = a.help.update(key("x"))
	if a.overlay != overlayNone {
		t.Fatalf("any other key must close help, overlay=%d", a.overlay)
	}
}
