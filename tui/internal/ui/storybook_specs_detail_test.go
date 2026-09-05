package ui

// Frames of the detail screen — one download opened from the list with enter.
// The screen shows a different left-hand panel depending on what it is looking
// at: bittorrent peers for a torrent, the connected mirrors for an HTTP
// download.

import (
	"testing"

	"aria2t/internal/rpc"
)

func init() {
	registerFrames(
		frameSpec{
			ID: "screens-detail-torrent", Title: "Screens/Detail", Name: "Torrent",
			Doc:  "A multi-file torrent mid-download. Below the summary the piece bitfield shows which parts have landed, the peer table names each client with its per-peer progress and choke flags, and the file panel marks which files were selected.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				f.status = demoSnapshot().Active[1] // Blender Open Movies
				press(t, a, "j", "enter")
			},
		},
		frameSpec{
			ID: "screens-detail-http", Title: "Screens/Detail", Name: "HTTP download",
			Doc:  "A plain HTTP download. There are no peers and no pieces here, so the left panel lists the mirrors aria2 is actually pulling from and what each one is contributing.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				f.status = demoSnapshot().Active[0] // debian ISO
				press(t, a, "enter")
			},
		},
		frameSpec{
			ID: "screens-detail-seeding", Title: "Screens/Detail", Name: "Seeding",
			Doc:  "A torrent that finished downloading and is now giving back. Progress is full, the summary leads with upload and ratio rather than speed and ETA, and every piece in the bitfield is filled.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				// A seeding torrent's peers are leechers pulling from us, so the
				// DOWN column is genuinely zero — the shared demo peers are the
				// other side of that exchange.
				f.peers = []rpc.Peer{
					{PeerID: "-TR4060-", IP: "203.0.113.9", Port: "51413", Seeder: "false",
						DownloadSpeed: "0", UploadSpeed: "1258291", AmChoking: "false", PeerChoking: "false"},
					{PeerID: "-qB5010-", IP: "198.51.100.31", Port: "6881", Seeder: "false",
						DownloadSpeed: "0", UploadSpeed: "838860", AmChoking: "false", PeerChoking: "true"},
					{PeerID: "-DE13F0-", IP: "2001:db8::7a2", Port: "6889", Seeder: "false",
						DownloadSpeed: "0", UploadSpeed: "209715", AmChoking: "false", PeerChoking: "true"},
				}
				f.status = demoSnapshot().Active[2] // archlinux ISO, seeding
				press(t, a, "j", "j", "enter")
			},
		},
		frameSpec{
			ID: "screens-detail-error", Title: "Screens/Detail", Name: "Failed",
			Doc:  "A download that failed. The aria2 error code is spelled out in plain English under the title, and the panels stay in place but have nothing to report — no mirror is connected and nothing was written.",
			Cols: 115,
			Rows: 24,
			Build: func(t *testing.T, a *App, f *fakeAPI) {
				f.servers = nil // nothing is connected to a download that failed
				f.status = demoSnapshot().Stopped[1]
				press(t, a, "4", "j", "enter")
			},
		},
	)
}
