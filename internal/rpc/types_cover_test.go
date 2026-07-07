package rpc

import "testing"

func TestErrorFormatting(t *testing.T) {
	e := &Error{Code: 1, Message: "Unauthorized"}
	if got := e.Error(); got != "aria2: Unauthorized (1)" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestFileGetters(t *testing.T) {
	f := File{Length: "2048", CompletedLength: "1024", Selected: "true"}
	if f.Len() != 2048 || f.Completed() != 1024 || !f.IsSelected() {
		t.Fatalf("bad file getters: %+v", f)
	}
	if (File{Selected: "false"}).IsSelected() {
		t.Fatal("Selected=false must not be selected")
	}
}

func TestStatusGetters(t *testing.T) {
	s := Status{
		UploadLength:  "512",
		UploadSpeed:   "64",
		NumPieces:     "10",
		PieceLength:   "16384",
		NumSeeders:    "3",
		Connections:   "5",
		InfoHash:      "deadbeef",
		DownloadSpeed: "128",
	}
	if s.Uploaded() != 512 || s.UpSpeed() != 64 || s.Pieces() != 10 ||
		s.PieceLen() != 16384 || s.Seeds() != 3 || s.Conns() != 5 || s.DownSpeed() != 128 {
		t.Fatalf("bad status getters: %+v", s)
	}
	if !s.IsTorrent() {
		t.Fatal("infoHash set must mean torrent")
	}
	if (Status{}).IsTorrent() {
		t.Fatal("empty infoHash must not mean torrent")
	}
}

func TestStatusRatio(t *testing.T) {
	if r := (Status{CompletedLength: "0", UploadLength: "100"}).Ratio(); r != 0 {
		t.Fatalf("ratio with zero completed = %f", r)
	}
	if r := (Status{CompletedLength: "100", UploadLength: "50"}).Ratio(); r != 0.5 {
		t.Fatalf("ratio = %f, want 0.5", r)
	}
}

func TestStatusNameMemoryPlaceholder(t *testing.T) {
	// aria2 reports "[MEMORY]xxxx" paths for in-memory control files; with no
	// URIs the name falls through to the gid. ([METADATA] is handled
	// separately by IsMetadata — see TestStatusMetadata.)
	s := Status{GID: "gid9", Files: []File{{Path: "[MEMORY]control"}}}
	if got := s.Name(); got != "gid9" {
		t.Fatalf("Name() = %q, want gid fallback", got)
	}
}

func TestPeerGetters(t *testing.T) {
	p := Peer{DownloadSpeed: "100", UploadSpeed: "200"}
	if p.DownSpeed() != 100 || p.UpSpeed() != 200 {
		t.Fatalf("bad peer getters: %+v", p)
	}
}

func TestGlobalStatGetters(t *testing.T) {
	g := GlobalStat{
		DownloadSpeed: "1000",
		UploadSpeed:   "2000",
		NumActive:     "1",
		NumWaiting:    "2",
		NumStopped:    "3",
	}
	if g.DownSpeed() != 1000 || g.UpSpeed() != 2000 ||
		g.Active() != 1 || g.Waiting() != 2 || g.Stopped() != 3 {
		t.Fatalf("bad global stat getters: %+v", g)
	}
}
