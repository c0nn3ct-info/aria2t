package ui

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// Tests must never touch the real system clipboard or terminal: TestMain
// swaps the integration points for inert stand-ins. The real defaults are
// saved first so their bodies can still be covered below.
var realClipboardRead, realClipboardWrite = clipboardRead, clipboardWrite

func TestMain(m *testing.M) {
	clipboardRead = func() string { return "" }
	clipboardWrite = func(string) error { return nil }
	bellOut = io.Discard
	os.Exit(m.Run())
}

func TestClipboardDefaultBodies(t *testing.T) {
	origC, origP := clipCopyArgv, clipPasteArgv
	t.Cleanup(func() { clipCopyArgv, clipPasteArgv = origC, origP })

	clipCopyArgv = []string{"cat"} // absorbs stdin, exits 0
	if err := realClipboardWrite("hello"); err != nil {
		t.Fatalf("write via cat: %v", err)
	}
	clipCopyArgv = []string{"/nonexistent-clipboard-bin"}
	if err := realClipboardWrite("x"); err == nil {
		t.Fatal("missing binary must error")
	}

	clipPasteArgv = []string{"printf", "abc"}
	if got := realClipboardRead(); got != "abc" {
		t.Fatalf("read = %q", got)
	}
	clipPasteArgv = []string{"/nonexistent-clipboard-bin"}
	if got := realClipboardRead(); got != "" {
		t.Fatalf("missing binary must read empty, got %q", got)
	}
}

func TestClipboardArgvFor(t *testing.T) {
	c, p := clipboardArgvFor("darwin")
	if c[0] != "pbcopy" || p[0] != "pbpaste" {
		t.Fatalf("darwin argv = %v %v", c, p)
	}
	c, p = clipboardArgvFor("linux")
	if c[0] != "xclip" || p[len(p)-1] != "-o" {
		t.Fatalf("linux argv = %v %v", c, p)
	}
}

func TestBellWritesBEL(t *testing.T) {
	var buf bytes.Buffer
	orig := bellOut
	bellOut = &buf
	t.Cleanup(func() { bellOut = orig })
	bell()
	if buf.String() != "\a" {
		t.Fatalf("bell wrote %q", buf.String())
	}
}

func TestLooksLikeSource(t *testing.T) {
	for s, want := range map[string]bool{
		"https://mirror/x.iso": true,
		"http://mirror/x.iso":  true,
		"ftp://mirror/x":       true,
		"ftps://mirror/x":      true,
		"sftp://mirror/x":      true,
		"magnet:?xt=urn:btih:": true,
		"meeting notes":        false,
		"":                     false,
	} {
		if got := looksLikeSource(s); got != want {
			t.Fatalf("looksLikeSource(%q) = %v", s, got)
		}
	}
}
