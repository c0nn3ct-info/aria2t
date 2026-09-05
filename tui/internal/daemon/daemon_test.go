package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeBin writes an executable shell script and returns its path.
func fakeBin(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-aria2c")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFreePort(t *testing.T) {
	p1, err := FreePort()
	if err != nil || p1 <= 0 || p1 > 65535 {
		t.Fatalf("p1=%d err=%v", p1, err)
	}
}

func TestRandomSecret(t *testing.T) {
	s1, err := randomSecret()
	if err != nil || len(s1) != 32 {
		t.Fatalf("s1=%q err=%v", s1, err)
	}
	s2, _ := randomSecret()
	if s1 == s2 {
		t.Fatal("secrets must differ")
	}
}

func TestBuildArgs(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "session.txt")
	args := buildArgs("/dl", session, "/tmp/l.log", "/tmp/a.conf", 1234)
	joined := strings.Join(args, " ")
	for _, want := range []string{"--enable-rpc", "--rpc-listen-port=1234", "--conf-path=/tmp/a.conf",
		"--save-session=" + session, "--force-save=true", "--dir=/dl", "--rpc-listen-all=false",
		"--bt-save-metadata=true", "--bt-load-saved-metadata=true"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q in %q", want, joined)
		}
	}
	if strings.Contains(joined, "rpc-secret") {
		t.Error("secret must not appear on argv")
	}
	if strings.Contains(joined, "--input-file") {
		t.Error("input-file must be absent when session file does not exist")
	}
	// existing session → resumed
	if err := os.WriteFile(session, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	joined = strings.Join(buildArgs("", session, "/tmp/l.log", "/tmp/a.conf", 1234), " ")
	if !strings.Contains(joined, "--input-file="+session) {
		t.Error("input-file must be passed for an existing session")
	}
	if strings.Contains(joined, "--dir=") {
		t.Error("empty dir must not produce --dir flag")
	}
}

func TestFindBinary(t *testing.T) {
	// A stub named aria2c placed on PATH must be found.
	dir := t.TempDir()
	stub := filepath.Join(dir, "aria2c")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := FindBinary()
	if err != nil || got != stub {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestFindBinaryMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if runtimeHasCommonAria2c(t) {
		t.Skip("aria2c present in a common path on this machine")
	}
	if _, err := FindBinary(); err == nil {
		t.Fatal("want error")
	}
}

func runtimeHasCommonAria2c(t *testing.T) bool {
	t.Helper()
	for _, p := range commonPaths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func TestStartSuccessAndStopViaSignal(t *testing.T) {
	bin := fakeBin(t, `trap 'exit 0' TERM INT
while true; do sleep 0.1; done`)
	d, err := Start(Options{
		Bin:        bin,
		DataDir:    t.TempDir(),
		ReadyProbe: func(port int, secret string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Port <= 0 || len(d.Secret) != 32 {
		t.Fatalf("port=%d secret=%q", d.Port, d.Secret)
	}
	if !strings.HasPrefix(d.URL(), "ws://localhost:") {
		t.Fatalf("url=%q", d.URL())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestStartExitsDuringStartup(t *testing.T) {
	bin := fakeBin(t, `echo "boom: bad option" >&2
exit 1`)
	_, err := Start(Options{
		Bin:        bin,
		DataDir:    t.TempDir(),
		ReadyProbe: func(int, string) error { return errors.New("not yet") },
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestStartReadyTimeout(t *testing.T) {
	bin := fakeBin(t, `trap 'exit 0' TERM
while true; do sleep 0.1; done`)
	_, err := Start(Options{
		Bin:         bin,
		DataDir:     t.TempDir(),
		ReadyProbe:  func(int, string) error { return errors.New("never ready") },
		ReadyWithin: 300 * time.Millisecond,
	})
	if err == nil || !strings.Contains(err.Error(), "did not become ready") {
		t.Fatalf("err = %v", err)
	}
}

func TestStartMissingBinary(t *testing.T) {
	_, err := Start(Options{Bin: filepath.Join(t.TempDir(), "nope"), DataDir: t.TempDir()})
	if err == nil {
		t.Fatal("want error")
	}
}

// TestRealAria2c drives a genuine aria2c end to end when it is installed:
// spawn on a free port, wait ready via RPC, stop gracefully.
func TestRealAria2c(t *testing.T) {
	if _, err := FindBinary(); err != nil {
		t.Skip("aria2c not installed")
	}
	dataDir := t.TempDir()
	d, err := Start(Options{Dir: t.TempDir(), DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	// graceful stop must have saved a session file
	if _, err := os.Stat(filepath.Join(dataDir, "session.txt")); err != nil {
		t.Fatalf("session file not written: %v", err)
	}
}

// A detached child (the native host's spawn) has no stderr pipe to us, so its
// last words are in the log file it was told to write. Without reading it the
// error was the bare "exited during startup: " the extension then showed.
func TestStartExitsDuringStartupDetachedReadsTheLog(t *testing.T) {
	bin := fakeBin(t, `for a in "$@"; do
  case "$a" in --log=*) printf 'first line\n[ERROR] boom from the log\n' >> "${a#--log=}" ;; esac
done
exit 1`)
	_, err := Start(Options{
		Bin:        bin,
		DataDir:    t.TempDir(),
		Detach:     true,
		ReadyProbe: func(int, string) error { return errors.New("not yet") },
	})
	if err == nil || !strings.Contains(err.Error(), "boom from the log") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "no output") {
		t.Fatalf("a log with content must not read as silence: %v", err)
	}
}

func TestStartExitsDuringStartupWithNothingToSay(t *testing.T) {
	bin := fakeBin(t, `exit 1`)
	dataDir := t.TempDir()
	_, err := Start(Options{
		Bin:        bin,
		DataDir:    dataDir,
		Detach:     true,
		ReadyProbe: func(int, string) error { return errors.New("not yet") },
	})
	want := "no output (see " + filepath.Join(dataDir, "aria2.log") + ")"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

func TestLogTail(t *testing.T) {
	dir := t.TempDir()
	if got := LogTail(filepath.Join(dir, "missing.log"), 5); got != nil {
		t.Fatalf("missing file: got %v", got)
	}
	p := filepath.Join(dir, "aria2.log")
	if err := os.WriteFile(p, []byte("one\r\n\n  \ntwo\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Blank lines are skipped, CR is trimmed, and the count is of what is kept.
	if got := LogTail(p, 10); strings.Join(got, "|") != "one|two|three" {
		t.Fatalf("got %v", got)
	}
	if got := LogTail(p, 2); strings.Join(got, "|") != "two|three" {
		t.Fatalf("got %v", got)
	}
	// A log larger than the read window yields only its end.
	big := strings.Repeat("padding line that is long enough to matter\n", logTailBytes/40+10) + "last\n"
	if err := os.WriteFile(p, []byte(big), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LogTail(p, 1)
	if len(got) != 1 || got[0] != "last" {
		t.Fatalf("got %v", got)
	}
}
