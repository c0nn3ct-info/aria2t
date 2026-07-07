package daemon

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestReapStaleNoStateFile(t *testing.T) {
	// Default seam over a missing file → read error → no-op (no panic).
	reapStale(t.TempDir())
}

func TestReapStaleMalformedOrPortless(t *testing.T) {
	origR := daemonStateRead
	defer func() { daemonStateRead = origR }()

	daemonStateRead = func(string) ([]byte, error) { return []byte("{bad"), nil }
	reapStale("x") // unmarshal error → no-op

	daemonStateRead = func(string) ([]byte, error) { return []byte(`{"port":0}`), nil }
	reapStale("x") // port 0 → no-op
}

func TestReapStaleShutsDownLiveDaemon(t *testing.T) {
	var got atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "shutdown") || strings.Contains(string(body), "saveSession") {
			got.Add(1)
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":"OK"}`))
	}))
	defer srv.Close()

	// httptest URL is http://127.0.0.1:PORT — extract the port for the state.
	port, err := strconv.Atoi(srv.URL[strings.LastIndex(srv.URL, ":")+1:])
	if err != nil {
		t.Fatal(err)
	}
	origR := daemonStateRead
	defer func() { daemonStateRead = origR }()
	raw, _ := json.Marshal(daemonState{PID: 1, Port: port, Secret: "s"})
	daemonStateRead = func(string) ([]byte, error) { return raw, nil }

	reapStale("x")
	if got.Load() == 0 {
		t.Fatal("reap must call saveSession/shutdown on a live daemon")
	}
}

func TestWriteStateAndPath(t *testing.T) {
	dir := t.TempDir()
	path := stateFilePath(dir)
	if filepath.Base(path) != "daemon.json" {
		t.Fatalf("state path = %q", path)
	}
	writeState(path, daemonState{PID: 42, Port: 6800, Secret: "abc"})
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var st daemonState
	if json.Unmarshal(raw, &st) != nil || st.PID != 42 || st.Port != 6800 || st.Secret != "abc" {
		t.Fatalf("state roundtrip: %s", raw)
	}
}

func TestStartWritesStateAndStopRemovesIt(t *testing.T) {
	dir := t.TempDir()
	bin := fakeBin(t, `trap 'exit 0' TERM INT
while true; do sleep 0.1; done`)
	d, err := Start(Options{
		Bin:        bin,
		DataDir:    dir,
		ReadyProbe: func(int, string) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stateFilePath(dir)); err != nil {
		t.Fatalf("Start must write the reap state: %v", err)
	}
	if err := d.Stop(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stateFilePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("Stop must remove the reap state, err=%v", err)
	}
}
