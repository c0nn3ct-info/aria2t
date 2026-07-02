package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/config"
	"aria2t/internal/sched"
)

// schedulerModel edits time-of-day bandwidth rules.
type schedulerModel struct {
	a      *App
	cursor int

	editing bool
	editIdx int // -1 = new rule
	// form: start, end, label, down, up
	form    [5]textinput.Model
	days    [7]bool
	formFoc int
}

func newSchedulerModel(a *App) schedulerModel {
	m := schedulerModel{a: a, editIdx: -1}
	placeholders := []string{"09:00", "18:00", "Working hours", "5M", "256K"}
	for i := range m.form {
		m.form[i] = textinput.New()
		m.form[i].Width = 16
		m.form[i].Placeholder = placeholders[i]
	}
	return m
}

func (m schedulerModel) update(msg tea.KeyMsg) (schedulerModel, tea.Cmd) {
	a := m.a
	if m.editing {
		return m.updateForm(msg)
	}
	switch msg.String() {
	case "esc", "q":
		a.screen = screenList
	case "j", "down":
		if m.cursor < len(a.cfg.Rules)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case " ":
		a.cfg.SchedulerEnabled = !a.cfg.SchedulerEnabled
		a.lastSchedKey = "" // force re-apply on next tick
		a.saveConfig()
		if !a.cfg.SchedulerEnabled {
			return m, a.rpcCmd("scheduler disabled — limits lifted", func(ctx context.Context, c api) error {
				return c.ChangeGlobalOption(ctx, map[string]string{
					"max-overall-download-limit": "0", "max-overall-upload-limit": "0",
				})
			})
		}
	case "+", "a":
		m.editing = true
		m.editIdx = -1
		for i := range m.form {
			m.form[i].SetValue("")
		}
		m.days = [7]bool{true, true, true, true, true, true, true}
		m.formFoc = 0
		return m, m.form[0].Focus()
	case "e":
		if m.cursor < len(a.cfg.Rules) {
			r := a.cfg.Rules[m.cursor]
			m.editing = true
			m.editIdx = m.cursor
			for i, v := range []string{r.Start, r.End, r.Label, r.Down, r.Up} {
				m.form[i].SetValue(v)
			}
			m.days = r.Days
			m.formFoc = 0
			return m, m.form[0].Focus()
		}
	case "-", "d":
		if m.cursor < len(a.cfg.Rules) {
			a.cfg.Rules = append(a.cfg.Rules[:m.cursor], a.cfg.Rules[m.cursor+1:]...)
			if m.cursor >= len(a.cfg.Rules) && m.cursor > 0 {
				m.cursor--
			}
			a.lastSchedKey = ""
			a.saveConfig()
		}
	}
	return m, nil
}

func (m schedulerModel) updateForm(msg tea.KeyMsg) (schedulerModel, tea.Cmd) {
	a := m.a
	key := msg.String()
	switch key {
	case "esc":
		m.editing = false
		m.form[m.formFoc].Blur()
		return m, nil
	case "tab":
		m.form[m.formFoc].Blur()
		m.formFoc = (m.formFoc + 1) % len(m.form)
		return m, m.form[m.formFoc].Focus()
	case "1", "2", "3", "4", "5", "6", "7":
		// 1=Mon … 7=Sun, matching how people say it; index time.Weekday.
		n := int(key[0] - '1') // 0=Mon
		idx := (n + 1) % 7     // time.Weekday: 0=Sun
		m.days[idx] = !m.days[idx]
		return m, nil
	case "enter":
		start, end := strings.TrimSpace(m.form[0].Value()), strings.TrimSpace(m.form[1].Value())
		if !validHM(start) || !validHM(end) {
			return m, a.flash("times must be HH:MM", true)
		}
		down, err := ParseLimit(m.form[3].Value())
		if err != nil {
			return m, a.flash(err.Error(), true)
		}
		up, err := ParseLimit(m.form[4].Value())
		if err != nil {
			return m, a.flash(err.Error(), true)
		}
		rule := config.Rule{
			Start: start, End: end,
			Label: strings.TrimSpace(m.form[2].Value()),
			Down:  down, Up: up, Days: m.days,
		}
		if m.editIdx == -1 {
			a.cfg.Rules = append(a.cfg.Rules, rule)
			m.cursor = len(a.cfg.Rules) - 1
		} else {
			a.cfg.Rules[m.editIdx] = rule
		}
		a.lastSchedKey = ""
		a.saveConfig()
		m.editing = false
		m.form[m.formFoc].Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.form[m.formFoc], cmd = m.form[m.formFoc].Update(msg)
	return m, cmd
}

func validHM(s string) bool {
	var h, mn int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &mn); err != nil {
		return false
	}
	return h >= 0 && h <= 24 && mn >= 0 && mn <= 59
}

func fmtDays(d [7]bool) string {
	all := true
	weekdays := [7]bool{false, true, true, true, true, true, false}
	for i, v := range d {
		if !v {
			all = false
		}
		_ = i
	}
	if all {
		return "every day"
	}
	if d == weekdays {
		return "Mon–Fri"
	}
	names := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	var on []string
	for i, v := range d {
		if v {
			on = append(on, names[i])
		}
	}
	if len(on) == 0 {
		return "never"
	}
	return strings.Join(on, ",")
}

func (m schedulerModel) view() string {
	a := m.a
	st := a.styles
	if m.editing {
		labels := []string{"Window start (HH:MM)", "Window end (HH:MM)", "Label", "Limit ▼", "Limit ▲"}
		var fields []string
		for i, l := range labels {
			box := st.Input
			if m.formFoc == i {
				box = st.InputHot
			}
			fields = append(fields, st.Dim.Render(l)+"\n"+box.Render(m.form[i].View()))
		}
		names := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
		var days []string
		for n, label := range names {
			idx := (n + 1) % 7
			if m.days[idx] {
				days = append(days, st.Green.Render("["+label+"]"))
			} else {
				days = append(days, st.Dim.Render(" "+label+" "))
			}
		}
		body := lipgloss.JoinVertical(lipgloss.Left,
			st.Title.Render("Scheduler rule"),
			"",
			strings.Join(fields, "\n"),
			"",
			st.Dim.Render("Days (1–7 toggle)  ")+strings.Join(days, " "),
			"",
			st.Dim.Render("tab next · ")+st.Green.Render("↵ save")+st.Dim.Render(" · esc cancel"),
		)
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, st.Modal.Render(body))
	}

	var b strings.Builder
	enabled := st.Dim.Render("[ ] enabled")
	if a.cfg.SchedulerEnabled {
		enabled = st.Green.Render("[x] enabled")
	}
	b.WriteString(" " + st.Dim.Render("← esc") + st.Faint.Render(" │ ") + st.Title.Render("Scheduler") + "   " + enabled + "\n")

	// 24h strip for today.
	now := time.Now()
	segs := sched.Segments(a.cfg.Rules, now.Weekday())
	stripW := a.width - 8
	if stripW < 24 {
		stripW = 24
	}
	var strip, axis strings.Builder
	activeLabel := "∞ (no rule)"
	if r, ok := sched.Active(a.cfg.Rules, now); ok {
		activeLabel = FmtLimit(r.Down) + " (" + r.Label + ")"
	}
	for _, seg := range segs {
		w := (seg.To - seg.From) * stripW / 1440
		if w < 1 {
			w = 1
		}
		label := FmtLimit(seg.Down)
		cell := label
		if len(cell) > w {
			cell = ""
		}
		padTotal := w - len([]rune(cell))
		block := strings.Repeat(" ", padTotal/2) + cell + strings.Repeat(" ", padTotal-padTotal/2)
		style := st.Green // unlimited
		if seg.Down != "0" {
			style = st.Yellow
		}
		strip.WriteString(style.Reverse(true).Render(block))
	}
	for h := 0; h <= 24; h += 6 {
		lbl := fmt.Sprintf("%02d", h)
		if h != 0 {
			axis.WriteString(strings.Repeat(" ", stripW/4-2))
		}
		axis.WriteString(lbl)
	}
	stripPanel := []string{
		st.Dim.Render("TODAY · limit applied now: ") + st.Yellow.Render(activeLabel),
		strip.String(),
		st.Dim.Render(axis.String()),
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(stripPanel, "\n")) + "\n")

	// Rules table.
	rows := []string{st.Dim.Render(pad("WINDOW", 16) + pad("DAYS", 12) + pad("RULE", 34) + lpad("LIMIT ▼/▲", 14))}
	for i, r := range a.cfg.Rules {
		marker, style := "  ", st.Text
		if i == m.cursor {
			marker, style = st.Brand.Render("▸ "), st.Title
		}
		limit := FmtLimit(r.Down) + " / " + FmtLimit(r.Up)
		line := marker + style.Render(pad(r.Start+" – "+r.End, 14)) +
			st.Text.Render(pad(fmtDays(r.Days), 12)) +
			st.Text.Render(pad(r.Label, 34)) +
			st.Yellow.Render(lpad(limit, 14))
		if i == m.cursor {
			line = st.RowSel.Render(line)
		}
		rows = append(rows, line)
	}
	if len(a.cfg.Rules) == 0 {
		rows = append(rows, st.Dim.Render("  no rules — press + to add one"))
	}
	b.WriteString(st.Panel.Width(a.width-2).Render(strings.Join(rows, "\n")) + "\n")

	key := func(k, label string) string { return st.Key.Render(k) + " " + st.Dim.Render(label) }
	b.WriteString(" " + strings.Join([]string{
		key("+", "add rule"), key("e", "edit"), key("-", "remove"),
		key("space", "toggle scheduler"), key("esc", "back"),
	}, "  "))
	b.WriteString(a.statusLine())
	return b.String()
}
