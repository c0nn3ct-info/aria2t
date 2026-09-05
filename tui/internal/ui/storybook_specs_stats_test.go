package ui

// Frames of the global stats screen — the `g` view: sparklines over the last
// sixty seconds of transfer, session totals, and a bar per active download.

import "testing"

func init() {
	registerFrames(
		frameSpec{
			ID: "screens-stats-overview", Title: "Screens/Stats", Name: "Overview",
			Doc: "Half an hour into a busy session. The cyan download sparkline and the magenta upload one are drawn against a single " +
				"shared peak, so the dip in the download curve and the much smaller upload line stay comparable; underneath, the " +
				"session totals and a bar per active download show where the bandwidth is actually going.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "g")
			},
		},
		frameSpec{
			ID: "screens-stats-empty", Title: "Screens/Stats", Name: "No history yet",
			Doc: "The same screen a moment after connecting, before the first poll has returned. The sparklines have nothing to plot " +
				"and the speed readouts fall back to dashes, the tiles sit at zero, and the bandwidth panel says \"no active " +
				"downloads\" rather than presenting an empty box.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				// Both the samples and the snapshot arrive on the same poll, so
				// the honest "nothing has arrived yet" state clears both. No key
				// expresses it: it is the absence of a message.
				a.downHist, a.upHist = newRing(60), newRing(60)
				a.snap = snapshot{}
				press(t, a, "g")
			},
		},
	)
}
