//go:build windows

package daemon

import "syscall"

// detachedProcess starts the child without a console; syscall lacks the
// constant (it lives in x/sys/windows as DETACHED_PROCESS).
const detachedProcess = 0x00000008

// detachTemplate starts the child in its own process group, outside the
// parent's console, so console control events aimed at the parent never
// reach it and it survives the parent's exit.
//
// A value, not a function — see the note in detach_unix.go: a statement in a
// platform-gated file is invisible to the other platform's coverage profile.
var detachTemplate = syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess}
