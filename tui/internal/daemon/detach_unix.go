//go:build !windows

package daemon

import "syscall"

// detachTemplate places the child in its own session (and thus its own process
// group, with no controlling terminal), so signals delivered to the parent's
// group never reach it and it survives the parent's exit.
//
// A package-level value rather than a function on purpose: this file is not
// compiled on Windows and its counterpart is not compiled here, so a statement
// in either would be absent from the other platform's coverage profile — a
// 100% report that never saw the code. Var initializers are not instrumented,
// so the platform split costs no coverage blind spot; the statements that
// consume it live in daemon.go, which is compiled everywhere.
var detachTemplate = syscall.SysProcAttr{Setsid: true}
