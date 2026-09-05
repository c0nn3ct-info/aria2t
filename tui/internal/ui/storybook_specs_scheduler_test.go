package ui

// Frames of the bandwidth scheduler — the day's rules, and the form that edits
// one of them.

import "testing"

func init() {
	registerFrames(
		frameSpec{
			ID: "screens-scheduler-rules", Title: "Screens/Scheduler", Name: "Rules",
			Doc: "Bandwidth rules for the day. The strip is today's 24 hours — green where nothing is capped, " +
				"yellow where a rule throttles — and the line above it names the limit actually in force at this " +
				"moment, which on a Tuesday afternoon is the workday rule's 2M.",
			Cols:  115,
			Rows:  24,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "S") },
		},
		frameSpec{
			ID: "screens-scheduler-edit", Title: "Screens/Scheduler", Name: "Edit rule",
			Doc: "The workday rule opened for editing, over the dimmed strip it belongs to. Its window, label and " +
				"the two limits are loaded into the form, and the day chips show it running Monday to Friday with " +
				"the weekend left alone.",
			Cols:  115,
			Rows:  30,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "S", "j", "e") },
		},
		frameSpec{
			ID: "screens-scheduler-new", Title: "Screens/Scheduler", Name: "New rule",
			Doc: "The same form for a rule that does not exist yet. Every field offers the shape it expects as a " +
				"placeholder — a start and end time, a name, a download and an upload cap — and a fresh rule begins " +
				"life applying to all seven days.",
			Cols:  115,
			Rows:  30,
			Build: func(t *testing.T, a *App, f *fakeAPI) { press(t, a, "S", "+") },
		},
	)
}
