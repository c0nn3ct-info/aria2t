package ui

// Frames of the dialogs — the modals that ask one question over a dimmed
// backdrop and then get out of the way: the throttle popup, the server
// switcher and its form, the checksum prompt, the destructive confirm,
// keyboard help, the file picker and the command palette.

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
	"aria2t/internal/daemon"
)

// openSwitcherFrame takes the app to the server switcher with both endpoints
// probed. The dial seam is swapped first so no frame ever touches the network,
// and the round-trips are then delivered as the probe's own message, so the
// readout is the same on every run.
func openSwitcherFrame(t *testing.T, a *App) {
	t.Helper()
	// The managed daemon's handle arrives with the connect message, which no
	// key can deliver: replaying it would re-poll the fake server and wipe the
	// demo downloads.
	a.daemon = &daemon.Daemon{Port: 6800, Secret: "demo"}
	a.dial = func(config.Server) (api, string, error) { return nil, "", errors.New("offline") }
	press(t, a, "s")
	_, _ = a.Update(latencyMsg{index: 0, d: 2 * time.Millisecond})
	_, _ = a.Update(latencyMsg{index: 1, d: 38 * time.Millisecond})
}

func init() {
	registerFrames(
		frameSpec{
			ID: "overlays-throttle-limits", Title: "Overlays/Dialogs", Name: "Throttle",
			Doc: "A speed cap for one download, without touching the global limits. " +
				"The chips are the values aria2 takes; the last one accepts a typed figure, and the download's current limit is the chip that comes up selected.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "l")
				// aria2 reports limits as plain bytes; the popup normalises them
				// back to the compact form before matching a chip.
				_, _ = a.Update(gidOptionsMsg{gid: a.throttle.gid, opts: map[string]string{
					"max-download-limit": "5242880",
					"max-upload-limit":   "262144",
				}})
			},
		},
		frameSpec{
			ID: "overlays-servers-list", Title: "Overlays/Dialogs", Name: "Server switcher",
			Doc: "Every aria2 endpoint aria2t knows, each with its measured round-trip, the built-in managed daemon first. " +
				"Enter connects, and the whole list reloads against the new server.",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { openSwitcherFrame(t, a) },
		},
		frameSpec{
			ID: "overlays-servers-edit", Title: "Overlays/Dialogs", Name: "Edit server",
			Doc: "The form behind e: name, host, port and the RPC secret, which is masked because it is a password. " +
				"The protocol chips pick the transport, and saving re-probes the server straight away.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				openSwitcherFrame(t, a)
				press(t, a, "j", "e") // the remote seedbox, not the managed daemon
			},
		},
		frameSpec{
			ID: "overlays-prompt-checksum", Title: "Overlays/Dialogs", Name: "Checksum prompt",
			Doc: "The one-line prompt, here collecting the sha-256 a finished download is supposed to match. " +
				"Storing it is separate from checking it, so a wrong paste costs nothing.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "4", "c",
					"type:9c56cc51b374c3ba189210d5b6d4bf57790d351c96c47c02190ecf1e430635ab")
			},
		},
		frameSpec{
			ID: "overlays-confirm-remove", Title: "Overlays/Dialogs", Name: "Confirm remove",
			Doc: "The guard in front of anything that cannot be undone, naming the download it is about to drop. " +
				"The buttons flip here — green is Cancel, red is the action — so red always marks the consequential choice.",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "d") },
		},
		frameSpec{
			ID: "overlays-help-keys", Title: "Overlays/Dialogs", Name: "Keyboard help",
			Doc: "Every binding on one card, grouped by where it applies — the reference behind ?, reachable from any screen. " +
				"Mouse is a first-class row here: every hint in the app is clickable, and any other key closes the card.",
			Cols:  115,
			Rows:  36,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "?") },
		},
		frameSpec{
			ID: "overlays-files-torrent", Title: "Overlays/Dialogs", Name: "File picker",
			Doc: "Which files of a multi-file torrent to fetch, as a foldable tree with per-folder sizes and progress. " +
				"A folder whose children are only partly chosen reads [~], and the header keeps a running count of what the selection weighs.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				f.status = demoSnapshot().Active[1] // the multi-file torrent
				press(t, a, "j", "enter", "f")
			},
		},
		frameSpec{
			ID: "overlays-commands-palette", Title: "Overlays/Dialogs", Name: "Command palette",
			Doc: "ctrl+p searches the actions by name, for when the key is not on the tip of the tongue. " +
				"Every row carries the shortcut it stands for, so using the palette teaches the keyboard.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				// ctrl+p has no name in the shared key table; the message is the
				// same one bubbletea would deliver for the key.
				_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
				drain(t, a, cmd)
			},
		},
	)
}
