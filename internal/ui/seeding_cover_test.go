package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/rpc"
)

// globalRecAPI additionally records ChangeGlobalOption payloads.
type globalRecAPI struct {
	*fakeAPI
	global map[string]string
}

func (f *globalRecAPI) ChangeGlobalOption(_ context.Context, opts map[string]string) error {
	f.global = opts
	return nil
}

func TestSeedingTrackersStart(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	if m.trackersStart() != 2+len(m.toggles) {
		t.Fatalf("trackersStart = %d", m.trackersStart())
	}
}

func TestSeedingLoadCmd(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	m.gid = "a1"
	cmd := m.loadCmd()
	if cmd == nil {
		t.Fatal("loadCmd with client must return a command")
	}
	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("msg = %#v", cmd())
	}
	var sawGID, sawGlobal bool
	for _, c := range batch {
		switch msg := c().(type) {
		case gidOptionsMsg:
			sawGID = msg.gid == "a1" && msg.err == nil
		case globalOptionsMsg:
			sawGlobal = msg.err == nil
		}
	}
	if !sawGID || !sawGlobal {
		t.Fatalf("gid=%v global=%v", sawGID, sawGlobal)
	}

	a.client = nil
	if m.loadCmd() != nil {
		t.Fatal("nil client must yield nil cmd")
	}
}

func TestSeedingAbsorbOptions(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	m.gid = "a1"

	// gid mismatch is ignored.
	m.absorbOptions(gidOptionsMsg{gid: "zz", opts: map[string]string{"seed-ratio": "9"}})
	if m.ratio.Value() != "" {
		t.Fatal("mismatched gid must be ignored")
	}

	// Ratio, time, and trackers from the bt-tracker CSV.
	m.absorbOptions(gidOptionsMsg{gid: "a1", opts: map[string]string{
		"seed-ratio": "2.0", "seed-time": "45", "bt-tracker": "t1,t2",
	}})
	if m.ratio.Value() != "2.0" || m.stime.Value() != "45" {
		t.Fatalf("ratio=%q stime=%q", m.ratio.Value(), m.stime.Value())
	}
	if len(m.trackers) != 2 || m.trackers[0] != "t1" {
		t.Fatalf("trackers = %v", m.trackers)
	}

	// No bt-tracker → fall back to the snapshot's announce list.
	a.snap.Active[0].BitTorrent = &rpc.BTInfo{AnnounceList: [][]string{{"u1", "u2"}, {"u3"}}}
	m.absorbOptions(gidOptionsMsg{gid: "a1", opts: map[string]string{}})
	if len(m.trackers) != 3 || m.trackers[2] != "u3" {
		t.Fatalf("trackers = %v", m.trackers)
	}

	// Dirty trackers are preserved.
	m.trackers = []string{"mine"}
	m.trackersDirty = true
	m.absorbOptions(gidOptionsMsg{gid: "a1", opts: map[string]string{"bt-tracker": "t9"}})
	if len(m.trackers) != 1 || m.trackers[0] != "mine" {
		t.Fatalf("trackers = %v", m.trackers)
	}
}

func TestSeedingAbsorbGlobal(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	m.absorbGlobal(map[string]string{"enable-dht": "true", "bt-require-crypto": "false"})
	if !m.toggles[0].on || m.toggles[3].on {
		t.Fatalf("toggles = %+v", m.toggles)
	}
}

func TestSeedingUpdateEscAndTabCycle(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)

	a.screen = screenSeeding
	m, _ = m.update(key("esc"))
	if a.screen != screenList {
		t.Fatal("esc must return to list")
	}
	a.screen = screenSeeding
	m, _ = m.update(key("q"))
	if a.screen != screenList {
		t.Fatal("q must return to list")
	}

	// tab: 0 → 1 (stime focus), → toggles, → trackers, → wrap to ratio.
	m.focus = 0
	m, cmd := m.update(key("tab"))
	if m.focus != 1 || cmd == nil {
		t.Fatalf("focus = %d", m.focus)
	}
	for want := 2; want <= m.trackersStart(); want++ {
		m, cmd = m.update(key("tab"))
		if m.focus != want || cmd != nil {
			t.Fatalf("focus = %d want %d", m.focus, want)
		}
	}
	m, cmd = m.update(key("tab")) // wrap
	if m.focus != 0 || cmd == nil {
		t.Fatalf("focus = %d", m.focus)
	}
}

func TestSeedingUpdateSpace(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	m.focus = 2
	m, cmd := m.update(key(" "))
	if m.toggles[0].on {
		t.Fatal("startup options are read-only; space must not toggle")
	}
	if cmd == nil || a.status == "" {
		t.Fatal("space on a startup option must flash an explanation")
	}
	// Guard: focus in trackers → not a toggle, falls through.
	a.status = ""
	m.focus = m.trackersStart()
	m, cmd = m.update(key(" "))
	if m.toggles[0].on || cmd != nil {
		t.Fatal("space outside toggles must be inert")
	}
	// Space over an input routes to the input update path.
	m.focus = 0
	m, _ = m.update(key(" "))
	m.focus = 1
	m, _ = m.update(key(" "))
}

func TestSeedingTypingIntoInputs(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	m.focus = 0
	m.ratio.Focus()
	m, _ = m.update(key("2"))
	if m.ratio.Value() != "2" {
		t.Fatalf("ratio = %q", m.ratio.Value())
	}
	m.ratio.Blur()
	m.focus = 1
	m.stime.Focus()
	m, _ = m.update(key("7"))
	if m.stime.Value() != "7" {
		t.Fatalf("stime = %q", m.stime.Value())
	}
}

func TestSeedingSaveFull(t *testing.T) {
	a, fake := testApp(t)
	rec := &globalRecAPI{fakeAPI: fake}
	a.client = rec
	m := newSeedingModel(a)
	m.gid = "a1"
	m.ratio.SetValue("2.0")
	m.stime.SetValue("30")
	m.trackers = []string{"x", "y"}
	m.trackersDirty = true
	m.toggles[0].on = true

	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	if cmd == nil {
		t.Fatal("ctrl+s must save")
	}
	msg := cmd()
	if dm, ok := msg.(actionDoneMsg); !ok || dm.err != nil || dm.text != "seeding settings saved" {
		t.Fatalf("msg = %#v", msg)
	}
	per := fake.changedOptions["a1"]
	if per["seed-ratio"] != "2.0" || per["seed-time"] != "30" || per["bt-tracker"] != "x,y" {
		t.Fatalf("perGID = %v", per)
	}
	if rec.global != nil {
		t.Fatalf("startup toggles must never reach changeGlobalOption, got %v", rec.global)
	}
}

// seedFailOptAPI fails per-download option changes.
type seedFailOptAPI struct {
	*fakeAPI
}

func (f *seedFailOptAPI) ChangeOption(context.Context, string, map[string]string) error {
	return errors.New("opt boom")
}

func TestSeedingSaveChangeOptionError(t *testing.T) {
	a, fake := testApp(t)
	a.client = &seedFailOptAPI{fakeAPI: fake}
	m := newSeedingModel(a)
	m.gid = "a1"
	m.ratio.SetValue("2.0")
	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	_ = m
	if cmd == nil {
		t.Fatal("ctrl+s must save")
	}
	msg := cmd()
	if dm, ok := msg.(actionDoneMsg); !ok || dm.err == nil {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestSeedingSaveEmptyPerGID(t *testing.T) {
	a, fake := testApp(t)
	rec := &globalRecAPI{fakeAPI: fake}
	a.client = rec
	m := newSeedingModel(a)
	m.gid = "a1"
	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	_ = m
	if cmd == nil {
		t.Fatal("ctrl+s must report something")
	}
	if a.status != "nothing to save" {
		t.Fatalf("status = %q", a.status)
	}
	if _, ok := fake.changedOptions["a1"]; ok {
		t.Fatal("blank fields must skip ChangeOption")
	}
	if rec.global != nil {
		t.Fatalf("no RPC expected, got global %v", rec.global)
	}
}

func TestSeedingTrackerCursorKeys(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	m.focus = m.trackersStart()
	m.trackers = []string{"t1", "t2"}

	m, _ = m.update(key("j"))
	if m.tCursor != 1 {
		t.Fatalf("tCursor = %d", m.tCursor)
	}
	m, _ = m.update(key("j")) // guard at end
	if m.tCursor != 1 {
		t.Fatalf("tCursor = %d", m.tCursor)
	}
	m, _ = m.update(key("k"))
	if m.tCursor != 0 {
		t.Fatalf("tCursor = %d", m.tCursor)
	}
	m, _ = m.update(key("k")) // guard at start
	if m.tCursor != 0 {
		t.Fatalf("tCursor = %d", m.tCursor)
	}
}

func TestSeedingTrackerRemove(t *testing.T) {
	a, _ := testApp(t)
	m := newSeedingModel(a)
	m.focus = m.trackersStart()
	m.trackers = []string{"t1", "t2"}
	m.tCursor = 1
	m, _ = m.update(key("-")) // remove last → cursor steps back
	if len(m.trackers) != 1 || m.trackers[0] != "t1" || m.tCursor != 0 || !m.trackersDirty {
		t.Fatalf("trackers=%v cursor=%d", m.trackers, m.tCursor)
	}
	m, _ = m.update(key("-")) // remove the only one, cursor stays 0
	if len(m.trackers) != 0 || m.tCursor != 0 {
		t.Fatalf("trackers=%v cursor=%d", m.trackers, m.tCursor)
	}
	m, _ = m.update(key("-")) // guard: nothing left
	if len(m.trackers) != 0 {
		t.Fatal("remove on empty must be inert")
	}
}

func TestSeedingTrackerAddPrompt(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenSeeding
	a.seeding.gid = "a1"
	a.seeding.focus = a.seeding.trackersStart()
	_, cmd := a.Update(key("+"))
	if a.overlay != overlayPrompt || cmd == nil {
		t.Fatal("+ must open the tracker prompt")
	}
	// Empty input is ignored.
	if a.prompt.onSubmit("   ") != nil {
		t.Fatal("empty submit must return nil cmd")
	}
	if len(a.seeding.trackers) != 0 {
		t.Fatalf("trackers = %v", a.seeding.trackers)
	}
	// Valid input appends.
	_ = a.prompt.onSubmit(" http://tr ")
	if len(a.seeding.trackers) != 1 || a.seeding.trackers[0] != "http://tr" || !a.seeding.trackersDirty {
		t.Fatalf("trackers = %v", a.seeding.trackers)
	}
}

func TestSeedingTrackerEditPrompt(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenSeeding
	a.seeding.gid = "a1"
	a.seeding.focus = a.seeding.trackersStart()
	a.seeding.trackers = []string{"old1", "old2"}
	a.seeding.tCursor = 1
	_, cmd := a.Update(key("e"))
	if a.overlay != overlayPrompt || cmd == nil {
		t.Fatal("e must open the edit prompt")
	}
	if a.prompt.input.Value() != "old2" {
		t.Fatalf("initial = %q", a.prompt.input.Value())
	}
	// Empty input is ignored.
	_ = a.prompt.onSubmit("  ")
	if a.seeding.trackers[1] != "old2" {
		t.Fatalf("trackers = %v", a.seeding.trackers)
	}
	// Valid input replaces in place.
	_ = a.prompt.onSubmit("new2")
	if a.seeding.trackers[1] != "new2" || !a.seeding.trackersDirty {
		t.Fatalf("trackers = %v", a.seeding.trackers)
	}
	// Stale index: the list shrank while the prompt was open.
	onSubmit := a.prompt.onSubmit
	a.seeding.trackers = []string{"only"}
	_ = onSubmit("late")
	if len(a.seeding.trackers) != 1 || a.seeding.trackers[0] != "only" {
		t.Fatalf("trackers = %v", a.seeding.trackers)
	}

	// Cursor beyond the list: e is inert.
	a.overlay = overlayNone
	a.seeding.tCursor = 9
	_, cmd = a.Update(key("e"))
	if a.overlay != overlayNone || cmd != nil {
		t.Fatal("e out of range must be inert")
	}
}

func TestSeedingViewVariants(t *testing.T) {
	a, _ := testApp(t)
	a.snap.Active[0].UploadLength = "75" // ratio 1.5
	m := newSeedingModel(a)
	m.gid = "a1"
	m.name = "payload"
	m.ratio.SetValue("2.0")
	m.toggles[0].on = true
	m.focus = 0
	m.trackers = []string{"t1", "t2"}
	if out := m.view(); !strings.Contains(out, "1.50 / 2.0") || !strings.Contains(out, "TRACKERS") {
		t.Fatalf("out = %q", out)
	}

	// Toggle focus marker and tracker cursor.
	m.focus = 2
	if out := m.view(); !strings.Contains(out, "DHT ◂") {
		t.Fatalf("out = %q", out)
	}
	m.focus = m.trackersStart()
	m.tCursor = 1
	if out := m.view(); !strings.Contains(out, "▸ ") {
		t.Fatalf("out = %q", out)
	}

	// Unknown gid, no target, no trackers.
	m2 := newSeedingModel(a)
	m2.gid = "zz"
	m2.focus = 1
	if out := m2.view(); !strings.Contains(out, "0.00 / ∞") || !strings.Contains(out, "no trackers") {
		t.Fatalf("out = %q", out)
	}
}

func TestParseFloatAndOrDefault(t *testing.T) {
	if parseFloat("1.5") != 1.5 || parseFloat("junk") != 0 {
		t.Fatal("parseFloat")
	}
	if orDefault("", "d") != "d" || orDefault("x", "d") != "x" {
		t.Fatal("orDefault")
	}
}

func TestStatusByGIDAllGroups(t *testing.T) {
	a, _ := testApp(t)
	for _, gid := range []string{"a1", "w1", "s1"} {
		if s, ok := a.statusByGID(gid); !ok || s.GID != gid {
			t.Fatalf("statusByGID(%q) = %v %v", gid, s, ok)
		}
	}
	if _, ok := a.statusByGID("zz"); ok {
		t.Fatal("unknown gid must miss")
	}
}
