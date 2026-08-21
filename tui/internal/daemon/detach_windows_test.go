//go:build windows

package daemon

import (
	"syscall"
	"testing"
)

// Counterpart to detach_unix_test.go: the flags aria2c must be spawned with so
// it outlives this process (own process group, no console). Neither file's
// subject contributes a statement to the coverage profile — see detach_unix.go
// — but the value still deserves an assertion on the platform that has it.
func TestDetachTemplate(t *testing.T) {
	if detachTemplate.CreationFlags&syscall.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Error("detachTemplate must request a new process group")
	}
	if detachTemplate.CreationFlags&detachedProcess == 0 {
		t.Error("detachTemplate must detach from the parent console")
	}
}
