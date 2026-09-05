package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rivo/uniseg"
)

// updateInput adds grapheme-boundary cursor movement and deletion to Bubbles'
// text input. The upstream widget stores rune offsets; without this adapter a
// backspace can leave a detached combining mark or half of an emoji sequence.
func updateInput(in textinput.Model, msg tea.KeyMsg) (textinput.Model, tea.Cmd) {
	pos := in.Position()
	bounds := graphemeRuneBounds(in.Value())
	previous := func() int {
		prev := 0
		for _, b := range bounds {
			if b >= pos {
				break
			}
			prev = b
		}
		return prev
	}
	next := func() int {
		return nextGraphemeBoundary(bounds, pos, len([]rune(in.Value())))
	}
	switch msg.String() {
	case "left":
		in.SetCursor(previous())
		return in, nil
	case "right":
		in.SetCursor(next())
		return in, nil
	case "backspace", "ctrl+h":
		if pos > 0 {
			runes := []rune(in.Value())
			start := previous()
			in.SetValue(string(append(runes[:start], runes[pos:]...)))
			in.SetCursor(start)
		}
		return in, nil
	case "delete", "ctrl+d":
		runes := []rune(in.Value())
		if pos < len(runes) {
			end := next()
			in.SetValue(string(append(runes[:pos], runes[end:]...)))
			in.SetCursor(pos)
		}
		return in, nil
	}
	return in.Update(msg)
}

func updateTextArea(in textarea.Model, msg tea.KeyMsg) (textarea.Model, tea.Cmd) {
	row := in.Line()
	lines := strings.Split(in.Value(), "\n")
	lineInfo := in.LineInfo()
	pos := lineInfo.StartColumn + lineInfo.ColumnOffset
	bounds := graphemeRuneBounds(lines[row])
	previous := func() int {
		prev := 0
		for _, b := range bounds {
			if b >= pos {
				break
			}
			prev = b
		}
		return prev
	}
	next := func() int {
		return nextGraphemeBoundary(bounds, pos, len([]rune(lines[row])))
	}
	restore := func(col int) textarea.Model {
		in.SetValue(strings.Join(lines, "\n"))
		for in.Line() > row {
			in.CursorUp()
		}
		in.SetCursor(col)
		return in
	}
	switch msg.String() {
	case "left":
		if pos > 0 {
			in.SetCursor(previous())
			return in, nil
		}
	case "right":
		if pos < len([]rune(lines[row])) {
			in.SetCursor(next())
			return in, nil
		}
	case "backspace", "ctrl+h":
		if pos > 0 {
			runes := []rune(lines[row])
			start := previous()
			lines[row] = string(append(runes[:start], runes[pos:]...))
			return restore(start), nil
		}
	case "delete", "ctrl+d":
		runes := []rune(lines[row])
		if pos < len(runes) {
			end := next()
			lines[row] = string(append(runes[:pos], runes[end:]...))
			return restore(pos), nil
		}
	}
	return in.Update(msg)
}

func graphemeRuneBounds(s string) []int {
	bounds := []int{0}
	pos := 0
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		pos += len(g.Runes())
		bounds = append(bounds, pos)
	}
	return bounds
}

func nextGraphemeBoundary(bounds []int, pos, end int) int {
	for _, b := range bounds {
		if b > pos {
			return b
		}
	}
	return end
}
