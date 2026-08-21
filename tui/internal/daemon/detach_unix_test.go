//go:build !windows

package daemon

import (
	"syscall"
	"testing"
)

// The detach assertions are unix-only: syscall.SysProcAttr.Setsid, Getpgid and
// Getpgrp do not exist on Windows, so keeping them in an untagged file made the
// whole package fail to build there (and a package that does not build cannot
// be covered). The Windows template is asserted by detach_windows_test.go.
func TestDetachTemplate(t *testing.T) {
	if !detachTemplate.Setsid {
		t.Fatal("detachTemplate must request a new session")
	}
}

// TestStartDetached asserts a Detach child lands in its own process group,
// so signals aimed at this process's group can never reach it and it
// survives our exit (the native-messaging-host requirement).
func TestStartDetached(t *testing.T) {
	bin := fakeBin(t, `trap 'exit 0' TERM INT
while true; do sleep 0.1; done`)
	d, err := Start(Options{
		Bin:        bin,
		DataDir:    t.TempDir(),
		Detach:     true,
		ReadyProbe: func(int, string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { d.kill(); <-d.done }()
	pgid, err := syscall.Getpgid(d.cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	if self := syscall.Getpgrp(); pgid == self {
		t.Fatalf("child pgid %d equals parent pgid %d — not detached", pgid, self)
	}
	if pgid != d.cmd.Process.Pid {
		t.Fatalf("child pgid %d != child pid %d — setsid did not make it a session leader", pgid, d.cmd.Process.Pid)
	}
}
