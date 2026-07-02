package config

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsDefault(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].Host != "localhost" || cfg.Theme != "dark" {
		t.Fatalf("not default: %+v", cfg)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	want := Default()
	want.Servers = append(want.Servers, Server{Name: "seedbox", Host: "sb.example.net", Port: 6800, Secret: "x", Protocol: "wss"})
	want.Active = 1
	want.SchedulerEnabled = true
	want.Rules = []Rule{{Start: "09:00", End: "18:00", Days: [7]bool{false, true, true, true, true, true, false}, Label: "work", Down: "5M", Up: "256K"}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Active != 1 || len(got.Servers) != 2 || got.Servers[1].Name != "seedbox" ||
		!got.SchedulerEnabled || len(got.Rules) != 1 || got.Rules[0].Down != "5M" || !got.Rules[0].Days[1] {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}

func TestServerURL(t *testing.T) {
	cases := []struct {
		s    Server
		want string
	}{
		{Server{Host: "localhost", Port: 6800, Protocol: "ws"}, "ws://localhost:6800/jsonrpc"},
		{Server{Host: "localhost", Port: 6800, Protocol: "http"}, "http://localhost:6800/jsonrpc"},
		{Server{Host: "sb.example.net", Port: 443, Protocol: "wss"}, "wss://sb.example.net:443/jsonrpc"},
		{Server{Host: "x", Port: 1, Protocol: ""}, "ws://x:1/jsonrpc"},
	}
	for _, tc := range cases {
		if got := tc.s.URL(); got != tc.want {
			t.Errorf("URL() = %q, want %q", got, tc.want)
		}
	}
}

func TestActiveServerClampsStaleIndex(t *testing.T) {
	cfg := Default()
	cfg.Active = 5
	if got := cfg.ActiveServer(); got.Name != "local" {
		t.Fatalf("got %+v", got)
	}
}
