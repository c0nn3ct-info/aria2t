package ui

import (
	"strings"
	"testing"
	"time"
)

func TestFmtBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{48 << 20, "48 MiB"},
		{658505728, "628 MiB"},
		{6227702579, "5.8 GiB"},
		{1503238553, "1.4 GiB"},
		{912 << 20, "912 MiB"},
	}
	for _, tc := range cases {
		if got := FmtBytes(tc.in); got != tc.want {
			t.Errorf("FmtBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFmtSpeed(t *testing.T) {
	if got := FmtSpeed(0); got != "—" {
		t.Errorf("idle = %q", got)
	}
	if got := FmtSpeed(13002342); got != "12.4 MiB/s" {
		t.Errorf("got %q", got)
	}
}

func TestFmtETA(t *testing.T) {
	if got := FmtETA(0, 100); got != "—" {
		t.Errorf("done = %q", got)
	}
	if got := FmtETA(100, 0); got != "—" {
		t.Errorf("stalled = %q", got)
	}
	if got := FmtETA(1900, 100); got != "19s" {
		t.Errorf("got %q", got)
	}
	if got := FmtETA(43200, 100); got != "7m 12s" {
		t.Errorf("got %q", got)
	}
}

func TestFmtAgo(t *testing.T) {
	now := time.Now()
	if got := FmtAgo(now.Add(-2*time.Minute), now); got != "2m ago" {
		t.Errorf("got %q", got)
	}
	if got := FmtAgo(now.Add(-10*time.Second), now); got != "just now" {
		t.Errorf("got %q", got)
	}
}

func TestBar(t *testing.T) {
	filled, empty := Bar(0.62, 20)
	if filled != "━━━━━━━━━━━╸" || empty != "────────" {
		t.Errorf("62%%: %q + %q", filled, empty)
	}
	filled, empty = Bar(0, 5)
	if filled != "" || empty != "─────" {
		t.Errorf("0%%: %q + %q", filled, empty)
	}
	filled, empty = Bar(1, 5)
	if filled != "━━━━━" || empty != "" {
		t.Errorf("100%%: %q + %q", filled, empty)
	}
}

func TestPieces(t *testing.T) {
	// 8 pieces, bitfield 0xF0 → first 4 complete, last 4 missing, 4 cells
	got := Pieces("f0", 8, 4)
	if got != "██░░" {
		t.Errorf("got %q", got)
	}
	// partial bucket: 0xA0 = 10100000 → cells of 2: [1,0][1,0][0,0][0,0]
	got = Pieces("a0", 8, 4)
	if got != "▓▓░░" {
		t.Errorf("got %q", got)
	}
	if got := Pieces("", 0, 4); got != "░░░░" {
		t.Errorf("empty: %q", got)
	}
}

func TestSpark(t *testing.T) {
	got := Spark([]int64{0, 50, 100}, 100, 3)
	if []rune(got)[0] != '▁' || []rune(got)[2] != '█' {
		t.Errorf("got %q", got)
	}
	if got := Spark([]int64{100}, 100, 3); len([]rune(got)) != 3 || !strings.HasPrefix(got, "  ") {
		t.Errorf("padding: %q", got)
	}
}

func TestParseLimit(t *testing.T) {
	for in, want := range map[string]string{"∞": "0", "": "0", "0": "0", "5M": "5M", "256k": "256K", "1.5m": "1.5M", "1024": "1024"} {
		got, err := ParseLimit(in)
		if err != nil || got != want {
			t.Errorf("ParseLimit(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseLimit("fast"); err == nil {
		t.Error("want error for junk")
	}
	if got := FmtLimit("0"); got != "∞" {
		t.Errorf("FmtLimit: %q", got)
	}
}
