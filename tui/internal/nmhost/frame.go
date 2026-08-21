// Package nmhost implements the native-messaging host that bridges the
// aria2t browser extension to the managed aria2c daemon: 4-byte
// little-endian length-prefixed JSON frames on stdin/stdout, one ack per
// request. The host is spawned by the browser per extension connection and
// exits on EOF; the daemon it ensures keeps running.
package nmhost

import (
	"encoding/binary"
	"fmt"
	"io"
)

// maxMessageSize caps a frame body at 1 MiB — Chrome enforces the same
// limit on messages sent to the browser.
const maxMessageSize = 1 << 20

// readFrame reads one length-prefixed message from r.
func readFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(lenBuf[:])
	if length == 0 {
		return nil, fmt.Errorf("zero-length frame")
	}
	if length > maxMessageSize {
		return nil, fmt.Errorf("frame too large: %d bytes", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// writeFrame writes one length-prefixed message to w.
func writeFrame(w io.Writer, body []byte) error {
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(body)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}
