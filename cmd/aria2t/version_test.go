package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var buf bytes.Buffer
	if code := run([]string{"--version"}, &buf); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	out := buf.String()
	if !strings.Contains(out, "aria2t") || !strings.Contains(out, appVersion) {
		t.Fatalf("version output = %q", out)
	}
}
