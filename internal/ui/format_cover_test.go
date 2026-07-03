package ui

import (
	"strings"
	"testing"
	"time"
)

func TestFmtBytesEdges(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{-5, "0 B"},
		{0, "0 B"},
		{999, "999 B"},
		{5 * 1024, "5.0 KiB"},
		{48 * 1024 * 1024, "48 MiB"},
		{6 << 50, "6.0 PiB"},
	}
	for _, c := range cases {
		if got := FmtBytes(c.n); got != c.want {
			t.Errorf("FmtBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestFmtSpeedEdges(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{-1, "—"},
		{0, "—"},
		{500, "500 B/s"},
		{720 * 1024, "720 KiB/s"},
		{13002342, "12.4 MiB/s"},
		{3 << 40, "3.0 TiB/s"},
	}
	for _, c := range cases {
		if got := FmtSpeed(c.n); got != c.want {
			t.Errorf("FmtSpeed(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestFmtETAEdges(t *testing.T) {
	if got := FmtETA(0, 100); got != "—" {
		t.Errorf("no remaining = %q", got)
	}
	if got := FmtETA(100, 0); got != "—" {
		t.Errorf("no speed = %q", got)
	}
	if got := FmtETA(100, 10); got != "10s" {
		t.Errorf("eta = %q", got)
	}
}

func TestFmtDurationEdges(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{19 * time.Second, "19s"},
		{7*time.Minute + 12*time.Second, "7m 12s"},
		{time.Hour + 4*time.Minute, "1h 4m"},
	}
	for _, c := range cases {
		if got := FmtDuration(c.d); got != c.want {
			t.Errorf("FmtDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestFmtAgoEdges(t *testing.T) {
	now := time.Now()
	cases := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-10 * time.Second), "just now"},
		{now.Add(-2 * time.Minute), "2m ago"},
		{now.Add(-3 * time.Hour), "3h ago"},
		{now.Add(-49 * time.Hour), "2d ago"},
	}
	for _, c := range cases {
		if got := FmtAgo(c.t, now); got != c.want {
			t.Errorf("FmtAgo = %q, want %q", got, c.want)
		}
	}
}

func TestBarEdges(t *testing.T) {
	if f, e := Bar(0.5, 0); f != "" || e != "" {
		t.Errorf("zero width: %q %q", f, e)
	}
	if f, e := Bar(-1, 4); f != "" || e != "────" {
		t.Errorf("negative frac: %q %q", f, e)
	}
	if f, e := Bar(2, 4); f != "━━━━" || e != "" {
		t.Errorf("overflowing frac: %q %q", f, e)
	}
	if f, e := Bar(0.5, 10); f != "━━━━╸" || e != "─────" {
		t.Errorf("half: %q %q", f, e)
	}
}

func TestPiecesEdges(t *testing.T) {
	if got := Pieces("ff", 0, 4); got != "░░░░" {
		t.Errorf("no pieces = %q", got)
	}
	if got := Pieces("ff", 0, -1); got != "" {
		t.Errorf("negative width = %q", got)
	}
	// Malformed hex digits are skipped.
	if got := Pieces("zz", 8, 2); got != "░░" {
		t.Errorf("malformed hex = %q", got)
	}
	// Fewer pieces than cells exercises the hi<=lo guard.
	if got := Pieces("8", 2, 4); !strings.Contains(got, "█") {
		t.Errorf("sparse pieces = %q", got)
	}
	// Full, partial, and empty buckets.
	if got := Pieces("f0", 8, 2); got != "█░" {
		t.Errorf("buckets = %q", got)
	}
	if got := Pieces("8", 2, 1); got != "▓" {
		t.Errorf("partial bucket = %q", got)
	}
}

func TestSparkEdges(t *testing.T) {
	if got := Spark([]int64{1}, 1, 0); got != "" {
		t.Errorf("zero width = %q", got)
	}
	// More samples than width keeps the tail.
	if got := Spark([]int64{0, 0, 8}, 8, 1); got != "█" {
		t.Errorf("trimmed = %q", got)
	}
	// Zero max renders the floor rune for every sample.
	if got := Spark([]int64{5}, 0, 1); got != "▁" {
		t.Errorf("zero max = %q", got)
	}
	// Sample above max clamps to the top level.
	if got := Spark([]int64{20}, 10, 1); got != "█" {
		t.Errorf("clamped = %q", got)
	}
	// Tiny non-zero sample gets a visible floor.
	if got := Spark([]int64{1}, 1000000, 1); got != "▂" {
		t.Errorf("floor = %q", got)
	}
	// Shorter than width pads with leading spaces.
	if got := Spark([]int64{8}, 8, 3); got != "  █" {
		t.Errorf("padded = %q", got)
	}
}

func TestParseLimitEdges(t *testing.T) {
	ok := []struct{ in, want string }{
		{"", "0"},
		{"∞", "0"},
		{"0", "0"},
		{"5m", "5M"},
		{"256K", "256K"},
		{"1.5G", "1536M"},
		{"1234", "1234"},
	}
	for _, c := range ok {
		got, err := ParseLimit(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseLimit(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"abc", "K", "1.2.3M"} {
		if _, err := ParseLimit(bad); err == nil {
			t.Errorf("ParseLimit(%q) must fail", bad)
		}
	}
}

func TestFmtLimitEdges(t *testing.T) {
	if got := FmtLimit(""); got != "∞" {
		t.Errorf("empty = %q", got)
	}
	if got := FmtLimit("0"); got != "∞" {
		t.Errorf("zero = %q", got)
	}
	if got := FmtLimit("5M"); got != "5M" {
		t.Errorf("value = %q", got)
	}
}
