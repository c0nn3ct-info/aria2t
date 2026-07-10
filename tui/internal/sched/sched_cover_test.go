package sched

import (
	"testing"
	"time"

	"aria2t/internal/config"
)

func TestParseHMMalformed(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", -1},           // no colon
		{"0900", -1},       // no colon
		{"ab:cd", -1},      // junk numbers
		{"12:xx", -1},      // junk minute
		{"xx:30", -1},      // junk hour
		{"25:00", -1},      // hour out of range
		{"-1:00", -1},      // negative hour
		{"12:60", -1},      // minute out of range
		{"12:-5", -1},      // negative minute
		{"24:00", 24 * 60}, // 24:00 allowed
		{"09:30", 9*60 + 30},
		{"00:00", 0},
	}
	for _, c := range cases {
		if got := parseHM(c.in); got != c.want {
			t.Errorf("parseHM(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestActiveSkipsMalformedRules(t *testing.T) {
	rules := []config.Rule{
		{Start: "bogus", End: "18:00", Days: allDays, Down: "1M"}, // malformed start
		{Start: "09:00", End: "junk", Days: allDays, Down: "2M"},  // malformed end
		{Start: "09:00", End: "18:00", Days: allDays, Down: "5M"}, // valid
	}
	r, ok := Active(rules, at(time.Wednesday, 12, 0))
	if !ok || r.Down != "5M" {
		t.Fatalf("malformed rules must be skipped, got ok=%v r=%+v", ok, r)
	}
	// only malformed rules: nothing matches
	if _, ok := Active(rules[:2], at(time.Wednesday, 12, 0)); ok {
		t.Fatal("malformed-only rules must never match")
	}
}
