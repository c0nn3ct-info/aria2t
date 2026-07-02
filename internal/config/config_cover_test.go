package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadUnreadableFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	// A directory is not readable as a file: ReadFile fails with an error
	// that is not fs.ErrNotExist.
	cfg, err := Load(dir)
	if err == nil {
		t.Fatal("want error for unreadable path, got nil")
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].Name != "built-in" {
		t.Fatalf("want Default config on read error, got %+v", cfg)
	}
}

func TestLoadCorruptJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err == nil {
		t.Fatal("want parse error, got nil")
	}
	if !strings.Contains(err.Error(), "config: parse") {
		t.Fatalf("want wrapped parse error, got %v", err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].Name != "built-in" {
		t.Fatalf("want Default config on parse error, got %+v", cfg)
	}
}

func TestSaveMkdirAllFails(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Parent of the target path is an existing regular file.
	if err := Save(filepath.Join(blocker, "config.json"), Default()); err == nil {
		t.Fatal("want MkdirAll error, got nil")
	}
}

func TestSaveWriteFileFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.json")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	// Target is an existing directory: WriteFile must fail.
	if err := Save(target, Default()); err == nil {
		t.Fatal("want WriteFile error, got nil")
	}
}

func TestDefaultPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, ".config", "aria2t", "config.json")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPathNoHomeFallsBack(t *testing.T) {
	t.Setenv("HOME", "")
	if got := DefaultPath(); got != "config.json" {
		t.Fatalf("DefaultPath() = %q, want %q", got, "config.json")
	}
}

func TestActiveServerBranches(t *testing.T) {
	// Empty server list falls back to the built-in default.
	empty := Config{}
	if got := empty.ActiveServer(); got.Name != "built-in" || !got.Managed {
		t.Fatalf("empty servers: got %+v", got)
	}

	// Negative index clamps to the first server.
	cfg := Config{
		Servers: []Server{{Name: "first"}, {Name: "second"}},
		Active:  -1,
	}
	if got := cfg.ActiveServer(); got.Name != "first" {
		t.Fatalf("negative index: got %+v", got)
	}

	// Valid index returns the selected server.
	cfg.Active = 1
	if got := cfg.ActiveServer(); got.Name != "second" {
		t.Fatalf("valid index: got %+v", got)
	}
}

func TestServerURLHTTPS(t *testing.T) {
	s := Server{Host: "sb.example.net", Port: 443, Protocol: "https"}
	if got := s.URL(); got != "https://sb.example.net:443/jsonrpc" {
		t.Fatalf("URL() = %q", got)
	}
}
