package ui

// Frames of the settings screen — the sidebar of sections, the fields inside
// one, and the read-only switches aria2c fixes at startup.

import "testing"

// settingsFrameGlobals is what a running aria2c answers `getGlobalOption`
// with: the option-backed fields are daemon state, not config, so without it
// half of the settings screen would render empty.
func settingsFrameGlobals() map[string]string {
	return map[string]string{
		"max-concurrent-downloads":   "5",
		"max-overall-download-limit": "5242880",
		"max-overall-upload-limit":   "1048576",
		"dir":                        "/home/ivan/Downloads",
		"enable-dht":                 "true",
		"enable-peer-exchange":       "true",
		"bt-enable-lpd":              "false",
		"bt-require-crypto":          "false",
		"seed-ratio":                 "1.5",
		"seed-time":                  "60",
	}
}

// settingsFrameOpen presses `,` from the list and lets the daemon answer with the
// options the screen displays.
func settingsFrameOpen(t *testing.T, a *App) {
	t.Helper()
	press(t, a, ",")
	_, cmd := a.Update(globalOptionsMsg{opts: settingsFrameGlobals()})
	drain(t, a, cmd)
}

func init() {
	registerFrames(
		frameSpec{
			ID: "screens-settings-connection", Title: "Screens/Settings", Name: "Connection",
			Doc: "Settings as it opens: the sections down the left, the selected one's fields to the right. " +
				"The built-in daemon has nothing to connect to — Aria2t chooses its endpoint and secret at launch — so this section explains itself and points at the server switcher instead.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				settingsFrameOpen(t, a)
			},
		},
		frameSpec{
			ID: "screens-settings-limits", Title: "Screens/Settings", Name: "Limits",
			Doc: "The speed and concurrency limits, read live from the running aria2c and edited in place. " +
				"The focused field is outlined, and the moment a value changes the header grows an unsaved-changes marker that stays until ^s pushes it back to the daemon.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				settingsFrameOpen(t, a)
				press(t, a, "j", "tab", "backspace", "8")
			},
		},
		frameSpec{
			ID: "screens-settings-bittorrent", Title: "Screens/Settings", Name: "BitTorrent",
			Doc: "DHT, peer exchange, local peer discovery and encryption are fixed when aria2c starts, so they read as state rather than as switches — the focused one says so, and space on it explains instead of toggling. " +
				"Below them, the seed ratio and seed time every new torrent inherits.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				settingsFrameOpen(t, a)
				press(t, a, "j", "j", "j", "tab")
			},
		},
		frameSpec{
			ID: "screens-settings-interface", Title: "Screens/Settings", Name: "Interface",
			Doc: "The theme lives here and only here — there is no global shortcut for it. " +
				"The checkbox is focused, so it advertises the key that flips it; saving repaints the whole app and stores the choice in the config.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				settingsFrameOpen(t, a)
				press(t, a, "j", "j", "j", "j", "tab")
			},
		},
	)
}
