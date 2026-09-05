package ui

// Frames of the download list — the screen the app opens on.

import (
	"testing"

	"aria2t/internal/daemon"
)

func init() {
	registerFrames(
		frameSpec{
			ID: "screens-list-all", Title: "Screens/List", Name: "All",
			Doc:   "The default tab. Active, waiting and stopped downloads concatenated, so a download stays on screen as its state changes.",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "1") },
		},
		frameSpec{
			ID: "screens-list-active", Title: "Screens/List", Name: "Active",
			Doc:   "Only what aria2 is working on. STATUS is a fixed-width coloured word, and a torrent that finished downloading reads \"seeding\" rather than \"active\".",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "2") },
		},
		frameSpec{
			ID: "screens-list-waiting", Title: "Screens/List", Name: "Waiting",
			Doc: "The queue: downloads aria2 has accepted but not started yet. Every row is an empty dashed track with no speed and no ETA, and the " +
				"key-bar gains the J/K reorder hint that only this tab offers.",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "3") },
		},
		frameSpec{
			ID: "screens-list-stopped", Title: "Screens/List", Name: "Stopped",
			Doc: "Finished and failed downloads. Progress and speed give way to an INTEGRITY column: a failure states its reason in plain English rather " +
				"than an aria2 error number, and a completed file can be checksummed and re-downloaded from here.",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "4") },
		},
		frameSpec{
			ID: "screens-list-reorder", Title: "Screens/List", Name: "Reorder",
			Doc: "Reordering the queue. One download is grabbed and carried with J/K while the rest shift around it; the grabbed row is magenta and " +
				"remembers where it started, so the move can be dropped with ↵ or abandoned with esc.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "3", "j", "j", "K")
			},
		},
		frameSpec{
			ID: "screens-list-filter", Title: "Screens/List", Name: "Filter",
			Doc: "A name filter, committed with ↵ and shown as a ⌕ badge beside the tabs. It narrows whichever tab is open and stays applied while " +
				"you act on what is left; esc clears it.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "/", "type:iso", "enter")
			},
		},
		frameSpec{
			ID: "screens-list-welcome", Title: "Screens/List", Name: "Welcome",
			Doc: "A first run: connected to the daemon with nothing downloading yet. Instead of an empty table the list names the two ways to start — " +
				"the add form, or one keystroke to take the link already on the clipboard.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				_, cmd := a.Update(pollMsg{}) // an empty poll: connected, nothing to show
				drain(t, a, cmd)
			},
		},
		frameSpec{
			ID: "screens-list-not-connected", Title: "Screens/List", Name: "Not connected",
			Doc: "The daemon could not be reached, and the list says why rather than leaving a red word in the header over an empty table. The usual " +
				"cause is aria2 not being installed; the app keeps retrying every second, so fixing it needs no restart.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				_, cmd := a.Update(pollMsg{})
				drain(t, a, cmd)
				_, cmd = a.Update(connectErrMsg{err: daemon.ErrBinaryNotFound})
				drain(t, a, cmd)
			},
		},
		frameSpec{
			ID: "screens-list-flash", Title: "Screens/List", Name: "Status flash",
			Doc: "The transient status line under the key-bar, here confirming that a download's source URL was copied. Every action answers on this " +
				"line — green for a success, red for a refusal — and it clears itself a few seconds later.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				// The only seam a key cannot express: yanking talks to the real
				// system clipboard, which no frame may depend on.
				orig := clipboardWrite
				clipboardWrite = func(string) error { return nil }
				t.Cleanup(func() { clipboardWrite = orig })
				press(t, a, "y")
			},
		},
	)
}
