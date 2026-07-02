package ui

import (
	"strings"
	"testing"

	"aria2t/internal/rpc"
)

func TestStatsViewActiveWithSpeed(t *testing.T) {
	a, _ := testApp(t)
	a.snap.Active[0].DownloadSpeed = "500"
	a.snap.Stat = rpc.GlobalStat{
		DownloadSpeed: "100", UploadSpeed: "50",
		NumActive: "1", NumWaiting: "3",
	}
	a.downHist.Push(10)
	a.upHist.Push(100) // upload peak wins
	out := a.stats.view()
	for _, want := range []string{"Global stats", "FINISHED", "BANDWIDTH BY DOWNLOAD", "█", "SESSION DOWNLOADED"} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q", want)
		}
	}
}

func TestStatsViewActiveZeroSpeed(t *testing.T) {
	a, _ := testApp(t)
	a.downHist.Push(100)
	a.upHist.Push(10) // download peak wins
	out := a.stats.view()
	if !strings.Contains(out, "BANDWIDTH BY DOWNLOAD") {
		t.Fatalf("view missing bandwidth panel: %q", out)
	}
	if strings.Contains(out, "no active downloads") {
		t.Fatal("active snapshot must list downloads")
	}
}

func TestStatsViewEmptyNarrow(t *testing.T) {
	a, _ := testApp(t)
	a.snap = snapshot{}
	out := a.stats.view()
	if !strings.Contains(out, "no active downloads") {
		t.Fatalf("view missing empty note: %q", out)
	}

	a.width = 10 // trips the minimum-width guard
	if out := a.stats.view(); out == "" {
		t.Fatal("narrow view must still render")
	}
}
