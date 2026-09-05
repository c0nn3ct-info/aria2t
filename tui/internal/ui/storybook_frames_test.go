package ui

// The Storybook frame generator, and the gate that keeps its output honest.
//
// `tui-storybook/` renders the TUI from committed ANSI frames rather than from
// a second, hand-written implementation of the same screens: every frame here
// is one real `App.View()` of one real state, so a story cannot drift from the
// product without this test failing first.
//
// Each `storybook_specs_*_test.go` registers its own frames from an `init()`;
// this file owns the world they start in, the render, and the golden compare.
//
//	go test ./internal/ui/ -run TestStorybookFrames                  # the gate
//	ARIA2T_FRAMES_UPDATE=1 go test ./internal/ui/ -run TestStorybookFrames
//
// The second form rewrites `tui-storybook/frames/*.json`; the diff it produces
// is the review surface for a rendering change.
//
// Test-only, so none of it enters the coverage denominator — the product code
// it drives is measured by the suites that already drive it. The one seam it
// needed in product code is `schedNow` (scheduler.go): today's strip is read
// off the wall clock, so a real clock draws a different frame every run.

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"aria2t/internal/config"
	"aria2t/internal/rpc"
)

// framesDir is `tui-storybook/frames` from this package. It is deliberately
// outside the Go module: `tui/` is mirrored to the public repository on its own
// and a Storybook payload has no business travelling with it. The test skips
// when the directory is absent, which is exactly the mirrored case.
const framesDir = "../../../tui-storybook/frames"

// frameSpec is one story: a named state of the TUI, rendered at a fixed size.
type frameSpec struct {
	// ID is the file stem and the id a story asks for — kebab-case, unique.
	ID string
	// Title is the Storybook sidebar path, e.g. "Screens/List".
	Title string
	// Name is the story name under that title, e.g. "Active".
	Name string
	// Doc is the one-paragraph description shown under the story. It says what
	// the state is and what to look at, not how it was built.
	Doc string
	// Cols/Rows are the terminal the frame is rendered at. Screens that pin a
	// key-bar look different at different heights, so every spec states both.
	Cols, Rows int
	// Build takes the shared world to the state this frame is of. Prefer real
	// key presses (`press`) over reaching into a child model: a frame reached
	// the way a user reaches it cannot show a state the app cannot.
	Build func(t *testing.T, a *App, f *fakeAPI)
}

// frameFile is the on-disk shape one spec renders to.
type frameFile struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Name  string `json:"name"`
	Doc   string `json:"doc"`
	Cols  int    `json:"cols"`
	Rows  int    `json:"rows"`
	// Dark and Light are the same state under both palettes, ANSI included
	// (24-bit SGR — the generator pins the profile, so the bytes carry the
	// palette's own hex values rather than whatever the runner's terminal is).
	Dark  string `json:"dark"`
	Light string `json:"light"`
}

var frameSpecs []frameSpec

// registerFrames adds specs from a spec file's init(). One file per group keeps
// the groups independent; the order they end up in does not matter, since every
// frame is written to its own file and the test sorts by id.
func registerFrames(specs ...frameSpec) { frameSpecs = append(frameSpecs, specs...) }

// demoClock is the wall clock every frame is drawn against: a Tuesday
// afternoon, inside the demo scheduler's "night" rule being off and its
// "workday" rule being on, so the scheduler strip has something to show.
var demoClock = time.Date(2026, 3, 17, 14, 25, 0, 0, time.UTC)

// demoConfig is the configuration the frames are rendered against: two servers
// so the switcher has a list, scheduler rules so its screen is not empty, and
// stored caps so Settings shows real values.
func demoConfig() config.Config {
	cfg := config.Default()
	cfg.Servers = []config.Server{
		{Name: "built-in", Host: "localhost", Protocol: "ws", Managed: true},
		{Name: "seedbox", Host: "box.example.net", Port: 6800, Protocol: "https", Secret: "s3cr3t"},
	}
	cfg.Dir = "~/Downloads"
	cfg.Split = 16
	cfg.GlobalDown = "5M"
	cfg.GlobalUp = "1M"
	cfg.SeedRatio = "1.5"
	cfg.SeedTime = "60"
	cfg.SchedulerEnabled = true
	cfg.Rules = []config.Rule{
		{Label: "night", Start: "00:00", End: "07:00", Days: [7]bool{true, true, true, true, true, true, true}, Down: "0", Up: "0"},
		{Label: "workday", Start: "09:00", End: "18:00", Days: [7]bool{false, true, true, true, true, true, false}, Down: "2M", Up: "512K"},
	}
	return cfg
}

// demoSnapshot is the download world every frame starts from — one of each kind
// the list has to tell apart: a plain HTTP download, a torrent still
// downloading, a torrent that finished and is seeding, a paused one, a queue,
// a completed download and a failed one. One name is Cyrillic on purpose: a
// column that only lines up over ASCII lines up by accident.
func demoSnapshot() snapshot {
	debian := rpc.Status{
		GID: "2089b05ecca3d829", Status: "active",
		TotalLength: "5804064768", CompletedLength: "2401239040",
		DownloadSpeed: "13002342", Connections: "16",
		Dir: "/home/ivan/Downloads",
		Files: []rpc.File{{
			Index: "1", Path: "/home/ivan/Downloads/debian-13.1.0-amd64-DVD-1.iso",
			Length: "5804064768", CompletedLength: "2401239040", Selected: "true",
			URIs: []rpc.URI{{URI: "https://cdimage.debian.org/debian-cd/13.1.0/amd64/iso-dvd/debian-13.1.0-amd64-DVD-1.iso", Status: "used"}},
		}},
	}
	blender := rpc.Status{
		GID: "7c9e1f2a4b6d8e03", Status: "active",
		TotalLength: "4294967296", CompletedLength: "1288490188",
		UploadLength: "204010946", DownloadSpeed: "8281948", UploadSpeed: "1048576",
		InfoHash:   "88f2a6c1d3e4b57908fa1c2d3e4b5f60718293a4",
		NumSeeders: "12", Connections: "34", NumPieces: "2048", PieceLength: "2097152",
		Bitfield: strings.Repeat("ff", 96) + strings.Repeat("00", 160),
		Dir:      "/home/ivan/Downloads",
		Files: []rpc.File{
			{Index: "1", Path: "/home/ivan/Downloads/Blender Open Movies/Sintel.2010.1080p.mkv",
				Length: "2147483648", CompletedLength: "1073741824", Selected: "true"},
			{Index: "2", Path: "/home/ivan/Downloads/Blender Open Movies/Tears of Steel.2012.1080p.mkv",
				Length: "1610612736", CompletedLength: "214748364", Selected: "true"},
			{Index: "3", Path: "/home/ivan/Downloads/Blender Open Movies/extras/making-of.mkv",
				Length: "536870912", CompletedLength: "0", Selected: "false"},
		},
		BitTorrent: btInfo("Blender Open Movies", "multi", "Creative Commons Attribution 3.0"),
	}
	seeding := rpc.Status{
		GID: "d41d8cd98f00b204", Status: "active", Seeder: "true",
		TotalLength: "1073741824", CompletedLength: "1073741824",
		UploadLength: "1610612736", UploadSpeed: "2306867",
		InfoHash:   "a5b6c7d8e9f0011223344556677889900aabbccd",
		NumSeeders: "3", Connections: "9", NumPieces: "512", PieceLength: "2097152",
		Bitfield: strings.Repeat("ff", 64),
		Dir:      "/home/ivan/Downloads",
		Files: []rpc.File{{Index: "1", Path: "/home/ivan/Downloads/archlinux-2026.03.01-x86_64.iso",
			Length: "1073741824", CompletedLength: "1073741824", Selected: "true"}},
		BitTorrent: btInfo("archlinux-2026.03.01-x86_64.iso", "single", ""),
	}
	paused := rpc.Status{
		GID: "3f5a7c9e1b2d4068", Status: "paused",
		TotalLength: "6227702579", CompletedLength: "1806083748", Connections: "0",
		Dir: "/home/ivan/Downloads",
		Files: []rpc.File{{Index: "1", Path: "/home/ivan/Downloads/ubuntu-26.04-desktop-amd64.iso",
			Length: "6227702579", CompletedLength: "1806083748", Selected: "true",
			URIs: []rpc.URI{{URI: "https://releases.ubuntu.com/26.04/ubuntu-26.04-desktop-amd64.iso", Status: "used"}}}},
	}
	waiting := []rpc.Status{
		{GID: "5e7f9a1c3b5d7f91", Status: "waiting", TotalLength: "52428800",
			Files: []rpc.File{{Index: "1", Path: "/home/ivan/Downloads/node-v24.3.0-linux-x64.tar.xz", Length: "52428800"}}},
		{GID: "6a8c0e2f4a6c8e02", Status: "waiting", TotalLength: "213909504",
			Files: []rpc.File{{Index: "1", Path: "/home/ivan/Downloads/alpine-3.21.0-x86_64.iso", Length: "213909504"}}},
		{GID: "9b1d3f5a7c9e1b23", Status: "waiting", TotalLength: "1932735283",
			Files: []rpc.File{{Index: "1", Path: "/home/ivan/Downloads/Каталог образов 2026.zip", Length: "1932735283"}}},
	}
	stopped := []rpc.Status{
		{GID: "1c3e5a7b9d1f3a57", Status: "complete",
			TotalLength: "1073741824", CompletedLength: "1073741824",
			Dir:   "/home/ivan/Downloads",
			Files: []rpc.File{{Index: "1", Path: "/home/ivan/Downloads/nixos-25.11-x86_64-linux.iso", Length: "1073741824", CompletedLength: "1073741824", Selected: "true"}}},
		{GID: "8e0a2c4e6a8c0e24", Status: "error", ErrorCode: "3",
			ErrorMessage: "Resource was not found",
			TotalLength:  "0", CompletedLength: "0",
			Dir: "/home/ivan/Downloads",
			Files: []rpc.File{{Index: "1", Path: "/home/ivan/Downloads/openwrt-24.10.2-x86-64.img.gz",
				URIs: []rpc.URI{{URI: "https://downloads.example.org/openwrt-24.10.2-x86-64.img.gz", Status: "used"}}}},
		},
	}
	return snapshot{
		Active:  []rpc.Status{debian, blender, seeding, paused},
		Waiting: waiting,
		Stopped: stopped,
		Stat: rpc.GlobalStat{
			DownloadSpeed: "21284290", UploadSpeed: "3355443",
			NumActive: "4", NumWaiting: "3", NumStopped: "2", NumStoppedTotal: "2",
		},
		Taken: demoClock,
	}
}

// btInfo fills the bittorrent section, whose Info is an anonymous struct and so
// cannot be written as a literal at the call site.
func btInfo(name, mode, comment string) *rpc.BTInfo {
	bt := &rpc.BTInfo{
		Mode: mode, Comment: comment,
		AnnounceList: [][]string{{"udp://tracker.opentrackr.org:1337/announce"}, {"udp://tracker.example.net:6969/announce"}},
	}
	bt.Info.Name = name
	return bt
}

// demoPeers is the peer table the torrent detail screen shows.
func demoPeers() []rpc.Peer {
	return []rpc.Peer{
		// aria2 percent-encodes a peer id on the wire and `clientName` decodes
		// it, so an un-encoded fixture reads as "-q" rather than "qB".
		{PeerID: "%2DqB5010%2D", IP: "203.0.113.42", Port: "51413", Seeder: "true",
			DownloadSpeed: "3145728", UploadSpeed: "262144", AmChoking: "false", PeerChoking: "false",
			Bitfield: strings.Repeat("ff", 64)},
		{PeerID: "%2DTR4060%2D", IP: "198.51.100.7", Port: "6881", Seeder: "false",
			DownloadSpeed: "1572864", UploadSpeed: "524288", AmChoking: "false", PeerChoking: "false",
			Bitfield: strings.Repeat("f0", 64)},
		{PeerID: "%2Dlt0D60%2D", IP: "2001:db8::b0b", Port: "6889", Seeder: "false",
			DownloadSpeed: "262144", UploadSpeed: "0", AmChoking: "true", PeerChoking: "true",
			Bitfield: strings.Repeat("81", 64)},
	}
}

// demoServers is the mirror table the HTTP detail screen shows.
func demoServers() []rpc.ServerStat {
	return []rpc.ServerStat{{Index: "1", Servers: []rpc.ServerInfo{
		{URI: "https://cdimage.debian.org/debian-cd/13.1.0/amd64/iso-dvd/debian-13.1.0-amd64-DVD-1.iso",
			CurrentURI:    "https://ftp.nl.debian.org/debian-cd/13.1.0/amd64/iso-dvd/debian-13.1.0-amd64-DVD-1.iso",
			DownloadSpeed: "9437184"},
		{URI: "https://cdimage.debian.org/debian-cd/13.1.0/amd64/iso-dvd/debian-13.1.0-amd64-DVD-1.iso",
			CurrentURI:    "https://ftp.de.debian.org/debian-cd/13.1.0/amd64/iso-dvd/debian-13.1.0-amd64-DVD-1.iso",
			DownloadSpeed: "3565158"},
	}}}
}

// frameApp is the world every frame starts from: connected to the managed
// daemon, one poll in, speed history already accumulated so the stats
// sparklines have a shape.
func frameApp(t *testing.T, theme string, cols, rows int) (*App, *fakeAPI) {
	t.Helper()
	cfg := demoConfig()
	cfg.Theme = theme
	a := NewApp(cfg, filepath.Join(t.TempDir(), "config.json"))
	fake := newFakeAPI()
	fake.waiting = demoSnapshot().Waiting
	fake.servers = demoServers()
	fake.peers = demoPeers()
	a.client = fake
	a.connected = true
	a.version = "1.37.0"
	a.endpoint = "localhost:6800"
	a.snap = demoSnapshot()
	a.stoppedSeeded = true
	a.width, a.height = cols, rows
	// A plausible half hour of history: a rising download curve with a dip, and
	// a flatter upload one. Fixed, so the sparkline is the same on every run.
	for i := 0; i < 60; i++ {
		down := int64(6_000_000 + i*260_000)
		if i > 38 && i < 46 {
			down = int64(2_400_000 + i*40_000)
		}
		a.downHist.Push(down)
		a.upHist.Push(int64(900_000 + (i%7)*180_000))
	}
	return a, fake
}

// keyMsg turns a key name into the message bubbletea would deliver. Only the
// names the specs actually use; an unknown one fails the test rather than
// silently rendering the state before the key.
func keyMsg(t *testing.T, name string) tea.KeyMsg {
	t.Helper()
	switch name {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+o":
		return tea.KeyMsg{Type: tea.KeyCtrlO}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	case "ctrl+t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	}
	if strings.HasPrefix(name, "type:") {
		return key_(strings.TrimPrefix(name, "type:"))
	}
	if len([]rune(name)) == 1 {
		return key_(name)
	}
	t.Fatalf("frames: unknown key %q", name)
	return tea.KeyMsg{}
}

// press drives the app the way a user does — through Update, with each command
// drained — so a frame can only show a state the key path can actually reach.
// "type:foo" sends a whole string as runes (one message, which is what a paste
// looks like to bubbletea and what a text input needs to receive intact).
func press(t *testing.T, a *App, names ...string) {
	t.Helper()
	for _, name := range names {
		_, cmd := a.Update(keyMsg(t, name))
		drain(t, a, cmd)
	}
}

// renderFrame builds the world, runs the spec's Build and returns the frame.
// The color profile is pinned by the caller, once, around the whole run.
func renderFrame(t *testing.T, spec frameSpec, theme string) string {
	t.Helper()
	lipgloss.SetHasDarkBackground(theme == "dark")
	a, fake := frameApp(t, theme, spec.Cols, spec.Rows)
	if spec.Build != nil {
		spec.Build(t, a, fake)
	}
	return a.View()
}

func TestStorybookFrames(t *testing.T) {
	update := os.Getenv("ARIA2T_FRAMES_UPDATE") == "1"
	if _, err := os.Stat(framesDir); errors.Is(err, fs.ErrNotExist) && !update {
		// The public repository carries `tui/` on its own; `tui-storybook/` is
		// monorepo-only, so there is nothing to check against there.
		t.Skip("no " + framesDir + " — tui-storybook/ is monorepo-only")
	}

	if len(frameSpecs) == 0 {
		t.Fatal("no frames registered: every storybook_specs_*_test.go calls registerFrames from init()")
	}
	seen := map[string]string{}
	for _, spec := range frameSpecs {
		if prev, dup := seen[spec.ID]; dup {
			t.Fatalf("duplicate frame id %q (%s and %s)", spec.ID, prev, spec.Title+"/"+spec.Name)
		}
		seen[spec.ID] = spec.Title + "/" + spec.Name
		if spec.Title == "" || spec.Name == "" || spec.Doc == "" {
			t.Fatalf("frame %q needs a title, a name and a doc", spec.ID)
		}
		if spec.Cols <= 0 || spec.Rows <= 0 {
			t.Fatalf("frame %q needs a terminal size", spec.ID)
		}
	}

	// The frames carry colour, so the renderer has to be told it can emit it:
	// under `go test` stdout is not a terminal and lipgloss degrades to plain
	// text. Restored afterwards — the profile is global, and the rest of the
	// package asserts on frames it expects to be uncoloured.
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	// And the background lightness the bubbles widgets resolve their own
	// adaptive colours against — the text inputs' cursor and placeholder come
	// from the library, not from `theme.go`. Without pinning it, lipgloss asks
	// the terminal, and under `go test` there is none: the answer would then
	// depend on the machine the frames were generated on, which is the one
	// thing a golden file may not do. `renderFrame` sets it per palette.
	prevDark := lipgloss.HasDarkBackground()
	defer lipgloss.SetHasDarkBackground(prevDark)

	// Same for the clock the scheduler strip is drawn against.
	prevNow := schedNow
	schedNow = func() time.Time { return demoClock }
	defer func() { schedNow = prevNow }()

	if update {
		if err := os.MkdirAll(framesDir, 0o755); err != nil {
			t.Fatalf("frames: mkdir: %v", err)
		}
	}

	written := map[string]bool{}
	for _, spec := range frameSpecs {
		t.Run(spec.ID, func(t *testing.T) {
			got := frameFile{
				ID: spec.ID, Title: spec.Title, Name: spec.Name, Doc: spec.Doc,
				Cols: spec.Cols, Rows: spec.Rows,
				Dark:  renderFrame(t, spec, "dark"),
				Light: renderFrame(t, spec, "light"),
			}
			// A frame is a full terminal: the app pins every screen to the
			// terminal height, so a short render means the spec left the app in
			// a state no user could see (usually a size below `tooSmall`).
			for _, f := range []struct {
				theme, body string
			}{{"dark", got.Dark}, {"light", got.Light}} {
				if lines := strings.Count(f.body, "\n") + 1; lines != spec.Rows {
					t.Fatalf("%s: %s frame is %d lines, terminal is %d", spec.ID, f.theme, lines, spec.Rows)
				}
				if !strings.Contains(f.body, "\x1b[") {
					t.Fatalf("%s: %s frame carries no colour", spec.ID, f.theme)
				}
			}

			blob, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("frames: marshal %s: %v", spec.ID, err)
			}
			blob = append(blob, '\n')
			path := filepath.Join(framesDir, spec.ID+".json")
			written[spec.ID+".json"] = true

			if update {
				if err := os.WriteFile(path, blob, 0o644); err != nil {
					t.Fatalf("frames: write %s: %v", path, err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("frames: %v\nrun: ARIA2T_FRAMES_UPDATE=1 go test ./internal/ui/ -run TestStorybookFrames", err)
			}
			if string(want) != string(blob) {
				t.Errorf("frames: %s is stale — the TUI renders this state differently now.\n"+
					"Regenerate and review the diff:\n"+
					"  ARIA2T_FRAMES_UPDATE=1 go test ./internal/ui/ -run TestStorybookFrames", spec.ID)
			}
		})
	}

	// A frame file nobody produces is a story showing a state that no longer
	// exists — the failure mode a golden test cannot see by comparing forwards.
	entries, err := os.ReadDir(framesDir)
	if err != nil {
		t.Fatalf("frames: read dir: %v", err)
	}
	var orphans []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" || written[e.Name()] {
			continue
		}
		orphans = append(orphans, e.Name())
		if update {
			if err := os.Remove(filepath.Join(framesDir, e.Name())); err != nil {
				t.Fatalf("frames: remove orphan %s: %v", e.Name(), err)
			}
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 && !update {
		t.Errorf("frames: %v have no spec — delete them, or regenerate with ARIA2T_FRAMES_UPDATE=1", orphans)
	}
}

// paletteFile is the design tokens `tui-storybook` paints its terminal chrome
// and its Foundations page with. Generated from `theme.go` rather than copied
// into the Storybook, for the same reason the frames are generated: a palette
// restated by hand is a palette that drifts.
const paletteFile = "../../../tui-storybook/palette.json"

// paletteRoles is one palette as `{role: "#rrggbb"}`. The roles are listed
// rather than reflected so the file states what each colour is *for* — that
// text is the whole point of the Foundations page — and so a role added to
// `Palette` fails this test until it is described here.
func paletteRoles(p Palette) []map[string]string {
	roles := []struct{ name, role, use string }{
		{"Accent", string(p.Accent), "brand blue: the wordmark, selected markers, every key glyph"},
		{"Bg", string(p.Bg), "input and backdrop background; the terminal ground the frames assume"},
		{"Surface", string(p.Surface), "modal background"},
		{"Fg", string(p.Fg), "primary text"},
		{"FgBright", string(p.FgBright), "emphasised text: titles, the selected row"},
		{"FgDim", string(p.FgDim), "secondary text, column headers, hint labels"},
		{"FgFaint", string(p.FgFaint), "empty progress track, separators, the dimmed backdrop"},
		{"Border", string(p.Border), "panel borders"},
		{"BorderDim", string(p.BorderDim), "inner rules"},
		{"Sel", string(p.Sel), "selected row background"},
		{"Green", string(p.Green), "active and done, the safe button"},
		{"Red", string(p.Red), "error, the consequential button"},
		{"Yellow", string(p.Yellow), "paused and waiting"},
		{"Cyan", string(p.Cyan), "download speed"},
		{"Magenta", string(p.Magenta), "upload speed, seeding, reorder mode"},
	}
	out := make([]map[string]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, map[string]string{"name": r.name, "hex": r.role, "use": r.use})
	}
	return out
}

func TestStorybookPalette(t *testing.T) {
	update := os.Getenv("ARIA2T_FRAMES_UPDATE") == "1"
	if _, err := os.Stat(filepath.Dir(paletteFile)); errors.Is(err, fs.ErrNotExist) && !update {
		t.Skip("no tui-storybook/ — it is monorepo-only")
	}
	payload := map[string]any{
		"dark":  map[string]any{"name": TokyoNight.Name, "title": "Tokyo Night", "roles": paletteRoles(TokyoNight)},
		"light": map[string]any{"name": TokyoNightDay.Name, "title": "Tokyo Night Day", "roles": paletteRoles(TokyoNightDay)},
	}
	blob, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("palette: marshal: %v", err)
	}
	blob = append(blob, '\n')
	if update {
		if err := os.MkdirAll(filepath.Dir(paletteFile), 0o755); err != nil {
			t.Fatalf("palette: mkdir: %v", err)
		}
		if err := os.WriteFile(paletteFile, blob, 0o644); err != nil {
			t.Fatalf("palette: write: %v", err)
		}
		return
	}
	want, err := os.ReadFile(paletteFile)
	if err != nil {
		t.Fatalf("palette: %v\nrun: ARIA2T_FRAMES_UPDATE=1 go test ./internal/ui/ -run TestStorybook", err)
	}
	if string(want) != string(blob) {
		t.Error("palette: tui-storybook/palette.json is stale — regenerate with " +
			"ARIA2T_FRAMES_UPDATE=1 go test ./internal/ui/ -run TestStorybook")
	}
}
