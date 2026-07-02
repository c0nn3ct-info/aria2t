package ui

import (
	"testing"

	"aria2t/internal/config"
)

func TestThrottleChipToOptions(t *testing.T) {
	a := NewApp(config.Default(), t.TempDir()+"/config.json")
	m := newThrottleModel(a)
	m.downSel = 2 // "5M"
	m.upSel = 1   // "256K"
	opts, err := m.values()
	if err != nil {
		t.Fatal(err)
	}
	if opts["max-download-limit"] != "5M" || opts["max-upload-limit"] != "256K" {
		t.Fatalf("opts = %v", opts)
	}
}

func TestThrottleCustomValue(t *testing.T) {
	a := NewApp(config.Default(), t.TempDir()+"/config.json")
	m := newThrottleModel(a)
	m.downSel = len(downPresets)
	m.custom[0].SetValue("8m")
	m.upSel = 0
	opts, err := m.values()
	if err != nil {
		t.Fatal(err)
	}
	if opts["max-download-limit"] != "8M" || opts["max-upload-limit"] != "0" {
		t.Fatalf("opts = %v", opts)
	}
}

func TestThrottleCustomJunkRejected(t *testing.T) {
	a := NewApp(config.Default(), t.TempDir()+"/config.json")
	m := newThrottleModel(a)
	m.downSel = len(downPresets)
	m.custom[0].SetValue("warp9")
	if _, err := m.values(); err == nil {
		t.Fatal("want error")
	}
}

func TestThrottleAbsorbSelectsMatchingChip(t *testing.T) {
	a := NewApp(config.Default(), t.TempDir()+"/config.json")
	m := newThrottleModel(a)
	m.gid = "g1"
	m.absorbOptions(gidOptionsMsg{gid: "g1", opts: map[string]string{
		"max-download-limit": "5M",
		"max-upload-limit":   "3M", // not a preset → custom
	}})
	if m.downSel != 2 {
		t.Fatalf("downSel = %d", m.downSel)
	}
	if m.upSel != len(upPresets) || m.custom[1].Value() != "3M" {
		t.Fatalf("upSel = %d custom=%q", m.upSel, m.custom[1].Value())
	}
}
