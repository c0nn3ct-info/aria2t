package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
	"aria2t/internal/daemon"
	"aria2t/internal/rpc"
)

// coverErrAPI makes selected RPC methods fail.
type coverErrAPI struct {
	*fakeAPI
	failActive, failWaiting, failStopped, failStat bool
}

func (f *coverErrAPI) TellActive(ctx context.Context) ([]rpc.Status, error) {
	if f.failActive {
		return nil, errors.New("active boom")
	}
	return f.fakeAPI.TellActive(ctx)
}

func (f *coverErrAPI) TellWaiting(ctx context.Context, o, n int) ([]rpc.Status, error) {
	if f.failWaiting {
		return nil, errors.New("waiting boom")
	}
	return f.fakeAPI.TellWaiting(ctx, o, n)
}

func (f *coverErrAPI) TellStopped(ctx context.Context, o, n int) ([]rpc.Status, error) {
	if f.failStopped {
		return nil, errors.New("stopped boom")
	}
	return f.fakeAPI.TellStopped(ctx, o, n)
}

func (f *coverErrAPI) GetGlobalStat(ctx context.Context) (rpc.GlobalStat, error) {
	if f.failStat {
		return rpc.GlobalStat{}, errors.New("stat boom")
	}
	return f.fakeAPI.GetGlobalStat(ctx)
}

// coverNotifAPI delivers notifications from a real channel.
type coverNotifAPI struct {
	*fakeAPI
	ch chan rpc.Notification
}

func (f *coverNotifAPI) Notifications() <-chan rpc.Notification { return f.ch }

// coverClosedPort returns a localhost TCP port that nothing listens on.
func coverClosedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// coverRPCServer serves one canned JSON-RPC response body.
func coverRPCServer(t *testing.T, respond func(id string) string) (host string, port int) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		_, _ = io.WriteString(w, respond(req.ID))
	}))
	t.Cleanup(ts.Close)
	addr := ts.Listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

func TestDialServerHTTPSuccess(t *testing.T) {
	host, port := coverRPCServer(t, func(id string) string {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":{"version":"9.9"}}`, id)
	})
	c, v, err := dialServer(config.Server{Host: host, Port: port, Protocol: "http"})
	if err != nil || v != "9.9" || c == nil {
		t.Fatalf("c=%v v=%q err=%v", c, v, err)
	}
	c.Close()
}

func TestDialServerHTTPError(t *testing.T) {
	host, port := coverRPCServer(t, func(id string) string {
		return fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"error":{"code":1,"message":"denied"}}`, id)
	})
	if _, _, err := dialServer(config.Server{Host: host, Port: port, Protocol: "http"}); err == nil {
		t.Fatal("want error from RPC error response")
	}
}

func TestDialServerWSError(t *testing.T) {
	port := coverClosedPort(t)
	if _, _, err := dialServer(config.Server{Host: "127.0.0.1", Port: port, Protocol: "ws"}); err == nil {
		t.Fatal("want dial error on closed port")
	}
}

func TestInitBuildsCommands(t *testing.T) {
	a, _ := testApp(t)
	if cmd := a.Init(); cmd == nil {
		t.Fatal("Init must return a command")
	}
}

func TestTickMsgAt(t *testing.T) {
	now := time.Now()
	if got := tickMsgAt(now); time.Time(got.(tickMsg)) != now {
		t.Fatalf("tickMsgAt = %v", got)
	}
	if cmd := tickCmd(); cmd == nil {
		t.Fatal("tickCmd must build")
	}
}

func TestConnectCmdUnmanaged(t *testing.T) {
	a, fake := testApp(t)
	a.cfg.Servers = []config.Server{{Name: "ext", Host: "h", Port: 42, Protocol: "ws"}}
	a.cfg.Active = 0
	a.dial = func(srv config.Server) (api, string, error) { return fake, "7.7", nil }
	msg := a.connectCmd()()
	got, ok := msg.(connectedMsg)
	if !ok || got.version != "7.7" || got.endpoint != "h:42" || got.daemon != nil {
		t.Fatalf("msg = %#v", msg)
	}

	a.dial = func(srv config.Server) (api, string, error) { return nil, "", errors.New("refused") }
	if _, ok := a.connectCmd()().(connectErrMsg); !ok {
		t.Fatal("want connectErrMsg")
	}
}

func TestConnectManagedCmd(t *testing.T) {
	a, fake := testApp(t)
	// Default config's server is managed; connectCmd must route here.
	var spawned daemon.Options
	d := &daemon.Daemon{Port: 7777, Secret: "sec"}
	a.spawn = func(o daemon.Options) (*daemon.Daemon, error) { spawned = o; return d, nil }
	var dialed config.Server
	a.dial = func(srv config.Server) (api, string, error) { dialed = srv; return fake, "8.8", nil }

	msg := a.connectCmd()()
	got, ok := msg.(connectedMsg)
	if !ok || got.daemon != d || got.endpoint != "localhost:7777" {
		t.Fatalf("msg = %#v", msg)
	}
	if dialed.Port != 7777 || dialed.Secret != "sec" || dialed.Protocol != "ws" {
		t.Fatalf("dialed = %+v", dialed)
	}
	if spawned.DataDir == "" {
		t.Fatalf("spawn options = %+v", spawned)
	}
}

func TestConnectManagedCmdSpawnError(t *testing.T) {
	a, _ := testApp(t)
	a.spawn = func(o daemon.Options) (*daemon.Daemon, error) { return nil, errors.New("no aria2c") }
	if _, ok := a.connectCmd()().(connectErrMsg); !ok {
		t.Fatal("want connectErrMsg on spawn failure")
	}
}

func TestConnectManagedCmdDialError(t *testing.T) {
	a, _ := testApp(t)
	a.daemon = &daemon.Daemon{Port: 7777, Secret: "sec"} // reuse path: spawn must not run
	a.spawn = func(o daemon.Options) (*daemon.Daemon, error) {
		t.Fatal("spawn must not be called when daemon exists")
		return nil, nil
	}
	a.dial = func(srv config.Server) (api, string, error) { return nil, "", errors.New("refused") }
	msg := a.connectCmd()()
	em, ok := msg.(connectErrMsg)
	if !ok || !strings.Contains(em.err.Error(), "built-in daemon dial") {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestShutdownWithClientOnly(t *testing.T) {
	a, _ := testApp(t)
	a.Shutdown()
	if a.client != nil {
		t.Fatal("client must be released")
	}
	a.Shutdown() // both nil: no-op branch
}

func TestShutdownWithDaemon(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fake-aria2c")
	script := "#!/bin/sh\ntrap 'exit 0' TERM INT\nwhile true; do sleep 0.1; done\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := daemon.Start(daemon.Options{
		Bin:        bin,
		DataDir:    t.TempDir(),
		ReadyProbe: func(int, string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := testApp(t)
	a.daemon = d
	a.Shutdown()
	if a.daemon != nil || a.client != nil {
		t.Fatal("daemon and client must be released")
	}
}

func TestPollCmdSuccessAndErrors(t *testing.T) {
	a, fake := testApp(t)
	msg := a.pollCmd()()
	if pm, ok := msg.(pollMsg); !ok || pm.err != nil {
		t.Fatalf("msg = %#v", msg)
	}
	for _, e := range []*coverErrAPI{
		{fakeAPI: fake, failActive: true},
		{fakeAPI: fake, failWaiting: true},
		{fakeAPI: fake, failStopped: true},
		{fakeAPI: fake, failStat: true},
	} {
		a.client = e
		msg := a.pollCmd()()
		if pm, ok := msg.(pollMsg); !ok || pm.err == nil {
			t.Fatalf("want error pollMsg, got %#v", msg)
		}
	}
	a.client = nil
	if a.pollCmd() != nil {
		t.Fatal("nil client must yield nil cmd")
	}
}

func TestListenCmd(t *testing.T) {
	a, fake := testApp(t)
	a.client = nil
	if a.listenCmd() != nil {
		t.Fatal("nil client must yield nil cmd")
	}

	napi := &coverNotifAPI{fakeAPI: fake, ch: make(chan rpc.Notification, 1)}
	a.client = napi
	napi.ch <- rpc.Notification{Method: "aria2.onDownloadComplete", GIDs: []string{"g1"}}
	msg := a.listenCmd()()
	if nm, ok := msg.(notifMsg); !ok || nm.Method != "aria2.onDownloadComplete" {
		t.Fatalf("msg = %#v", msg)
	}
	close(napi.ch)
	if msg := a.listenCmd()(); msg != nil {
		t.Fatalf("closed channel must yield nil, got %#v", msg)
	}
}

func TestRPCCmdNotConnected(t *testing.T) {
	a, _ := testApp(t)
	a.client = nil
	cmd := a.rpcCmd("x", func(ctx context.Context, c api) error { return nil })
	if cmd == nil || a.status != "not connected" || !a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
}

func TestFlashAndClearStatus(t *testing.T) {
	a, _ := testApp(t)
	if cmd := a.flash("hello", false); cmd == nil {
		t.Fatal("flash must schedule a clear")
	}
	seq := a.statusSeq
	if msg := clearStatusAt(seq)(time.Time{}); msg.(clearStatusMsg).seq != seq {
		t.Fatalf("clearStatusAt = %#v", msg)
	}
	_, _ = a.Update(clearStatusMsg{seq: seq - 1}) // stale: keeps message
	if a.status != "hello" {
		t.Fatalf("status = %q", a.status)
	}
	_, _ = a.Update(clearStatusMsg{seq: seq})
	if a.status != "" {
		t.Fatalf("status = %q", a.status)
	}
}

func TestSaveConfigError(t *testing.T) {
	a, _ := testApp(t)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a.cfgPath = filepath.Join(blocker, "config.json") // parent is a file → MkdirAll fails
	a.saveConfig()
	if !strings.Contains(a.status, "config save failed") || !a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
}

func TestReconnect(t *testing.T) {
	a, _ := testApp(t)
	a.spawn = func(o daemon.Options) (*daemon.Daemon, error) { return nil, errors.New("nope") }
	cmd := a.reconnect()
	if a.client != nil || a.connected || cmd == nil {
		t.Fatal("reconnect must drop the client and redial")
	}
}

func TestApplyScheduleBranches(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	a, _ := testApp(t)
	if a.applySchedule(now) != nil {
		t.Fatal("disabled scheduler must be a no-op")
	}
	a.cfg.SchedulerEnabled = true
	c := a.client
	a.client = nil
	if a.applySchedule(now) != nil {
		t.Fatal("nil client must be a no-op")
	}
	a.client = c

	// No rules → unlimited.
	cmd := a.applySchedule(now)
	if cmd == nil || a.lastSchedKey != "0/0" {
		t.Fatalf("key = %q", a.lastSchedKey)
	}
	if msg := cmd(); !strings.Contains(msg.(actionDoneMsg).text, "unlimited") {
		t.Fatalf("msg = %#v", msg)
	}
	// Same key again → dedupe.
	if a.applySchedule(now) != nil {
		t.Fatal("same key must dedupe")
	}
	// Active rule.
	var days [7]bool
	for i := range days {
		days[i] = true
	}
	a.cfg.Rules = []config.Rule{{Start: "00:00", End: "23:59", Days: days, Label: "day", Down: "5M", Up: "1M"}}
	cmd = a.applySchedule(now)
	if cmd == nil || a.lastSchedKey != "5M/1M" {
		t.Fatalf("key = %q", a.lastSchedKey)
	}
	if msg := cmd(); !strings.Contains(msg.(actionDoneMsg).text, "day") {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestUpdateWindowSize(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if a.width != 80 || a.height != 24 {
		t.Fatalf("size = %dx%d", a.width, a.height)
	}
}

func TestUpdateConnectedAndErr(t *testing.T) {
	a, fake := testApp(t)
	d := &daemon.Daemon{Port: 1, Secret: "s"}
	_, cmd := a.Update(connectedMsg{client: fake, version: "9", endpoint: "x:1", daemon: d})
	if !a.connected || a.daemon != d || a.version != "9" || a.endpoint != "x:1" || cmd == nil {
		t.Fatalf("connected=%v daemon=%v", a.connected, a.daemon)
	}
	_, _ = a.Update(connectErrMsg{err: errors.New("down")})
	if a.connected || a.connErr == nil {
		t.Fatal("connectErrMsg must mark disconnected")
	}
}

func TestUpdateTickBranches(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	a, _ := testApp(t)
	a.cfg.SchedulerEnabled = true // applySchedule wiring
	if _, cmd := a.Update(tickMsg(now)); cmd == nil {
		t.Fatal("tick with client must poll")
	}

	b, _ := testApp(t)
	b.client = nil
	b.connErr = errors.New("down")
	if _, cmd := b.Update(tickMsg(now)); cmd == nil {
		t.Fatal("tick with connErr must retry connect")
	}

	c, _ := testApp(t)
	c.client = nil
	c.connErr = nil
	if _, cmd := c.Update(tickMsg(now)); cmd == nil {
		t.Fatal("tick must always re-arm")
	}
}

func TestUpdatePollError(t *testing.T) {
	a, _ := testApp(t)
	_, cmd := a.Update(pollMsg{err: errors.New("gone")})
	if a.connected || cmd != nil {
		t.Fatal("poll error must mark disconnected")
	}
}

func TestUpdatePollDetailRefresh(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "a1"
	_, cmd := a.Update(pollMsg{snap: snapshot{}})
	if cmd == nil {
		t.Fatal("detail screen must refresh on poll")
	}
}

func TestUpdateNotif(t *testing.T) {
	a, _ := testApp(t)
	if _, cmd := a.Update(notifMsg{Method: "m"}); cmd == nil {
		t.Fatal("notif must re-listen and poll")
	}
	a.client = nil
	// listenCmd is nil and no poll is queued: the batch collapses to nil.
	if _, cmd := a.Update(notifMsg{Method: "m"}); cmd != nil {
		t.Fatal("notif without client must produce no work")
	}
}

func TestUpdateActionDone(t *testing.T) {
	a, _ := testApp(t)
	_, cmd := a.Update(actionDoneMsg{err: errors.New("bad")})
	if cmd == nil || a.status != "bad" || !a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
	_, cmd = a.Update(actionDoneMsg{text: "did it"})
	if cmd == nil || a.status != "did it" || a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
	a.status = ""
	_, cmd = a.Update(actionDoneMsg{})
	if cmd == nil || a.status != "" {
		t.Fatalf("silent success must not flash, status = %q", a.status)
	}
}

func TestUpdateVerifyProgress(t *testing.T) {
	a, _ := testApp(t)
	a.verify["g"] = &verifyState{}
	_, _ = a.Update(verifyProgressMsg{gid: "g", done: 3, total: 9})
	if a.verify["g"].Done != 3 || a.verify["g"].Total != 9 {
		t.Fatalf("state = %+v", a.verify["g"])
	}
	_, _ = a.Update(verifyProgressMsg{gid: "unknown", done: 1, total: 1})
}

func TestUpdateVerifyDone(t *testing.T) {
	a, _ := testApp(t)
	// Unknown gid + error outcome.
	_, cmd := a.Update(verifyDoneMsg{gid: "new", err: errors.New("io")})
	if cmd == nil || !strings.Contains(a.status, "verify: io") {
		t.Fatalf("status = %q", a.status)
	}
	if v := a.verify["new"]; v == nil || v.Running || !v.Finished {
		t.Fatalf("state = %+v", a.verify["new"])
	}
	// OK outcome.
	a.verify["g"] = &verifyState{Running: true}
	_, cmd = a.Update(verifyDoneMsg{gid: "g", ok: true, computed: "c"})
	if cmd == nil || a.status != "checksum verified" {
		t.Fatalf("status = %q", a.status)
	}
	// Mismatch outcome.
	_, cmd = a.Update(verifyDoneMsg{gid: "g", ok: false, computed: "c2"})
	if cmd == nil || a.status != "checksum MISMATCH" || !a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
}

func TestUpdateDataMessages(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(detailDataMsg{status: rpc.Status{GID: "a1"}})
	_, _ = a.Update(gidOptionsMsg{gid: "a1", err: errors.New("x")})
	_, _ = a.Update(gidOptionsMsg{gid: "a1", opts: map[string]string{"max-download-limit": "0"}})
	_, _ = a.Update(globalOptionsMsg{err: errors.New("x")})
	_, _ = a.Update(globalOptionsMsg{opts: map[string]string{"dir": "/tmp"}})
	_, _ = a.Update(latencyMsg{index: 0, d: time.Millisecond})
	_, _ = a.Update(statusMsg{}) // unhandled type → default return
}

func TestHandleKeyCtrlC(t *testing.T) {
	a, _ := testApp(t)
	_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+c must produce QuitMsg")
	}
}

func TestHandleKeyOverlayRouting(t *testing.T) {
	a, _ := testApp(t)
	for _, ov := range []overlay{overlayAdd, overlayThrottle, overlayServers, overlayPrompt} {
		a.overlay = ov
		_, _ = a.Update(key("x"))
		if a.screen != screenList {
			t.Fatalf("overlay %d must swallow keys", ov)
		}
	}
}

func TestHandleKeyScreenRouting(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayNone

	a.screen = screenDetail
	_, _ = a.Update(key("x"))

	a.screen = screenStats
	_, _ = a.Update(key("x")) // no-op key
	if a.screen != screenStats {
		t.Fatal("x must not leave stats")
	}
	_, _ = a.Update(key("esc"))
	if a.screen != screenList {
		t.Fatal("esc must leave stats")
	}

	for _, sc := range []screen{screenSettings, screenSeeding, screenScheduler} {
		a.screen = sc
		_, _ = a.Update(key("x"))
	}

	a.screen = screen(99) // unknown screen → fallthrough return
	if _, cmd := a.Update(key("x")); cmd != nil {
		t.Fatal("unknown screen must be inert")
	}
	a.screen = screenList
}

func TestStartVerifyGuards(t *testing.T) {
	a, _ := testApp(t)
	s := rpc.Status{GID: "s1"}
	if cmd := a.startVerify(s); cmd == nil || !strings.Contains(a.status, "no expected checksum") {
		t.Fatalf("status = %q", a.status)
	}
	a.verify["s1"] = &verifyState{Expected: "e", Running: true}
	if cmd := a.startVerify(s); cmd == nil || !strings.Contains(a.status, "already running") {
		t.Fatalf("status = %q", a.status)
	}
	a.verify["s1"] = &verifyState{Expected: "e"}
	if cmd := a.startVerify(s); cmd == nil || !strings.Contains(a.status, "no local file path") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestStartVerifySuccess(t *testing.T) {
	a, _ := testApp(t)
	path := filepath.Join(t.TempDir(), "payload.bin")
	content := []byte("hello aria2t")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	a.verify["s1"] = &verifyState{Expected: hex.EncodeToString(sum[:])}
	s := rpc.Status{GID: "s1", Files: []rpc.File{{Path: path}}}
	cmd := a.startVerify(s)
	if cmd == nil || !a.verify["s1"].Running {
		t.Fatal("verification must start")
	}
	msg := cmd()
	done, ok := msg.(verifyDoneMsg)
	if !ok || !done.ok || done.gid != "s1" || done.err != nil {
		t.Fatalf("msg = %#v", msg)
	}
	if a.verify["s1"].Done != int64(len(content)) {
		t.Fatalf("progress = %+v", a.verify["s1"])
	}
}

func TestRedownload(t *testing.T) {
	a, _ := testApp(t)
	s := rpc.Status{
		GID: "s1",
		Dir: "/dl",
		Files: []rpc.File{
			{URIs: []rpc.URI{{URI: "http://x/a"}, {URI: "http://x/a"}}}, // dedupe
			{URIs: []rpc.URI{{URI: "http://x/b"}}},
		},
	}
	cmd := a.redownload(s)
	if cmd == nil {
		t.Fatal("redownload must return a command")
	}
	msg := cmd()
	if dm, ok := msg.(actionDoneMsg); !ok || dm.err != nil || dm.text != "re-download queued" {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestOpenBinFor(t *testing.T) {
	if openBinFor("darwin") != "open" {
		t.Fatal("darwin must use open")
	}
	if openBinFor("linux") != "xdg-open" {
		t.Fatal("linux must use xdg-open")
	}
}

func TestOpenDir(t *testing.T) {
	a, _ := testApp(t)
	old := openDirBin
	t.Cleanup(func() { openDirBin = old })

	openDirBin = "true"
	cmd := a.openDir(t.TempDir())
	if cmd == nil {
		t.Fatal("openDir must return a command")
	}
	if msg := cmd(); msg.(actionDoneMsg).err != nil {
		t.Fatalf("msg = %#v", msg)
	}

	openDirBin = filepath.Join(t.TempDir(), "missing-opener")
	if msg := a.openDir("/x")(); msg.(actionDoneMsg).err == nil {
		t.Fatal("missing opener must error")
	}
}

func TestHeaderVariants(t *testing.T) {
	a, _ := testApp(t)
	a.endpoint = ""
	if h := a.header(); !strings.Contains(h, "connected") || !strings.Contains(h, "built-in") {
		t.Fatalf("header = %q", h)
	}
	a.connected = false
	a.endpoint = "somewhere:1"
	if h := a.header(); !strings.Contains(h, "disconnected") || !strings.Contains(h, "somewhere:1") {
		t.Fatalf("header = %q", h)
	}
	a.width = 0 // gap clamp
	if h := a.header(); h == "" {
		t.Fatal("narrow header empty")
	}
}

func TestStatusLineVariants(t *testing.T) {
	a, _ := testApp(t)
	a.status = ""
	if a.statusLine() != "" {
		t.Fatal("empty status must render nothing")
	}
	a.status, a.statusErr = "bad", true
	if !strings.Contains(a.statusLine(), "bad") {
		t.Fatal("error status missing")
	}
	a.statusErr = false
	if !strings.Contains(a.statusLine(), "bad") {
		t.Fatal("ok status missing")
	}
}

func TestViewAllScreensAndOverlays(t *testing.T) {
	a, _ := testApp(t)
	for _, sc := range []screen{screenList, screenDetail, screenStats, screenSettings, screenSeeding, screenScheduler} {
		a.screen = sc
		_ = a.View() // must not panic
	}
	a.screen = screenList
	a.prompt = newPromptModel(a, "t", "", nil) // zero promptModel has no styles
	for _, ov := range []overlay{overlayAdd, overlayThrottle, overlayServers, overlayPrompt} {
		a.overlay = ov
		if a.View() == "" {
			t.Fatalf("overlay %d view empty", ov)
		}
	}
}
