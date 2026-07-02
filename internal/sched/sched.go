// Package sched resolves time-of-day bandwidth rules.
package sched

import (
	"strconv"
	"strings"
	"time"

	"aria2t/internal/config"
)

// Active returns the rule covering t, if any. For windows crossing midnight
// (End <= Start) the rule matches when t is after Start on an enabled day,
// or before End on the day following an enabled day. Earlier rules win.
func Active(rules []config.Rule, t time.Time) (config.Rule, bool) {
	minutes := t.Hour()*60 + t.Minute()
	day := int(t.Weekday())
	prevDay := (day + 6) % 7
	for _, r := range rules {
		start, end := parseHM(r.Start), parseHM(r.End)
		if start < 0 || end < 0 {
			continue
		}
		if start < end { // same-day window
			if r.Days[day] && minutes >= start && minutes < end {
				return r, true
			}
		} else { // crosses midnight
			if r.Days[day] && minutes >= start {
				return r, true
			}
			if r.Days[prevDay] && minutes < end {
				return r, true
			}
		}
	}
	return config.Rule{}, false
}

// Segment is a half-open minute range [From,To) of one day with the
// download limit that applies. Gaps between rules become unlimited segments.
type Segment struct {
	From, To int // minutes since midnight, 0..1440
	Down     string
	Up       string
	Label    string
}

// Segments flattens the rules into a full 00:00–24:00 cover for one weekday,
// resolving each minute through the same precedence as Active.
func Segments(rules []config.Rule, day time.Weekday) []Segment {
	// Sample per minute, then merge runs. 1440 iterations is cheap and keeps
	// midnight-crossing and overlap semantics identical to Active.
	base := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC) // a Sunday
	base = base.AddDate(0, 0, int(day))
	var out []Segment
	for m := 0; m < 24*60; m++ {
		r, ok := Active(rules, base.Add(time.Duration(m)*time.Minute))
		down, up, label := "0", "0", ""
		if ok {
			down, up, label = r.Down, r.Up, r.Label
		}
		if len(out) > 0 && out[len(out)-1].Down == down && out[len(out)-1].Up == up && out[len(out)-1].Label == label {
			out[len(out)-1].To = m + 1
			continue
		}
		out = append(out, Segment{From: m, To: m + 1, Down: down, Up: up, Label: label})
	}
	return out
}

// parseHM converts "HH:MM" to minutes since midnight, -1 on malformed input.
func parseHM(s string) int {
	h, m, ok := strings.Cut(s, ":")
	if !ok {
		return -1
	}
	hh, err1 := strconv.Atoi(h)
	mm, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hh < 0 || hh > 24 || mm < 0 || mm > 59 {
		return -1
	}
	return hh*60 + mm
}
