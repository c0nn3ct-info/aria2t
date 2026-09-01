package daemon

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// swapOSExecutable replaces the os.Executable seam for the duration of a test.
func swapOSExecutable(t *testing.T, fn func() (string, error)) {
	t.Helper()
	old := osExecutable
	osExecutable = fn
	t.Cleanup(func() { osExecutable = old })
}

// swapCommonPaths replaces commonPaths for the duration of a test.
func swapCommonPaths(t *testing.T, paths []string) {
	t.Helper()
	old := commonPaths
	commonPaths = paths
	t.Cleanup(func() { commonPaths = old })
}

func TestFindBinaryCommonPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // LookPath must miss
	dir := t.TempDir()
	real := filepath.Join(dir, "aria2c")
	if err := os.WriteFile(real, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	swapCommonPaths(t, []string{
		filepath.Join(dir, "missing"), // Stat error
		dir,                           // exists but is a directory
		real,                          // the one to find
	})
	got, err := FindBinary()
	if err != nil || got != real {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestFindBinaryNotFoundAnywhere(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	swapCommonPaths(t, []string{filepath.Join(t.TempDir(), "missing")})
	// The sentinel, not just any error: the native host tags its ack from it
	// so the extension can tell a missing aria2c from a missing host.
	if _, err := FindBinary(); !errors.Is(err, ErrBinaryNotFound) {
		t.Fatalf("err = %v, want ErrBinaryNotFound", err)
	}
}

func TestFreePortListenError(t *testing.T) {
	old := netListen
	netListen = func(network, address string) (net.Listener, error) {
		return nil, errors.New("no sockets")
	}
	t.Cleanup(func() { netListen = old })
	if _, err := FreePort(); err == nil || !strings.Contains(err.Error(), "no sockets") {
		t.Fatalf("err = %v", err)
	}
}

func TestRandomSecretReadError(t *testing.T) {
	old := randRead
	randRead = func([]byte) (int, error) { return 0, errors.New("entropy drought") }
	t.Cleanup(func() { randRead = old })
	if _, err := randomSecret(); err == nil || !strings.Contains(err.Error(), "entropy drought") {
		t.Fatalf("err = %v", err)
	}
}

func TestDefaultProbeError(t *testing.T) {
	// Grab a port and close it so nothing listens there.
	port, err := FreePort()
	if err != nil {
		t.Fatal(err)
	}
	if err := defaultProbe(port, "secret"); err == nil {
		t.Fatal("want error probing a closed port")
	}
}

func TestStartFindBinaryError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	swapCommonPaths(t, nil)
	if _, err := Start(Options{DataDir: t.TempDir()}); err == nil {
		t.Fatal("want error")
	}
}

func TestStartFreePortError(t *testing.T) {
	old := netListen
	netListen = func(network, address string) (net.Listener, error) {
		return nil, errors.New("no sockets")
	}
	t.Cleanup(func() { netListen = old })
	_, err := Start(Options{Bin: fakeBin(t, "exit 0"), DataDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "no sockets") {
		t.Fatalf("err = %v", err)
	}
}

func TestStartRandomSecretError(t *testing.T) {
	old := randRead
	randRead = func([]byte) (int, error) { return 0, errors.New("entropy drought") }
	t.Cleanup(func() { randRead = old })
	_, err := Start(Options{Bin: fakeBin(t, "exit 0"), DataDir: t.TempDir(), Port: 6800})
	if err == nil || !strings.Contains(err.Error(), "entropy drought") {
		t.Fatalf("err = %v", err)
	}
}

func TestStartDefaultDataDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp) // os.TempDir() honors TMPDIR
	bin := fakeBin(t, `trap 'exit 0' TERM INT
while true; do sleep 0.1; done`)
	d, err := Start(Options{
		Bin:        bin,
		ReadyProbe: func(int, string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { d.kill(); <-d.done }()
	if st, err := os.Stat(filepath.Join(os.TempDir(), "aria2t")); err != nil || !st.IsDir() {
		t.Fatalf("default data dir not created: st=%v err=%v", st, err)
	}
}

func TestStartMkdirAllError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Start(Options{
		Bin:        fakeBin(t, "exit 0"),
		DataDir:    filepath.Join(file, "sub"), // parent is a file
		ReadyProbe: func(int, string) error { return nil },
	})
	if err == nil {
		t.Fatal("want error")
	}
}

func TestStopEscalatesToKill(t *testing.T) {
	// The child ignores every catchable shutdown signal, so Stop must walk
	// the whole escalation and SIGKILL it. Two deterministic guards:
	//   - The ready file makes Start block until the trap is actually
	//     installed. Without it, macOS can stall the exec of a freshly
	//     written script for seconds (syspolicyd assessment of new
	//     executables), SIGTERM then lands on the default handler and the
	//     child dies before the escalation this test exists to cover.
	//   - The duration check fails loudly if the child still died early,
	//     instead of silently passing with the SIGKILL path unexercised.
	ready := filepath.Join(t.TempDir(), "ready")
	bin := fakeBin(t, `trap '' TERM INT HUP
touch "`+ready+`"
while true; do sleep 0.1; done`)
	d, err := Start(Options{
		Bin:     bin,
		DataDir: t.TempDir(),
		ReadyProbe: func(int, string) error {
			_, err := os.Stat(ready)
			return err
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	start := time.Now()
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 4*time.Second {
		t.Fatalf("Stop returned after %v — child died before the SIGKILL escalation", elapsed)
	}
}

func TestStopReturnsWhenProcessExitsEarly(t *testing.T) {
	// The child exits on its own shortly after start; Stop's first wait
	// observes done before any signal is needed.
	bin := fakeBin(t, "sleep 1")
	d, err := Start(Options{
		Bin:        bin,
		DataDir:    t.TempDir(),
		ReadyProbe: func(int, string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestAria2cName(t *testing.T) {
	if got := aria2cName("windows"); got != "aria2c.exe" {
		t.Fatalf("windows: got %q", got)
	}
	if got := aria2cName("darwin"); got != "aria2c" {
		t.Fatalf("darwin: got %q", got)
	}
}

// The Windows installer drops aria2c beside aria2t rather than relying on a
// PATH the current process has not re-read; FindBinary must see it there.
func TestFindBinaryBesideExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // LookPath must miss
	swapCommonPaths(t, nil)
	dir := t.TempDir()
	beside := filepath.Join(dir, aria2cName(runtime.GOOS))
	if err := os.WriteFile(beside, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	swapOSExecutable(t, func() (string, error) { return filepath.Join(dir, "aria2t"), nil })
	got, err := FindBinary()
	if err != nil || got != beside {
		t.Fatalf("got %q err=%v", got, err)
	}
}

// A directory named aria2c beside the binary is not a usable aria2c, and an
// unresolvable executable path is not fatal — both fall through to commonPaths.
func TestFindBinaryBesideRejected(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, aria2cName(runtime.GOOS)), 0o755); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(t.TempDir(), "aria2c")
	if err := os.WriteFile(real, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	swapCommonPaths(t, []string{real})

	swapOSExecutable(t, func() (string, error) { return filepath.Join(dir, "aria2t"), nil })
	if got, err := FindBinary(); err != nil || got != real {
		t.Fatalf("dir beside: got %q err=%v", got, err)
	}

	swapOSExecutable(t, func() (string, error) { return "", errors.New("no exe") })
	if got, err := FindBinary(); err != nil || got != real {
		t.Fatalf("exe error: got %q err=%v", got, err)
	}
}
