package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVerifyMatch(t *testing.T) {
	data := []byte("hello aria2t")
	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	ok, got, err := Verify(writeTemp(t, data), strings.ToUpper(want), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != want {
		t.Fatalf("ok=%v got=%s want=%s", ok, got, want)
	}
}

func TestVerifyMismatch(t *testing.T) {
	ok, _, err := Verify(writeTemp(t, []byte("data")), strings.Repeat("0", 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("must mismatch")
	}
}

func TestProgressReported(t *testing.T) {
	data := make([]byte, 3<<20) // 3 MiB → several 1 MiB chunks
	var calls int
	var lastDone, lastTotal int64
	_, _, err := Verify(writeTemp(t, data), strings.Repeat("0", 64), func(done, total int64) {
		calls++
		lastDone, lastTotal = done, total
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls < 3 || lastDone != int64(len(data)) || lastTotal != int64(len(data)) {
		t.Fatalf("calls=%d done=%d total=%d", calls, lastDone, lastTotal)
	}
}

func TestVerifyMissingFile(t *testing.T) {
	if _, _, err := Verify(filepath.Join(t.TempDir(), "absent"), "00", nil); err == nil {
		t.Fatal("want error for missing file")
	}
}
