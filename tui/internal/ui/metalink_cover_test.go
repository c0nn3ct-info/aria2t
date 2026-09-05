package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aria2t/internal/rpc"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

type tellStatusErrAPI struct{ *fakeAPI }

func (tellStatusErrAPI) TellStatus(context.Context, string) (rpc.Status, error) {
	return rpc.Status{}, errors.New("status unavailable")
}

func metalinkStatuses() []rpc.Status {
	return []rpc.Status{
		{GID: "g1", TotalLength: "100", Files: []rpc.File{{Path: "/dl/a.iso"}}},
		{GID: "g2", TotalLength: "200", Files: []rpc.File{{Path: "/dl/b.iso"}}},
		{GID: "g3", TotalLength: "300", Files: []rpc.File{{Path: "/dl/c.iso"}}},
	}
}

func TestBuildForestAndSelectedGids(t *testing.T) {
	root := buildForest(metalinkStatuses())
	if len(root.children) != 3 {
		t.Fatalf("forest children = %d", len(root.children))
	}
	if got := selectedGids(root); len(got) != 3 {
		t.Fatalf("all selected by default: %v", got)
	}
	root.children[1].selected = false
	got := selectedGids(root)
	if len(got) != 2 || got[0] != "g1" || got[1] != "g3" {
		t.Fatalf("selectedGids = %v", got)
	}
}

func TestAddSubmitMetalinkMulti(t *testing.T) {
	a, rec := addTestApp(t)
	rec.metaGids = []string{"g1", "g2"}
	m := newAddModel(a)
	m.tab = addTabMetalink
	m.startNow = true
	p := writeTempFile(t, "m.meta4", "<metalink/>")
	m.file.SetValue(p)
	a.overlay = overlayAdd
	_, cmd := m.submit()
	drain(t, a, cmd)
	if rec.metaOpts["pause"] != "true" {
		t.Fatalf("metalink must be added paused: %v", rec.metaOpts)
	}
	if a.overlay != overlayFiles || len(a.files.gids) != 2 || !a.files.fromAdd {
		t.Fatalf("multi-file metalink must open the picker: overlay=%d gids=%v", a.overlay, a.files.gids)
	}
}

func TestMetalinkAddedSingleAndError(t *testing.T) {
	// Single download → no picker, unpause when start-now.
	a, fake := testApp(t)
	_, cmd := a.Update(metalinkAddedMsg{gids: []string{"g1"}, unpause: true})
	drain(t, a, cmd)
	if a.overlay != overlayNone || len(fake.unpaused) != 1 || fake.unpaused[0] != "g1" {
		t.Fatalf("single metalink start: overlay=%d unpaused=%v", a.overlay, fake.unpaused)
	}
	// Empty result → just a flash.
	_, _ = a.Update(metalinkAddedMsg{gids: nil})
	if a.overlay != overlayNone {
		t.Fatal("empty metalink must not open a picker")
	}
	// Error → flash.
	_, _ = a.Update(metalinkAddedMsg{err: errors.New("bad metalink")})
	if !a.statusErr {
		t.Fatal("metalink error must flash")
	}
}

func TestFilesMultiLoad(t *testing.T) {
	a, fake := testApp(t)
	fake.status = rpc.Status{GID: "x", TotalLength: "10", Files: []rpc.File{{Path: "/dl/x.iso"}}}
	m := newFilesModel(a)
	m.gids = []string{"g1", "g2"}
	msg, ok := m.loadCmd()().(filesMultiMsg)
	if !ok || len(msg.statuses) != 2 {
		t.Fatalf("multi loadCmd = %#v", msg)
	}
	// Error path.
	a.client = tellStatusErrAPI{fake}
	msg = m.loadCmd()().(filesMultiMsg)
	if msg.err == nil {
		t.Fatal("multi loadCmd must surface a TellStatus error")
	}
}

func TestFilesMultiAbsorbConfirmCancel(t *testing.T) {
	// Absorb error.
	a, _ := testApp(t)
	m := newFilesModel(a)
	m.gids = []string{"g1"}
	m2, _ := m.absorbMulti(filesMultiMsg{err: errors.New("boom")})
	if m2.err == nil {
		t.Fatal("absorbMulti must record errors")
	}

	// Confirm: drop the deselected download, unpause the kept ones.
	a, fake := testApp(t)
	m = newFilesModel(a)
	m.gids = []string{"g1", "g2", "g3"}
	m.unpauseAfter = true
	m, _ = m.absorbMulti(filesMultiMsg{gids: m.gids, statuses: metalinkStatuses()})
	if len(m.rows) != 3 {
		t.Fatalf("rows = %d", len(m.rows))
	}
	m.cursor = 1
	m, _ = m.update(key(" ")) // deselect g2
	a.overlay = overlayFiles
	_, cmd := m.update(key("enter"))
	drain(t, a, cmd)
	if len(fake.removed) != 1 || fake.removed[0] != "g2" {
		t.Fatalf("dropped downloads must be removed: %v", fake.removed)
	}
	if len(fake.unpaused) != 2 {
		t.Fatalf("kept downloads must start: %v", fake.unpaused)
	}

	// Nothing selected → flash, stays open.
	a2, _ := testApp(t)
	m = newFilesModel(a2)
	m.gids = []string{"g1", "g2"}
	m, _ = m.absorbMulti(filesMultiMsg{gids: m.gids, statuses: metalinkStatuses()[:2]})
	setSelected(m.root, false)
	a2.overlay = overlayFiles
	_, _ = m.update(key("enter"))
	if a2.overlay != overlayFiles || !a2.statusErr {
		t.Fatalf("empty metalink selection must warn: overlay=%d", a2.overlay)
	}

	// Cancel from the add flow removes every provisional download.
	a3, fake3 := testApp(t)
	m = newFilesModel(a3)
	m.gids = []string{"g1", "g2"}
	m.fromAdd = true
	m.unpauseAfter = true
	m, _ = m.absorbMulti(filesMultiMsg{gids: m.gids, statuses: metalinkStatuses()[:2]})
	a3.overlay = overlayFiles
	_, cmd = m.update(key("esc"))
	drain(t, a3, cmd)
	if len(fake3.removed) != 2 || len(fake3.unpaused) != 0 {
		t.Fatalf("metalink cancel must remove all: removed=%v unpaused=%v", fake3.removed, fake3.unpaused)
	}
}

func TestFilesMultiMsgOffOverlay(t *testing.T) {
	a, _ := testApp(t)
	a.overlay = overlayNone
	if _, cmd := a.Update(filesMultiMsg{gids: []string{"g"}}); cmd != nil {
		t.Fatal("filesMultiMsg off the picker must be inert")
	}
}

func TestFilesMultiMsgOnOverlay(t *testing.T) {
	a, _ := testApp(t)
	a.files = newFilesModel(a)
	a.files.gids = []string{"g1"}
	a.overlay = overlayFiles
	_, _ = a.Update(filesMultiMsg{gids: []string{"g1"}, statuses: metalinkStatuses()[:1]})
	if a.files.root == nil {
		t.Fatal("filesMultiMsg on the picker must build the forest")
	}
}

func TestAddSubmitMetalinkNotConnected(t *testing.T) {
	a, _ := testApp(t)
	a.client = nil
	m := newAddModel(a)
	m.tab = addTabMetalink
	m.file.SetValue(writeTempFile(t, "m.meta4", "<metalink/>"))
	_, cmd := m.submit()
	if cmd == nil || !a.statusErr {
		t.Fatalf("metalink with no client must flash, status=%q", a.status)
	}
}

func TestMetalinkAddedSinglePaused(t *testing.T) {
	a, fake := testApp(t)
	_, _ = a.Update(metalinkAddedMsg{gids: []string{"g1"}, unpause: false})
	if a.overlay != overlayNone || len(fake.unpaused) != 0 {
		t.Fatalf("start-paused single metalink must not unpause: %v", fake.unpaused)
	}
}

// opErrAPI injects Remove / Unpause failures for the metalink error paths.
type opErrAPI struct {
	*fakeAPI
	removeErr, unpauseErr bool
}

func (f opErrAPI) Remove(ctx context.Context, gid string) error {
	if f.removeErr {
		return errors.New("remove failed")
	}
	return f.fakeAPI.Remove(ctx, gid)
}

func (f opErrAPI) Unpause(ctx context.Context, gid string) error {
	if f.unpauseErr {
		return errors.New("unpause failed")
	}
	return f.fakeAPI.Unpause(ctx, gid)
}

func TestFilesMultiConfirmAndCancelErrors(t *testing.T) {
	// Remove failure during confirm.
	a, fake := testApp(t)
	a.client = opErrAPI{fakeAPI: fake, removeErr: true}
	m := newFilesModel(a)
	m.gids = []string{"g1", "g2"}
	m.unpauseAfter = true
	m, _ = m.absorbMulti(filesMultiMsg{gids: m.gids, statuses: metalinkStatuses()[:2]})
	m.cursor = 0
	m, _ = m.update(key(" ")) // deselect g1 → a drop exists
	a.overlay = overlayFiles
	_, cmd := m.update(key("enter"))
	drain(t, a, cmd)
	if !a.statusErr {
		t.Fatal("Remove failure must flash")
	}

	// Unpause failure during confirm (keep all → no drop).
	a2, fake2 := testApp(t)
	a2.client = opErrAPI{fakeAPI: fake2, unpauseErr: true}
	m = newFilesModel(a2)
	m.gids = []string{"g1", "g2"}
	m.unpauseAfter = true
	m, _ = m.absorbMulti(filesMultiMsg{gids: m.gids, statuses: metalinkStatuses()[:2]})
	a2.overlay = overlayFiles
	_, cmd = m.update(key("enter"))
	drain(t, a2, cmd)
	if !a2.statusErr {
		t.Fatal("Unpause failure must flash")
	}

	// Remove failure during cancel.
	a3, fake3 := testApp(t)
	a3.client = opErrAPI{fakeAPI: fake3, removeErr: true}
	m = newFilesModel(a3)
	m.gids = []string{"g1"}
	m.fromAdd = true
	m.unpauseAfter = true
	m, _ = m.absorbMulti(filesMultiMsg{gids: m.gids, statuses: metalinkStatuses()[:1]})
	a3.overlay = overlayFiles
	_, cmd = m.update(key("esc"))
	drain(t, a3, cmd)
	if !a3.statusErr {
		t.Fatal("cancel Remove failure must flash")
	}
}
