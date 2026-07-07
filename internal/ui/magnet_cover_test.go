package ui

import (
	"errors"
	"testing"

	"aria2t/internal/rpc"
)

func TestAddSubmitMagnet(t *testing.T) {
	a, rec := addTestApp(t)
	m := newAddModel(a)
	m.uris.SetValue("magnet:?xt=urn:btih:deadbeef")
	m.startNow = true
	a.overlay = overlayAdd
	_, cmd := m.submit()
	if a.overlay != overlayNone {
		t.Fatalf("submit must close the add overlay, overlay=%d", a.overlay)
	}
	drain(t, a, cmd)
	if rec.uriOpts["pause-metadata"] != "true" {
		t.Fatalf("magnet must set pause-metadata: %v", rec.uriOpts)
	}
	if !a.pendingMagnets["gid"] {
		t.Fatalf("magnet gid must be pending with start=true: %v", a.pendingMagnets)
	}

	// Not connected.
	a2, _ := testApp(t)
	a2.client = nil
	m2 := newAddModel(a2)
	m2.uris.SetValue("magnet:?xt=urn:btih:abc")
	_, cmd = m2.submit()
	if cmd == nil || !a2.statusErr {
		t.Fatalf("magnet with no client must flash, status=%q", a2.status)
	}
}

func TestMagnetAddedMsg(t *testing.T) {
	a, _ := testApp(t)
	_, _ = a.Update(magnetAddedMsg{gid: "m1", unpause: true})
	if !a.pendingMagnets["m1"] {
		t.Fatalf("magnet must register as pending: %v", a.pendingMagnets)
	}
	_, _ = a.Update(magnetAddedMsg{err: errors.New("bad magnet")})
	if !a.statusErr {
		t.Fatal("magnet add error must flash")
	}
}

func TestResolveMagnets(t *testing.T) {
	a, _ := testApp(t)
	if a.resolveMagnets(snapshot{}) != nil {
		t.Fatal("no pending magnets → nil cmd")
	}

	// Pending but not yet seen / metadata still going → stays pending.
	a.pendingMagnets["m1"] = true
	a.resolveMagnets(snapshot{})
	if !a.pendingMagnets["m1"] {
		t.Fatal("unseen magnet must stay pending")
	}
	a.resolveMagnets(snapshot{Active: []rpc.Status{{GID: "m1", Status: "active"}}})
	if !a.pendingMagnets["m1"] {
		t.Fatal("magnet without followedBy must stay pending")
	}

	// Metadata error → dropped + flash, never queued.
	a.resolveMagnets(snapshot{Stopped: []rpc.Status{{GID: "m1", Status: "error"}}})
	if a.pendingMagnets["m1"] || !a.statusErr || len(a.magnetQueue) != 0 {
		t.Fatalf("errored magnet must drop and flash: pending=%v queue=%v", a.pendingMagnets, a.magnetQueue)
	}

	// Resolved → queued (paused), pending dropped, but NOT opened by resolve.
	a.pendingMagnets["m2"] = true
	a.resolveMagnets(snapshot{Stopped: []rpc.Status{{GID: "m2", Status: "complete", FollowedBy: []string{"c2"}}}})
	if a.pendingMagnets["m2"] || a.overlay != overlayNone {
		t.Fatalf("resolve must queue, not open: overlay=%d", a.overlay)
	}
	if len(a.magnetQueue) != 1 || a.magnetQueue[0].gid != "c2" {
		t.Fatalf("resolved magnet must be queued: %v", a.magnetQueue)
	}

	// presentNextMagnet opens it when idle and drains the queue.
	a.screen, a.overlay = screenList, overlayNone
	cmd := a.presentNextMagnet()
	if a.overlay != overlayFiles || a.files.gid != "c2" || !a.files.fromAdd || cmd == nil {
		t.Fatalf("present must open the picker: overlay=%d gid=%q", a.overlay, a.files.gid)
	}
	if len(a.magnetQueue) != 0 {
		t.Fatal("queue must drain")
	}

	// Present is a no-op with an empty queue, or when the UI is busy.
	if a.presentNextMagnet() != nil {
		t.Fatal("empty queue → nil")
	}
	a.magnetQueue = []magnetReady{{gid: "c9"}}
	a.overlay = overlayAdd
	if a.presentNextMagnet() != nil || len(a.magnetQueue) != 1 {
		t.Fatal("busy UI must leave the magnet queued")
	}
}

// TestMagnetsPresentedSequentially is the reported scenario: several magnets
// finish while the first picker is unanswered — they queue and appear one by one.
func TestMagnetsPresentedSequentially(t *testing.T) {
	a, fake := testApp(t)
	a.screen, a.overlay = screenList, overlayNone
	// Multi-file torrents, so the picker stays open (single-file magnets would
	// auto-finish and not need a picker).
	fake.status = rpc.Status{Dir: "/dl", Files: []rpc.File{
		{Index: "1", Path: "/dl/a.iso"}, {Index: "2", Path: "/dl/b.iso"}}}
	a.pendingMagnets["m1"] = true
	a.pendingMagnets["m2"] = true

	// One poll: both magnets' metadata resolves.
	_, cmd := a.Update(pollMsg{snap: snapshot{Stopped: []rpc.Status{
		{GID: "m1", Status: "complete", FollowedBy: []string{"c1"}, Files: []rpc.File{{Path: "[METADATA]m1"}}},
		{GID: "m2", Status: "complete", FollowedBy: []string{"c2"}, Files: []rpc.File{{Path: "[METADATA]m2"}}},
	}}})
	drain(t, a, cmd)
	if a.overlay != overlayFiles {
		t.Fatal("the first magnet's picker must open")
	}
	if len(a.magnetQueue) != 1 || a.files.moreQueued != 1 {
		t.Fatalf("the second must wait in the queue: queue=%d moreQueued=%d", len(a.magnetQueue), a.files.moreQueued)
	}
	first := a.files.gid
	if len(fake.paused) != 2 {
		t.Fatalf("both torrents must be paused while waiting: %v", fake.paused)
	}

	// Answer the first picker (esc closes it).
	_, cmd = a.files.update(key("esc"))
	drain(t, a, cmd)
	if a.overlay != overlayNone {
		t.Fatal("first picker must close")
	}

	// The next poll presents the second magnet.
	_, cmd = a.Update(pollMsg{snap: snapshot{}})
	drain(t, a, cmd)
	if a.overlay != overlayFiles || a.files.gid == first || len(a.magnetQueue) != 0 {
		t.Fatalf("the second magnet's picker must follow: gid=%q queue=%d", a.files.gid, len(a.magnetQueue))
	}
}
