package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aria2t/internal/config"
	"aria2t/internal/daemon"
	"aria2t/internal/rpc"
)

// closeRecAPI records Close calls on top of fakeAPI.
type closeRecAPI struct {
	*fakeAPI
	closed *bool
}

func (c closeRecAPI) Close() error {
	*c.closed = true
	return nil
}

// uiFakeDaemon spawns a real *daemon.Daemon backed by a harmless script.
func uiFakeDaemon(t *testing.T) *daemon.Daemon {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake-aria2c")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\ntrap 'exit 0' TERM INT\nwhile true; do sleep 0.1; done\n"), 0o755); err != nil {
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
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = d.Stop(ctx)
	})
	return d
}

func stopDaemon(t *testing.T, d *daemon.Daemon) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestConnectManagedRespawnsDeadDaemon(t *testing.T) {
	a, _ := testApp(t)
	dead := uiFakeDaemon(t)
	stopDaemon(t, dead)
	a.daemon = dead

	fresh := uiFakeDaemon(t)
	spawns := 0
	a.spawn = func(o daemon.Options) (*daemon.Daemon, error) {
		spawns++
		return fresh, nil
	}
	a.dial = func(srv config.Server) (api, string, error) { return newFakeAPI(), "1.37.0", nil }

	msg := a.connectCmd()()
	got, ok := msg.(connectedMsg)
	if !ok || got.daemon != fresh || spawns != 1 {
		t.Fatalf("msg = %#v spawns = %d", msg, spawns)
	}
}

func TestConnectManagedDialFailureKeepsDaemon(t *testing.T) {
	a, _ := testApp(t)
	d := uiFakeDaemon(t)
	a.spawn = func(o daemon.Options) (*daemon.Daemon, error) { return d, nil }
	a.dial = func(srv config.Server) (api, string, error) { return nil, "", errors.New("refused") }

	msg := a.connectCmd()()
	ce, ok := msg.(connectErrMsg)
	if !ok || ce.daemon != d {
		t.Fatalf("dial failure must carry the daemon: %#v", msg)
	}
	_, _ = a.Update(msg)
	if a.daemon != d {
		t.Fatal("connectErrMsg must store the spawned daemon for reuse")
	}
	// Retry with the stored daemon: no second spawn.
	spawns := 0
	a.spawn = func(o daemon.Options) (*daemon.Daemon, error) {
		spawns++
		return nil, errors.New("must not respawn")
	}
	a.dial = func(srv config.Server) (api, string, error) { return newFakeAPI(), "1.37.0", nil }
	msg = a.connectCmd()()
	if _, ok := msg.(connectedMsg); !ok || spawns != 0 {
		t.Fatalf("retry must reuse the daemon: %#v spawns=%d", msg, spawns)
	}
}

func TestConnectedMsgClosesPreviousClient(t *testing.T) {
	a, _ := testApp(t)
	closed := false
	a.client = closeRecAPI{fakeAPI: newFakeAPI(), closed: &closed}
	next := newFakeAPI()
	_, _ = a.Update(connectedMsg{client: next, version: "v", endpoint: "x:1"})
	if !closed {
		t.Fatal("previous client must be closed on reconnect")
	}
	if a.client != api(next) {
		t.Fatal("new client must be stored")
	}
	// Same client again: must not close it.
	_, _ = a.Update(connectedMsg{client: next, version: "v", endpoint: "x:1"})
}

func TestConnectedMsgStopsStaleDaemon(t *testing.T) {
	a, _ := testApp(t)
	stale := uiFakeDaemon(t)
	fresh := uiFakeDaemon(t)
	a.daemon = stale
	_, _ = a.Update(connectedMsg{client: newFakeAPI(), version: "v", endpoint: "x:1", daemon: fresh})
	if a.daemon != fresh {
		t.Fatal("fresh daemon must be stored")
	}
	deadline := time.Now().Add(10 * time.Second)
	for stale.Alive() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if stale.Alive() {
		t.Fatal("stale daemon must be stopped in the background")
	}
}

func TestPollErrorTearsDownClient(t *testing.T) {
	a, _ := testApp(t)
	closed := false
	a.client = closeRecAPI{fakeAPI: newFakeAPI(), closed: &closed}
	a.connected = true
	_, _ = a.Update(pollMsg{err: errors.New("broken pipe")})
	if a.connected || a.client != nil || !closed || a.connErr == nil {
		t.Fatalf("poll failure must drop the client: connected=%v client=%v closed=%v err=%v",
			a.connected, a.client, closed, a.connErr)
	}
	// Idempotent when client already nil.
	_, _ = a.Update(pollMsg{err: errors.New("again")})
}

func TestActionErrorResetsScheduleKey(t *testing.T) {
	a, _ := testApp(t)
	a.lastSchedKey = "5M/1M"
	_, _ = a.Update(actionDoneMsg{err: errors.New("nope")})
	if a.lastSchedKey != "" {
		t.Fatal("failed action must reset the schedule key so limits retry")
	}
}

func TestDetailSpaceFlipsLocalSelection(t *testing.T) {
	a, _ := testApp(t)
	m := newDetailModel(a)
	m.gid = "g"
	m.filesFocused = true
	m.s = rpc.Status{GID: "g", Files: []rpc.File{{Index: "1", Selected: "true"}}}
	m, cmd := m.update(key(" "))
	if cmd == nil || m.s.Files[0].IsSelected() {
		t.Fatal("first toggle must flip local state off")
	}
	m, _ = m.update(key(" "))
	if !m.s.Files[0].IsSelected() {
		t.Fatal("second toggle must flip local state back on")
	}
}

func TestStoppedTabGuardsThrottleAndSeeding(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(key("3")) // stopped tab
	_, cmd := a.Update(key("l"))
	if cmd == nil || a.overlay != overlayNone {
		t.Fatalf("throttle on stopped must flash, status=%q", a.status)
	}
	_, cmd = a.Update(key("t"))
	if cmd == nil || a.screen != screenList {
		t.Fatalf("seeding on stopped must flash, status=%q", a.status)
	}
}

func TestServersDeleteAdjustsActiveIndex(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.Servers = []config.Server{
		{Name: "a", Host: "a", Port: 1, Protocol: "ws"},
		{Name: "b", Host: "b", Port: 2, Protocol: "ws"},
		{Name: "c", Host: "c", Port: 3, Protocol: "ws"},
	}
	a.cfg.Active = 2
	m := newServersModel(a)
	m.cursor = 1
	m, cmd := m.update(key("-"))
	if a.cfg.Active != 1 || len(a.cfg.Servers) != 2 || cmd == nil {
		t.Fatalf("active must follow its server: active=%d servers=%d", a.cfg.Active, len(a.cfg.Servers))
	}
	// Deleting the active server itself resets to 0.
	a.cfg.Active = 1
	m.cursor = 1
	m, cmd = m.update(key("-"))
	if a.cfg.Active != 0 || len(a.cfg.Servers) != 1 || cmd == nil {
		t.Fatalf("deleting active must reset: active=%d", a.cfg.Active)
	}
	_ = m
}

func TestSettingsSaveSurfacesConfigError(t *testing.T) {
	a, _ := testApp(t)
	// cfgPath under a regular file makes MkdirAll fail inside config.Save.
	block := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(block, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a.cfgPath = filepath.Join(block, "sub", "config.json")
	m := newSettingsModel(a)
	m, cmd := m.save()
	_ = m
	if cmd == nil || !a.statusErr {
		t.Fatalf("config save failure must flash an error, status=%q", a.status)
	}
}

func TestParseLimitRejectsNegativeAndZeroSuffix(t *testing.T) {
	if _, err := ParseLimit("-5M"); err == nil {
		t.Fatal("negative limit must be rejected")
	}
	if got, err := ParseLimit("0K"); err != nil || got != "0" {
		t.Fatalf("0K = %q, %v", got, err)
	}
	if got := NormalizeLimit("5242880"); got != "5M" {
		t.Fatalf("NormalizeLimit bytes = %q", got)
	}
	if got := NormalizeLimit("weird"); got != "weird" {
		t.Fatalf("NormalizeLimit passthrough = %q", got)
	}
}

func TestDetailRemoveStoppedUsesRemoveResult(t *testing.T) {
	a, fake := testApp(t)
	m := newDetailModel(a)
	m.gid = "s1"
	m.s = rpc.Status{GID: "s1", Status: "complete"}
	_, cmd := m.update(key("d"))
	if cmd == nil {
		t.Fatal("d must produce a command")
	}
	if msg := cmd(); msg.(actionDoneMsg).err != nil {
		t.Fatalf("msg = %#v", msg)
	}
	if len(fake.removed) != 0 {
		t.Fatal("stopped download must not go through aria2.remove")
	}
}
