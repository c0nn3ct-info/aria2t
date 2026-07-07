package ui

import (
	"context"
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

// ---- pause all / resume all / purge ----

func TestPauseAllResumeAllKeys(t *testing.T) {
	a, fake := testApp(t)
	_, cmd := a.Update(key("P"))
	drain(t, a, cmd)
	if !fake.pausedAll {
		t.Fatal("P must call aria2.pauseAll")
	}
	_, cmd = a.Update(key("U"))
	drain(t, a, cmd)
	if !fake.unpausedAll {
		t.Fatal("U must call aria2.unpauseAll")
	}
}

func TestPurgeStoppedNeedsConfirm(t *testing.T) {
	a, fake := testApp(t)
	a.list.tab = tabStopped
	_, _ = a.Update(key("D"))
	if a.overlay != overlayConfirm {
		t.Fatalf("D on stopped tab must ask first, overlay=%d", a.overlay)
	}
	if fake.purged {
		t.Fatal("purge must not run before confirmation")
	}
	_, cmd := a.Update(key("y"))
	drain(t, a, cmd)
	if !fake.purged {
		t.Fatal("confirming must call aria2.purgeDownloadResult")
	}
}

func TestPurgeIgnoredOutsideStoppedTab(t *testing.T) {
	a, fake := testApp(t)
	a.list.tab = tabActive
	_, _ = a.Update(key("D"))
	if a.overlay != overlayNone || fake.purged {
		t.Fatal("D must do nothing on the active tab")
	}
}

// ---- yank ----

func withClipboard(t *testing.T) *string {
	t.Helper()
	var got string
	orig := clipboardWrite
	clipboardWrite = func(s string) error { got = s; return nil }
	t.Cleanup(func() { clipboardWrite = orig })
	return &got
}

func TestYankCopiesFirstURI(t *testing.T) {
	a, _ := testApp(t)
	got := withClipboard(t)
	a.snap.Active[0].Files = []rpc.File{{URIs: []rpc.URI{{URI: "https://mirror.example/f.iso"}}}}
	_, cmd := a.Update(key("y"))
	drain(t, a, cmd)
	if *got != "https://mirror.example/f.iso" {
		t.Fatalf("clipboard = %q", *got)
	}
	if a.status == "" || a.statusErr {
		t.Fatalf("expected success flash, got %q err=%v", a.status, a.statusErr)
	}
}

func TestYankBuildsMagnetForTorrents(t *testing.T) {
	a, _ := testApp(t)
	got := withClipboard(t)
	a.snap.Active[0].InfoHash = "deadbeefcafe"
	_, cmd := a.Update(key("y"))
	drain(t, a, cmd)
	if *got != "magnet:?xt=urn:btih:deadbeefcafe" {
		t.Fatalf("clipboard = %q", *got)
	}
}

func TestYankWithoutSourceFlashesError(t *testing.T) {
	a, _ := testApp(t)
	got := withClipboard(t)
	_, cmd := a.Update(key("y"))
	drain(t, a, cmd)
	if *got != "" {
		t.Fatalf("nothing must be copied, got %q", *got)
	}
	if !a.statusErr {
		t.Fatal("expected an error flash")
	}
}

// ---- add-overlay clipboard prefill ----

func withClipboardText(t *testing.T, text string) {
	t.Helper()
	orig := clipboardRead
	clipboardRead = func() string { return text }
	t.Cleanup(func() { clipboardRead = orig })
}

func TestAddPrefillsFromClipboard(t *testing.T) {
	a, _ := testApp(t)
	withClipboardText(t, "https://mirror.example/x.iso\n")
	_, _ = a.Update(key("a"))
	if got := a.add.uris.Value(); got != "https://mirror.example/x.iso" {
		t.Fatalf("uris = %q", got)
	}
}

func TestAddIgnoresNonURLClipboard(t *testing.T) {
	a, _ := testApp(t)
	withClipboardText(t, "meeting notes: buy milk")
	_, _ = a.Update(key("a"))
	if got := a.add.uris.Value(); got != "" {
		t.Fatalf("uris must stay empty, got %q", got)
	}
}

// ---- completion / failure notifications ----

func withBell(t *testing.T) *int {
	t.Helper()
	rings := 0
	orig := bellOut
	bellOut = writerFunc(func(p []byte) (int, error) { rings += len(p); return len(p), nil })
	t.Cleanup(func() { bellOut = orig })
	return &rings
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func TestCompletionNotifiesOnceSeeded(t *testing.T) {
	a, _ := testApp(t)
	rings := withBell(t)
	base := snapshot{Stopped: []rpc.Status{{GID: "s1", Status: "complete"}}}
	_, _ = a.Update(pollMsg{snap: base})
	if *rings != 0 || a.status != "" {
		t.Fatalf("first poll must seed silently, rings=%d status=%q", *rings, a.status)
	}
	next := snapshot{Stopped: []rpc.Status{
		{GID: "s1", Status: "complete"},
		{GID: "s2", Status: "complete", Files: []rpc.File{{Path: "/dl/fedora.iso"}}},
	}}
	_, _ = a.Update(pollMsg{snap: next})
	if *rings == 0 {
		t.Fatal("a new completion must ring the bell")
	}
	if !strings.Contains(a.status, "fedora.iso") || a.statusErr {
		t.Fatalf("expected success flash with the name, got %q err=%v", a.status, a.statusErr)
	}
}

func TestFailureNotifiesWithMessage(t *testing.T) {
	a, _ := testApp(t)
	rings := withBell(t)
	_, _ = a.Update(pollMsg{snap: snapshot{}})
	next := snapshot{Stopped: []rpc.Status{
		{GID: "e1", Status: "error", ErrorMessage: "HTTP 404 Not Found",
			Files: []rpc.File{{Path: "/dl/gone.iso"}}},
	}}
	_, _ = a.Update(pollMsg{snap: next})
	if *rings == 0 {
		t.Fatal("a new failure must ring the bell")
	}
	if !a.statusErr || !strings.Contains(a.status, "gone.iso") || !strings.Contains(a.status, "404") {
		t.Fatalf("expected error flash with name and message, got %q", a.status)
	}
}

func TestReconnectReseedsSilently(t *testing.T) {
	a, fake := testApp(t)
	rings := withBell(t)
	_, _ = a.Update(pollMsg{snap: snapshot{}})
	_, _ = a.Update(connectedMsg{client: fake, version: "t", endpoint: "x"})
	full := snapshot{Stopped: []rpc.Status{{GID: "old", Status: "complete"}}}
	_, _ = a.Update(pollMsg{snap: full})
	if *rings != 0 {
		t.Fatal("history replay after reconnect must stay silent")
	}
}

// ---- error text surfaced ----

func TestStoppedTabShowsErrorMessage(t *testing.T) {
	a, _ := testApp(t)
	a.snap.Stopped = []rpc.Status{{GID: "e1", Status: "error",
		ErrorMessage: "HTTP authorization failed",
		Files:        []rpc.File{{Path: "/dl/secret.iso"}}}}
	a.list.tab = tabStopped
	if v := a.View(); !strings.Contains(v, "authorization failed") {
		t.Fatal("stopped tab must surface the aria2 error message")
	}
}

func TestDetailShowsErrorMessage(t *testing.T) {
	a, _ := testApp(t)
	a.screen = screenDetail
	a.detail.gid = "e1"
	a.detail.s = rpc.Status{GID: "e1", Status: "error", ErrorMessage: "HTTP 404 Not Found"}
	if v := a.View(); !strings.Contains(v, "404") {
		t.Fatal("detail must surface the aria2 error message")
	}
}

// ---- live filter ----

func namedStatus(gid, name, status string) rpc.Status {
	return rpc.Status{GID: gid, Status: status, Files: []rpc.File{{Path: "/dl/" + name}}}
}

func filterApp(t *testing.T) *App {
	a, _ := testApp(t)
	a.list.tab = tabActive // exercise the filter over a single known group
	a.snap.Active = []rpc.Status{
		namedStatus("a1", "ubuntu.iso", "active"),
		namedStatus("a2", "fedora.iso", "active"),
		namedStatus("a3", "debian.iso", "active"),
	}
	return a
}

func typeString(a *App, s string) {
	for _, r := range s {
		_, _ = a.Update(key(string(r)))
	}
}

func TestFilterNarrowsLive(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	if !a.list.filtering {
		t.Fatal("/ must enter filter typing mode")
	}
	typeString(a, "fed")
	rows := a.list.rows()
	if len(rows) != 1 || rows[0].Name() != "fedora.iso" {
		t.Fatalf("rows = %v", rows)
	}
	_, _ = a.Update(key("enter"))
	if a.list.filtering {
		t.Fatal("enter must leave typing mode")
	}
	if len(a.list.rows()) != 1 {
		t.Fatal("committed filter must stay applied")
	}
	_, _ = a.Update(key("esc"))
	if len(a.list.rows()) != 3 {
		t.Fatal("esc must clear a committed filter")
	}
}

func TestFilterEscWhileTypingClears(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	typeString(a, "ubu")
	_, _ = a.Update(key("esc"))
	if a.list.filtering || len(a.list.rows()) != 3 {
		t.Fatalf("esc while typing must cancel and clear, rows=%d", len(a.list.rows()))
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	typeString(a, "FED")
	if len(a.list.rows()) != 1 {
		t.Fatal("filter must be case-insensitive")
	}
}

func TestFilterQuestionMarkGoesIntoQuery(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	typeString(a, "?")
	if a.overlay == overlayHelp {
		t.Fatal("? while typing must not open help")
	}
	if got := a.list.filterInput.Value(); got != "?" {
		t.Fatalf("query = %q", got)
	}
}

func TestFilterCursorClamps(t *testing.T) {
	a := filterApp(t)
	a.list.cursor = 2
	_, _ = a.Update(key("/"))
	typeString(a, "fed")
	if s, ok := a.list.selected(); !ok || s.Name() != "fedora.iso" {
		t.Fatalf("cursor must clamp into the filtered rows, sel=%v ok=%v", s, ok)
	}
}

func TestFilterBlocksReorder(t *testing.T) {
	a := filterApp(t)
	a.list.tab = tabWaiting
	_, _ = a.Update(key("/"))
	typeString(a, "w")
	_, _ = a.Update(key("enter"))
	_, _ = a.Update(key("J"))
	if a.list.reordering {
		t.Fatal("reorder must be refused while a filter is active")
	}
	if !a.statusErr {
		t.Fatal("expected an explanatory error flash")
	}
}

func TestFilterClearsOnTabSwitch(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	typeString(a, "fed")
	_, _ = a.Update(key("enter"))
	_, _ = a.Update(key("tab"))
	if got := a.list.filterInput.Value(); got != "" {
		t.Fatalf("tab switch must clear the filter, got %q", got)
	}
}

func TestFilterBadgeRendered(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	typeString(a, "fed")
	_, _ = a.Update(key("enter"))
	if v := a.View(); !strings.Contains(v, "⌕ fed") {
		t.Fatal("tabs line must show the active filter")
	}
}

func TestWheelDoesNotTypeIntoFilter(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	if a.wheelNavigates() {
		t.Fatal("wheel must not synthesize j/k into the filter input")
	}
}

func TestYankClipboardFailureFlashes(t *testing.T) {
	a, _ := testApp(t)
	orig := clipboardWrite
	clipboardWrite = func(string) error { return context.DeadlineExceeded }
	t.Cleanup(func() { clipboardWrite = orig })
	a.snap.Active[0].InfoHash = "beef"
	_, cmd := a.Update(key("y"))
	drain(t, a, cmd)
	if !a.statusErr || !strings.Contains(a.status, "clipboard") {
		t.Fatalf("expected clipboard error flash, got %q", a.status)
	}
}

func TestNoticeSilentWhenNothingNew(t *testing.T) {
	a, _ := testApp(t)
	rings := withBell(t)
	snap := snapshot{Stopped: []rpc.Status{{GID: "s1", Status: "complete"}}}
	_, _ = a.Update(pollMsg{snap: snap})
	_, _ = a.Update(pollMsg{snap: snap})
	if *rings != 0 || a.status != "" {
		t.Fatalf("unchanged stopped list must stay silent, status=%q", a.status)
	}
}

func TestNoticeCountsExtras(t *testing.T) {
	a, _ := testApp(t)
	_ = withBell(t)
	_, _ = a.Update(pollMsg{snap: snapshot{}})
	next := snapshot{Stopped: []rpc.Status{
		{GID: "n1", Status: "complete", Files: []rpc.File{{Path: "/dl/one.iso"}}},
		{GID: "n2", Status: "complete", Files: []rpc.File{{Path: "/dl/two.iso"}}},
	}}
	_, _ = a.Update(pollMsg{snap: next})
	if !strings.Contains(a.status, "+1 more") {
		t.Fatalf("expected +1 more, got %q", a.status)
	}
}

func TestKeybarShowsFilterInputWhileTyping(t *testing.T) {
	a := filterApp(t)
	_, _ = a.Update(key("/"))
	if v := a.View(); !strings.Contains(v, "↵ keep") {
		t.Fatal("keybar must show the filter input while typing")
	}
}
