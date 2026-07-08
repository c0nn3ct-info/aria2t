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

func TestSettingsSavePersistsSeedDefaults(t *testing.T) {
	a, _ := testApp(t)
	m := newSettingsModel(a)
	m.fields[3][4].input.SetValue("1.5") // Default seed ratio
	m.fields[3][5].input.SetValue("120") // Default seed time (min)
	_, cmd := m.save()
	drain(t, a, cmd)
	if a.cfg.SeedRatio != "1.5" || a.cfg.SeedTime != "120" {
		t.Fatalf("seed defaults not stored: ratio=%q time=%q", a.cfg.SeedRatio, a.cfg.SeedTime)
	}
	got, err := config.Load(a.cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.SeedRatio != "1.5" || got.SeedTime != "120" {
		t.Fatalf("seed defaults not persisted: %+v", got)
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

	// Seed defaults apply regardless of the scheduler; caps are skipped when it
	// is enabled.
	a.cfg.GlobalDown = "9M"
	a.cfg.GlobalUp = "9M"
	a.cfg.SeedRatio = "2.0"
	a.cfg.SeedTime = "60"
	a.cfg.SchedulerEnabled = true
	fake.globalOpts = nil
	drain(t, a, a.applySavedLimitsCmd())
	if fake.globalOpts["seed-ratio"] != "2.0" || fake.globalOpts["seed-time"] != "60" {
		t.Fatalf("seed defaults not applied: %v", fake.globalOpts)
	}
	if _, ok := fake.globalOpts["max-overall-download-limit"]; ok {
		t.Fatalf("caps must be skipped when scheduler enabled: %v", fake.globalOpts)
	}
}
