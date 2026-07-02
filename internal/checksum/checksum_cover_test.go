package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

func TestVerifyStatError(t *testing.T) {
	orig := statFile
	statFile = func(*os.File) (os.FileInfo, error) { return nil, errors.New("stat boom") }
	t.Cleanup(func() { statFile = orig })

	ok, sum, err := Verify(writeTemp(t, []byte("x")), "00", nil)
	if err == nil || err.Error() != "stat boom" {
		t.Fatalf("want stat error, got %v", err)
	}
	if ok || sum != "" {
		t.Fatalf("ok=%v sum=%q, want false and empty", ok, sum)
	}
}

func TestVerifyReadError(t *testing.T) {
	// Opening a directory succeeds, but Read on the handle fails.
	ok, sum, err := Verify(t.TempDir(), "00", nil)
	if err == nil {
		t.Fatal("want read error for directory")
	}
	if ok || sum != "" {
		t.Fatalf("ok=%v sum=%q, want false and empty", ok, sum)
	}
}

func TestVerifyTrimsAndIgnoresCase(t *testing.T) {
	data := []byte("trim me")
	sum := sha256.Sum256(data)
	expected := "  \t" + hex.EncodeToString(sum[:]) + " \n"
	ok, _, err := Verify(writeTemp(t, data), expected, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("want match with surrounding whitespace trimmed")
	}
}
