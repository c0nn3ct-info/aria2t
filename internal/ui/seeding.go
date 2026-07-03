package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/rpc"
)

// seedToggle is one global BitTorrent switch shown on the seeding screen.
type seedToggle struct {
	label  string
	optKey string
	on     bool
}

// seedingModel is the per-torrent seeding & trackers screen.
type seedingModel struct {
	a    *App
	gid  string
	name string

	ratio textinput.Model
	stime textinput.Model

	toggles []seedToggle

	trackers      []string
	trackersDirty bool

	focus   int // 0 ratio, 1 time, 2..2+len(toggles)-1 toggles, then trackers
	tCursor int
}

func newSeedingModel(a *App) seedingModel {
	ratio := textinput.New()
	ratio.Width = 8
	stime := textinput.New()
	stime.Width = 8
	return seedingModel{
		a: a, ratio: ratio, stime: stime,
		toggles: []seedToggle{
			{label: "DHT", optKey: "enable-dht"},
			{label: "PEX", optKey: "enable-peer-exchange"},
			{label: "LPD", optKey: "bt-enable-lpd"},
			{label: "Encryption required", optKey: "bt-require-crypto"},
		},
	}
}

func (m seedingModel) trackersStart() int { return 2 + len(m.toggles) }

// loadCmd fetches per-download and global options.
func (m seedingModel) loadCmd() tea.Cmd {
	c := m.a.client
	gid := m.gid
	if c == nil {
		return nil
	}
	perGID := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		opts, err := c.GetOption(ctx, gid)
		return gidOptionsMsg{gid: gid, opts: opts, err: err}
	}
	global := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		opts, err := c.GetGlobalOption(ctx)
		return globalOptionsMsg{opts: opts, err: err}
	}
	return tea.Batch(perGID, global, m.ratio.Focus())
}

func (m *seedingModel) absorbOptions(msg gidOptionsMsg) {
	if msg.gid != m.gid {
		return
	}
	if v := msg.opts["seed-ratio"]; v != "" {
		m.ratio.SetValue(v)
	}
	if v := msg.opts["seed-time"]; v != "" {
		m.stime.SetValue(v)
	}
	if !m.trackersDirty {
		m.trackers = nil
		if v := msg.opts["bt-tracker"]; v != "" {
			m.trackers = strings.Split(v, ",")
		}
		if len(m.trackers) == 0 {
			if s, ok := m.a.statusByGID(m.gid); ok && s.BitTorrent != nil {
				for _, tier := range s.BitTorrent.AnnounceList {
					m.trackers = append(m.trackers, tier...)
				}
			}
		}
	}
}

func (m *seedingModel) absorbGlobal(opts map[string]string) {
	for i := range m.toggles {
		m.toggles[i].on = opts[m.toggles[i].optKey] == "true"
	}
}

func (m seedingModel) update(msg tea.KeyMsg) (seedingModel, tea.Cmd) {
	a := m.a
	key := msg.String()
	inTrackers := m.focus >= m.trackersStart()
	switch key {
	case "esc", "q":
		a.screen = screenList
		return m, nil
	case "tab":
		m.ratio.Blur()
		m.stime.Blur()
		m.focus = (m.focus + 1) % (m.trackersStart() + 1)
		switch m.focus {
		case 0:
			return m, m.ratio.Focus()
		case 1:
			return m, m.stime.Focus()
		}
		return m, nil
	case " ":
		if i := m.focus - 2; i >= 0 && i < len(m.toggles) {
			// aria2 fixes these at startup; changeGlobalOption silently
			// ignores them, so pretending to toggle would lie to the user.
			return m, a.flash(m.toggles[i].label+" is set at aria2 startup — not changeable at runtime", true)
		}
	case "ctrl+s":
		return m.save()
	}
	if inTrackers {
		switch key {
		case "j", "down":
			if m.tCursor < len(m.trackers)-1 {
				m.tCursor++
			}
			return m, nil
		case "k", "up":
			if m.tCursor > 0 {
				m.tCursor--
			}
			return m, nil
		case "-":
			if m.tCursor < len(m.trackers) {
				m.trackers = append(m.trackers[:m.tCursor], m.trackers[m.tCursor+1:]...)
				m.trackersDirty = true
				if m.tCursor >= len(m.trackers) && m.tCursor > 0 {
					m.tCursor--
				}
			}
			return m, nil
		case "+":
			a.prompt = newPromptModel(a, "Add tracker URL", "", func(v string) tea.Cmd {
				if v = strings.TrimSpace(v); v != "" {
					a.seeding.trackers = append(a.seeding.trackers, v)
					a.seeding.trackersDirty = true
				}
				return nil
			})
			a.overlay = overlayPrompt
			return m, a.prompt.focusCmd()
		case "e":
			if m.tCursor < len(m.trackers) {
				idx := m.tCursor
				a.prompt = newPromptModel(a, "Edit tracker URL", m.trackers[idx], func(v string) tea.Cmd {
					if v = strings.TrimSpace(v); v != "" && idx < len(a.seeding.trackers) {
						a.seeding.trackers[idx] = v
						a.seeding.trackersDirty = true
					}
					return nil
				})
				a.overlay = overlayPrompt
				return m, a.prompt.focusCmd()
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	switch m.focus {
	case 0:
		m.ratio, cmd = m.ratio.Update(msg)
	case 1:
		m.stime, cmd = m.stime.Update(msg)
	}
	return m, cmd
}

func (m seedingModel) save() (seedingModel, tea.Cmd) {
	a := m.a
	gid := m.gid
	perGID := map[string]string{}
	if v := strings.TrimSpace(m.ratio.Value()); v != "" {
		perGID["seed-ratio"] = v
	}
	if v := strings.TrimSpace(m.stime.Value()); v != "" {
		perGID["seed-time"] = v
	}
	if m.trackersDirty {
		perGID["bt-tracker"] = strings.Join(m.trackers, ",")
	}
	if len(perGID) == 0 {
		return m, a.flash("nothing to save", false)
	}
	return m, a.rpcCmd("seeding settings saved", func(ctx context.Context, c api) error {
		return c.ChangeOption(ctx, gid, perGID)
	})
}

func (m seedingModel) view() string {
	a := m.a
	st := a.styles
	var b strings.Builder
	b.WriteString(" " + st.Dim.Render("← esc") + st.Faint.Render(" │ ") + st.Title.Render("Seeding") +
		"  " + st.Dim.Render(m.name) + "\n")

	inputView := func(idx int, label string, in textinput.Model) string {
		box := st.Input
		if m.focus == idx {
			box = st.InputHot
		}
		return st.Dim.Render(label) + "\n" + box.Render(in.View())
	}
	var togglesLine []string
	for i, t := range m.toggles {
		box := st.Dim.Render("[ ]")
		if t.on {
			box = st.Green.Render("[x]")
		}
		label := " " + t.label
		if m.focus == 2+i {
			label = st.Title.Render(" " + t.label + " ◂")
		}
		togglesLine = append(togglesLine, box+label)
	}
	ratioNow := 0.0
	ratioText := "0.00"
	target := strings.TrimSpace(m.ratio.Value())
	if s, ok := a.statusByGID(m.gid); ok {
		ratioNow = s.Ratio()
		ratioText = fmt.Sprintf("%.2f", ratioNow)
	}
	frac := 0.0
	if t := parseFloat(target); t > 0 {
		frac = ratioNow / t
	}
	f, e := Bar(frac, 18)
	top := []string{
		inputView(0, "Stop at ratio", m.ratio) + "      " + inputView(1, "Or after seeding (min)", m.stime),
		"",
		st.Dim.Render("ratio now  ") + st.Green.Render(f) + st.Faint.Render(e) + " " +
			st.Title.Render(ratioText+" / "+orDefault(target, "∞")),
		"",
		strings.Join(togglesLine, "   ") + st.Dim.Render("  · startup options, read-only"),
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(top, "\n")) + "\n")

	rows := []string{st.Dim.Render("TRACKERS") + "  " + st.Dim.Render("e edit · + add · - remove")}
	for i, tr := range m.trackers {
		marker := "  "
		style := st.Text
		if m.focus >= m.trackersStart() && i == m.tCursor {
			marker = st.Brand.Render("▸ ")
			style = st.Title
		}
		rows = append(rows, marker+style.Render(tr))
	}
	if len(m.trackers) == 0 {
		rows = append(rows, st.Dim.Render("  no trackers"))
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(rows, "\n")) + "\n")

	key := func(k, label string) string { return st.Key.Render(k) + " " + st.Dim.Render(label) }
	b.WriteString(" " + strings.Join([]string{
		key("tab", "next field"), key("space", "toggle"), key("^s", "save"), key("esc", "back"),
	}, "  "))
	b.WriteString(a.statusLine())
	return b.String()
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// statusByGID finds a download in the current snapshot.
func (a *App) statusByGID(gid string) (rpc.Status, bool) {
	for _, group := range [][]rpc.Status{a.snap.Active, a.snap.Waiting, a.snap.Stopped} {
		for _, s := range group {
			if s.GID == gid {
				return s, true
			}
		}
	}
	return rpc.Status{}, false
}
