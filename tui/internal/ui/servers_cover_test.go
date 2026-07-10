package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
	"aria2t/internal/daemon"
)

// collectLatency executes a probe command tree and gathers latency messages.
func collectLatency(t *testing.T, cmd tea.Cmd) []latencyMsg {
	t.Helper()
	var out []latencyMsg
	var walk func(c tea.Cmd)
	walk = func(c tea.Cmd) {
		if c == nil {
			return
		}
		msg := c()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				walk(sub)
			}
			return
		}
		if lm, ok := msg.(latencyMsg); ok {
			out = append(out, lm)
		}
	}
	walk(cmd)
	return out
}

func TestServersProbeCmdManagedNoDaemon(t *testing.T) {
	a, _ := testApp(t) // default config: single managed server
	a.daemon = nil
	m := newServersModel(a)
	if m.probeCmd() != nil {
		t.Fatal("managed server without daemon must be skipped entirely")
	}
}

func TestServersProbeCmdManagedWithDaemon(t *testing.T) {
	a, fake := testApp(t)
	a.daemon = &daemon.Daemon{Port: 7171, Secret: "sec"}
	var dialed config.Server
	a.dial = func(srv config.Server) (api, string, error) { dialed = srv; return fake, "v", nil }
	m := newServersModel(a)
	msgs := collectLatency(t, m.probeCmd())
	if len(msgs) != 1 || msgs[0].err != nil || msgs[0].index != 0 {
		t.Fatalf("msgs = %#v", msgs)
	}
	if dialed.Port != 7171 || dialed.Secret != "sec" || dialed.Protocol != "ws" {
		t.Fatalf("dialed = %+v", dialed)
	}
}

func TestServersProbeCmdUnmanaged(t *testing.T) {
	a, fake := testApp(t)
	a.cfg.Servers = []config.Server{
		{Name: "ok", Host: "ok", Port: 1, Protocol: "ws"},
		{Name: "bad", Host: "bad", Port: 2, Protocol: "ws"},
	}
	a.cfg.Active = 0
	a.dial = func(srv config.Server) (api, string, error) {
		if srv.Host == "bad" {
			return nil, "", errors.New("refused")
		}
		return fake, "v", nil
	}
	m := newServersModel(a)
	msgs := collectLatency(t, m.probeCmd())
	if len(msgs) != 2 {
		t.Fatalf("msgs = %#v", msgs)
	}
	byIdx := map[int]latencyMsg{}
	for _, lm := range msgs {
		byIdx[lm.index] = lm
	}
	if byIdx[0].err != nil || byIdx[0].d < 0 {
		t.Fatalf("ok probe = %#v", byIdx[0])
	}
	if byIdx[1].err == nil {
		t.Fatalf("bad probe = %#v", byIdx[1])
	}
}

func TestServersAbsorbLatency(t *testing.T) {
	a, _ := testApp(t)
	m := newServersModel(a)
	m.absorbLatency(latencyMsg{index: 0, err: errors.New("down")})
	if !m.dead[0] {
		t.Fatal("error must mark dead")
	}
	m.absorbLatency(latencyMsg{index: 0, d: 3 * time.Millisecond})
	if m.dead[0] || m.latency[0] != 3*time.Millisecond {
		t.Fatalf("dead=%v latency=%v", m.dead[0], m.latency[0])
	}
}

func serversTestApp(t *testing.T) (*App, *fakeAPI) {
	t.Helper()
	a, fake := testApp(t)
	a.cfg.Servers = []config.Server{
		{Name: "one", Host: "h1", Port: 1, Protocol: "ws"},
		{Name: "two", Host: "h2", Port: 2, Protocol: "http"},
		{Name: "three", Host: "h3", Port: 3, Protocol: "ws"},
	}
	a.cfg.Active = 0
	a.dial = func(srv config.Server) (api, string, error) { return fake, "v", nil }
	return a, fake
}

func TestServersUpdateNavigation(t *testing.T) {
	a, _ := serversTestApp(t)
	m := newServersModel(a)

	a.overlay = overlayServers
	m, _ = m.update(key("esc"))
	if a.overlay != overlayNone {
		t.Fatal("esc must close")
	}

	m.cursor = 0
	m, _ = m.update(key("j"))
	m, _ = m.update(key("s"))
	if m.cursor != 2 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	m, _ = m.update(key("j")) // wrap to 0
	if m.cursor != 0 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	m, _ = m.update(key("k")) // wrap back to 2
	if m.cursor != 2 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	// Unknown key is inert.
	m, cmd := m.update(key("x"))
	if cmd != nil {
		t.Fatal("unknown key must be inert")
	}
}

func TestServersUpdateEnterSwitches(t *testing.T) {
	a, _ := serversTestApp(t)
	a.overlay = overlayServers
	m := newServersModel(a)
	m.cursor = 1
	m, cmd := m.update(key("enter"))
	if cmd == nil || a.overlay != overlayNone || a.cfg.Active != 1 {
		t.Fatalf("active = %d", a.cfg.Active)
	}
	if !strings.Contains(a.status, "switching to two") {
		t.Fatalf("status = %q", a.status)
	}
	saved, err := config.Load(a.cfgPath)
	if err != nil || saved.Active != 1 {
		t.Fatalf("persisted active = %d err=%v", saved.Active, err)
	}

	// Out-of-range cursor is a no-op.
	m.cursor = 9
	m, cmd = m.update(key("enter"))
	if cmd != nil {
		t.Fatal("out-of-range enter must be inert")
	}
}

func TestServersUpdateAddForm(t *testing.T) {
	a, _ := serversTestApp(t)
	m := newServersModel(a)
	m.form[0].SetValue("stale")
	m, cmd := m.update(key("+"))
	if !m.editing || m.editIdx != -1 || !m.formWS || cmd == nil {
		t.Fatal("+ must open a blank add form")
	}
	if m.form[0].Value() != "" || m.form[2].Value() != "6800" || m.formFoc != 0 {
		t.Fatalf("form = %q/%q", m.form[0].Value(), m.form[2].Value())
	}
}

func TestServersUpdateEditForm(t *testing.T) {
	a, _ := serversTestApp(t)
	m := newServersModel(a)
	m.cursor = 0 // ws server
	m, cmd := m.update(key("e"))
	if !m.editing || m.editIdx != 0 || !m.formWS || cmd == nil {
		t.Fatal("e must open the edit form")
	}
	if m.form[0].Value() != "one" || m.form[1].Value() != "h1" || m.form[2].Value() != "1" {
		t.Fatalf("form = %q/%q/%q", m.form[0].Value(), m.form[1].Value(), m.form[2].Value())
	}

	m2 := newServersModel(a)
	m2.cursor = 1 // http server
	m2, _ = m2.update(key("e"))
	if m2.formWS {
		t.Fatal("http server must open with formWS off")
	}

	m3 := newServersModel(a)
	m3.cursor = 9
	m3, cmd = m3.update(key("e"))
	if m3.editing || cmd != nil {
		t.Fatal("out-of-range e must be inert")
	}
}

func TestServersUpdateRemoveGuards(t *testing.T) {
	// Single server: removal refused.
	a, _ := testApp(t)
	m := newServersModel(a)
	m, _ = m.update(key("-"))
	if len(a.cfg.Servers) != 1 {
		t.Fatal("single server must not be removed")
	}

	// Active index clamp + cursor clamp when removing the last entry.
	b, _ := serversTestApp(t)
	b.cfg.Active = 2
	mb := newServersModel(b)
	mb.cursor = 2
	mb, _ = mb.update(key("d"))
	if len(b.cfg.Servers) != 2 || b.cfg.Active != 0 || mb.cursor != 1 {
		t.Fatalf("servers=%d active=%d cursor=%d", len(b.cfg.Servers), b.cfg.Active, mb.cursor)
	}

	// Removing a middle entry keeps active and cursor valid without clamping.
	c, _ := serversTestApp(t)
	mc := newServersModel(c)
	mc.cursor = 0
	mc, _ = mc.update(key("-"))
	if len(c.cfg.Servers) != 2 || mc.cursor != 0 || c.cfg.Active != 0 {
		t.Fatalf("servers=%d cursor=%d", len(c.cfg.Servers), mc.cursor)
	}
}

func TestServersUpdateFormKeys(t *testing.T) {
	a, _ := serversTestApp(t)
	m := newServersModel(a)
	m, _ = m.update(key("+"))

	// Typing goes to the focused field.
	m, _ = m.update(key("x"))
	if m.form[0].Value() != "x" {
		t.Fatalf("name = %q", m.form[0].Value())
	}
	// Tab cycles through all four fields and wraps.
	for i := 1; i <= 4; i++ {
		m, _ = m.update(key("tab"))
		if m.formFoc != i%4 {
			t.Fatalf("formFoc = %d after %d tabs", m.formFoc, i)
		}
	}
	// ctrl+w toggles the protocol.
	m, _ = m.update(ctrl(tea.KeyCtrlW))
	if m.formWS {
		t.Fatal("ctrl+w must toggle to http")
	}
	// esc leaves the form.
	m, _ = m.update(key("esc"))
	if m.editing {
		t.Fatal("esc must leave editing")
	}
}

func TestServersUpdateFormEnterValidation(t *testing.T) {
	a, _ := serversTestApp(t)
	m := newServersModel(a)
	m, _ = m.update(key("+"))

	m.form[2].SetValue("abc")
	m, _ = m.update(key("enter"))
	if !strings.Contains(a.status, "bad port") || !m.editing {
		t.Fatalf("status = %q", a.status)
	}

	m.form[2].SetValue("6800")
	m, _ = m.update(key("enter")) // name and host empty
	if !strings.Contains(a.status, "name and host required") || !m.editing {
		t.Fatalf("status = %q", a.status)
	}
}

func TestServersUpdateFormEnterAddsAndEdits(t *testing.T) {
	a, _ := serversTestApp(t)
	m := newServersModel(a)
	m, _ = m.update(key("+"))
	m.form[0].SetValue("new")
	m.form[1].SetValue("nh")
	m.form[2].SetValue("9000")
	m.form[3].SetValue("shh")
	m.formWS = false // http branch
	m, cmd := m.update(key("enter"))
	if m.editing || cmd == nil {
		t.Fatal("valid enter must save and reprobe")
	}
	if len(a.cfg.Servers) != 4 || m.cursor != 3 {
		t.Fatalf("servers=%d cursor=%d", len(a.cfg.Servers), m.cursor)
	}
	got := a.cfg.Servers[3]
	if got.Name != "new" || got.Host != "nh" || got.Port != 9000 || got.Secret != "shh" || got.Protocol != "http" {
		t.Fatalf("srv = %+v", got)
	}

	// Edit existing entry in place.
	m.cursor = 0
	m, _ = m.update(key("e"))
	m.form[0].SetValue("renamed")
	m, _ = m.update(key("enter"))
	if a.cfg.Servers[0].Name != "renamed" || a.cfg.Servers[0].Protocol != "ws" || len(a.cfg.Servers) != 4 {
		t.Fatalf("srv = %+v", a.cfg.Servers[0])
	}
}

func TestServersViewFormStates(t *testing.T) {
	a, _ := serversTestApp(t)
	m := newServersModel(a)
	m.editing = true
	m.editIdx = -1
	m.formWS = true
	if out := m.view(); !strings.Contains(out, "Add server") {
		t.Fatalf("out = %q", out)
	}
	m.editIdx = 0
	m.formWS = false
	if out := m.view(); !strings.Contains(out, "Edit server") {
		t.Fatalf("out = %q", out)
	}
}

func TestServersViewListStates(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.Servers = []config.Server{
		{Name: "builtin", Managed: true, Protocol: "ws"},
		{Name: "dead", Host: "h1", Port: 1, Protocol: "ws"},
		{Name: "fast", Host: "h2", Port: 2, Protocol: "ws"},
		{Name: "pending", Host: "h3", Port: 3, Protocol: "ws"},
	}
	a.cfg.Active = 2
	a.daemon = nil
	m := newServersModel(a)
	m.cursor = 1
	m.dead[1] = true
	m.latency[2] = 12 * time.Millisecond

	out := m.view()
	for _, want := range []string{"starts on demand", "unreachable", "12ms", "connected", "probing", "built-in · managed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q:\n%s", want, out)
		}
	}

	// Managed server with a live daemon shows its endpoint.
	a.daemon = &daemon.Daemon{Port: 7171, Secret: "s"}
	if out := m.view(); !strings.Contains(out, "localhost:7171") {
		t.Fatalf("out = %q", out)
	}
}
