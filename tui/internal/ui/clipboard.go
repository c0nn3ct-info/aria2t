package ui

import (
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// clipboardArgvFor picks the platform's copy and paste commands.
func clipboardArgvFor(goos string) (copyArgv, pasteArgv []string) {
	if goos == "darwin" {
		return []string{"pbcopy"}, []string{"pbpaste"}
	}
	return []string{"xclip", "-selection", "clipboard"},
		[]string{"xclip", "-selection", "clipboard", "-o"}
}

var clipCopyArgv, clipPasteArgv = clipboardArgvFor(runtime.GOOS)

// clipboardWrite copies text to the system clipboard; a var so tests can
// capture instead of touching the real clipboard.
var clipboardWrite = func(text string) error {
	cmd := exec.Command(clipCopyArgv[0], clipCopyArgv[1:]...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// clipboardRead returns the clipboard text, "" when unavailable.
var clipboardRead = func() string {
	out, err := exec.Command(clipPasteArgv[0], clipPasteArgv[1:]...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// bellOut is where the terminal bell byte goes; swapped in tests.
var bellOut io.Writer = os.Stdout

// bell rings the terminal bell (BEL is invisible, so it is safe to write
// mid-render).
func bell() { _, _ = bellOut.Write([]byte{7}) }

// looksLikeSource reports whether s is something the add form accepts.
func looksLikeSource(s string) bool {
	for _, p := range []string{"http://", "https://", "ftp://", "ftps://", "sftp://", "magnet:"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
