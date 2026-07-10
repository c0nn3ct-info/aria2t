package ui

import (
	"strings"

	"aria2t/internal/rpc"
)

// treeNode is one node in a download's file tree. Leaves are files (index
// set); interior nodes are directories synthesized from path segments. The
// synthetic root carries no name and is never rendered.
type treeNode struct {
	name      string // path segment; "" for the synthetic root
	index     string // aria2 File.Index (leaves only)
	gid       string // owning download (multi-download metalink picker only)
	length    int64
	completed int64
	selected  bool // leaves only
	collapsed bool // dirs only
	depth     int  // set by flatten
	children  []*treeNode
	parent    *treeNode
}

// buildTree turns aria2's flat file list into a directory tree, keeping the
// files' original order so the derived select-file indices stay aligned with
// what aria2 reported.
func buildTree(files []rpc.File, dir string) *treeNode {
	root := &treeNode{}
	for _, f := range files {
		segs := splitSegs(trimPathPrefix(f.Path, dir))
		if len(segs) == 0 {
			segs = []string{f.Path} // last resort: never drop a file
		}
		node := root
		for i, seg := range segs {
			child := findChild(node, seg)
			if child == nil {
				child = &treeNode{name: seg, parent: node}
				node.children = append(node.children, child)
			}
			if i == len(segs)-1 { // leaf carries the file's data
				child.index = f.Index
				child.length = f.Len()
				child.completed = f.Completed()
				child.selected = f.IsSelected()
			}
			node = child
		}
	}
	return root
}

// buildForest builds a flat picker for a metalink's downloads: one leaf per
// download (each aria2 gid is a separate file), tagged with its gid so the
// confirm step can keep or drop whole downloads.
func buildForest(statuses []rpc.Status) *treeNode {
	root := &treeNode{}
	for _, s := range statuses {
		root.children = append(root.children, &treeNode{
			name:      s.Name(),
			gid:       s.GID,
			length:    s.Total(),
			completed: s.Completed(),
			selected:  true,
			parent:    root,
		})
	}
	return root
}

// selectedGids returns the gids of selected top-level leaves (metalink picker).
func selectedGids(root *treeNode) []string {
	var out []string
	for _, c := range root.children {
		if c.isLeaf() && c.selected && c.gid != "" {
			out = append(out, c.gid)
		}
	}
	return out
}

// splitSegs splits a relative path into non-empty segments.
func splitSegs(rel string) []string {
	out := make([]string, 0, 4)
	for _, s := range strings.Split(rel, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func findChild(n *treeNode, name string) *treeNode {
	for _, c := range n.children {
		if c.name == name {
			return c
		}
	}
	return nil
}

func (n *treeNode) isLeaf() bool { return len(n.children) == 0 }

// flatten returns the visible nodes in display order, excluding the root and
// skipping the contents of collapsed directories. Depth is stamped for
// indentation.
func flatten(root *treeNode) []*treeNode {
	var out []*treeNode
	var walk func(n *treeNode, depth int)
	walk = func(n *treeNode, depth int) {
		for _, c := range n.children {
			c.depth = depth
			out = append(out, c)
			if !c.isLeaf() && !c.collapsed {
				walk(c, depth+1)
			}
		}
	}
	walk(root, 0)
	return out
}

// dirSel counts selected and total descendant leaves under a directory (a
// leaf counts as itself). It is only meaningful for interior nodes.
func dirSel(n *treeNode) (sel, total int) {
	if n.isLeaf() {
		if n.selected {
			return 1, 1
		}
		return 0, 1
	}
	for _, c := range n.children {
		s, t := dirSel(c)
		sel += s
		total += t
	}
	return sel, total
}

// checkGlyph returns the raw checkbox for a node: [x] all selected, [ ] none,
// [~] mixed (dirs only). Styling is the caller's job.
func checkGlyph(n *treeNode) string {
	if n.isLeaf() {
		if n.selected {
			return "[x]"
		}
		return "[ ]"
	}
	sel, total := dirSel(n)
	switch {
	case sel == 0:
		return "[ ]"
	case sel == total:
		return "[x]"
	default:
		return "[~]"
	}
}

// toggleNode flips a leaf, or drives a directory to all-selected when it is
// not already full, else to none.
func toggleNode(n *treeNode) {
	if n.isLeaf() {
		n.selected = !n.selected
		return
	}
	sel, total := dirSel(n)
	setSelected(n, sel < total)
}

func setSelected(n *treeNode, v bool) {
	if n.isLeaf() {
		n.selected = v
		return
	}
	for _, c := range n.children {
		setSelected(c, v)
	}
}

func nodeSize(n *treeNode) int64 {
	if n.isLeaf() {
		return n.length
	}
	var t int64
	for _, c := range n.children {
		t += nodeSize(c)
	}
	return t
}

func nodeDone(n *treeNode) int64 {
	if n.isLeaf() {
		return n.completed
	}
	var t int64
	for _, c := range n.children {
		t += nodeDone(c)
	}
	return t
}

// selectedIndices returns the aria2 indices of selected leaves in order —
// the value for the select-file option.
func selectedIndices(root *treeNode) []string {
	var out []string
	var walk func(n *treeNode)
	walk = func(n *treeNode) {
		for _, c := range n.children {
			if c.isLeaf() {
				if c.selected && c.index != "" {
					out = append(out, c.index)
				}
			} else {
				walk(c)
			}
		}
	}
	walk(root)
	return out
}

// leafCount is the number of files in the tree.
func leafCount(root *treeNode) int {
	n := 0
	var walk func(x *treeNode)
	walk = func(x *treeNode) {
		for _, c := range x.children {
			if c.isLeaf() {
				n++
			} else {
				walk(c)
			}
		}
	}
	walk(root)
	return n
}

// selectedLeaves is the number of selected files in the tree.
func selectedLeaves(root *treeNode) int {
	n := 0
	var walk func(x *treeNode)
	walk = func(x *treeNode) {
		for _, c := range x.children {
			if c.isLeaf() {
				if c.selected {
					n++
				}
			} else {
				walk(c)
			}
		}
	}
	walk(root)
	return n
}
