package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/config"
)

// Regressions caught by driving the real TUI (tmux e2e): initial focus was
// applied to a copy of the model, so overlays opened with dead inputs; and
// textinput values set before Width bypassed the overflow window, blowing
// modals past the terminal edge.

func TestAddOverlayTypeableOnOpen(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("a"))
	if a.overlay != overlayAdd {
		t.Fatalf("a must open the add overlay, got %d", a.overlay)
	}
	_, _ = a.Update(key("x"))
	if got := a.add.uris.Value(); got != "x" {
		t.Fatalf("typing after open must land in the URL field, got %q", got)
	}
}

func TestChecksumPromptTypeableOnOpen(t *testing.T) {
	a, _ := testApp(t)
	a.list.tab = tabStopped
	_, _ = a.Update(key("c"))
	if a.overlay != overlayPrompt {
		t.Fatalf("c must open the checksum prompt, got %d", a.overlay)
	}
	_, _ = a.Update(key("f"))
	if got := a.prompt.input.Value(); got != "f" {
		t.Fatalf("typing after open must land in the prompt input, got %q", got)
	}
}

func TestSeedingRatioTypeableOnOpen(t *testing.T) {
	a, _ := testApp(t)
	a.snap.Active[0].InfoHash = "deadbeef"
	_, cmd := a.Update(key("t"))
	drain(t, a, cmd)
	if a.screen != screenSeeding {
		t.Fatalf("t on a torrent must open seeding, got %d", a.screen)
	}
	_, _ = a.Update(key("9"))
	if got := a.seeding.ratio.Value(); got != "9" {
		t.Fatalf("typing after open must land in the ratio input, got %q", got)
	}
}

func TestAddModalWidthIndependentOfDirLength(t *testing.T) {
	short := NewApp(config.Default(), t.TempDir()+"/config.json")
	long := NewApp(config.Default(), t.TempDir()+"/config.json")
	long.cfg.Dir = "/very/long/path/" + strings.Repeat("sub/", 30) + "downloads"
	long.add = newAddModel(long)
	short.add = newAddModel(short)
	ws, wl := lipgloss.Width(short.add.view()), lipgloss.Width(long.add.view())
	if wl != ws {
		t.Fatalf("modal width must not grow with dir length: short=%d long=%d", ws, wl)
	}
}

func TestSettingsViewStaysInsideTerminal(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.Servers[0].Secret = strings.Repeat("s3cret-", 30)
	a.settings = newSettingsModel(a)
	for i, line := range strings.Split(a.settings.view(), "\n") {
		if w := lipgloss.Width(line); w > a.width {
			t.Fatalf("settings line %d is %d cells wide, terminal is %d", i, w, a.width)
		}
	}
}

func TestFlashVisibleWithinTerminalWidth(t *testing.T) {
	a, _ := testApp(t)
	a.status, a.statusErr = "limits applied", false
	found := false
	for _, line := range strings.Split(a.View(), "\n") {
		if !strings.Contains(line, "limits applied") {
			continue
		}
		found = true
		if w := lipgloss.Width(line); w > a.width {
			t.Fatalf("status line is %d cells wide, terminal is %d — flash would be clipped", w, a.width)
		}
	}
	if !found {
		t.Fatal("flash text missing from the rendered view")
	}
}

func TestStripHeightEmptyStoppedTab(t *testing.T) {
	a, _ := testApp(t)
	a.snap.Stopped = nil
	a.list.tab = tabStopped
	if h := a.list.stripHeight(); h != 0 {
		t.Fatalf("no selection → no strip, got %d", h)
	}
}

func TestOverlayDimsBackdrop(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("a"))
	v := a.View()
	if !strings.Contains(v, "Add download") {
		t.Fatal("modal missing from the view")
	}
	if !strings.Contains(v, "ubuntu.iso") {
		t.Fatal("the screen beneath the modal must stay visible (dimmed), not a black void")
	}
	for i, line := range strings.Split(v, "\n") {
		if w := lipgloss.Width(line); w > a.width {
			t.Fatalf("line %d is %d cells, terminal is %d", i, w, a.width)
		}
	}
}

func TestCompositeTinyTerminal(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 30, 5 // narrower than the list rows, shorter than the modal
	_, _ = a.Update(key("a"))
	lines := strings.Split(a.View(), "\n")
	if len(lines) != 5 {
		t.Fatalf("composite must emit exactly the terminal height, got %d lines", len(lines))
	}
}
