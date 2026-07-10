package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"aria2t/internal/rpc"
)

// picksRead and picksWrite are indirections for tests.
var (
	picksRead  = os.ReadFile
	picksWrite = os.WriteFile
)

// pendingPick is one download added paused and awaiting file selection. It is
// persisted so a picker left unanswered when the user quits is reopened on the
// next launch, honouring the "choose files before the download starts" rule.
// GID is the identifier aria2 preserves across --save-session/--input-file: the
// metadata gid for a magnet, the torrent gid for a torrent-file add. (Local
// metalink adds save no meaningful gid, so they are not persisted.)
type pendingPick struct {
	GID     string `json:"gid"`
	Kind    string `json:"kind"` // "magnet" | "torrent"
	Unpause bool   `json:"unpause"`
}

// picksPath is the sidecar next to the managed daemon's data.
func (a *App) picksPath() string {
	return filepath.Join(filepath.Dir(a.cfgPath), "daemon", "picks.json")
}

// loadPicks reads the persisted pending picks; any error yields none.
func (a *App) loadPicks() []pendingPick {
	raw, err := picksRead(a.picksPath())
	if err != nil {
		return nil
	}
	var p []pendingPick
	if json.Unmarshal(raw, &p) != nil {
		return nil
	}
	return p
}

// savePicks writes the current set (best effort). A nil slice is written as an
// empty JSON array, so a cleared set leaves a clean file rather than stale data.
func (a *App) savePicks() {
	p := a.picks
	if p == nil {
		p = []pendingPick{}
	}
	raw, _ := json.MarshalIndent(p, "", "  ") // cannot fail for this type
	_ = picksWrite(a.picksPath(), append(raw, '\n'), 0o600)
}

// addPick records a new pending pick (deduping by gid) and persists it.
func (a *App) addPick(p pendingPick) {
	for _, e := range a.picks {
		if e.GID == p.GID {
			return
		}
	}
	a.picks = append(a.picks, p)
	a.savePicks()
}

// clearPick removes an answered pick and persists the shrunk set.
func (a *App) clearPick(gid string) {
	if gid == "" {
		return
	}
	kept := make([]pendingPick, 0, len(a.picks))
	for _, e := range a.picks {
		if e.GID != gid {
			kept = append(kept, e)
		}
	}
	if len(kept) != len(a.picks) {
		a.picks = kept
		a.savePicks()
	}
}

// reconcilePicks runs once after a (re)connect: it matches the persisted picks
// against the first snapshot and re-drives the existing picker machinery — a
// magnet is fed back into pendingMagnets (so resolveMagnets reopens it on its
// new followedBy child), a still-paused torrent is queued directly. Entries
// whose download is gone, or whose torrent is no longer paused (already
// answered), are dropped.
func (a *App) reconcilePicks(snap snapshot) {
	if a.picksReconciled {
		return
	}
	a.picksReconciled = true
	if len(a.picks) == 0 {
		return
	}
	byGID := map[string]rpc.Status{}
	for _, lst := range [][]rpc.Status{snap.Active, snap.Waiting, snap.Stopped} {
		for _, s := range lst {
			byGID[s.GID] = s
		}
	}
	kept := make([]pendingPick, 0, len(a.picks))
	for _, e := range a.picks {
		s, ok := byGID[e.GID]
		if !ok {
			continue // download no longer present
		}
		switch e.Kind {
		case "magnet":
			a.pendingMagnets[e.GID] = e.Unpause
			kept = append(kept, e)
		case "torrent":
			if s.Status == "paused" {
				a.magnetQueue = append(a.magnetQueue, magnetReady{gid: e.GID, unpause: e.Unpause, parent: e.GID})
				kept = append(kept, e)
			}
		}
	}
	if len(kept) != len(a.picks) {
		a.picks = kept
		a.savePicks()
	}
}
