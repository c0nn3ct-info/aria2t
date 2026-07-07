package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

func TestDetectAddTab(t *testing.T) {
	cases := map[string]int{
		"magnet:?xt=urn":       addTabURL,
		"https://x/y.iso":      addTabURL,
		"/home/u/cool.torrent": addTabTorrent,
		"/home/u/a.METALINK":   addTabMetalink,
		"/home/u/a.meta4":      addTabMetalink,
		"just some text":       addTabURL,
	}
	for in, want := range cases {
		if got := detectAddTab(in); got != want {
			t.Fatalf("detectAddTab(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestAddDetectsTorrentPathFromClipboard(t *testing.T) {
	a, _ := testApp(t)
	withClipboardText(t, "/home/u/cool.torrent")
	m := newAddModel(a)
	if m.tab != addTabTorrent || m.file.Value() != "/home/u/cool.torrent" {
		t.Fatalf("tab=%d file=%q", m.tab, m.file.Value())
	}
	if m.uris.Value() != "" {
		t.Fatalf("uris must stay empty: %q", m.uris.Value())
	}
}

func TestAddSubmitTorrentNotConnected(t *testing.T) {
	a, _ := testApp(t)
	a.client = nil
	m := newAddModel(a)
	m.tab = addTabTorrent
	p := filepath.Join(t.TempDir(), "x.torrent")
	if err := os.WriteFile(p, []byte("d4:teste"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.file.SetValue(p)
	_, cmd := m.submit()
	if cmd == nil || !a.statusErr || !strings.Contains(a.status, "not connected") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestTorrentAddedMsgError(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(torrentAddedMsg{err: errors.New("bad torrent")})
	if a.overlay == overlayFiles || !a.statusErr || !strings.Contains(a.status, "bad torrent") {
		t.Fatalf("overlay=%d status=%q", a.overlay, a.status)
	}
}

func TestFilesDataMsgIgnoredOffOverlay(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayNone
	if _, cmd := a.Update(filesDataMsg{gid: "g1", files: sampleFiles()}); cmd != nil {
		t.Fatal("filesDataMsg off the picker must be inert")
	}
}

func TestFilesRetryMsgRouting(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayFiles
	a.files = newFilesModel(a)
	a.files.gid = "g1"
	if _, cmd := a.Update(filesRetryMsg{gid: "other"}); cmd != nil {
		t.Fatal("mismatched retry must be inert")
	}
	if _, cmd := a.Update(filesRetryMsg{gid: "g1"}); cmd == nil {
		t.Fatal("matching retry must re-load")
	}
}

func TestWheelNavigatesFilesOverlay(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayFiles
	if !a.wheelNavigates() {
		t.Fatal("files overlay must wheel-scroll")
	}
}

func TestFilesOverlayRoutedThroughApp(t *testing.T) {
	a, _ := testApp(t)
	a.files = loadedFiles(a)
	a.overlay = overlayFiles
	if v := a.View(); v == "" { // view compositing branch
		t.Fatal("files overlay view empty")
	}
	_, _ = a.Update(key("j")) // handleKey overlayFiles branch
	if a.files.cursor != 1 {
		t.Fatalf("key routing cursor = %d", a.files.cursor)
	}
	click(t, a, "row:2") // handleMouse overlayFiles branch
	if a.files.cursor != 2 {
		t.Fatalf("mouse routing cursor = %d", a.files.cursor)
	}
}

func TestFilesDataMsgOnOverlay(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayFiles
	a.files = newFilesModel(a)
	a.files.gid = "g1"
	_, _ = a.Update(filesDataMsg{gid: "g1", dir: "/d", files: sampleFiles()})
	if a.files.root == nil {
		t.Fatal("filesDataMsg on the picker must absorb the files")
	}
}

func TestDetailViewTinyHeight(t *testing.T) {
	a, _ := testApp(t)
	a.height = 8 // forces the rowCap floor
	a.detail.s = rpc.Status{GID: "g", Status: "active", InfoHash: "h",
		NumPieces: "8", Bitfield: "ff", Files: []rpc.File{{Index: "1", Path: "/d/a", Length: "1"}}}
	if out := a.detail.view(); out == "" {
		t.Fatal("tiny detail view must still render")
	}
}

func TestDetailViewMinimalTorrent(t *testing.T) {
	a, _ := testApp(t)
	// BitTorrent present but no mode/comment; no pieces; paused.
	a.detail.s = rpc.Status{GID: "g", Status: "paused", InfoHash: "abcd1234",
		BitTorrent: &rpc.BTInfo{}}
	if out := a.detail.view(); !strings.Contains(out, "torrent") {
		t.Fatalf("minimal torrent view: %q", out)
	}
}

func TestFilesRowStringZeroLength(t *testing.T) {
	a, _ := testApp(t)
	m := newFilesModel(a)
	m.gid = "g1"
	m.loading = false
	m.root = buildTree([]rpc.File{{Index: "1", Path: "/d/empty.bin", Length: "0"}}, "/d")
	m.rows = flatten(m.root)
	if out := m.view(); !strings.Contains(out, "empty.bin") {
		t.Fatalf("zero-length file view: %q", out)
	}
}
