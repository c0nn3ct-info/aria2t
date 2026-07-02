package sched

import (
	"testing"
	"time"

	"aria2t/internal/config"
)

var allDays = [7]bool{true, true, true, true, true, true, true}

// at builds a time on a known weekday: 2026-07-01 is a Wednesday.
func at(weekday time.Weekday, hh, mm int) time.Time {
	base := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC) // Sunday
	return base.AddDate(0, 0, int(weekday)).Add(time.Duration(hh*60+mm) * time.Minute)
}

func TestActiveInsideWindow(t *testing.T) {
	rules := []config.Rule{{Start: "09:00", End: "18:00", Days: allDays, Down: "5M", Up: "256K"}}
	r, ok := Active(rules, at(time.Wednesday, 12, 30))
	if !ok || r.Down != "5M" {
		t.Fatalf("ok=%v r=%+v", ok, r)
	}
	if _, ok := Active(rules, at(time.Wednesday, 18, 0)); ok {
		t.Fatal("end is exclusive")
	}
	if _, ok := Active(rules, at(time.Wednesday, 8, 59)); ok {
		t.Fatal("before start must not match")
	}
}

func TestWindowCrossingMidnight(t *testing.T) {
	rules := []config.Rule{{Start: "21:00", End: "06:00", Days: allDays, Down: "0", Up: "0"}}
	if _, ok := Active(rules, at(time.Wednesday, 23, 0)); !ok {
		t.Fatal("23:00 must match")
	}
	if _, ok := Active(rules, at(time.Thursday, 5, 0)); !ok {
		t.Fatal("05:00 next day must match")
	}
	if _, ok := Active(rules, at(time.Thursday, 6, 0)); ok {
		t.Fatal("06:00 must not match")
	}
}

func TestDayMask(t *testing.T) {
	weekdays := [7]bool{false, true, true, true, true, true, false}
	rules := []config.Rule{{Start: "09:00", End: "18:00", Days: weekdays, Down: "5M"}}
	if _, ok := Active(rules, at(time.Saturday, 12, 0)); ok {
		t.Fatal("saturday must not match")
	}
	if _, ok := Active(rules, at(time.Monday, 12, 0)); !ok {
		t.Fatal("monday must match")
	}
	// midnight-crossing rule enabled Friday only: Saturday 05:00 still matches
	night := []config.Rule{{Start: "21:00", End: "06:00", Days: [7]bool{5: true}, Down: "1M"}}
	if _, ok := Active(night, at(time.Saturday, 5, 0)); !ok {
		t.Fatal("early saturday belongs to friday's night window")
	}
	if _, ok := Active(night, at(time.Saturday, 22, 0)); ok {
		t.Fatal("saturday night not enabled")
	}
}

func TestFirstRuleWins(t *testing.T) {
	rules := []config.Rule{
		{Start: "09:00", End: "18:00", Days: allDays, Down: "5M"},
		{Start: "06:00", End: "21:00", Days: allDays, Down: "20M"},
	}
	r, ok := Active(rules, at(time.Wednesday, 12, 0))
	if !ok || r.Down != "5M" {
		t.Fatalf("first rule must win, got %+v", r)
	}
}

func TestSegmentsFullDayCoverage(t *testing.T) {
	rules := []config.Rule{
		{Start: "09:00", End: "18:00", Days: allDays, Down: "5M", Label: "work"},
		{Start: "06:00", End: "09:00", Days: allDays, Down: "20M", Label: "morning"},
		{Start: "21:00", End: "06:00", Days: allDays, Down: "0", Label: "night"},
	}
	segs := Segments(rules, time.Wednesday)
	if segs[0].From != 0 || segs[len(segs)-1].To != 1440 {
		t.Fatalf("segments must cover the day: %+v", segs)
	}
	for i := 1; i < len(segs); i++ {
		if segs[i].From != segs[i-1].To {
			t.Fatalf("gap between segments %d and %d", i-1, i)
		}
	}
	// expected: night(00-06) morning(06-09) work(09-18) gap(18-21) night(21-24)
	if len(segs) != 5 {
		t.Fatalf("want 5 segments, got %d: %+v", len(segs), segs)
	}
	if segs[3].Label != "" || segs[3].Down != "0" {
		t.Fatalf("18-21 must be an unlimited gap: %+v", segs[3])
	}
}
