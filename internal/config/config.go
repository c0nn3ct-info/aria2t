// Package config persists aria2t settings as JSON.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Server is one aria2 RPC endpoint. A managed server is a private aria2c
// child process that aria2t spawns on demand (host/port/secret are then
// decided at launch); an unmanaged one is an external daemon the user runs.
type Server struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"` // 0 on managed servers: picked at spawn
	Secret   string `json:"secret"`
	Protocol string `json:"protocol"` // ws | http
	Managed  bool   `json:"managed,omitempty"`
	// Transient marks a session-only entry (e.g. --url): usable like any
	// server but never written back to the config file.
	Transient bool `json:"-"`
}

// URL returns the JSON-RPC endpoint for the chosen protocol.
func (s Server) URL() string {
	scheme := s.Protocol
	if scheme != "http" && scheme != "https" && scheme != "wss" {
		scheme = "ws"
	}
	return fmt.Sprintf("%s://%s:%d/jsonrpc", scheme, s.Host, s.Port)
}

// Rule is one scheduler window. Start/End are "HH:MM"; End may be earlier
// than Start, meaning the window crosses midnight. Days index is
// time.Weekday (0 = Sunday). Down/Up are aria2 limit strings ("5M", "0" = ∞).
type Rule struct {
	Start string  `json:"start"`
	End   string  `json:"end"`
	Days  [7]bool `json:"days"`
	Label string  `json:"label"`
	Down  string  `json:"down"`
	Up    string  `json:"up"`
}

// Config is the persisted application state.
type Config struct {
	Servers          []Server `json:"servers"`
	Active           int      `json:"active"`
	Theme            string   `json:"theme"` // dark | light
	SchedulerEnabled bool     `json:"schedulerEnabled"`
	Rules            []Rule   `json:"rules"`
	Dir              string   `json:"dir"`   // default download dir for the add form
	Split            int      `json:"split"` // connections per server for the add form
}

// Default returns the out-of-the-box configuration: a managed built-in
// daemon, so the app works without the user running aria2c themselves.
func Default() Config {
	return Config{
		Servers: []Server{{Name: "built-in", Host: "localhost", Protocol: "ws", Managed: true}},
		Theme:   "dark",
		Dir:     "~/Downloads",
		Split:   16,
	}
}

// ActiveServer returns the selected server, clamping a stale index.
func (c Config) ActiveServer() Server {
	if len(c.Servers) == 0 {
		return Default().Servers[0]
	}
	if c.Active < 0 || c.Active >= len(c.Servers) {
		return c.Servers[0]
	}
	return c.Servers[c.Active]
}

// DefaultPath is ~/.config/aria2t/config.json.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".config", "aria2t", "config.json")
}

// Load reads path; a missing file yields Default without error.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	cfg := Default()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Default(), fmt.Errorf("config: parse %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes cfg to path, creating parent directories. Transient servers
// (CLI overrides) are stripped so saving never rewrites the user's server
// list with session-only entries or their secrets.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	kept := make([]Server, 0, len(cfg.Servers))
	newActive := 0
	for i, s := range cfg.Servers {
		if s.Transient {
			continue
		}
		if i == cfg.Active {
			newActive = len(kept)
		}
		kept = append(kept, s)
	}
	cfg.Servers = kept
	cfg.Active = newActive
	// Config is a plain struct of strings, ints, bools, and slices of the
	// same; MarshalIndent cannot fail on it.
	raw, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}
