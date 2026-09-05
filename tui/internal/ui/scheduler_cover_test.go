package ui

import (
	"strings"
	"testing"
	"time"

	"aria2t/internal/config"
)

func schedTestRules() []config.Rule {
	all := [7]bool{true, true, true, true, true, true, true}
	return []config.Rule{
		{Start: "09:00", End: "18:00", Days: all, Label: "work", Down: "5M", Up: "256K"},
		{Start: "22:00", End: "06:00", Days: all, Label: "night", Down: "0", Up: "0"},
	}
}

func TestSchedulerUpdateNavigation(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)

	a.screen = screenScheduler
	m, _ = m.update(key("esc"))
	if a.screen != screenList {
		t.Fatal("esc must return to list")
	}
	a.screen = screenScheduler
	m, _ = m.update(key("q"))
	if a.screen != screenList {
		t.Fatal("q must return to list")
	}

	// Guards with no rules.
	m, _ = m.update(key("j"))
	m, _ = m.update(key("k"))
	if m.cursor != 0 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	a.cfg.Rules = schedTestRules()
	m, _ = m.update(key("j"))
	if m.cursor != 1 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	m, _ = m.update(key("j")) // guard at end
	if m.cursor != 1 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	m, _ = m.update(key("k"))
	m, _ = m.update(key("k")) // guard at start
	if m.cursor != 0 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	// Unknown key is inert.
	m, cmd := m.update(key("x"))
	if cmd != nil {
		t.Fatal("unknown key must be inert")
	}
}

func TestSchedulerSpaceToggle(t *testing.T) {
	a, fake := testApp(t)
	rec := &globalRecAPI{fakeAPI: fake}
	a.client = rec
	m := newSchedulerModel(a)

	// Off → on: no RPC.
	a.lastSchedKey = "stale"
	m, cmd := m.update(key(" "))
	if !a.cfg.SchedulerEnabled || a.lastSchedKey != "" || cmd != nil {
		t.Fatal("space must enable without lifting limits")
	}

	// On → off: limits lifted via ChangeGlobalOption 0/0.
	m, cmd = m.update(key(" "))
	if a.cfg.SchedulerEnabled || cmd == nil {
		t.Fatal("space must disable and lift limits")
	}
	msg := cmd()
	if dm, ok := msg.(actionDoneMsg); !ok || dm.err != nil || !strings.Contains(dm.text, "limits lifted") {
		t.Fatalf("msg = %#v", msg)
	}
	if rec.global["max-overall-download-limit"] != "0" || rec.global["max-overall-upload-limit"] != "0" {
		t.Fatalf("global = %v", rec.global)
	}
}

func TestSchedulerAddFormInit(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)
	m.form[0].SetValue("stale")
	m.days = [7]bool{}
	m, cmd := m.update(key("+"))
	if !m.editing || m.editIdx != -1 || cmd == nil || m.formFoc != 0 {
		t.Fatal("+ must open a blank form")
	}
	if m.form[0].Value() != "" {
		t.Fatalf("form = %q", m.form[0].Value())
	}
	for i, d := range m.days {
		if !d {
			t.Fatalf("day %d must default on", i)
		}
	}
	// "a" opens it too.
	m2 := newSchedulerModel(a)
	m2, _ = m2.update(key("a"))
	if !m2.editing {
		t.Fatal("a must open the form")
	}
}

func TestSchedulerEditFormInit(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)
	m, cmd := m.update(key("e")) // no rules → guard
	if m.editing || cmd != nil {
		t.Fatal("e without rules must be inert")
	}

	a.cfg.Rules = schedTestRules()
	m, cmd = m.update(key("e"))
	if !m.editing || m.editIdx != 0 || cmd == nil {
		t.Fatal("e must open the edit form")
	}
	if m.form[0].Value() != "09:00" || m.form[1].Value() != "18:00" ||
		m.form[2].Value() != "work" || m.form[3].Value() != "5M" || m.form[4].Value() != "256K" {
		t.Fatalf("form = %+v", []string{m.form[0].Value(), m.form[1].Value(), m.form[2].Value(), m.form[3].Value(), m.form[4].Value()})
	}
}

func TestSchedulerRemoveGuards(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)
	m, _ = m.update(key("-")) // no rules → guard
	if len(a.cfg.Rules) != 0 {
		t.Fatal("nothing to remove")
	}

	a.cfg.Rules = schedTestRules()
	m.cursor = 1
	m, _ = m.update(key("d")) // remove last → cursor steps back
	if len(a.cfg.Rules) != 1 || m.cursor != 0 {
		t.Fatalf("rules=%d cursor=%d", len(a.cfg.Rules), m.cursor)
	}
	m, _ = m.update(key("-")) // remove the only one, cursor stays 0
	if len(a.cfg.Rules) != 0 || m.cursor != 0 {
		t.Fatalf("rules=%d cursor=%d", len(a.cfg.Rules), m.cursor)
	}
}

func TestSchedulerFormKeys(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)
	m, _ = m.update(key("+"))

	// Typing goes into the focused field.
	m, _ = m.update(key("0"))
	if m.form[0].Value() != "0" {
		t.Fatalf("form[0] = %q", m.form[0].Value())
	}
	// Tab cycles all five fields and wraps.
	for i := 1; i <= 5; i++ {
		m, _ = m.update(key("tab"))
		if m.formFoc != i%5 {
			t.Fatalf("formFoc = %d after %d tabs", m.formFoc, i)
		}
	}
	m, _ = m.update(key("shift+tab"))
	if m.formFoc != 4 {
		t.Fatalf("shift+tab formFoc = %d", m.formFoc)
	}
	// Alt+1..7 toggle days without stealing digits from focused text fields.
	m.days = [7]bool{}
	weekday := map[string]time.Weekday{
		"alt+1": time.Monday, "alt+2": time.Tuesday, "alt+3": time.Wednesday, "alt+4": time.Thursday,
		"alt+5": time.Friday, "alt+6": time.Saturday, "alt+7": time.Sunday,
	}
	for k, wd := range weekday {
		m, _ = m.update(key(k))
		if !m.days[int(wd)] {
			t.Fatalf("key %s must toggle %v", k, wd)
		}
	}
	// esc leaves the form.
	m, _ = m.update(key("esc"))
	if m.editing {
		t.Fatal("esc must leave editing")
	}
}

func TestSchedulerFormEnterValidation(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)
	m, _ = m.update(key("+"))

	m.form[0].SetValue("xx")
	m.form[1].SetValue("18:00")
	m, _ = m.update(key("enter"))
	if !strings.Contains(a.status, "times must be HH:MM") || !m.editing {
		t.Fatalf("status = %q", a.status)
	}

	m.form[0].SetValue("09:00")
	m.form[3].SetValue("junk")
	m, _ = m.update(key("enter"))
	if !strings.Contains(a.status, "bad limit") || !m.editing {
		t.Fatalf("status = %q", a.status)
	}

	m.form[3].SetValue("5M")
	m.form[4].SetValue("junk")
	m, _ = m.update(key("enter"))
	if !strings.Contains(a.status, "bad limit") || !m.editing {
		t.Fatalf("status = %q", a.status)
	}
}

func TestSchedulerFormEnterAddsAndEdits(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)
	m, _ = m.update(key("+"))
	m.form[0].SetValue("09:00")
	m.form[1].SetValue("18:00")
	m.form[2].SetValue(" work ")
	m.form[3].SetValue("5m")
	m.form[4].SetValue("256k")
	a.lastSchedKey = "stale"
	m, cmd := m.update(key("enter"))
	if m.editing || cmd != nil || a.lastSchedKey != "" {
		t.Fatal("valid enter must save")
	}
	if len(a.cfg.Rules) != 1 || m.cursor != 0 {
		t.Fatalf("rules = %+v", a.cfg.Rules)
	}
	r := a.cfg.Rules[0]
	if r.Start != "09:00" || r.End != "18:00" || r.Label != "work" || r.Down != "5M" || r.Up != "256K" {
		t.Fatalf("rule = %+v", r)
	}

	// Edit the rule in place.
	m, _ = m.update(key("e"))
	m.form[2].SetValue("late shift")
	m, _ = m.update(key("enter"))
	if len(a.cfg.Rules) != 1 || a.cfg.Rules[0].Label != "late shift" {
		t.Fatalf("rules = %+v", a.cfg.Rules)
	}
}

func TestValidHM(t *testing.T) {
	cases := map[string]bool{
		"09:00": true, "24:00": false, "0:59": false,
		"xx": false, "25:00": false, "10:99": false, "-1:00": false,
	}
	for in, want := range cases {
		if validHM(in) != want {
			t.Fatalf("validHM(%q) != %v", in, want)
		}
	}
}

func TestFmtDays(t *testing.T) {
	all := [7]bool{true, true, true, true, true, true, true}
	if fmtDays(all) != "every day" {
		t.Fatal("all days")
	}
	weekdays := [7]bool{false, true, true, true, true, true, false}
	if fmtDays(weekdays) != "Mon–Fri" {
		t.Fatal("weekdays")
	}
	if fmtDays([7]bool{true, true}) != "Su,Mo" {
		t.Fatalf("subset = %q", fmtDays([7]bool{true, true}))
	}
	if fmtDays([7]bool{}) != "never" {
		t.Fatal("never")
	}
}

func TestSchedulerViewEditing(t *testing.T) {
	a, _ := testApp(t)
	m := newSchedulerModel(a)
	m, _ = m.update(key("+"))
	m.formFoc = 2
	m.days = [7]bool{true, false, true, false, true, false, true}
	if out := m.view(); !strings.Contains(out, "Scheduler rule") {
		t.Fatalf("out = %q", out)
	}
}

func TestSchedulerViewStripAndRules(t *testing.T) {
	a, _ := testApp(t)
	all := [7]bool{true, true, true, true, true, true, true}

	// An always-active throttling rule: active-label branch and yellow strip.
	a.cfg.SchedulerEnabled = true
	a.cfg.Rules = []config.Rule{
		{Start: "00:00", End: "24:00", Days: all, Label: "cap", Down: "5M", Up: "1M"},
	}
	m := newSchedulerModel(a)
	out := m.view()
	if !strings.Contains(out, "5M (cap)") || !strings.Contains(out, "5M / 1M") {
		t.Fatalf("out = %q", out)
	}

	// Narrow width: stripW clamps to 24, a 30-minute segment collapses to
	// one cell too small for its label, the remainder is green/unlimited.
	b, _ := testApp(t)
	b.width = 30
	b.cfg.Rules = []config.Rule{
		{Start: "00:00", End: "00:30", Days: all, Label: "blip", Down: "5M", Up: "0"},
		{Start: "01:00", End: "02:00", Days: all, Label: "second", Down: "1M", Up: "0"},
	}
	mb := newSchedulerModel(b)
	mb.cursor = 1
	if out := mb.view(); out == "" {
		t.Fatal("narrow view empty")
	}

	// No rules at all.
	c, _ := testApp(t)
	mc := newSchedulerModel(c)
	if out := mc.view(); !strings.Contains(out, "no rules — press + to add one") || !strings.Contains(out, "∞ (no rule)") {
		t.Fatalf("out = %q", out)
	}
}
