package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/rpc"
)

// ring is a fixed-size ring buffer of speed samples.
type ring struct {
	buf  []int64
	head int // next write position
	n    int // stored count
}

func newRing(size int) *ring { return &ring{buf: make([]int64, size)} }

func (r *ring) Push(v int64) {
	r.buf[r.head] = v
	r.head = (r.head + 1) % len(r.buf)
	if r.n < len(r.buf) {
		r.n++
	}
}

// Slice returns samples oldest-first.
func (r *ring) Slice() []int64 {
	out := make([]int64, 0, r.n)
	start := (r.head - r.n + len(r.buf)) % len(r.buf)
	for i := 0; i < r.n; i++ {
		out = append(out, r.buf[(start+i)%len(r.buf)])
	}
	return out
}

func (r *ring) Max() int64 {
	var m int64
	for _, v := range r.Slice() {
		if v > m {
			m = v
		}
	}
	return m
}

// statsModel renders the global stats screen.
type statsModel struct {
	a *App
}

func newStatsModel(a *App) statsModel { return statsModel{a: a} }

func (m statsModel) view() string {
	a := m.a
	st := a.styles
	var b strings.Builder

	a.hits.line("back", 0, a.width)
	b.WriteString(" " + st.Dim.Render("← esc") + st.Faint.Render(" │ ") + st.Title.Render("Global stats") +
		"   " + st.Dim.Render("window: ") + st.Text.Render("60s") + "\n")

	down := a.downHist.Slice()
	up := a.upHist.Slice()
	peak := a.downHist.Max()
	if upPeak := a.upHist.Max(); upPeak > peak {
		peak = upPeak
	}
	width := a.width - 14
	if width < 20 {
		width = 20
	}
	graph := []string{
		st.Cyan.Render("▼ download "+FmtSpeed(a.snap.Stat.DownSpeed())) + "  " +
			st.Magenta.Render("▲ upload "+FmtSpeed(a.snap.Stat.UpSpeed())) + "  " +
			st.Dim.Render("peak "+FmtSpeed(peak)),
		st.Dim.Render(lpad(FmtBytes(peak), 8)) + " " + st.Cyan.Render(Spark(down, peak, width)),
		st.Dim.Render(lpad("0", 8)) + " " + st.Magenta.Render(Spark(up, peak, width)),
		st.Dim.Render(strings.Repeat(" ", 9) + "-60s" + strings.Repeat(" ", max(1, width-14)) + "now"),
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(graph, "\n")) + "\n")

	// Stat tiles.
	finished := 0
	for _, s := range a.snap.Stopped {
		if s.Status == "complete" {
			finished++
		}
	}
	var sessionDown, sessionUp int64
	for _, s := range append(append(append([]rpc.Status{}, a.snap.Active...), a.snap.Waiting...), a.snap.Stopped...) {
		sessionDown += s.Completed()
		sessionUp += s.Uploaded()
	}
	tile := func(label, value string) string {
		return st.Panel.Render(st.Dim.Render(label) + "\n" + st.Title.Render(value))
	}
	tiles := lipgloss.JoinHorizontal(lipgloss.Top,
		tile("SESSION DOWNLOADED", FmtBytes(sessionDown)),
		tile("SESSION UPLOADED", FmtBytes(sessionUp)),
		tile("ACTIVE / WAITING", fmt.Sprintf("%d / %d", a.snap.Stat.Active(), a.snap.Stat.Waiting())),
		tile("FINISHED", fmt.Sprintf("%d", finished)),
	)
	b.WriteString(tiles + "\n")

	// Bandwidth by download.
	var rows []string
	rows = append(rows, st.Dim.Render("BANDWIDTH BY DOWNLOAD"))
	var maxSpeed int64
	for _, s := range a.snap.Active {
		if v := s.DownSpeed(); v > maxSpeed {
			maxSpeed = v
		}
	}
	barW := a.width / 3
	for _, s := range a.snap.Active {
		frac := 0.0
		if maxSpeed > 0 {
			frac = float64(s.DownSpeed()) / float64(maxSpeed)
		}
		blocks := int(frac * float64(barW))
		rows = append(rows,
			st.Text.Render(pad(s.Name(), a.width-barW-20))+
				st.Cyan.Render(pad(strings.Repeat("█", blocks), barW))+
				st.Cyan.Render(lpad(FmtSpeed(s.DownSpeed()), 12)))
	}
	if len(a.snap.Active) == 0 {
		rows = append(rows, st.Dim.Render("no active downloads"))
	}
	b.WriteString(st.Panel.Width(a.width - 2).Render(strings.Join(rows, "\n")))
	b.WriteString("\n")
	b.WriteString(a.hintbar(strings.Count(b.String(), "\n"), []keyHint{{"esc", "esc", "back"}}))
	b.WriteString(a.statusLine())
	return b.String()
}

// mouse routes the back hint (and the header "back" region is handled globally).
func (m statsModel) mouse(id string) (statsModel, tea.Cmd) {
	if kind, arg := splitID(id); kind == "key" && arg == "esc" {
		m.a.screen = screenList
	}
	return m, nil
}
