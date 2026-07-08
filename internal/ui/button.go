package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// btnVariant selects a filled button style. Buttons are single-span labels, so
// unlike Modal/Input they carry a Background safely (the span resets before any
// sibling span, so it cannot band). Green = the safe/expected choice, Red = the
// consequential one; each dialog assigns them per action.
type btnVariant int

const (
	btnGreen btnVariant = iota // green fill
	btnRed                     // red fill
)

// button is one filled action control in an overlay footer. token is dispatched
// through keyFromToken on click (so a click is identical to pressing the key);
// key is the glyph shown INSIDE the label ("Add  ↵", "Cancel  esc").
type button struct {
	token   string
	label   string
	key     string
	variant btnVariant
}

func (b button) style(st Styles) lipgloss.Style {
	if b.variant == btnRed {
		return st.BtnRed
	}
	return st.BtnGreen
}

func (b button) render(st Styles) string { return b.style(st).Render(b.label + "  " + b.key) }

// buttonRow renders an ordered set of buttons as one left-aligned line
// (secondary/cancel first … primary last), separated by three spaces. It MUST
// be the modal body's last content line so registerButtons can anchor it.
func (a *App) buttonRow(buttons []button) string {
	parts := make([]string, len(buttons))
	for i, b := range buttons {
		parts[i] = b.render(a.styles)
	}
	return strings.Join(parts, "   ")
}

// registerButtons registers a "btn:<token>" region per button, mirroring
// buttonRow's geometry: x-origin offX+3 (rounded border + Padding(1,2)), the row
// is the modal's last content line at offY+Height-3. Call after the modal is
// composed. Re-rendering each button here (as buttonRow does) is the deliberate
// same-strings-feed-both-view-and-hitmap pattern so they can never drift.
func (a *App) registerButtons(offX, offY int, modal string, buttons []button) {
	y := offY + lipgloss.Height(modal) - 3
	x := offX + 3
	for _, b := range buttons {
		w := lipgloss.Width(b.render(a.styles))
		a.hits.add("btn:"+b.token, x, y, x+w-1, y)
		x += w + 3
	}
}

// dispatchBtn maps a clicked button token to its key message, so a button click
// runs exactly the same path as pressing the key.
func dispatchBtn(arg string) tea.KeyMsg { return keyFromToken(arg) }
