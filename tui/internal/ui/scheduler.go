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
				m.form[i].SetValue(safeText(v))
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
	case "shift+tab":
		m.form[m.formFoc].Blur()
		m.formFoc = (m.formFoc + len(m.form) - 1) % len(m.form)
		return m, m.form[m.formFoc].Focus()
	case "alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7":
		// 1=Mon … 7=Sun, matching how people say it; index time.Weekday.
		n := int(key[len(key)-1] - '1') // 0=Mon
		idx := (n + 1) % 7              // time.Weekday: 0=Sun
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
	m.form[m.formFoc], cmd = updateInput(m.form[m.formFoc], msg)
	return m, cmd
}

func validHM(s string) bool {
	if len(s) != 5 || s[2] != ':' || s[0] < '0' || s[0] > '9' || s[1] < '0' || s[1] > '9' || s[3] < '0' || s[3] > '9' || s[4] < '0' || s[4] > '9' {
		return false
	}
	_, err := time.Parse("15:04", s)
	return err == nil
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

// mouse handles clicks: rule rows select; while editing, form fields focus, day
// chips toggle, and the buttons save/cancel.
func (m schedulerModel) mouse(id string) (schedulerModel, tea.Cmd) {
	kind, arg := splitID(id)
	if m.editing {
		switch kind {
		case "field":
			if i := argInt(arg); i >= 0 && i < len(m.form) {
				m.form[m.formFoc].Blur()
				m.formFoc = i
				return m, m.form[i].Focus()
			}
		case "day":
			if i := argInt(arg); i >= 0 && i < 7 {
				m.days[i] = !m.days[i]
			}
		case "btn":
			return m.updateForm(dispatchBtn(arg))
		}
		return m, nil
	}
	switch kind {
	case "key":
		return m.update(keyFromToken(arg))
	case "rule":
		if i := argInt(arg); i >= 0 && i < len(m.a.cfg.Rules) {
			m.cursor = i
		}
	}
	return m, nil
}

// view renders the rules list, and when editing, composites the rule form over
// it (dimmed backdrop, like every overlay) so the form has proper click regions.
func (m schedulerModel) view() string {
	body := m.listBody()
	if !m.editing {
		return body
	}
	m.a.hits.reset() // only the form is interactive while editing
	modal := m.a.modalCard(false).Render(m.formBody())
	offX, offY := m.a.overlayOffset(modal)
	m.registerForm(offX, offY, modal)
	return m.a.composite(body, modal)
}

func (m schedulerModel) formButtons() []button {
	return []button{{"esc", "Cancel", "esc", btnRed}, {"enter", "Save", "↵", btnGreen}}
}

func (m schedulerModel) formBody() string {
	st := m.a.styles
	labels := []string{"Window start (HH:MM)", "Window end (HH:MM)", "Label", "Down limit", "Up limit"}
	field := func(i int) string {
		box := st.Input
		if m.formFoc == i {
			box = st.InputHot
		}
		return st.Dim.Render(labels[i]) + "\n" + box.Render(m.form[i].View())
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
	return lipgloss.JoinVertical(lipgloss.Left,
		st.Title.Render("Scheduler rule"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, field(0), "   ", field(1)),
		"",
		field(2),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, field(3), "   ", field(4)),
		"",
		st.Dim.Render("Days (click or alt+1–7)  ")+strings.Join(days, " "),
		"",
		m.a.buttonRow(m.formButtons()),
	)
}

// registerForm registers click regions for the composited rule form: each input
// row focuses, each day chip toggles, and the buttons save/cancel.
func (m schedulerModel) registerForm(offX, offY int, modal string) {
	st := m.a.styles
	contentX := offX + 3
	pairOffset := lipgloss.Width(st.Dim.Render("Window start (HH:MM)")) + 3
	for _, f := range []struct{ i, x, y int }{
		{0, contentX, offY + 4}, {1, contentX + pairOffset, offY + 4},
		{2, contentX, offY + 9},
		{3, contentX, offY + 14}, {4, contentX + pairOffset, offY + 14},
	} {
		m.a.hits.add(fmt.Sprintf("field:%d", f.i), f.x, f.y, f.x+19, f.y+3)
	}
	dayY := offY + 19
	prefix := st.Dim.Render("Days (click or alt+1–7)  ")
	dx := offX + 3 + lipgloss.Width(prefix)
	names := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	for n, label := range names {
		idx := (n + 1) % 7
		chip := st.Dim.Render(" " + label + " ")
		if m.days[idx] {
			chip = st.Green.Render("[" + label + "]")
		}
		w := lipgloss.Width(chip)
		m.a.hits.add(fmt.Sprintf("day:%d", idx), dx, dayY, dx+w-1, dayY)
		dx += w + 1 // chips joined with a space
	}
	m.a.registerButtons(offX, offY, modal, m.formButtons())
}

// schedNow is the wall clock the strip is drawn against, swapped by the
// Storybook frame generator: today's segments and the active-rule line are read
// off it, so a real clock would give a different frame on every run.
var schedNow = time.Now

func (m schedulerModel) listBody() string {
	a := m.a
	st := a.styles
	var b strings.Builder
	enabled := st.Dim.Render("[ ] enabled")
	if a.cfg.SchedulerEnabled {
		enabled = st.Green.Render("[x] enabled")
	}
	b.WriteString(" " + st.Dim.Render("← esc") + st.Faint.Render(" │ ") + st.Title.Render("Scheduler") + "   " + enabled + "\n")

	// 24h strip for today.
	now := schedNow()
	segs := sched.Segments(a.cfg.Rules, now.Weekday())
	stripW := a.width - 8
	if stripW < 24 {
		stripW = 24
	}
	var strip, axis strings.Builder
	activeLabel := "∞ (no rule)"
	if r, ok := sched.Active(a.cfg.Rules, now); ok {
		activeLabel = FmtLimit(r.Down) + " (" + safeText(r.Label) + ")"
	}
	for _, seg := range segs {
		w := (seg.To - seg.From) * stripW / 1440
		if w < 1 {
			w = 1
		}
		label := FmtLimit(seg.Down)
		cell := label
		if cellWidth(cell) > w {
			cell = ""
		}
		padTotal := w - cellWidth(cell)
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

	// Rules table. Rows sit below the header (1) + strip panel (5 lines)
	// + rules panel border (1) + column header (1). The label column
	// shrinks on narrow terminals so rows never wrap inside the panel.
	a.hits.line("back", 0, a.width)
	labelW := a.width - 48
	if labelW < 10 {
		labelW = 10
	}
	if labelW > 34 {
		labelW = 34
	}
	rows := []string{st.Dim.Render(pad("WINDOW", 16) + pad("DAYS", 12) + pad("RULE", labelW) + lpad("DOWN/UP", 14))}
	for i, r := range a.cfg.Rules {
		a.hits.add(fmt.Sprintf("rule:%d", i), 1, 8+i, a.width-2, 8+i)
		_ = r
		marker, style := "  ", st.Text
		if i == m.cursor {
			marker, style = st.Brand.Render("▸ "), st.Title
		}
		limit := FmtLimit(r.Down) + " / " + FmtLimit(r.Up)
		line := marker + style.Render(pad(safeText(r.Start)+" – "+safeText(r.End), 14)) +
			st.Text.Render(pad(fmtDays(r.Days), 12)) +
			st.Text.Render(pad(safeText(r.Label), labelW)) +
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

	return a.screenFrame(b.String(), []keyHint{
		{"+", "+", "add rule"}, {"e", "e", "edit"}, {"-", "-", "remove"},
		{" ", "space", "toggle scheduler"}, {"esc", "esc", "back"},
	})
}
