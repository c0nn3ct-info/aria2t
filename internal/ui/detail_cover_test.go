package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

// detailFakeAPI overrides status/peer/server calls on top of fakeAPI.
type detailFakeAPI struct {
	*fakeAPI
	status        rpc.Status
	statusErr     error
	peers         []rpc.Peer
	peersCalled   bool
	srv           []rpc.ServerStat
	serversCalled bool
}

func (f *detailFakeAPI) TellStatus(context.Context, string) (rpc.Status, error) {
	return f.status, f.statusErr
}

func (f *detailFakeAPI) GetPeers(context.Context, string) ([]rpc.Peer, error) {
	f.peersCalled = true
	return f.peers, nil
}

func (f *detailFakeAPI) GetServers(context.Context, string) ([]rpc.ServerStat, error) {
	f.serversCalled = true
	return f.srv, nil
}

func TestDetailRefreshCmdGuards(t *testing.T) {
	a, _ := testApp(t)
	a.detail.gid = "g"
	a.client = nil
	if a.detail.refreshCmd() != nil {
		t.Fatal("nil client must yield nil cmd")
	}
	a, _ = testApp(t)
	a.detail.gid = ""
	if a.detail.refreshCmd() != nil {
		t.Fatal("empty gid must yield nil cmd")
	}
}

func TestDetailRefreshCmdError(t *testing.T) {
	a, fake := testApp(t)
	a.client = &detailFakeAPI{fakeAPI: fake, statusErr: errors.New("boom")}
	a.detail.gid = "g"
	msg, ok := a.detail.refreshCmd()().(detailDataMsg)
	if !ok || msg.err == nil {
		t.Fatalf("want error msg, got %#v", msg)
	}
}

func TestDetailRefreshCmdTorrentFetchesPeers(t *testing.T) {
	a, fake := testApp(t)
	rec := &detailFakeAPI{
		fakeAPI: fake,
		status:  rpc.Status{GID: "g", InfoHash: "h"},
		peers:   []rpc.Peer{{IP: "1.2.3.4", Port: "51413"}},
	}
	a.client = rec
	a.detail.gid = "g"
	msg := a.detail.refreshCmd()().(detailDataMsg)
	if !rec.peersCalled || rec.serversCalled {
		t.Fatal("torrent status must fetch peers, not servers")
	}
	if len(msg.peers) != 1 || msg.status.GID != "g" {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestDetailRefreshCmdNonTorrentFetchesServers(t *testing.T) {
	a, fake := testApp(t)
	rec := &detailFakeAPI{
		fakeAPI: fake,
		status:  rpc.Status{GID: "g"},
		srv:     []rpc.ServerStat{{Index: "1", Servers: []rpc.ServerInfo{{URI: "http://m"}}}},
	}
	a.client = rec
	a.detail.gid = "g"
	msg := a.detail.refreshCmd()().(detailDataMsg)
	if rec.peersCalled || !rec.serversCalled {
		t.Fatal("non-torrent must fetch servers, not peers")
	}
	if len(msg.servers) != 1 || msg.peers != nil {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestDetailAbsorb(t *testing.T) {
	a, _ := testApp(t)
	m := &a.detail

	e := errors.New("boom")
	m.absorb(detailDataMsg{err: e})
	if m.err != e {
		t.Fatalf("err = %v", m.err)
	}

	m.absorb(detailDataMsg{
		status:  rpc.Status{Files: []rpc.File{{Index: "1"}}},
		peers:   []rpc.Peer{{IP: "1.1.1.1"}},
		servers: []rpc.ServerStat{{Index: "1"}},
	})
	if m.err != nil || len(m.s.Files) != 1 || len(m.peers) != 1 || len(m.servers) != 1 {
		t.Fatalf("absorb state: %#v", m)
	}
}

func TestDetailEscAndQ(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	_, _ = a.Update(key("esc"))
	if a.screen != screenList {
		t.Fatalf("esc must return to list, screen=%d", a.screen)
	}
	a.screen = screenDetail
	_, _ = a.Update(key("q"))
	if a.screen != screenList {
		t.Fatalf("q screen = %d", a.screen)
	}
}

func TestDetailFOpensPicker(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "g1"
	// No files → friendly flash, no overlay.
	a.detail.s = rpc.Status{GID: "g1"}
	_, _ = a.Update(key("f"))
	if a.overlay != overlayNone || a.status != "no files to select" {
		t.Fatalf("f without files: overlay=%d status=%q", a.overlay, a.status)
	}
	// With files → the picker opens on this gid.
	a.detail.s = rpc.Status{GID: "g1", Files: []rpc.File{{Index: "1", Path: "/dl/a"}}}
	_, cmd := a.Update(key("f"))
	if a.overlay != overlayFiles || a.files.gid != "g1" || cmd == nil {
		t.Fatalf("f must open picker: overlay=%d gid=%q", a.overlay, a.files.gid)
	}
}

func TestDetailPauseResume(t *testing.T) {
	a, fake := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "a1"
	a.detail.s = rpc.Status{Status: "active"}
	_, cmd := a.Update(key("p"))
	drain(t, a, cmd)
	if len(fake.paused) != 1 || fake.paused[0] != "a1" {
		t.Fatalf("paused = %v", fake.paused)
	}

	a.screen = screenDetail
	a.detail.gid = "a1"
	a.detail.s = rpc.Status{Status: "paused"}
	_, cmd = a.Update(key("p"))
	drain(t, a, cmd)
	if len(fake.paused) != 1 || len(fake.unpaused) != 1 || fake.unpaused[0] != "a1" {
		t.Fatalf("paused=%v unpaused=%v", fake.paused, fake.unpaused)
	}
}

func TestDetailRemove(t *testing.T) {
	a, fake := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "a1"
	_, _ = a.Update(key("d"))
	if a.overlay != overlayConfirm {
		t.Fatalf("d must ask first, overlay = %d", a.overlay)
	}
	_, cmd := a.Update(key("y"))
	if a.screen != screenList || a.overlay != overlayNone {
		t.Fatalf("screen = %d overlay = %d", a.screen, a.overlay)
	}
	drain(t, a, cmd)
	if len(fake.removed) != 1 || fake.removed[0] != "a1" {
		t.Fatalf("removed = %v", fake.removed)
	}
	// Active removal also purges the result (so --force-save can't resurrect it).
	if len(fake.removedResults) != 1 || fake.removedResults[0] != "a1" {
		t.Fatalf("active remove must purge the result, removedResults = %v", fake.removedResults)
	}
}

// TestDetailRemoveActiveErrorSkipsPurge covers the detail early-return arm when
// aria2.remove fails.
func TestDetailRemoveActiveErrorSkipsPurge(t *testing.T) {
	a, fake := testApp(t)
	fake.removeErr = errors.New("boom")
	a.screen = screenDetail
	a.detail.gid = "a1"
	a.detail.s = rpc.Status{GID: "a1", Status: "active"}
	_, _ = a.Update(key("d"))
	_, cmd := a.Update(key("y"))
	drain(t, a, cmd)
	if len(fake.removedResults) != 0 {
		t.Fatalf("detail purge must be skipped on error, removedResults = %v", fake.removedResults)
	}
}

func TestDetailTrackersKey(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "a1"
	a.detail.s = rpc.Status{GID: "a1", InfoHash: "h"}
	_, cmd := a.Update(key("t"))
	if a.screen != screenSeeding || a.seeding.gid != "a1" || cmd == nil {
		t.Fatalf("screen=%d gid=%q", a.screen, a.seeding.gid)
	}

	a.screen = screenDetail
	a.detail.s = rpc.Status{GID: "a1"}
	_, _ = a.Update(key("t"))
	if a.status != "not a torrent" || !a.statusErr {
		t.Fatalf("status = %q", a.status)
	}
}

func TestDetailOpenDirAndUnknownKey(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.s = rpc.Status{Dir: ""}
	_, _ = a.Update(key("o"))
	if a.status != "no directory" {
		t.Fatalf("status = %q", a.status)
	}
	_, cmd := a.Update(key("x"))
	if cmd != nil {
		t.Fatal("unknown key must be inert")
	}
}

func TestDetailViewError(t *testing.T) {
	a, _ := testApp(t)
	a.detail.err = errors.New("kaput")
	if out := a.detail.view(); !strings.Contains(out, "kaput") {
		t.Fatalf("view must show error: %q", out)
	}
}

func TestDetailViewErrorStatus(t *testing.T) {
	a, _ := testApp(t)
	a.detail.s = rpc.Status{GID: "g", Status: "error", ErrorCode: "9"}
	if out := a.detail.view(); !strings.Contains(out, "not enough disk space") {
		t.Fatalf("error status must read plainly: %q", out)
	}
}

func TestDetailViewTorrentFull(t *testing.T) {
	a, _ := testApp(t)
	a.detail.s = rpc.Status{
		GID: "g1", Status: "active", Seeder: "true",
		TotalLength: "100", CompletedLength: "50", UploadLength: "25",
		InfoHash: "0123456789abcdef0123456789abcdef01234567",
		Bitfield: "ff", NumPieces: "8", PieceLength: "16384",
		Dir: "/dl",
		Files: []rpc.File{
			{Index: "1", Path: "/dl/a.txt", Length: "10", CompletedLength: "5", Selected: "true"},
			{Index: "2", Path: "/other/b.txt", Length: "20", Selected: "false"},
		},
		BitTorrent: &rpc.BTInfo{Mode: "multi", Comment: "a nice torrent"},
	}
	for i := 0; i < 8; i++ {
		a.detail.peers = append(a.detail.peers, rpc.Peer{
			IP: "10.0.0.1", Port: "6881", PeerID: "%2DqB4520-", Bitfield: "ff",
			DownloadSpeed: "100", UploadSpeed: "50", Seeder: "true",
			AmChoking: "false", PeerChoking: "false",
		})
	}
	out := a.detail.view()
	for _, want := range []string{"PEERS", "seeding", "FILES", "1 of 2 selected", "torrent", "multi", "a.txt", "ratio"} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q:\n%s", want, out)
		}
	}
}

func TestDetailViewServers(t *testing.T) {
	a, _ := testApp(t)
	a.detail.s = rpc.Status{
		GID: "g1", Status: "active", TotalLength: "100", CompletedLength: "50",
		Dir: "/dl", Files: []rpc.File{{Index: "1", Path: "/dl/x.iso", Length: "100"}},
	}
	a.detail.servers = []rpc.ServerStat{{Index: "1", Servers: []rpc.ServerInfo{
		{URI: "https://mirror.example/x.iso", DownloadSpeed: "999000"},
	}}}
	out := a.detail.view()
	for _, want := range []string{"SERVER", "mirror.example"} {
		if !strings.Contains(out, want) {
			t.Fatalf("server view missing %q:\n%s", want, out)
		}
	}
}

func TestDetailViewEmptyLists(t *testing.T) {
	a, _ := testApp(t)
	// Torrent, no peers.
	a.detail.s = rpc.Status{GID: "g1", Status: "waiting", InfoHash: "h"}
	if out := a.detail.view(); !strings.Contains(out, "no peers") {
		t.Fatalf("view must note missing peers: %q", out)
	}
	// Non-torrent, no servers.
	a.detail.s = rpc.Status{GID: "g1", Status: "waiting"}
	if out := a.detail.view(); !strings.Contains(out, "no active servers") {
		t.Fatalf("view must note missing servers: %q", out)
	}
	a.width = 10 // trips the minimum pieces-width guard path
	a.detail.s = rpc.Status{GID: "g1", NumPieces: "4", Bitfield: "f"}
	if out := a.detail.view(); out == "" {
		t.Fatal("narrow view must still render")
	}
}

func TestDetailPeersOverflow(t *testing.T) {
	a, _ := testApp(t)
	a.height = 16 // tight, forces the peer list to cap
	a.detail.s = rpc.Status{GID: "g1", Status: "active", InfoHash: "h", NumPieces: "8", Bitfield: "ff"}
	for i := 0; i < 30; i++ {
		a.detail.peers = append(a.detail.peers, rpc.Peer{IP: "10.0.0.9", Port: "1"})
	}
	if out := a.detail.view(); !strings.Contains(out, "more") {
		t.Fatalf("capped peer list must show an overflow line:\n%s", out)
	}
}

func TestPeerFlags(t *testing.T) {
	seed := rpc.Peer{Seeder: "true", PeerChoking: "false", AmChoking: "false"}
	if got := peerFlags(seed); got != "Sdu" {
		t.Fatalf("flags = %q", got)
	}
	leech := rpc.Peer{Seeder: "false", PeerChoking: "true", AmChoking: "true"}
	if got := peerFlags(leech); got != "---" {
		t.Fatalf("flags = %q", got)
	}
}

func TestServerHost(t *testing.T) {
	if got := serverHost("https://host.example/path/file.iso"); got != "https://host.example" {
		t.Fatalf("got %q", got)
	}
	if got := serverHost("not a url with spaces"); got != "not a url with spaces" {
		t.Fatalf("got %q", got)
	}
}

func TestClientName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"%2DqB4520-", "qB"},
		{"TR2940", "TR"},
		{"lt0D60", "lt"},
		{"LT1234", "lt"},
		{"AZ2504", "AZ"},
		{"DE13F0", "DE"},
		{"UT355S", "µT"},
		{"A2xxxx", "aria2"},
		{"ar1234", "aria2"},
		{"XYzzzz", "XY"},
		{"a", "?"},
		{"", "?"},
		{"%2D", "?"},
	}
	for _, c := range cases {
		if got := clientName(c.in); got != c.want {
			t.Fatalf("clientName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTrimPathPrefix(t *testing.T) {
	if got := trimPathPrefix("/dl/sub/a.txt", "/dl"); got != "sub/a.txt" {
		t.Fatalf("got %q", got)
	}
	if got := trimPathPrefix("/other/a.txt", "/dl"); got != "/other/a.txt" {
		t.Fatalf("got %q", got)
	}
	if got := trimPathPrefix("a.txt", ""); got != "a.txt" {
		t.Fatalf("got %q", got)
	}
}
