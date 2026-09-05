package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestGraphemeAwareTextInput(t *testing.T) {
	in := textinput.New()
	in.SetValue("Ae\u0301👩‍💻Z")
	in.CursorEnd()

	in, _ = updateInput(in, tea.KeyMsg{Type: tea.KeyLeft})
	if in.Position() != 6 { // before Z; the emoji occupies three runes
		t.Fatalf("left cursor = %d", in.Position())
	}
	in, _ = updateInput(in, tea.KeyMsg{Type: tea.KeyLeft})
	if in.Position() != 3 { // before the complete emoji sequence
		t.Fatalf("left over emoji = %d", in.Position())
	}
	in, _ = updateInput(in, tea.KeyMsg{Type: tea.KeyBackspace})
	if in.Value() != "A👩‍💻Z" || in.Position() != 1 {
		t.Fatalf("combining cluster backspace = %q @ %d", in.Value(), in.Position())
	}
	in, _ = updateInput(in, tea.KeyMsg{Type: tea.KeyRight})
	if in.Position() != 4 {
		t.Fatalf("right over emoji = %d", in.Position())
	}
	in.SetCursor(1)
	in, _ = updateInput(in, tea.KeyMsg{Type: tea.KeyDelete})
	if in.Value() != "AZ" || in.Position() != 1 {
		t.Fatalf("emoji delete = %q @ %d", in.Value(), in.Position())
	}
	in.CursorStart()
	in, _ = updateInput(in, tea.KeyMsg{Type: tea.KeyBackspace})
	in.CursorEnd()
	in, _ = updateInput(in, tea.KeyMsg{Type: tea.KeyDelete})
	if in.Value() != "AZ" {
		t.Fatal("boundary delete changed text")
	}
	_ = in.Focus()
	in, _ = updateInput(in, key("x"))
	if in.Value() != "AZx" {
		t.Fatalf("ordinary input = %q", in.Value())
	}
}

func TestGraphemeAwareTextArea(t *testing.T) {
	in := textarea.New()
	in.SetWidth(40)
	in.SetValue("Ae\u0301👩‍💻Z\nQ")
	in.CursorUp()
	in.SetCursor(7)

	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyLeft})
	if textareaPos(in) != 6 {
		t.Fatalf("textarea left = %d", textareaPos(in))
	}
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyLeft})
	if textareaPos(in) != 3 {
		t.Fatalf("textarea emoji left = %d", textareaPos(in))
	}
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyBackspace})
	if !strings.HasPrefix(in.Value(), "A👩‍💻Z") || in.Line() != 0 || textareaPos(in) != 1 {
		t.Fatalf("textarea backspace = %q line=%d col=%d", in.Value(), in.Line(), textareaPos(in))
	}
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyRight})
	if textareaPos(in) != 4 {
		t.Fatalf("textarea right = %d", textareaPos(in))
	}
	in.SetCursor(1)
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyDelete})
	if !strings.HasPrefix(in.Value(), "AZ") {
		t.Fatalf("textarea delete = %q", in.Value())
	}
	in.CursorStart()
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyLeft})
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyBackspace})
	in.CursorEnd()
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyRight})
	in, _ = updateTextArea(in, tea.KeyMsg{Type: tea.KeyDelete})
	_ = in.Focus()
	in, _ = updateTextArea(in, key("x"))
	if !strings.Contains(in.Value(), "x") {
		t.Fatalf("textarea ordinary input = %q", in.Value())
	}
}

func textareaPos(in textarea.Model) int {
	info := in.LineInfo()
	return info.StartColumn + info.ColumnOffset
}

func TestGraphemeBoundsEmpty(t *testing.T) {
	if got := graphemeRuneBounds(""); len(got) != 1 || got[0] != 0 {
		t.Fatalf("empty bounds = %v", got)
	}
	if got := nextGraphemeBoundary([]int{0}, 1, 2); got != 2 {
		t.Fatalf("fallback boundary = %d", got)
	}
}
