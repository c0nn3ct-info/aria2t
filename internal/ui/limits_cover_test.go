package ui

import (
	"testing"

	"aria2t/internal/config"
)

func TestSettingsSavePersistsGlobalCaps(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m.fields[1][1].input.SetValue("5M")   // Global download limit
	m.fields[1][2].input.SetValue("512K") // Global upload limit
	_, cmd := m.save()
	drain(t, a, cmd)
	if a.cfg.GlobalDown != "5M" || a.cfg.GlobalUp != "512K" {
		t.Fatalf("caps not stored in cfg: down=%q up=%q", a.cfg.GlobalDown, a.cfg.GlobalUp)
	}
	got, err := config.Load(a.cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.GlobalDown != "5M" || got.GlobalUp != "512K" {
		t.Fatalf("caps not persisted: %+v", got)
	}
}

func TestApplySavedLimits(t *testing.T) {
	a, fake := testApp(t)

	// No caps stored → nothing to do.
	if a.applySavedLimitsCmd() != nil {
		t.Fatal("no caps → nil")
	}

	// Scheduler owns the limits when enabled → skip even with caps.
	a.cfg.GlobalDown = "5M"
	a.cfg.SchedulerEnabled = true
	if a.applySavedLimitsCmd() != nil {
		t.Fatal("scheduler enabled → nil")
	}

	// Scheduler off, download cap only.
	a.cfg.SchedulerEnabled = false
	a.cfg.GlobalUp = ""
	drain(t, a, a.applySavedLimitsCmd())
	if fake.globalOpts["max-overall-download-limit"] != "5M" {
		t.Fatalf("download cap not applied: %v", fake.globalOpts)
	}
	if _, ok := fake.globalOpts["max-overall-upload-limit"]; ok {
		t.Fatalf("upload cap must be absent: %v", fake.globalOpts)
	}

	// Scheduler off, upload cap only.
	a.cfg.GlobalDown = ""
	a.cfg.GlobalUp = "256K"
	fake.globalOpts = nil
	drain(t, a, a.applySavedLimitsCmd())
	if fake.globalOpts["max-overall-upload-limit"] != "256K" {
		t.Fatalf("upload cap not applied: %v", fake.globalOpts)
	}
	if _, ok := fake.globalOpts["max-overall-download-limit"]; ok {
		t.Fatalf("download cap must be absent: %v", fake.globalOpts)
	}
}
