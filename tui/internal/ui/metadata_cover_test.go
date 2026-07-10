package ui

import (
	"testing"

	"aria2t/internal/rpc"
)

func TestMetadataLeftoverStrippedAndPurged(t *testing.T) {
	a, fake := testApp(t)
	meta := rpc.Status{GID: "m1", Status: "complete", InfoHash: "deadbeefcafe",
		Files: []rpc.File{{Path: "[METADATA]deadbeefcafe"}}}
	real := rpc.Status{GID: "r1", Status: "complete", Files: []rpc.File{{Path: "/dl/x.iso"}}}

	_, cmd := a.Update(pollMsg{snap: snapshot{Stopped: []rpc.Status{meta, real}}})
	if len(a.snap.Stopped) != 1 || a.snap.Stopped[0].GID != "r1" {
		t.Fatalf("metadata leftover must be hidden: %+v", a.snap.Stopped)
	}
	drain(t, a, cmd)
	if len(fake.removedResults) != 1 || fake.removedResults[0] != "m1" {
		t.Fatalf("metadata must be purged once: %v", fake.removedResults)
	}
	// A second identical poll must not re-purge.
	_, cmd = a.Update(pollMsg{snap: snapshot{Stopped: []rpc.Status{meta, real}}})
	drain(t, a, cmd)
	if len(fake.removedResults) != 1 {
		t.Fatalf("purge must fire once per gid: %v", fake.removedResults)
	}
}

func TestActiveMetadataStaysVisible(t *testing.T) {
	a, _ := testApp(t)
	active := rpc.Status{GID: "m1", Status: "active", InfoHash: "abc",
		Files: []rpc.File{{Path: "[METADATA]abc"}}}
	_, _ = a.Update(pollMsg{snap: snapshot{Active: []rpc.Status{active}}})
	if len(a.snap.Active) != 1 {
		t.Fatal("an in-progress metadata download must stay visible")
	}
}

func TestRemoveResultsCmdGuards(t *testing.T) {
	a, _ := testApp(t)
	if a.removeResultsCmd(nil) != nil {
		t.Fatal("no gids → nil cmd")
	}
	a.client = nil
	if a.removeResultsCmd([]string{"x"}) != nil {
		t.Fatal("nil client → nil cmd")
	}
}
