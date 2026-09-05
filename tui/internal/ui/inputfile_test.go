package ui

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseInputFile(t *testing.T) {
	content := "# a comment\n" +
		"\n" +
		"http://example.org/a.iso\thttp://mirror/a.iso\n" +
		"  dir=/downloads\n" +
		"  out=a.iso\n" +
		"  malformed-no-equals\n" + // indented but no '=', skipped
		"magnet:?xt=urn:btih:abc\n" +
		"# trailing comment\n" +
		"  \n" + // blank-ish, ignored
		"http://example.org/b.iso\n"
	entries := parseInputFile(content)
	if len(entries) != 3 {
		t.Fatalf("entries = %d", len(entries))
	}
	if len(entries[0].uris) != 2 || entries[0].uris[1] != "http://mirror/a.iso" {
		t.Fatalf("mirrors = %v", entries[0].uris)
	}
	if entries[0].opts["dir"] != "/downloads" || entries[0].opts["out"] != "a.iso" {
		t.Fatalf("opts = %v", entries[0].opts)
	}
	if entries[1].uris[0] != "magnet:?xt=urn:btih:abc" || len(entries[1].opts) != 0 {
		t.Fatalf("entry1 = %+v", entries[1])
	}
	// An option line before any URL is ignored (no panic, no entry).
	if got := parseInputFile("  dir=/x\nhttp://y\n"); len(got) != 1 || got[0].uris[0] != "http://y" {
		t.Fatalf("leading option = %+v", got)
	}
}

func TestSplitURIsAndMergeOpts(t *testing.T) {
	if got := splitURIs("a\t\tb \t c"); strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("split = %v", got)
	}
	merged := mergeOpts(map[string]string{"dir": "/base", "split": "8"}, map[string]string{"dir": "/override"})
	if merged["dir"] != "/override" || merged["split"] != "8" {
		t.Fatalf("merged = %v", merged)
	}
}

type addURIErrAPI struct{ *fakeAPI }

func (addURIErrAPI) AddURI(context.Context, []string, map[string]string) (string, error) {
	return "", errors.New("add failed")
}

func TestAddSubmitInputFile(t *testing.T) {
	a, fake := testApp(t)
	m := newAddModel(a)
	m.tab = addTabInput

	// Empty path.
	m.file.SetValue("  ")
	if _, cmd := m.submit(); cmd == nil || !a.statusErr {
		t.Fatalf("empty path must flash, status=%q", a.status)
	}
	// Unreadable file.
	m.file.SetValue("/no/such/list.txt")
	if _, cmd := m.submit(); cmd == nil {
		t.Fatal("missing file must flash")
	} else {
		drain(t, a, cmd)
		if !a.statusErr {
			t.Fatal("missing file must flash")
		}
	}
	// No downloads in file.
	m.file.SetValue(writeTempFile(t, "empty.txt", "# only a comment\n"))
	if _, cmd := m.submit(); cmd == nil {
		t.Fatalf("empty list must flash, status=%q", a.status)
	} else {
		drain(t, a, cmd)
		if !strings.Contains(a.status, "no valid links") {
			t.Fatalf("empty list must flash, status=%q", a.status)
		}
	}
	// A binary file (e.g. a .torrent / .aria2) is rejected before parsing.
	m.file.SetValue(writeTempFile(t, "bin.aria2", "\x00\x01binary\x00stuff"))
	if _, cmd := m.submit(); cmd == nil {
		t.Fatalf("binary file must flash, status=%q", a.status)
	} else {
		drain(t, a, cmd)
		if !strings.Contains(a.status, "binary file") {
			t.Fatalf("binary file must flash, status=%q", a.status)
		}
	}
	// A file whose only lines are non-URIs (a stray path) is rejected.
	m.file.SetValue(writeTempFile(t, "junk.txt", "/Users/ivan/Movies/Films/Смешарики.aria2\n"))
	if _, cmd := m.submit(); cmd == nil {
		t.Fatalf("non-URI lines must flash, status=%q", a.status)
	} else {
		drain(t, a, cmd)
		if !strings.Contains(a.status, "no valid links") {
			t.Fatalf("non-URI lines must flash, status=%q", a.status)
		}
	}
	// Mixed valid + junk: only the valid entry is added.
	a.overlay = overlayAdd
	m.file.SetValue(writeTempFile(t, "mixed.txt", "/local/path\nhttp://x/ok.iso\n"))
	_, cmd := m.submit()
	drain(t, a, cmd)
	if len(fake.addedURIs) != 1 || fake.addedURIs[0][0] != "http://x/ok.iso" {
		t.Fatalf("only the valid URI must be added: %v", fake.addedURIs)
	}
	fake.addedURIs = nil
	// Two downloads → two adds.
	path := writeTempFile(t, "list.txt", "http://x/1.iso\n  dir=/d\nhttp://x/2.iso\n")
	m.file.SetValue(path)
	a.overlay = overlayAdd
	_, cmd = m.submit()
	if a.overlay != overlayAdd {
		t.Fatalf("form must stay visible while async work runs, overlay=%d", a.overlay)
	}
	drain(t, a, cmd)
	if len(fake.addedURIs) != 2 {
		t.Fatalf("batch must add each entry: %v", fake.addedURIs)
	}
	if !strings.Contains(a.status, "added 2") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestAddSubmitInputFileAddError(t *testing.T) {
	a, fake := testApp(t)
	a.client = addURIErrAPI{fake}
	m := newAddModel(a)
	m.tab = addTabInput
	m.file.SetValue(writeTempFile(t, "list.txt", "http://x/1.iso\n"))
	_, cmd := m.submit()
	drain(t, a, cmd)
	if !a.statusErr {
		t.Fatal("a failing batch add must flash")
	}
}

func TestAddViewInputTab(t *testing.T) {
	a, _ := testApp(t)
	m := newAddModel(a)
	m.tab = addTabInput
	out := m.view()
	if !strings.Contains(out, "Input file") || !strings.Contains(out, "input file (batch)") {
		t.Fatalf("input tab view missing labels:\n%s", out)
	}
}

func TestAddCtrlOInputExts(t *testing.T) {
	a, _ := testApp(t)
	withReadDir(t, func(string) ([]os.DirEntry, error) { return sampleDir(), nil })
	m := newAddModel(a)
	m.tab = addTabInput
	_, _ = m.update(ctrl(tea.KeyCtrlO))
	if a.overlay != overlayBrowse || a.browse.exts != nil {
		t.Fatalf("input tab browse must show all files: exts=%v", a.browse.exts)
	}
}
