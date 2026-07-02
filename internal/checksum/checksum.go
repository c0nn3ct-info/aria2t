// Package checksum verifies downloaded files against expected digests.
package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// Verify streams path through sha-256 and compares against expectedHex
// (case-insensitive). progress, if non-nil, is called after every chunk
// with bytes hashed so far and the file size.
func Verify(path, expectedHex string, progress func(done, total int64)) (ok bool, computedHex string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return false, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return false, "", err
	}
	total := info.Size()

	h := sha256.New()
	buf := make([]byte, 1<<20)
	var done int64
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
			done += int64(n)
			if progress != nil {
				progress(done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return false, "", rerr
		}
	}
	computedHex = hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(computedHex, strings.TrimSpace(expectedHex)), computedHex, nil
}
