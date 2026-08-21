package daemon

import "testing"

func TestStateRoundtrip(t *testing.T) {
	dir := t.TempDir()
	if _, _, _, ok := State(dir); ok {
		t.Fatal("missing state file must report ok=false")
	}
	writeState(stateFilePath(dir), daemonState{PID: 7, Port: 6801, Secret: "s3"})
	pid, port, secret, ok := State(dir)
	if !ok || pid != 7 || port != 6801 || secret != "s3" {
		t.Fatalf("State = %d %d %q %v", pid, port, secret, ok)
	}
}

func TestStateMalformedOrPortless(t *testing.T) {
	origR := daemonStateRead
	defer func() { daemonStateRead = origR }()

	daemonStateRead = func(string) ([]byte, error) { return []byte("{bad"), nil }
	if _, _, _, ok := State("x"); ok {
		t.Fatal("malformed state must report ok=false")
	}
	daemonStateRead = func(string) ([]byte, error) { return []byte(`{"port":0}`), nil }
	if _, _, _, ok := State("x"); ok {
		t.Fatal("portless state must report ok=false")
	}
}
