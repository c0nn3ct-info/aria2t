package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

// detailFakeAPI overrides status/peer calls on top of fakeAPI.
type detailFakeAPI struct {
	*fakeAPI
	status      rpc.Status
	statusErr   error
	peers       []rpc.Peer
	peersCalled bool
}

func (f *detailFakeAPI) TellStatus(context.Context, string) (rpc.Status, error) {
	return f.status, f.statusErr
}

func (f *detailFakeAPI) GetPeers(context.Context, string) ([]rpc.Peer, error) {
	f.peersCalled = true
	return f.peers, nil
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
	if !rec.peersCalled {
		t.Fatal("torrent status must fetch peers")
	}
	if len(msg.peers) != 1 || msg.status.GID != "g" {
		t.Fatalf("msg = %#v", msg)
	}
}

func TestDetailRefreshCmdNonTorrentSkipsPeers(t *testing.T) {
	a, fake := testApp(t)
	rec := &detailFakeAPI{fakeAPI: fake, status: rpc.Status{GID: "g"}}
	a.client = rec
	a.detail.gid = "g"
	msg := a.detail.refreshCmd()().(detailDataMsg)
	if rec.peersCalled {
		t.Fatal("non-torrent must not fetch peers")
	}
	if msg.peers != nil {
		t.Fatalf("peers = %#v", msg.peers)
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

	m.fileCursor = 5
	m.absorb(detailDataMsg{status: rpc.Status{Files: []rpc.File{{Index: "1"}, {Index: "2"}}}})
	if m.err != nil || m.fileCursor != 1 {
		t.Fatalf("err = %v cursor = %d", m.err, m.fileCursor)
	}

	m.fileCursor = 5
	m.absorb(detailDataMsg{status: rpc.Status{}})
	if m.fileCursor != 5 {
		t.Fatalf("zero files must not clamp, cursor = %d", m.fileCursor)
	}
}

func TestDetailEscAndQ(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.filesFocused = true
	_, _ = a.Update(key("esc"))
	if a.detail.filesFocused || a.screen != screenDetail {
		t.Fatalf("esc must only unfocus files, focused=%v screen=%d", a.detail.filesFocused, a.screen)
	}
	_, _ = a.Update(key("q"))
	if a.screen != screenList {
		t.Fatalf("screen = %d", a.screen)
	}
}

func TestDetailFileNavigation(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.s = rpc.Status{Files: []rpc.File{{Index: "1"}, {Index: "2"}}}

	// Unfocused: j/k are inert.
	_, _ = a.Update(key("j"))
	_, _ = a.Update(key("k"))
	if a.detail.fileCursor != 0 {
		t.Fatalf("cursor = %d", a.detail.fileCursor)
	}

	_, _ = a.Update(key("f"))
	if !a.detail.filesFocused {
		t.Fatal("f must focus files")
	}
	_, _ = a.Update(key("j"))
	if a.detail.fileCursor != 1 {
		t.Fatalf("cursor = %d", a.detail.fileCursor)
	}
	_, _ = a.Update(key("j")) // at bottom, guard
	if a.detail.fileCursor != 1 {
		t.Fatalf("cursor = %d", a.detail.fileCursor)
	}
	_, _ = a.Update(key("k"))
	_, _ = a.Update(key("k")) // at top, guard
	if a.detail.fileCursor != 0 {
		t.Fatalf("cursor = %d", a.detail.fileCursor)
	}
	_, _ = a.Update(key("f"))
	if a.detail.filesFocused {
		t.Fatal("f must toggle focus off")
	}
}

func TestDetailSpaceTogglesFile(t *testing.T) {
	a, fake := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "g1"
	a.detail.s = rpc.Status{Files: []rpc.File{
		{Index: "1", Selected: "true"},
		{Index: "2", Selected: "true"},
	}}

	// Not focused: no command.
	_, cmd := a.Update(key(" "))
	if cmd != nil {
		t.Fatal("space without focus must be inert")
	}

	a.detail.filesFocused = true
	a.detail.fileCursor = 0
	_, cmd = a.Update(key(" "))
	drain(t, a, cmd)
	got := fake.changedOptions["g1"]
	if got == nil || got["select-file"] != "2" {
		t.Fatalf("changedOptions = %#v", fake.changedOptions)
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
	a.detail.s = rpc.Status{Status: "paused"}
	_, cmd = a.Update(key("p"))
	drain(t, a, cmd)
	if len(fake.paused) != 1 {
		t.Fatalf("paused download must be unpaused, paused = %v", fake.paused)
	}
}

func TestDetailRemove(t *testing.T) {
	a, fake := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "a1"
	_, cmd := a.Update(key("d"))
	if a.screen != screenList {
		t.Fatalf("screen = %d", a.screen)
	}
	drain(t, a, cmd)
	if len(fake.removed) != 1 || fake.removed[0] != "a1" {
		t.Fatalf("removed = %v", fake.removed)
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

func TestDetailViewTorrentFull(t *testing.T) {
	a, _ := testApp(t)
	a.detail.s = rpc.Status{
		GID: "g1", Status: "active",
		TotalLength: "100", CompletedLength: "50",
		InfoHash: "h", Bitfield: "ff", NumPieces: "8", PieceLength: "16384",
		Dir: "/dl",
		Files: []rpc.File{
			{Index: "1", Path: "/dl/a.txt", Length: "10", Selected: "true"},
			{Index: "2", Path: "/other/b.txt", Length: "20", Selected: "false"},
		},
		BitTorrent: &rpc.BTInfo{AnnounceList: [][]string{
			{"http://t1", "http://t2"},
			{"http://t3", "http://t4"},
		}},
	}
	for i := 0; i < 8; i++ {
		a.detail.peers = append(a.detail.peers, rpc.Peer{
			IP: "10.0.0.1", Port: "6881", PeerID: "%2DqB4520-",
			DownloadSpeed: "100", UploadSpeed: "50",
		})
	}
	a.detail.filesFocused = true
	a.detail.fileCursor = 0
	out := a.detail.view()
	for _, want := range []string{"2 more", "1 selected of 2", "ANNOUNCE", "http://t3", "j/k + space", "a.txt"} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q", want)
		}
	}
	if strings.Contains(out, "http://t4") {
		t.Fatal("announce list must truncate at 3 entries")
	}
}

func TestDetailViewNoPeers(t *testing.T) {
	a, _ := testApp(t)
	a.detail.s = rpc.Status{GID: "g1", Status: "waiting"}
	if out := a.detail.view(); !strings.Contains(out, "no peers") {
		t.Fatalf("view must note missing peers: %q", out)
	}

	a.width = 10 // trips the minimum pieces-width guard
	if out := a.detail.view(); out == "" {
		t.Fatal("narrow view must still render")
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
