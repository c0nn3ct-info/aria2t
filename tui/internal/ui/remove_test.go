package ui

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"aria2t/internal/rpc"
)

// stopped and active are the two shapes removeFn branches on.
func stoppedDL() rpc.Status { return rpc.Status{Status: "complete"} }
func activeDL() rpc.Status  { return rpc.Status{Status: "active"} }

// A stopped download is already a result: remove it with a single purge and no
// aria2.remove call.
func TestRemoveFnStopped(t *testing.T) {
	a, _ := testApp(t)
	fake := newFakeAPI()
	if err := a.removeFn("s1", stoppedDL())(context.Background(), fake); err != nil {
		t.Fatalf("stopped remove err: %v", err)
	}
	if len(fake.removed) != 0 {
		t.Fatalf("stopped remove must not call aria2.remove: %v", fake.removed)
	}
	if len(fake.removedResults) != 1 || fake.removedResults[0] != "s1" {
		t.Fatalf("stopped remove must purge the result: %v", fake.removedResults)
	}
}

// A stopped download whose purge aria2 refuses is reported rather than silently
// treated as removed — and nothing on disk is touched, since it is still there.
func TestRemoveFnStoppedPurgeError(t *testing.T) {
	a, _ := testApp(t)
	rec := stubClean(t, nil)
	fake := newFakeAPI()
	fake.removeResultFailsN = 1
	s := rpc.Status{Status: "complete", Dir: "/dl", Files: []rpc.File{{Path: "/dl/a.iso"}}}
	if err := a.removeFn("s1", s)(context.Background(), fake); err == nil {
		t.Fatal("a refused purge must be returned")
	}
	if rec.called {
		t.Fatal("nothing may be deleted when the download is still there")
	}
}

// A failing aria2.remove is surfaced and the result is never purged.
func TestRemoveFnRemoveError(t *testing.T) {
	a, _ := testApp(t)
	fake := newFakeAPI()
	fake.removeErr = errors.New("boom")
	if err := a.removeFn("a1", activeDL())(context.Background(), fake); err == nil {
		t.Fatal("remove error must be returned")
	}
	if len(fake.removedResults) != 0 {
		t.Fatalf("purge must not run when remove fails: %v", fake.removedResults)
	}
}

// The common case: aria2.remove stops the download and the result is purgeable
// immediately, so a single purge attempt deletes it.
func TestRemoveFnActiveImmediate(t *testing.T) {
	a, _ := testApp(t)
	fake := newFakeAPI()
	if err := a.removeFn("a1", activeDL())(context.Background(), fake); err != nil {
		t.Fatalf("active remove err: %v", err)
	}
	if len(fake.removed) != 1 || fake.removed[0] != "a1" {
		t.Fatalf("active remove must call aria2.remove: %v", fake.removed)
	}
	if len(fake.removedResults) != 1 || fake.removedResults[0] != "a1" {
		t.Fatalf("active remove must purge the result: %v", fake.removedResults)
	}
}

// A seeding torrent: aria2.remove is async, so the first purge attempts fail
// until teardown completes. removeFn retries until the result lands.
func TestRemoveFnSeedingRetries(t *testing.T) {
	defer swapPurgeInterval(time.Millisecond)()
	a, _ := testApp(t)
	fake := newFakeAPI()
	fake.removeResultFailsN = 3 // three misses, then the result exists
	if err := a.removeFn("seed1", activeDL())(context.Background(), fake); err != nil {
		t.Fatalf("seeding remove err: %v", err)
	}
	if len(fake.removedResults) != 1 || fake.removedResults[0] != "seed1" {
		t.Fatalf("seeding remove must eventually purge: %v", fake.removedResults)
	}
}

// If the download never becomes a purgeable result, removeFn gives up when the
// context is done rather than blocking forever.
func TestRemoveFnGivesUpOnCtxDone(t *testing.T) {
	defer swapPurgeInterval(time.Hour)()
	a, _ := testApp(t)
	fake := newFakeAPI()
	fake.removeResultFailsN = 1 << 30 // never succeeds
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.removeFn("x1", activeDL())(ctx, fake); err != nil {
		t.Fatalf("give-up path must return nil, got %v", err)
	}
	if len(fake.removedResults) != 0 {
		t.Fatalf("nothing should be purged when it never settles: %v", fake.removedResults)
	}
}

// swapPurgeInterval overrides removePurgeInterval and returns a restore func.
func swapPurgeInterval(d time.Duration) func() {
	old := removePurgeInterval
	removePurgeInterval = d
	return func() { removePurgeInterval = old }
}

// Removing forgets the download in aria2 but touches nothing on disk, so the
// .aria2 control file and the <infohash>.torrent the daemon saves under
// --bt-save-metadata stayed behind in the download folder.
func TestRemoveFnTakesTheLeftoversWithIt(t *testing.T) {
	a, _ := testApp(t)
	rec := stubClean(t, nil)
	bt := &rpc.BTInfo{}
	bt.Info.Name = "Show"
	s := rpc.Status{
		Status:     "complete",
		Dir:        "/dl",
		InfoHash:   "1e838bf16c32054f7d7a02197a20deb0545e74b7",
		BitTorrent: bt,
		Files:      []rpc.File{{Path: "/dl/Show/a.mkv"}},
	}
	if err := a.removeFn("g1", s)(context.Background(), newFakeAPI()); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	want := []string{
		"/dl/Show/a.mkv.aria2",
		"/dl/Show.aria2",
		"/dl/1e838bf16c32054f7d7a02197a20deb0545e74b7.torrent",
	}
	if !rec.called || rec.gid != "g1" || !reflect.DeepEqual(rec.files, want) {
		t.Fatalf("clean = %+v, want files %v", rec, want)
	}
	if rec.dataDir != filepath.Join(filepath.Dir(a.cfgPath), "daemon") {
		t.Fatalf("dataDir = %q", rec.dataDir)
	}
}

// The same for a live download, whose removal is asynchronous.
func TestRemoveFnCleansAfterAnActiveRemoval(t *testing.T) {
	defer swapPurgeInterval(time.Millisecond)()
	a, _ := testApp(t)
	rec := stubClean(t, nil)
	fake := newFakeAPI()
	fake.removeResultFailsN = 2
	s := rpc.Status{Status: "active", Dir: "/dl", Files: []rpc.File{{Path: "/dl/a.iso"}}}
	if err := a.removeFn("a1", s)(context.Background(), fake); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	if !rec.called || !reflect.DeepEqual(rec.files, []string{"/dl/a.iso.aria2"}) {
		t.Fatalf("clean = %+v", rec)
	}
}

// A purge that never lands still leaves nothing behind: aria2.remove succeeded,
// so the download is stopped and its bookkeeping is dead either way.
func TestRemoveFnCleansEvenWhenThePurgeNeverLands(t *testing.T) {
	defer swapPurgeInterval(time.Hour)()
	a, _ := testApp(t)
	rec := stubClean(t, nil)
	fake := newFakeAPI()
	fake.removeResultFailsN = 1 << 30
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := rpc.Status{Status: "active", Dir: "/dl", Files: []rpc.File{{Path: "/dl/a.iso"}}}
	if err := a.removeFn("a1", s)(ctx, fake); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	if !rec.called {
		t.Fatal("the leftovers outlive a purge that never settles")
	}
}

// A removal aria2 refused changes nothing on disk either.
func TestRemoveFnCleansNothingWhenTheRemovalFails(t *testing.T) {
	a, _ := testApp(t)
	rec := stubClean(t, nil)
	fake := newFakeAPI()
	fake.removeErr = errors.New("boom")
	s := rpc.Status{Status: "active", Dir: "/dl", Files: []rpc.File{{Path: "/dl/a.iso"}}}
	if err := a.removeFn("a1", s)(context.Background(), fake); err == nil {
		t.Fatal("remove error must be returned")
	}
	if rec.called {
		t.Fatal("nothing may be deleted when the download is still there")
	}
}

// The opt-out is honoured on removal too, the same rule the completion path and
// the native host both follow.
func TestRemoveFnHonoursTheControlOptOut(t *testing.T) {
	a, _ := testApp(t)
	a.cfg.KeepControl = true
	rec := stubClean(t, nil)
	s := rpc.Status{Status: "complete", Dir: "/dl", Files: []rpc.File{{Path: "/dl/a.iso"}}}
	if err := a.removeFn("g1", s)(context.Background(), newFakeAPI()); err != nil {
		t.Fatalf("remove err: %v", err)
	}
	if rec.called {
		t.Fatal("cleanup must not run when the user opted out")
	}
}
