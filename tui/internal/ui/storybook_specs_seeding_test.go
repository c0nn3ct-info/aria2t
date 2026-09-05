package ui

// Frames of the seeding screen — where a torrent's give-back is bounded.

import (
	"context"
	"testing"
)

// seedingFrameAPI answers the two option reads the seeding screen makes when it
// opens: aria2's per-download seed limits and extra trackers, and the global
// BitTorrent switches. The shared fake answers both with an empty map, which
// would draw the screen with no target, no extra trackers and every switch off.
type seedingFrameAPI struct{ *fakeAPI }

func (seedingFrameAPI) GetOption(context.Context, string) (map[string]string, error) {
	return map[string]string{
		"seed-ratio": "2.0",
		"seed-time":  "60",
		"bt-tracker": "udp://tracker.torrent.eu.org:451/announce,https://tracker.example.org/announce",
	}, nil
}

func (seedingFrameAPI) GetGlobalOption(context.Context) (map[string]string, error) {
	return map[string]string{
		"enable-dht":           "true",
		"enable-peer-exchange": "true",
		"bt-enable-lpd":        "false",
		"bt-require-crypto":    "false",
	}, nil
}

// openSeeding walks the list to the seeding archlinux torrent and opens `t` on
// it, against an API that answers the option reads with realistic values.
func openSeeding(t *testing.T, a *App, f *fakeAPI) {
	t.Helper()
	a.client = seedingFrameAPI{f}
	press(t, a, "j", "j", "t")
}

func init() {
	registerFrames(
		frameSpec{
			ID: "screens-seeding-limits", Title: "Screens/Seeding", Name: "Seed limits",
			Doc: "Where a finished torrent's give-back is bounded: stop at a ratio, or after so many minutes, whichever comes first. " +
				"The bar reads the live ratio against the target — this one has uploaded one and a half times what it took, three quarters of the way to 2.0.",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { openSeeding(t, a, f) },
		},
		frameSpec{
			ID: "screens-seeding-ratio", Title: "Screens/Seeding", Name: "Raising the target",
			Doc: "A new target typed into the ratio field. The bar under it re-reads against the new number as you type, so the cost of a bigger " +
				"target is visible before ^s writes it to the download.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				openSeeding(t, a, f)
				press(t, a, "backspace", "backspace", "backspace", "type:3.0")
			},
		},
		frameSpec{
			ID: "screens-seeding-startup", Title: "Screens/Seeding", Name: "Startup switches",
			Doc: "DHT, peer exchange, local discovery and encryption decide how this torrent finds peers, so the screen shows them — but aria2 fixes " +
				"them when it starts. Selecting one and pressing space says so instead of pretending to flip it.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				openSeeding(t, a, f)
				press(t, a, "tab", "tab") // ratio → time → the DHT switch
				// Not `press`: the explanation is a flash, and flash's own
				// command is the four-second tick that clears it again.
				_, _ = a.Update(keyMsg(t, "space"))
			},
		},
	)
}
