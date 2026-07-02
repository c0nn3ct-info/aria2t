package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
)

// settingsTestApp swaps in an unmanaged active server and rebuilds the
// settings model against it.
func settingsTestApp(t *testing.T) (*App, *fakeAPI) {
	t.Helper()
	a, fake := testApp(t)
	a.cfg.Servers = []config.Server{{Name: "ext", Host: "h", Port: 42, Secret: "s", Protocol: "ws"}}
	a.cfg.Active = 0
	a.settings = newSettingsModel(a)
	a.dial = func(srv config.Server) (api, string, error) { return fake, "v", nil }
	return a, fake
}

func TestNewSettingsModelProtocolAndTheme(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a) // managed built-in server defaults to ws
	if !m.fields[0][3].on || m.fields[4][0].on {
		t.Fatalf("toggles = ws:%v light:%v", m.fields[0][3].on, m.fields[4][0].on)
	}

	a.cfg.Servers = []config.Server{{Name: "e", Host: "h", Port: 1, Protocol: "http"}}
	a.cfg.Theme = "light"
	m = newSettingsModel(a)
	if m.fields[0][3].on || !m.fields[4][0].on {
		t.Fatalf("toggles = ws:%v light:%v", m.fields[0][3].on, m.fields[4][0].on)
	}
}

func TestSettingsLoadCmd(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	cmd := m.loadCmd()
	if cmd == nil {
		t.Fatal("loadCmd with client must return a command")
	}
	if msg, ok := cmd().(globalOptionsMsg); !ok || msg.err != nil {
		t.Fatalf("msg = %#v", msg)
	}
	a.client = nil
	if m.loadCmd() != nil {
		t.Fatal("nil client must yield nil cmd")
	}
}

func TestSettingsAbsorbGlobal(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m.absorbGlobal(map[string]string{"enable-dht": "true", "dir": "/x"})
	if !m.fields[3][0].on || m.fields[2][0].input.Value() != "/x" {
		t.Fatalf("dht=%v dir=%q", m.fields[3][0].on, m.fields[2][0].input.Value())
	}
	// Dirty inputs are not overwritten; toggles still track the server.
	m.dirty = true
	m.absorbGlobal(map[string]string{"enable-dht": "false", "dir": "/y"})
	if m.fields[3][0].on || m.fields[2][0].input.Value() != "/x" {
		t.Fatalf("dht=%v dir=%q", m.fields[3][0].on, m.fields[2][0].input.Value())
	}
}

func TestSettingsBlurAll(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m.fields[0][0].input.Focus()
	m.blurAll()
	if m.fields[0][0].input.Focused() {
		t.Fatal("blurAll must blur every input")
	}
}

func TestSettingsSidebarNavigation(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)

	a.screen = screenSettings
	m, _ = m.update(key("esc"))
	if a.screen != screenList {
		t.Fatal("esc must return to list")
	}
	a.screen = screenSettings
	m, _ = m.update(key("q"))
	if a.screen != screenList {
		t.Fatal("q must return to list")
	}

	for i := 0; i < 10; i++ {
		m, _ = m.update(key("j"))
	}
	if m.section != len(m.sections)-1 {
		t.Fatalf("section = %d", m.section)
	}
	for i := 0; i < 10; i++ {
		m, _ = m.update(key("k"))
	}
	if m.section != 0 {
		t.Fatalf("section = %d", m.section)
	}
	// Unknown key in the sidebar is inert.
	m, cmd := m.update(key("x"))
	if cmd != nil {
		t.Fatal("unknown sidebar key must be inert")
	}
}

func TestSettingsEnterFieldsInputVsToggle(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)

	// Connection's first field is an input → focus command.
	m.section = 0
	m, cmd := m.update(key("enter"))
	if m.inSide || cmd == nil {
		t.Fatal("enter must focus the first input")
	}

	// BitTorrent's first field is a toggle → no focus command.
	m2 := newSettingsModel(a)
	m2.section = 3
	m2, cmd = m2.update(key("l"))
	if m2.inSide || cmd != nil {
		t.Fatal("toggle-first section must not emit a focus cmd")
	}
}

func TestSettingsFieldTabCycle(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m.section = 0
	m, _ = m.update(key("tab")) // into fields, focus 0
	m, cmd := m.update(key("tab"))
	if m.focus != 1 || cmd == nil {
		t.Fatalf("focus = %d", m.focus)
	}
	m, _ = m.update(key("tab")) // 2 (secret input)
	m, cmd = m.update(key("tab"))
	if m.focus != 3 || cmd != nil { // toggle: no focus cmd
		t.Fatalf("focus = %d", m.focus)
	}
	m, _ = m.update(key("tab")) // wrap back to the sidebar
	if !m.inSide || m.focus != 0 {
		t.Fatalf("inSide=%v focus=%d", m.inSide, m.focus)
	}
}

func TestSettingsShiftTabAndLeft(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m, _ = m.update(key("tab"))
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if !m.inSide {
		t.Fatal("shift+tab must return to the sidebar")
	}
	m, _ = m.update(key("tab"))
	m, _ = m.update(key("left"))
	if !m.inSide {
		t.Fatal("left must return to the sidebar")
	}
}

func TestSettingsSpaceToggleAndInput(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m.section = 3
	m.inSide = false
	m.focus = 0
	m, _ = m.update(key(" "))
	if !m.fields[3][0].on || !m.dirty {
		t.Fatal("space must toggle and mark dirty")
	}

	// Space over an input goes to the input update path.
	m2 := newSettingsModel(a)
	m2.section = 0
	m2, _ = m2.update(key("tab")) // focus Host input
	m2, _ = m2.update(key(" "))
	if m2.fields[0][0].input.Value() != "h " && !strings.HasSuffix(m2.fields[0][0].input.Value(), " ") {
		t.Fatalf("host = %q", m2.fields[0][0].input.Value())
	}
}

func TestSettingsTypingDirtyFlag(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m, _ = m.update(key("tab")) // focus Host input

	// A key the input ignores leaves the model clean.
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyUp})
	if m.dirty {
		t.Fatal("no value change must stay clean")
	}
	m, _ = m.update(key("x"))
	if !m.dirty {
		t.Fatal("typing must mark dirty")
	}
	// A toggle focused swallows plain keys.
	m.section = 3
	m.focus = 0
	m, cmd := m.update(key("x"))
	if cmd != nil {
		t.Fatal("typing on a toggle must be inert")
	}
}

func TestSettingsSaveManagedSkips(t *testing.T) {
	a, fake := testApp(t) // managed built-in server
	rec := &globalRecAPI{fakeAPI: fake}
	a.client = rec
	m := newSettingsModel(a)
	// Match loaded options so nothing is pushed: flash-only branch.
	m.absorbGlobal(map[string]string{
		"enable-dht": "false", "enable-peer-exchange": "false",
		"bt-enable-lpd": "false", "bt-require-crypto": "false",
	})
	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	if cmd == nil || m.dirty {
		t.Fatal("ctrl+s must save")
	}
	if a.status != "settings saved" || a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
	if a.cfg.Theme != "dark" {
		t.Fatalf("theme = %q", a.cfg.Theme)
	}
}

func TestSettingsSaveBadPort(t *testing.T) {
	a, _ := settingsTestApp(t)
	m := a.settings
	m.fields[0][1].input.SetValue("abc")
	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	_ = m
	if cmd == nil || !strings.Contains(a.status, "bad port") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestSettingsSaveConnectionChangeReconnects(t *testing.T) {
	a, _ := settingsTestApp(t)
	m := a.settings
	m.fields[0][0].input.SetValue("h2")
	m.fields[0][3].on = false // http now
	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	_ = m
	if cmd == nil {
		t.Fatal("save must return commands")
	}
	if a.client != nil || a.connected {
		t.Fatal("connection change must reconnect")
	}
	got := a.cfg.Servers[0]
	if got.Host != "h2" || got.Port != 42 || got.Protocol != "http" {
		t.Fatalf("srv = %+v", got)
	}
}

func TestSettingsSaveChangedOptions(t *testing.T) {
	a, fake := settingsTestApp(t)
	rec := &globalRecAPI{fakeAPI: fake}
	a.client = rec
	m := a.settings
	m.absorbGlobal(map[string]string{
		"enable-dht": "false", "enable-peer-exchange": "false",
		"bt-enable-lpd": "false", "bt-require-crypto": "false",
	})
	m.fields[1][0].input.SetValue("5") // max-concurrent-downloads
	// A config-backed field (optKey "") inside an option section is skipped.
	m.fields[1] = append(m.fields[1], setField{label: "phantom"})
	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	_ = m
	if cmd == nil {
		t.Fatal("save must return commands")
	}
	if a.client == nil {
		t.Fatal("unchanged connection must not reconnect")
	}
	drain(t, a, cmd)
	if rec.global["max-concurrent-downloads"] != "5" || len(rec.global) != 1 {
		t.Fatalf("global = %v", rec.global)
	}
}

func TestSettingsSaveStaleActiveClamp(t *testing.T) {
	a, _ := settingsTestApp(t)
	a.cfg.Active = 7 // stale index
	m := a.settings
	m, cmd := m.update(ctrl(tea.KeyCtrlS))
	_ = m
	if cmd == nil || a.cfg.Active != 0 {
		t.Fatalf("active = %d", a.cfg.Active)
	}
	if a.cfg.Servers[0].Host != "h" {
		t.Fatalf("srv = %+v", a.cfg.Servers[0])
	}
}

func TestSettingsSaveThemeLight(t *testing.T) {
	a, _ := settingsTestApp(t)
	m := a.settings
	m.fields[4][0].on = true
	m, _ = m.update(ctrl(tea.KeyCtrlS))
	_ = m
	if a.cfg.Theme != "light" {
		t.Fatalf("theme = %q", a.cfg.Theme)
	}
}

func TestStoreActiveServerEmptyList(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.Servers = nil
	srv := config.Server{Name: "n", Host: "h", Port: 1, Protocol: "ws"}
	a.storeActiveServer(srv)
	if len(a.cfg.Servers) != 1 || a.cfg.Active != 0 || a.cfg.Servers[0].Name != "n" {
		t.Fatalf("cfg = %+v", a.cfg)
	}
}

func TestSettingsViewManagedConnection(t *testing.T) {
	a, _ := testApp(t) // managed active server
	m := newSettingsModel(a)
	m.section = 0
	if out := m.view(); !strings.Contains(out, "Built-in daemon") {
		t.Fatalf("out = %q", out)
	}
}

func TestSettingsViewUnmanagedSections(t *testing.T) {
	a, _ := settingsTestApp(t)
	m := a.settings
	m.dirty = true
	for s := range m.sections {
		m.section = s
		m.inSide = true
		if out := m.view(); !strings.Contains(out, "unsaved changes") {
			t.Fatalf("section %d missing dirty marker", s)
		}
	}

	// Focused input.
	m.section = 0
	m.inSide = false
	m.focus = 0
	if out := m.view(); out == "" {
		t.Fatal("focused input view empty")
	}
	// Focused and unfocused toggles, one switched on.
	m.section = 3
	m.focus = 0
	m.fields[3][1].on = true
	out := m.view()
	if !strings.Contains(out, "space toggles") || !strings.Contains(out, "[x]") {
		t.Fatalf("out = %q", out)
	}
}
