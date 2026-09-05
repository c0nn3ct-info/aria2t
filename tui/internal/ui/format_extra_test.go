package ui

import (
	"strings"
	"testing"
)

func TestBitfieldProgress(t *testing.T) {
	if got := bitfieldProgress("ff", 0); got != 0 {
		t.Fatalf("zero pieces = %f", got)
	}
	if got := bitfieldProgress("ff", 8); got != 1 {
		t.Fatalf("full = %f", got)
	}
	if got := bitfieldProgress("80", 8); got < 0.12 || got > 0.13 {
		t.Fatalf("one bit = %f", got)
	}
	if got := bitfieldProgress("zz", 8); got != 0 {
		t.Fatalf("garbage hex must count nothing = %f", got)
	}
	// Trailing bits beyond numPieces are ignored (no over-count).
	if got := bitfieldProgress("ffff", 8); got != 1 {
		t.Fatalf("over-length bitfield = %f", got)
	}
}

func TestTrunc(t *testing.T) {
	if got := trunc("hello", 10); got != "hello" {
		t.Fatalf("short = %q", got)
	}
	if got := trunc("hello", 1); got != "…" {
		t.Fatalf("w=1 = %q", got)
	}
	if got := trunc("hello world", 5); got != "hell…" {
		t.Fatalf("long = %q", got)
	}
}

func TestFriendlyError(t *testing.T) {
	cases := map[string]string{
		"2":  "timed out",
		"3":  "404",
		"4":  "404",
		"5":  "too slow",
		"6":  "network",
		"8":  "resuming",
		"9":  "disk space",
		"11": "same file",
		"12": "same torrent",
		"13": "already exists",
		"15": "open the destination",
		"16": "create or truncate",
		"17": "read/write",
		"18": "download directory",
		"19": "resolve host",
		"20": "metalink",
		"22": "rejected",
		"23": "redirects",
		"24": "login",
		"25": "parse the .torrent",
		"26": "corrupt",
		"27": "magnet",
	}
	for code, want := range cases {
		if got := friendlyError(code, "raw"); !strings.Contains(got, want) {
			t.Fatalf("friendlyError(%q) = %q, want substring %q", code, got, want)
		}
	}
	// Unknown code falls back to the raw message, then to a generic string.
	if got := friendlyError("999", "custom message"); got != "custom message" {
		t.Fatalf("fallback msg = %q", got)
	}
	if got := friendlyError("999", ""); got != "error 999" {
		t.Fatalf("fallback code = %q", got)
	}
	if got := friendlyError("", ""); got != "error" {
		t.Fatalf("fallback empty = %q", got)
	}
	if got := friendlyError("0", ""); got != "error" {
		t.Fatalf("code zero = %q", got)
	}
}

func TestCapRows(t *testing.T) {
	st := NewStyles(TokyoNight)
	if got := capRows(nil, 3, st.Dim, "empty"); len(got) != 1 || !strings.Contains(got[0], "empty") {
		t.Fatalf("empty = %v", got)
	}
	rows := []string{"a", "b", "c", "d"}
	if got := capRows(rows, 9, st.Dim, ""); len(got) != 4 {
		t.Fatalf("under cap = %v", got)
	}
	if got := capRows(rows, 2, st.Dim, ""); len(got) != 2 || !strings.Contains(got[1], "more") {
		t.Fatalf("over cap = %v", got)
	}
	if got := capRows(rows, 0, st.Dim, ""); len(got) != 1 || !strings.Contains(got[0], "more") {
		t.Fatalf("cap<1 = %v", got)
	}
}
