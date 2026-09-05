package ui

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
	"aria2t/internal/rpc"
)

func TestThemeTextContrast(t *testing.T) {
	for _, p := range []Palette{TokyoNight, TokyoNightDay} {
		for name, colour := range map[string]string{
			"foreground": string(p.Fg), "bright": string(p.FgBright), "dim": string(p.FgDim),
			"faint": string(p.FgFaint), "accent": string(p.Accent), "green": string(p.Green),
			"red": string(p.Red), "yellow": string(p.Yellow), "cyan": string(p.Cyan), "magenta": string(p.Magenta),
		} {
			if ratio := contrastRatio(colour, string(p.Bg)); ratio < 4.5 {
				t.Errorf("%s %s contrast %.2f < 4.5", p.Name, name, ratio)
			}
		}
	}
}

func contrastRatio(a, b string) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(hex string) float64 {
	component := func(i int) float64 {
		var value uint64
		_, _ = fmt.Sscanf(hex[i:i+2], "%x", &value)
		v := float64(value) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*component(1) + 0.7152*component(3) + 0.0722*component(5)
}

func TestSafeTerminalTextAndUnicodeCells(t *testing.T) {
	unsafe := "ok\x1b[2J\nnext\t\x00end"
	if got := safeText(unsafe); got != "ok next end" {
		t.Fatalf("safeText = %q", got)
	}
	if got := asciiText("✓ ┌─┐ ▸ █▓░ ∞"); strings.ContainsAny(got, "✓┌─┐▸█▓░∞") {
		t.Fatalf("ASCII fallback leaked decorative glyphs: %q", got)
	}
	if got := cellWidth("界e\u0301"); got != 3 {
		t.Fatalf("cell width = %d", got)
	}
	if got := pad("界e\u0301", 4); cellWidth(got) != 4 {
		t.Fatalf("cell-aware pad width = %d (%q)", cellWidth(got), got)
	}
	if got := trunc("x", 0); got != "" {
		t.Fatalf("zero truncation = %q", got)
	}
	if got := pad("x", 0); got != "" {
		t.Fatalf("zero pad = %q", got)
	}
}

func TestSafePasteAndRPCData(t *testing.T) {
	plain := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
	if got := safeKey(plain); got.String() != "x" {
		t.Fatalf("ordinary key changed: %q", got.String())
	}
	paste := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a\r\nb\t\x00\x1b[2Jc"), Paste: true}
	got := safeKey(paste)
	if string(got.Runes) != "a\n\nb\tc" || !got.Paste {
		t.Fatalf("safe paste = %q paste=%v", string(got.Runes), got.Paste)
	}

	bt := &rpc.BTInfo{Comment: "c\x1b[2J", Mode: "m\n"}
	bt.Info.Name = "n\x00"
	bt.AnnounceList = [][]string{{"https://x/\x1b[2J"}}
	s := rpc.Status{
		GID: "g\x00", Status: "active\n", InfoHash: "h\t", ErrorCode: "1\r",
		ErrorMessage: "bad\x1b[31m", Following: "f\x00", FollowedBy: []string{"c\n"},
		Dir: "/tmp\x1b[2J", Files: []rpc.File{{Path: "p\n", URIs: []rpc.URI{{URI: "u\x1b[2J", Status: "used\n"}}}},
		BitTorrent: bt,
	}
	clean := safeStatus(s)
	if strings.ContainsAny(clean.GID+clean.Status+clean.ErrorMessage+clean.Dir+clean.Files[0].Path+clean.BitTorrent.Info.Name, "\x00\x1b\n\r\t") {
		t.Fatalf("status was not neutralized: %#v", clean)
	}
	snap := safeSnapshot(snapshot{Active: []rpc.Status{s}, Waiting: []rpc.Status{s}, Stopped: []rpc.Status{s}})
	if strings.Contains(snap.Active[0].ErrorMessage, "\x1b") || strings.Contains(snap.Waiting[0].Dir, "\x1b") || strings.Contains(snap.Stopped[0].GID, "\x00") {
		t.Fatal("snapshot was not neutralized")
	}
	detail := safeDetailData(detailDataMsg{
		status:  s,
		peers:   []rpc.Peer{{PeerID: "id\n", IP: "ip\x1b[2J", Port: "p\t"}},
		servers: []rpc.ServerStat{{Servers: []rpc.ServerInfo{{URI: "u\n", CurrentURI: "c\x1b[2J"}}}},
	})
	if strings.ContainsAny(detail.peers[0].PeerID+detail.peers[0].IP+detail.peers[0].Port+detail.servers[0].Servers[0].CurrentURI, "\x1b\n\t") {
		t.Fatal("detail data was not neutralized")
	}
}

func TestAccessibleOutputAndEventHistory(t *testing.T) {
	a, _ := testApp(t)
	a.SetAccessible(true)
	for i := 0; i < 55; i++ {
		a.flash("event", i == 54)
	}
	if len(a.events) != 50 {
		t.Fatalf("event history = %d", len(a.events))
	}
	out := a.outputView(a.styles.Green.Render("✓ ready"))
	if strings.Contains(out, "\x1b") || !strings.Contains(out, "[OK] ready") || !strings.Contains(out, "Activity log") || !strings.Contains(out, "ERROR: event") {
		t.Fatalf("accessible output:\n%s", out)
	}
	a.SetAccessible(false)
	styled := a.styles.Green.Render("ok")
	if a.outputView(styled) != styled {
		t.Fatal("normal output changed")
	}
}

func TestMinimumViewportAndStalePoll(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 40, 6
	if got := strings.Split(a.View(), "\n"); len(got) != 6 || got[0] != "Terminal too small" {
		t.Fatalf("fallback = %#v", got)
	}
	_, cmd := a.handleKey(key("j"))
	if cmd != nil || a.list.cursor != 0 {
		t.Fatal("hidden UI accepted input")
	}
	if _, cmd = a.handleKey(key("q")); cmd == nil {
		t.Fatal("q must quit the fallback")
	}
	a.width, a.height = 0, 0
	if a.tooSmallView() != "" {
		t.Fatal("zero-sized viewport must render nothing")
	}
	a.pollSeq = 2
	a.polling = true
	_, _ = a.Update(pollMsg{seq: 1, snap: snapshot{Active: []rpc.Status{{GID: "stale"}}}})
	if !a.polling || len(a.snap.Active) != 1 || a.snap.Active[0].GID == "stale" {
		t.Fatal("stale poll was accepted")
	}
}

func TestPollingCadenceAndResizeClamps(t *testing.T) {
	a, _ := testApp(t)
	if got := a.pollInterval(); got != time.Second {
		t.Fatalf("active cadence = %s", got)
	}
	a.snap = snapshot{}
	a.verify["v"] = &verifyState{Running: true}
	if got := a.pollInterval(); got != time.Second {
		t.Fatalf("verification cadence = %s", got)
	}
	a.verify["v"].Running = false
	if got := a.pollInterval(); got != 5*time.Second {
		t.Fatalf("idle cadence = %s", got)
	}
	a.accessible = true
	if got := a.pollInterval(); got != 10*time.Second {
		t.Fatalf("accessible cadence = %s", got)
	}

	a.files = newFilesModel(a)
	a.files.rows = []*treeNode{{name: "one"}}
	a.files.cursor = 9
	a.browse = newBrowseModelAsync(a, "/tmp", nil)
	a.browse.entries = []browseEntry{{name: "one"}}
	a.browse.cursor = 9
	a.cfg.Rules = []config.Rule{{Label: "one"}}
	a.scheduler.cursor = 9
	a.seeding.trackers = []string{"one"}
	a.seeding.tCursor = 9
	_, _ = a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if a.files.cursor != 0 || a.browse.cursor != 0 || a.scheduler.cursor != 0 || a.seeding.tCursor != 0 {
		t.Fatalf("resize clamps: files=%d browse=%d scheduler=%d trackers=%d", a.files.cursor, a.browse.cursor, a.scheduler.cursor, a.seeding.tCursor)
	}
}

func TestCompactLayoutsAndEdgeGeometry(t *testing.T) {
	a, _ := testApp(t)
	a.width, a.height = 80, 24
	a.settings = newSettingsModel(a)
	a.screen = screenSettings
	if out := a.View(); !strings.Contains(out, "Connection") || !strings.Contains(out, "Built-in daemon") {
		t.Fatalf("compact managed settings:\n%s", out)
	}
	a.cfg.Servers[0].Managed = false
	a.settings = newSettingsModel(a)
	a.settings.section = 3
	a.settings.inSide = false
	a.settings.focus = 0
	a.settings.fields[3][0].on = true
	_ = a.View()
	a.settings.focus = 1
	_ = a.View()
	a.settings.section = 4
	a.settings.focus = 0
	_ = a.View()

	a.height = 0
	_ = a.screenFrame("x", nil)
	a.width, a.height = 3, 1
	_ = a.composite("abcdef", "one\ntwo\nthree")
}

func TestAsyncAddAndBrowseStates(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	m.submitting = true
	if next, cmd := m.update(key("x")); cmd != nil || !next.submitting {
		t.Fatal("busy add form accepted input")
	}
	if out := m.view(); !strings.Contains(out, "Working: reading and adding") {
		t.Fatal("busy state is not visible")
	}

	m = newAddModel(a)
	m.tab = addTabInput
	m.file.SetValue("/tmp/list")
	a.client = nil
	if _, cmd := m.submit(); cmd == nil || !a.statusErr {
		t.Fatal("input add without client must fail visibly")
	}
	a.client = newFakeAPI()
	m.tab = addTabMetalink
	m.file.SetValue(filepath.Join(t.TempDir(), "missing.meta4"))
	_, cmd := m.submit()
	if msg, ok := cmd().(metalinkAddedMsg); !ok || msg.err == nil {
		t.Fatalf("missing metalink result = %#v", msg)
	}

	b := newBrowseModelAsync(a, "/tmp", nil)
	if out := b.view(); !strings.Contains(out, "loading directory") {
		t.Fatal("browser loading state is not visible")
	}
}

func TestOpenDirAbsolutePathFailure(t *testing.T) {
	a, _ := testApp(t)
	orig := absolutePath
	absolutePath = func(string) (string, error) { return "", errors.New("no cwd") }
	t.Cleanup(func() { absolutePath = orig })
	if cmd := a.openDir("relative"); cmd == nil || !strings.Contains(a.status, "no cwd") {
		t.Fatalf("absolute path error status = %q", a.status)
	}
}

func TestAsyncBrowseIgnoresOldDirectory(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayBrowse
	a.browse = newBrowseModelAsync(a, "/new", nil)
	_, _ = a.Update(browseDataMsg{dir: "/old", entries: []browseEntry{{name: "old"}}})
	if len(a.browse.entries) != 0 {
		t.Fatal("stale directory listing was accepted")
	}
	_, _ = a.Update(browseDataMsg{dir: "/new", entries: []browseEntry{{name: "new"}}})
	if len(a.browse.entries) != 1 || a.browse.loading {
		t.Fatal("current directory listing was not absorbed")
	}
}

func TestAsyncInputFilePreservesFormOnError(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayAdd
	a.add.submitting = true
	_, _ = a.Update(addBatchDoneMsg{err: os.ErrNotExist})
	if a.overlay != overlayAdd || a.add.submitting || !a.statusErr {
		t.Fatal("failed add did not preserve the form")
	}
	_, _ = a.Update(addBatchDoneMsg{text: "added"})
	if a.overlay != overlayNone {
		t.Fatal("successful batch add did not close the form")
	}
}

func TestReverseFocusNavigation(t *testing.T) {
	a, _ := testApp(t)
	add := newAddModel(a)
	add.rename = true
	add.focus = 0
	add, _ = add.update(key("shift+tab"))
	if add.focus != 3 {
		t.Fatalf("add reverse focus = %d", add.focus)
	}

	settings := newSettingsModel(a)
	settings.inSide = false
	settings.focus = 2
	settings, _ = settings.update(key("shift+tab"))
	if settings.inSide || settings.focus != 1 {
		t.Fatalf("settings reverse focus: sidebar=%v focus=%d", settings.inSide, settings.focus)
	}
	settings.section = 3
	settings.focus = 2
	settings, _ = settings.update(key("shift+tab"))
	if settings.focus != 1 {
		t.Fatalf("settings reverse focus to toggle = %d", settings.focus)
	}

	seeding := newSeedingModel(a)
	seeding.focus = 2
	seeding, _ = seeding.update(key("shift+tab"))
	if seeding.focus != 1 || !seeding.stime.Focused() {
		t.Fatalf("seeding reverse focus = %d", seeding.focus)
	}
	seeding, _ = seeding.update(key("shift+tab"))
	if seeding.focus != 0 || !seeding.ratio.Focused() {
		t.Fatalf("seeding reverse focus to ratio = %d", seeding.focus)
	}
	seeding, _ = seeding.update(key("shift+tab"))
	if seeding.focus != seeding.trackersStart() {
		t.Fatalf("seeding reverse wrap = %d", seeding.focus)
	}

	servers := newServersModel(a)
	servers.editing = true
	servers.formFoc = 0
	servers, _ = servers.updateForm(key("shift+tab"))
	if servers.formFoc != len(servers.form)-1 {
		t.Fatalf("server reverse focus = %d", servers.formFoc)
	}

	throttle := newThrottleModel(a)
	throttle.gid, throttle.name = "g", "name"
	throttle, _ = throttle.update(key("shift+tab"))
	if throttle.row != 1 {
		t.Fatalf("throttle reverse focus = %d", throttle.row)
	}

	list := newListModel(a)
	list, _ = list.update(key("shift+tab"))
	if list.tab != 3 {
		t.Fatalf("list reverse tab = %d", list.tab)
	}
}

func TestURLAddWithoutClientPreservesForm(t *testing.T) {
	a, _ := testApp(t)
	a.client = nil
	a.overlay = overlayAdd
	m := newAddModel(a)
	m.uris.SetValue("https://example.test/file")
	m, cmd := m.submit()
	if cmd == nil || m.submitting || a.overlay != overlayAdd || !a.statusErr {
		t.Fatalf("disconnected URL add: busy=%v overlay=%d status=%q", m.submitting, a.overlay, a.status)
	}
}
