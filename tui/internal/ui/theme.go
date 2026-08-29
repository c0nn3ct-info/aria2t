package ui

import "github.com/charmbracelet/lipgloss"

// Palette holds the design's color roles. Hex values are copied verbatim
// from the design spec (Tokyo Night / Tokyo Night Day).
type Palette struct {
	Name      string
	Accent    lipgloss.Color // brand blue, selected markers, keys
	Bg        lipgloss.Color // input/backdrop background
	Surface   lipgloss.Color // modal background
	Fg        lipgloss.Color // primary text
	FgBright  lipgloss.Color // emphasized text
	FgDim     lipgloss.Color // secondary text, labels
	FgFaint   lipgloss.Color // empty bar track, separators
	Border    lipgloss.Color // panel borders
	BorderDim lipgloss.Color // inner rules
	Sel       lipgloss.Color // selected row background
	Green     lipgloss.Color
	Red       lipgloss.Color
	Yellow    lipgloss.Color
	Cyan      lipgloss.Color // download speed
	Magenta   lipgloss.Color // upload speed, reorder accent
}

var TokyoNight = Palette{
	Name:      "dark",
	Accent:    "#7aa2f7",
	Bg:        "#16161e",
	Surface:   "#1f2335",
	Fg:        "#c0caf5",
	FgBright:  "#e6e9f2",
	FgDim:     "#565f89",
	FgFaint:   "#3b4261",
	Border:    "#3b4261",
	BorderDim: "#24283b",
	Sel:       "#2a2f45",
	Green:     "#9ece6a",
	Red:       "#f7768e",
	Yellow:    "#e0af68",
	Cyan:      "#7dcfee",
	Magenta:   "#bb9af7",
}

var TokyoNightDay = Palette{
	Name:      "light",
	Accent:    "#2e7de9",
	Bg:        "#e1e2e7",
	Surface:   "#e9eaf0",
	Fg:        "#343b58",
	FgBright:  "#1a1f39",
	FgDim:     "#8990b3",
	FgFaint:   "#a8aecb",
	Border:    "#a8aecb",
	BorderDim: "#c4c8da",
	Sel:       "#c4c8da",
	Green:     "#587539",
	Red:       "#f52a65",
	Yellow:    "#8c6c3e",
	Cyan:      "#007197",
	Magenta:   "#9854f1",
}

// PaletteByName maps a config theme name to a palette.
func PaletteByName(name string) Palette {
	if name == "light" {
		return TokyoNightDay
	}
	return TokyoNight
}

// Styles are prebuilt lipgloss styles for one palette.
type Styles struct {
	P Palette

	Brand     lipgloss.Style // "Aria2t" wordmark
	Title     lipgloss.Style // bold bright text
	Text      lipgloss.Style
	Dim       lipgloss.Style
	Faint     lipgloss.Style
	Green     lipgloss.Style
	Red       lipgloss.Style
	Yellow    lipgloss.Style
	Cyan      lipgloss.Style
	Magenta   lipgloss.Style
	Key       lipgloss.Style // keybar key letter
	Badge     lipgloss.Style // filled accent badge (active tab, "active" pill)
	BadgeWarn lipgloss.Style // filled magenta badge (REORDER MODE)
	TabIdle   lipgloss.Style // idle tab/chip text
	// Filled dialog buttons. Unlike Modal/Input these DO carry a Background —
	// safe because each is a single-span short label whose SGR resets before any
	// neighbour, so it cannot band the way multi-span modal content does. Do not
	// "fix" them by removing the background. Green = the safe/expected choice,
	// Red = the consequential one; dialogs assign them per action (a destructive
	// dialog flips them — green Cancel, red proceed).
	BtnGreen lipgloss.Style // green fill
	BtnRed   lipgloss.Style // red fill
	Panel    lipgloss.Style // rounded bordered panel
	RowSel   lipgloss.Style // selected row background
	Input    lipgloss.Style // text input box
	InputHot lipgloss.Style // focused input box
	Modal    lipgloss.Style // overlay window
}

// NewStyles builds the style set for p.
func NewStyles(p Palette) Styles {
	border := lipgloss.RoundedBorder()
	return Styles{
		P:         p,
		Brand:     lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		Title:     lipgloss.NewStyle().Foreground(p.FgBright).Bold(true),
		Text:      lipgloss.NewStyle().Foreground(p.Fg),
		Dim:       lipgloss.NewStyle().Foreground(p.FgDim),
		Faint:     lipgloss.NewStyle().Foreground(p.FgFaint),
		Green:     lipgloss.NewStyle().Foreground(p.Green),
		Red:       lipgloss.NewStyle().Foreground(p.Red),
		Yellow:    lipgloss.NewStyle().Foreground(p.Yellow),
		Cyan:      lipgloss.NewStyle().Foreground(p.Cyan),
		Magenta:   lipgloss.NewStyle().Foreground(p.Magenta),
		Key:       lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		Badge:     lipgloss.NewStyle().Background(p.Accent).Foreground(p.Bg).Bold(true).Padding(0, 1),
		BadgeWarn: lipgloss.NewStyle().Background(p.Magenta).Foreground(p.Bg).Bold(true).Padding(0, 1),
		TabIdle:   lipgloss.NewStyle().Foreground(p.FgDim),
		// Single-span filled buttons — background is safe here (see Styles doc).
		BtnGreen: lipgloss.NewStyle().Background(p.Green).Foreground(p.Bg).Bold(true).Padding(0, 2),
		BtnRed:   lipgloss.NewStyle().Background(p.Red).Foreground(p.Bg).Bold(true).Padding(0, 2),
		Panel:    lipgloss.NewStyle().Border(border).BorderForeground(p.Border).Padding(0, 1),
		RowSel:   lipgloss.NewStyle().Background(p.Sel),
		Input:    lipgloss.NewStyle().Foreground(p.Fg).Border(border).BorderForeground(p.Border).Padding(0, 1),
		InputHot: lipgloss.NewStyle().Foreground(p.Fg).Border(border).BorderForeground(p.Accent).Padding(0, 1),
		// No Background: the modal is transparent so it shows the terminal's own
		// colours ("normal colours") like the rest of the UI, framed by the
		// Accent border over App.composite's dimmed backdrop. An opaque fill
		// reads as too dark (Bg) or a clashing surface (Surface → blue on
		// non-truecolor), and a block bg would also band (drops after resets).
		Modal: lipgloss.NewStyle().Border(border).BorderForeground(p.Accent).Padding(1, 2),
	}
}
