package config

import (
	"path/filepath"
	"testing"
)

func TestSaveStripsTransientServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	cfg.Servers = []Server{
		{Name: "cli", Host: "x", Port: 1, Secret: "top-secret", Protocol: "ws", Transient: true},
		{Name: "real", Host: "y", Port: 2, Protocol: "http"},
	}
	cfg.Active = 1
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Servers) != 1 || got.Servers[0].Name != "real" {
		t.Fatalf("transient server must be stripped: %+v", got.Servers)
	}
	if got.Active != 0 {
		t.Fatalf("active must be remapped to the kept server, got %d", got.Active)
	}
}

func TestSaveActiveOnTransientFallsBackToZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := Default()
	cfg.Servers = []Server{
		{Name: "cli", Host: "x", Port: 1, Protocol: "ws", Transient: true},
		{Name: "a", Host: "y", Port: 2, Protocol: "ws"},
		{Name: "b", Host: "z", Port: 3, Protocol: "ws"},
	}
	cfg.Active = 0 // pointing at the transient entry
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, _ := Load(path)
	if len(got.Servers) != 2 || got.Active != 0 || got.Servers[0].Name != "a" {
		t.Fatalf("got %+v active=%d", got.Servers, got.Active)
	}
}

func TestServerURLPathOverride(t *testing.T) {
	s := Server{Host: "h", Port: 1, Protocol: "ws", Path: "/rpc/aria2"}
	if got := s.URL(); got != "ws://h:1/rpc/aria2" {
		t.Fatalf("got %q", got)
	}
	s.Path = "noslash"
	if got := s.URL(); got != "ws://h:1/noslash" {
		t.Fatalf("got %q", got)
	}
}
