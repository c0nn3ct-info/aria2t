package ui

import (
	"errors"
	"path/filepath"
	"testing"

	"aria2t/internal/rpc"
)

type cleanCall struct {
	dataDir string
	gid     string
	files   []string
	called  bool
}

// stubClean swaps the control.Clean seam, recording the call.
func stubClean(t *testing.T, err error) *cleanCall {
	t.Helper()
	rec := &cleanCall{}
	orig := cleanControlFiles
	cleanControlFiles = func(dataDir, gid string, files []string) error {
		rec.dataDir, rec.gid, rec.files, rec.called = dataDir, gid, files, true
		return err
	}
	t.Cleanup(func() { cleanControlFiles = orig })
	return rec
}

// A finished download's control file is deleted, and the deletion is recorded
// against the daemon's stable state dir so the next launch drops it from the
// session.
func TestNoticeStoppedCleansControlFile(t *testing.T) {
	a, _ := testApp(t)
	rec := stubClean(t, errors.New("ignored: cleanup is best effort"))
	a.stoppedSeeded = true
	a.knownStopped = map[string]bool{}

	a.noticeStopped([]rpc.Status{{
		GID:    "g1",
		Status: "complete",
		Dir:    "/dl",
		Files:  []rpc.File{{Path: "/dl/ubuntu.iso"}},
	}})

	if !rec.called {
		t.Fatal("control file was not cleaned")
	}
	if want := filepath.Join(filepath.Dir(a.cfgPath), "daemon"); rec.dataDir != want {
		t.Fatalf("dataDir = %q, want %q", rec.dataDir, want)
	}
	if rec.gid != "g1" {
		t.Fatalf("gid = %q, want g1", rec.gid)
	}
	if len(rec.files) != 1 || rec.files[0] != "/dl/ubuntu.iso.aria2" {
		t.Fatalf("files = %v", rec.files)
	}
}

// A multi-file torrent's control file sits beside its folder, named after the
// torrent — so the torrent name has to reach control.Paths.
func TestCleanControlIncludesTorrentName(t *testing.T) {
	a, _ := testApp(t)
	rec := stubClean(t, nil)
	s := rpc.Status{
		GID:      "g2",
		Status:   "complete",
		Dir:      "/dl",
		InfoHash: "ff",
		Files:    []rpc.File{{Path: "/dl/Movie/a.mkv"}},
	}
	s.BitTorrent = &rpc.BTInfo{}
	s.BitTorrent.Info.Name = "Movie"
	a.cleanControl(s)

	want := []string{"/dl/Movie/a.mkv.aria2", "/dl/Movie.aria2"}
	if len(rec.files) != len(want) {
		t.Fatalf("files = %v, want %v", rec.files, want)
	}
	for i := range want {
		if rec.files[i] != want[i] {
			t.Fatalf("files = %v, want %v", rec.files, want)
		}
	}
}

// The setting is the user's opt-out; with it off nothing is deleted.
func TestCleanControlRespectsOptOut(t *testing.T) {
	a, _ := testApp(t)
	rec := stubClean(t, nil)
	a.cfg.KeepControl = true

	a.cleanControl(rpc.Status{GID: "g1", Files: []rpc.File{{Path: "/dl/f.iso"}}})

	if rec.called {
		t.Fatal("keepControl must suppress the deletion")
	}
}
