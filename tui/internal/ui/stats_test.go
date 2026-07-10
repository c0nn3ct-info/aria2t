package ui

import "testing"

func TestRingWraps(t *testing.T) {
	r := newRing(3)
	for i := int64(1); i <= 5; i++ {
		r.Push(i)
	}
	got := r.Slice()
	if len(got) != 3 || got[0] != 3 || got[1] != 4 || got[2] != 5 {
		t.Fatalf("got %v", got)
	}
}

func TestRingPartial(t *testing.T) {
	r := newRing(5)
	r.Push(7)
	r.Push(9)
	got := r.Slice()
	if len(got) != 2 || got[0] != 7 || got[1] != 9 {
		t.Fatalf("got %v", got)
	}
}

func TestRingMax(t *testing.T) {
	r := newRing(4)
	for _, v := range []int64{2, 9, 1} {
		r.Push(v)
	}
	if r.Max() != 9 {
		t.Fatalf("max = %d", r.Max())
	}
}
