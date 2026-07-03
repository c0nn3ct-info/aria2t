package ui

import (
	"strconv"
	"strings"
)

// region is one clickable rectangle (inclusive cell bounds) tagged with a
// semantic action id like "tab:1", "row:4", "key:add" or "btn:submit".
type region struct {
	id             string
	x0, y0, x1, y1 int
}

// hitmap collects the regions of the most recent render. Views rebuild it
// on every View() call, so it always matches what is on screen.
type hitmap struct {
	regions []region
}

func (h *hitmap) reset() { h.regions = h.regions[:0] }

// add registers a rectangle; later additions win over earlier ones so
// overlays naturally shadow the screen beneath them.
func (h *hitmap) add(id string, x0, y0, x1, y1 int) {
	h.regions = append(h.regions, region{id: id, x0: x0, y0: y0, x1: x1, y1: y1})
}

// line registers a full-width single-line region.
func (h *hitmap) line(id string, y, width int) { h.add(id, 0, y, width-1, y) }

// hit resolves a click position to the topmost region id.
func (h *hitmap) hit(x, y int) (string, bool) {
	for i := len(h.regions) - 1; i >= 0; i-- {
		r := h.regions[i]
		if x >= r.x0 && x <= r.x1 && y >= r.y0 && y <= r.y1 {
			return r.id, true
		}
	}
	return "", false
}

// splitID separates "row:4" into ("row", "4").
func splitID(id string) (kind, arg string) {
	kind, arg, _ = strings.Cut(id, ":")
	return kind, arg
}

// argInt parses the numeric argument of a region id, -1 on garbage.
func argInt(arg string) int {
	n, err := strconv.Atoi(arg)
	if err != nil {
		return -1
	}
	return n
}
