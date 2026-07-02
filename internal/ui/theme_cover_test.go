package ui

import "testing"

func TestPaletteByName(t *testing.T) {
	if p := PaletteByName("light"); p.Name != "light" {
		t.Fatalf("light palette = %q", p.Name)
	}
	if p := PaletteByName("dark"); p.Name != "dark" {
		t.Fatalf("dark palette = %q", p.Name)
	}
	if p := PaletteByName("nonsense"); p.Name != "dark" {
		t.Fatalf("fallback palette = %q", p.Name)
	}
}

func TestNewStyles(t *testing.T) {
	st := NewStyles(TokyoNightDay)
	if st.P.Name != "light" {
		t.Fatalf("styles palette = %q", st.P.Name)
	}
	if st.Brand.Render("x") == "" {
		t.Fatal("brand style renders empty")
	}
}
