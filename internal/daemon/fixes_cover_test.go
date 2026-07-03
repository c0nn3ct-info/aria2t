package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAliveTransitions(t *testing.T) {
	bin := fakeBin(t, `trap 'exit 0' TERM INT
while true; do sleep 0.1; done`)
	d, err := Start(Options{
		Bin:        bin,
		DataDir:    t.TempDir(),
		ReadyProbe: func(int, string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Alive() {
		t.Fatal("freshly started daemon must be alive")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if d.Alive() {
		t.Fatal("stopped daemon must not be alive")
	}
}

func TestStartConfWriteError(t *testing.T) {
	dataDir := t.TempDir()
	// Existing conf path as a DIRECTORY makes the secret write fail.
	if err := os.MkdirAll(filepath.Join(dataDir, "aria2t.conf"), 0o700); err != nil {
		t.Fatal(err)
	}
	bin := fakeBin(t, `exit 0`)
	_, err := Start(Options{Bin: bin, DataDir: dataDir})
	if err == nil || !strings.Contains(err.Error(), "aria2t.conf") {
		t.Fatalf("err = %v", err)
	}
}

func TestSecretLandsInConfFileNotArgs(t *testing.T) {
	dataDir := t.TempDir()
	bin := fakeBin(t, `trap 'exit 0' TERM
while true; do sleep 0.1; done`)
	d, err := Start(Options{
		Bin:        bin,
		DataDir:    dataDir,
		Secret:     "sekret",
		ReadyProbe: func(int, string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		d.Stop(ctx)
	}()
	raw, err := os.ReadFile(filepath.Join(dataDir, "aria2t.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "rpc-secret=sekret") {
		t.Fatalf("conf = %q", raw)
	}
	if info, _ := os.Stat(filepath.Join(dataDir, "aria2t.conf")); info.Mode().Perm() != 0o600 {
		t.Fatalf("conf mode = %v", info.Mode())
	}
}
