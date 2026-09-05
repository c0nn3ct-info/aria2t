package ui

// Frames of the "Add download" overlay — the four sources aria2 can be handed,
// the guard that catches a paste which is not one of them, and the file browser
// that fills a path in without typing it.

import (
	"os"
	"testing"
)

func init() {
	registerFrames(
		frameSpec{
			ID: "overlays-add-url", Title: "Overlays/Add", Name: "URL",
			Doc: "The tab the overlay opens on, holding a link that is ready to add. " +
				"The box takes one URI per line and extra lines are mirrors of the same file rather than separate downloads; Save to and Connections apply to whatever is added.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "a",
					"type:https://cdimage.debian.org/debian-cd/13.1.0/amd64/iso-dvd/debian-13.1.0-amd64-DVD-1.iso")
			},
		},
		frameSpec{
			ID: "overlays-add-torrent", Title: "Overlays/Add", Name: "Torrent file",
			Doc:  "Adding a local .torrent. The torrent is added paused so its file tree can be pruned before anything downloads, and \"Start immediately\" decides whether it resumes once the picker is answered.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "a", "ctrl+t",
					"type:/home/ivan/Downloads/blender-open-movies.torrent")
			},
		},
		frameSpec{
			ID: "overlays-add-metalink", Title: "Overlays/Add", Name: "Metalink",
			Doc:  "A local .metalink or .meta4. One metalink can describe several downloads at once, so it is added paused too and the tree picker then offers the whole set to keep or drop.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "a", "ctrl+t", "ctrl+t",
					"type:/home/ivan/Downloads/debian-13.1.0-amd64.metalink")
			},
		},
		frameSpec{
			ID: "overlays-add-input", Title: "Overlays/Add", Name: "Input file",
			Doc:  "An aria2 --input-file list, added as a batch. Each entry keeps its own per-download options, merged over the defaults set here, and any line that is not a link is skipped rather than queued.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "a", "ctrl+t", "ctrl+t", "ctrl+t",
					"type:/home/ivan/Downloads/weekly-mirrors.txt")
			},
		},
		frameSpec{
			ID: "overlays-add-invalid", Title: "Overlays/Add", Name: "Not a link",
			Doc:  "What a paste that is not a link gets. A file path, an .aria2 control file or plain text is refused before anything is queued, and the message names the schemes that do work — aria2 would otherwise accept the add and fail the download a second later.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				press(t, a, "a",
					"type:/home/ivan/Downloads/debian-13.1.0-amd64-DVD-1.iso.aria2")
				// Submitted, but not drained: the refusal returns a four-second
				// timer whose message would clear the very error this frame is of.
				_, _ = a.Update(keyMsg(t, "ctrl+d"))
			},
		},
		frameSpec{
			ID: "overlays-add-browse", Title: "Overlays/Add", Name: "File browser",
			Doc:  "The file picker, reached with ^o from the Torrent tab so a path can be chosen instead of typed. Directories come first, and only files the tab accepts are listed — the .iso and the notes sitting beside these torrents are not offered.",
			Cols: 115,
			Rows: 26,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				withReadDir(t, func(string) ([]os.DirEntry, error) {
					return []os.DirEntry{
						fakeDirEntry{"debian-13.1.0-amd64-DVD-1.iso", false},
						fakeDirEntry{"iso", true},
						fakeDirEntry{"blender-open-movies.torrent", false},
						fakeDirEntry{".DS_Store", false},
						fakeDirEntry{"Blender Open Movies", true},
						fakeDirEntry{"notes.txt", false},
						fakeDirEntry{"archlinux-2026.03.01-x86_64.torrent", false},
						fakeDirEntry{"torrents", true},
						fakeDirEntry{"ubuntu-26.04-desktop-amd64.torrent", false},
					}, nil
				})
				press(t, a, "a", "ctrl+t", "tab")
				// Retype "Save to" as an absolute path: the browser opens on it,
				// and an expanded ~ would put this machine's home in the frame.
				for range "~/Downloads" {
					press(t, a, "backspace")
				}
				press(t, a, "type:/home/ivan/Downloads", "ctrl+o", "j", "j", "j", "j")
			},
		},
	)
}
