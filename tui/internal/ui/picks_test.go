package ui

import (
	"errors"
	"os"
	"testing"

	"aria2t/internal/rpc"
)

// withPicksSeams swaps the persistence seams for an in-memory store.
func withPicksSeams(t *testing.T) *[]byte {
	t.Helper()
	store := new([]byte)
	missing := true
	origR, origW := picksRead, picksWrite
	picksRead = func(string) ([]byte, error) {
		if missing {
			return nil, os.ErrNotExist
		}
		return *store, nil
	}
	picksWrite = func(_ string, data []byte, _ os.FileMode) error {
		b := make([]byte, len(data))
		copy(b, data)
		*store = b
		missing = false
		return nil
	}
	t.Cleanup(func() { picksRead, picksWrite = origR, origW })
	return store
}

func TestLoadPicks(t *testing.T) {
	a, _ := testApp(t)

	// read error → nil
	origR := picksRead
	picksRead = func(string) ([]byte, error) { return nil, errors.New("boom") }
	if a.loadPicks() != nil {
		t.Fatal("read error must yield nil")
	}
	// bad json → nil
	picksRead = func(string) ([]byte, error) { return []byte("{not json"), nil }
	if a.loadPicks() != nil {
		t.Fatal("bad json must yield nil")
	}
	// ok
	picksRead = func(string) ([]byte, error) {
		return []byte(`[{"gid":"g1","kind":"torrent","unpause":true}]`), nil
	}
	got := a.loadPicks()
	if len(got) != 1 || got[0].GID != "g1" || got[0].Kind != "torrent" || !got[0].Unpause {
		t.Fatalf("loadPicks = %+v", got)
	}
	picksRead = origR
}

func TestSavePicksNilWritesEmptyArray(t *testing.T) {
	store := withPicksSeams(t)
	a, _ := testApp(t)
	a.picks = nil
	a.savePicks()
	if string(*store) != "[]\n" {
		t.Fatalf("nil picks must persist as empty array, got %q", string(*store))
	}
}

func TestAddClearPick(t *testing.T) {
	store := withPicksSeams(t)
	a, _ := testApp(t)

	a.addPick(pendingPick{GID: "g1", Kind: "torrent"})
	a.addPick(pendingPick{GID: "g1", Kind: "torrent"}) // dedupe
	if len(a.picks) != 1 {
		t.Fatalf("dedupe failed: %+v", a.picks)
	}
	a.addPick(pendingPick{GID: "g2", Kind: "magnet"})
	if len(a.picks) != 2 {
		t.Fatalf("second add failed: %+v", a.picks)
	}

	// clear with empty gid is a no-op
	*store = nil
	a.clearPick("")
	if *store != nil || len(a.picks) != 2 {
		t.Fatal("empty gid must not touch picks")
	}
	// clear a gid not present → no save, no change
	a.clearPick("nope")
	if *store != nil || len(a.picks) != 2 {
		t.Fatal("absent gid must not save")
	}
	// clear an existing gid → shrinks + persists
	a.clearPick("g1")
	if len(a.picks) != 1 || a.picks[0].GID != "g2" || *store == nil {
		t.Fatalf("clear g1 failed: picks=%+v store=%v", a.picks, *store)
	}
}

func TestReconcilePicks(t *testing.T) {
	withPicksSeams(t)
	a, _ := testApp(t)

	// already reconciled → no-op even with entries
	a.picksReconciled = true
	a.picks = []pendingPick{{GID: "x", Kind: "magnet"}}
	a.reconcilePicks(snapshot{})
	if _, ok := a.pendingMagnets["x"]; ok {
		t.Fatal("reconciled=true must skip")
	}

	// empty picks → no-op (just sets the flag)
	a.picksReconciled = false
	a.picks = nil
	a.reconcilePicks(snapshot{})
	if !a.picksReconciled {
		t.Fatal("must set reconciled")
	}

	// full reconcile: magnet present→pending+kept; torrent paused→queued+kept;
	// torrent active→dropped; absent→dropped.
	a.picksReconciled = false
	a.pendingMagnets = map[string]bool{}
	a.magnetQueue = nil
	a.picks = []pendingPick{
		{GID: "m1", Kind: "magnet", Unpause: true},
		{GID: "t1", Kind: "torrent", Unpause: true},
		{GID: "t2", Kind: "torrent"}, // present but active → drop
		{GID: "gone", Kind: "torrent"},
	}
	snap := snapshot{
		Active:  []rpc.Status{{GID: "m1", Status: "active"}, {GID: "t2", Status: "active"}},
		Waiting: []rpc.Status{{GID: "t1", Status: "paused"}},
	}
	a.reconcilePicks(snap)
	if v, ok := a.pendingMagnets["m1"]; !ok || !v {
		t.Fatalf("magnet must seed pending: %v", a.pendingMagnets)
	}
	if len(a.magnetQueue) != 1 || a.magnetQueue[0].gid != "t1" || a.magnetQueue[0].parent != "t1" {
		t.Fatalf("paused torrent must queue with parent: %+v", a.magnetQueue)
	}
	if len(a.picks) != 2 {
		t.Fatalf("stale + answered entries must drop, kept=%+v", a.picks)
	}
}

func TestTorrentAddedWritesPick(t *testing.T) {
	withPicksSeams(t)
	a, _ := testApp(t)
	_, _ = a.Update(torrentAddedMsg{gid: "T", unpause: true})
	if a.overlay != overlayFiles || a.files.pickKey != "T" {
		t.Fatalf("torrent add must open picker with pickKey: overlay=%d key=%q", a.overlay, a.files.pickKey)
	}
	if len(a.picks) != 1 || a.picks[0].GID != "T" || a.picks[0].Kind != "torrent" {
		t.Fatalf("torrent add must persist a pick: %+v", a.picks)
	}
	// error path writes nothing
	_, _ = a.Update(torrentAddedMsg{err: errors.New("bad")})
	if len(a.picks) != 1 {
		t.Fatalf("errored add must not persist: %+v", a.picks)
	}
}

func TestMagnetAddedWritesPick(t *testing.T) {
	withPicksSeams(t)
	a, _ := testApp(t)
	_, _ = a.Update(magnetAddedMsg{gid: "M", unpause: false})
	if len(a.picks) != 1 || a.picks[0].GID != "M" || a.picks[0].Kind != "magnet" {
		t.Fatalf("magnet add must persist a pick: %+v", a.picks)
	}
}

func TestConnectedLoadsPicks(t *testing.T) {
	withPicksSeams(t)
	a, fake := testApp(t)
	// seed a persisted pick, then connect
	a.picks = []pendingPick{{GID: "keep", Kind: "magnet"}}
	a.savePicks()
	a.picksReconciled = true
	_, _ = a.Update(connectedMsg{client: fake, version: "v"})
	if a.picksReconciled {
		t.Fatal("connect must reset picksReconciled")
	}
	if len(a.picks) != 1 || a.picks[0].GID != "keep" {
		t.Fatalf("connect must reload picks: %+v", a.picks)
	}
}

// TestRestartReopensPicker is the reported scenario: aria2t quit with a picker
// unanswered; on relaunch the persisted pick reopens it.
func TestRestartReopensPicker(t *testing.T) {
	withPicksSeams(t)
	a, fake := testApp(t)
	a.screen, a.overlay = screenList, overlayNone
	fake.status = rpc.Status{Dir: "/dl", Files: []rpc.File{
		{Index: "1", Path: "/dl/a.iso"}, {Index: "2", Path: "/dl/b.iso"}}}
	// Persisted from the previous run: a paused torrent awaiting selection.
	a.picks = []pendingPick{{GID: "t1", Kind: "torrent", Unpause: true}}
	a.savePicks()

	// Connect reloads picks; the first poll reconciles + presents.
	_, _ = a.Update(connectedMsg{client: fake, version: "v"})
	_, cmd := a.Update(pollMsg{snap: snapshot{Waiting: []rpc.Status{{GID: "t1", Status: "paused"}}}})
	drain(t, a, cmd)
	if a.overlay != overlayFiles || a.files.gid != "t1" {
		t.Fatalf("restart must reopen the picker: overlay=%d gid=%q", a.overlay, a.files.gid)
	}
	if a.files.pickKey != "t1" {
		t.Fatalf("reopened picker must carry its pick key: %q", a.files.pickKey)
	}
	// Answering it clears the persisted pick.
	_, cmd = a.files.update(key("esc"))
	drain(t, a, cmd)
	if len(a.picks) != 0 {
		t.Fatalf("answering must clear the persisted pick: %+v", a.picks)
	}
}
