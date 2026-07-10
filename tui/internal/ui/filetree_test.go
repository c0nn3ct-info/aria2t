package ui

import (
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

func sampleFiles() []rpc.File {
	return []rpc.File{
		{Index: "1", Path: "/d/A/x.bin", Length: "100", CompletedLength: "50", Selected: "true"},
		{Index: "2", Path: "/d/A/y.bin", Length: "200", CompletedLength: "0", Selected: "false"},
		{Index: "3", Path: "/d/z.bin", Length: "300", CompletedLength: "300", Selected: "true"},
	}
}

func TestBuildTreeStructure(t *testing.T) {
	root := buildTree(sampleFiles(), "/d")
	if len(root.children) != 2 {
		t.Fatalf("root children = %d", len(root.children))
	}
	dir := root.children[0]
	if dir.name != "A" || dir.isLeaf() || len(dir.children) != 2 {
		t.Fatalf("A dir = %+v", dir)
	}
	if dir.children[0].index != "1" || dir.children[0].parent != dir {
		t.Fatalf("leaf x = %+v", dir.children[0])
	}
	z := root.children[1]
	if z.name != "z.bin" || !z.isLeaf() || z.index != "3" {
		t.Fatalf("z = %+v", z)
	}
}

func TestBuildTreeFallbackWhenPathEqualsDir(t *testing.T) {
	root := buildTree([]rpc.File{{Index: "1", Path: "/d"}}, "/d")
	if len(root.children) != 1 || root.children[0].name != "/d" {
		t.Fatalf("fallback leaf = %+v", root.children)
	}
}

func TestSplitSegs(t *testing.T) {
	if got := splitSegs("//a///b/"); strings.Join(got, ",") != "a,b" {
		t.Fatalf("segs = %v", got)
	}
	if got := splitSegs(""); len(got) != 0 {
		t.Fatalf("empty = %v", got)
	}
}

func TestFlattenAndCollapse(t *testing.T) {
	root := buildTree(sampleFiles(), "/d")
	rows := flatten(root)
	if len(rows) != 4 { // A, x, y, z
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].depth != 0 || rows[1].depth != 1 {
		t.Fatalf("depths = %d %d", rows[0].depth, rows[1].depth)
	}
	root.children[0].collapsed = true
	rows = flatten(root)
	if len(rows) != 2 || rows[0].name != "A" || rows[1].name != "z.bin" {
		t.Fatalf("collapsed rows = %v", rows)
	}
}

func TestCheckGlyphAndDirSel(t *testing.T) {
	root := buildTree(sampleFiles(), "/d")
	dir := root.children[0]
	if g := checkGlyph(dir); g != "[~]" { // one of two selected
		t.Fatalf("mixed glyph = %q", g)
	}
	if g := checkGlyph(root.children[1]); g != "[x]" { // z selected leaf
		t.Fatalf("leaf glyph = %q", g)
	}
	setSelected(dir, true)
	if g := checkGlyph(dir); g != "[x]" {
		t.Fatalf("all glyph = %q", g)
	}
	setSelected(dir, false)
	if g := checkGlyph(dir); g != "[ ]" {
		t.Fatalf("none glyph = %q", g)
	}
	if g := checkGlyph(dir.children[0]); g != "[ ]" {
		t.Fatalf("unselected leaf glyph = %q", g)
	}
}

func TestToggleNode(t *testing.T) {
	root := buildTree(sampleFiles(), "/d")
	leaf := root.children[1] // z, selected
	toggleNode(leaf)
	if leaf.selected {
		t.Fatal("leaf toggle must clear selection")
	}
	dir := root.children[0] // mixed → select all
	toggleNode(dir)
	if s, total := dirSel(dir); s != total {
		t.Fatalf("dir toggle to full: %d/%d", s, total)
	}
	toggleNode(dir) // full → none
	if s, _ := dirSel(dir); s != 0 {
		t.Fatalf("dir toggle to empty: sel=%d", s)
	}
}

func TestNodeSizeDoneAndCounts(t *testing.T) {
	root := buildTree(sampleFiles(), "/d")
	if nodeSize(root) != 600 || nodeDone(root) != 350 {
		t.Fatalf("size=%d done=%d", nodeSize(root), nodeDone(root))
	}
	if leafCount(root) != 3 || selectedLeaves(root) != 2 {
		t.Fatalf("leaves=%d sel=%d", leafCount(root), selectedLeaves(root))
	}
}

func TestSelectedIndices(t *testing.T) {
	root := buildTree(sampleFiles(), "/d")
	if got := strings.Join(selectedIndices(root), ","); got != "1,3" {
		t.Fatalf("indices = %q", got)
	}
	setSelected(root, false)
	if got := selectedIndices(root); len(got) != 0 {
		t.Fatalf("none selected = %v", got)
	}
	// A leaf with an empty index is never emitted.
	root = buildTree([]rpc.File{{Index: "", Path: "/d/a", Selected: "true"}}, "/d")
	if got := selectedIndices(root); len(got) != 0 {
		t.Fatalf("empty-index leaf must not emit: %v", got)
	}
}
