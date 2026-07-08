package ui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
	"aria2t/internal/rpc"
)

// fakeAPI records calls; every method succeeds with zero values.
type fakeAPI struct {
	changePosition []struct {
		GID string
		Pos int
		How string
	}
	changedOptions map[string]map[string]string // gid → opts
	globalOpts     map[string]string            // last ChangeGlobalOption
	paused         []string
	unpaused       []string
	removed        []string
	removedResults []string
	removeErr      error            // when set, Remove returns it (exercises the purge early-return)
	waiting        []rpc.Status     // returned by TellWaiting
	status         rpc.Status       // returned by TellStatus
	servers        []rpc.ServerStat // returned by GetServers
	addTorrentOpts map[string]string
	addedURIs      [][]string
	metalinkGids   []string
	pausedAll      bool
	unpausedAll    bool
	purged         bool
}

func newFakeAPI() *fakeAPI { return &fakeAPI{changedOptions: map[string]map[string]string{}} }

func (f *fakeAPI) TellActive(context.Context) ([]rpc.Status, error) { return nil, nil }
func (f *fakeAPI) TellWaiting(context.Context, int, int) ([]rpc.Status, error) {
	return f.waiting, nil
}
func (f *fakeAPI) TellStopped(context.Context, int, int) ([]rpc.Status, error) {
	return nil, nil
}
func (f *fakeAPI) TellStatus(context.Context, string) (rpc.Status, error) {
	return f.status, nil
}
func (f *fakeAPI) AddURI(_ context.Context, uris []string, _ map[string]string) (string, error) {
	f.addedURIs = append(f.addedURIs, uris)
	return "gid", nil
}
func (f *fakeAPI) AddTorrent(_ context.Context, _ string, opts map[string]string) (string, error) {
	f.addTorrentOpts = opts
	return "tgid", nil
}
func (f *fakeAPI) AddMetalink(context.Context, string, map[string]string) ([]string, error) {
	return f.metalinkGids, nil
}
func (f *fakeAPI) Pause(_ context.Context, gid string) error {
	f.paused = append(f.paused, gid)
	return nil
}
func (f *fakeAPI) PauseAll(context.Context) error {
	f.pausedAll = true
	return nil
}
func (f *fakeAPI) Unpause(_ context.Context, gid string) error {
	f.unpaused = append(f.unpaused, gid)
	return nil
}
func (f *fakeAPI) UnpauseAll(context.Context) error {
	f.unpausedAll = true
	return nil
}
func (f *fakeAPI) PurgeDownloadResult(context.Context) error {
	f.purged = true
	return nil
}
func (f *fakeAPI) Remove(_ context.Context, gid string) error {
	f.removed = append(f.removed, gid)
	return f.removeErr
}
func (f *fakeAPI) RemoveDownloadResult(_ context.Context, gid string) error {
	f.removedResults = append(f.removedResults, gid)
	return nil
}
func (f *fakeAPI) ChangePosition(_ context.Context, gid string, pos int, how string) (int, error) {
	f.changePosition = append(f.changePosition, struct {
		GID string
		Pos int
		How string
	}{gid, pos, how})
	return pos, nil
}
func (f *fakeAPI) ChangeOption(_ context.Context, gid string, opts map[string]string) error {
	f.changedOptions[gid] = opts
	return nil
}
func (f *fakeAPI) GetOption(context.Context, string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (f *fakeAPI) ChangeGlobalOption(_ context.Context, opts map[string]string) error {
	f.globalOpts = opts
	return nil
}
func (f *fakeAPI) GetGlobalOption(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (f *fakeAPI) GetGlobalStat(context.Context) (rpc.GlobalStat, error) {
	return rpc.GlobalStat{}, nil
}
func (f *fakeAPI) GetPeers(context.Context, string) ([]rpc.Peer, error) { return nil, nil }
func (f *fakeAPI) GetServers(context.Context, string) ([]rpc.ServerStat, error) {
	return f.servers, nil
}
func (f *fakeAPI) GetVersion(context.Context) (string, error) { return "test", nil }
func (f *fakeAPI) Notifications() <-chan rpc.Notification     { return nil }
func (f *fakeAPI) Close() error                               { return nil }

func testApp(t *testing.T) (*App, *fakeAPI) {
	t.Helper()
	fake := newFakeAPI()
	a := NewApp(config.Default(), t.TempDir()+"/config.json")
	a.client = fake
	a.connected = true
	fake.waiting = []rpc.Status{
		{GID: "w1", Status: "waiting"},
		{GID: "w2", Status: "waiting"},
		{GID: "w3", Status: "waiting"},
	}
	a.snap = snapshot{
		Active: []rpc.Status{
			{GID: "a1", Status: "active", TotalLength: "100", CompletedLength: "50"},
		},
		Waiting: []rpc.Status{
			{GID: "w1", Status: "waiting"},
			{GID: "w2", Status: "waiting"},
			{GID: "w3", Status: "waiting"},
		},
		Stopped: []rpc.Status{
			{GID: "s1", Status: "complete"},
		},
	}
	return a, fake
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// drain executes a command tree synchronously and feeds each resulting
// message into Update once. Follow-up commands (ticks, polls) are dropped —
// tests assert on fake-API state, not on the full async loop.
func drain(t *testing.T, a *App, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			drain(t, a, c)
		}
		return
	}
	if msg == nil {
		return
	}
	_, _ = a.Update(msg)
}

func TestTabSwitch(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("tab")) // All → Active
	if a.list.tab != tabActive {
		t.Fatalf("tab = %d", a.list.tab)
	}
	_, _ = a.Update(key("4"))
	if a.list.tab != tabStopped {
		t.Fatalf("tab = %d", a.list.tab)
	}
}

func TestReorderEnterCommitsPosition(t *testing.T) {
	a, fake := testApp(t)
	_, _ = a.Update(key("3")) // waiting tab, cursor on w1
	_, _ = a.Update(key("J"))
	if !a.list.reordering {
		t.Fatal("J must enter reorder mode")
	}
	if a.list.localOrder[1].GID != "w1" {
		t.Fatalf("w1 must have moved down: %+v", a.list.localOrder)
	}
	_, cmd := a.Update(key("enter"))
	drain(t, a, cmd)
	if a.list.reordering {
		t.Fatal("enter must leave reorder mode")
	}
	if len(fake.changePosition) != 1 {
		t.Fatalf("changePosition calls = %d", len(fake.changePosition))
	}
	got := fake.changePosition[0]
	if got.GID != "w1" || got.Pos != 1 || got.How != "POS_SET" {
		t.Fatalf("bad call: %+v", got)
	}
}

func TestReorderEscRestores(t *testing.T) {
	a, fake := testApp(t)
	_, _ = a.Update(key("3"))
	_, _ = a.Update(key("J"))
	_, _ = a.Update(key("J")) // w1 now at index 2
	_, _ = a.Update(key("esc"))
	if a.list.reordering {
		t.Fatal("esc must leave reorder mode")
	}
	if a.snap.Waiting[0].GID != "w1" {
		t.Fatalf("order must be restored: %+v", a.snap.Waiting)
	}
	if len(fake.changePosition) != 0 {
		t.Fatal("esc must not call changePosition")
	}
}

func TestReorderGGMovesToTop(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("3"))
	_, _ = a.Update(key("j")) // cursor to w2
	_, _ = a.Update(key("J")) // grab w2, move to index 2
	_, _ = a.Update(key("g"))
	_, _ = a.Update(key("g")) // gg → top
	if a.list.localOrder[0].GID != "w2" || a.list.cursor != 0 {
		t.Fatalf("gg must move to top: %+v cursor=%d", a.list.localOrder, a.list.cursor)
	}
}

func TestPauseCallsRPC(t *testing.T) {
	a, fake := testApp(t)
	_, cmd := a.Update(key("p"))
	drain(t, a, cmd)
	if len(fake.paused) != 1 || fake.paused[0] != "a1" {
		t.Fatalf("paused = %v", fake.paused)
	}
}

func TestEnterOpensDetailEscReturns(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("enter"))
	if a.screen != screenDetail {
		t.Fatalf("screen = %d", a.screen)
	}
	_, _ = a.Update(key("esc"))
	if a.screen != screenList {
		t.Fatalf("screen = %d", a.screen)
	}
}

func TestSelectionClamped(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("2")) // Active tab: a single row
	for i := 0; i < 10; i++ {
		_, _ = a.Update(key("j"))
	}
	if a.list.cursor != 0 { // one active download → single row
		t.Fatalf("cursor = %d", a.list.cursor)
	}
	_, _ = a.Update(key("k"))
	if a.list.cursor != 0 {
		t.Fatalf("cursor = %d", a.list.cursor)
	}
}

func TestThemeToggle(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("T"))
	if a.cfg.Theme != "light" || a.styles.P.Name != "light" {
		t.Fatalf("theme = %s", a.cfg.Theme)
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	a, _ := testApp(t)
	for _, k := range []string{"", "2", "3"} {
		if k != "" {
			_, _ = a.Update(key(k))
		}
		if a.View() == "" {
			t.Fatal("empty view")
		}
	}
	_, _ = a.Update(key("g")) // stats screen (from stopped tab g is not bound; go back)
	_ = a.View()
}
