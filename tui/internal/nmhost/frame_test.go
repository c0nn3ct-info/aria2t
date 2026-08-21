package nmhost

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"
)

// failWriter accepts allow bytes, then errors.
type failWriter struct{ allow int }

func (f *failWriter) Write(p []byte) (int, error) {
	if f.allow >= len(p) {
		f.allow -= len(p)
		return len(p), nil
	}
	n := f.allow
	f.allow = 0
	return n, errors.New("sink closed")
}

// frame appends one length-prefixed message to buf.
func frame(buf *bytes.Buffer, body string) {
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(body)))
	buf.Write(lenBuf[:])
	buf.WriteString(body)
}

func TestFrameRoundtrip(t *testing.T) {
	var buf bytes.Buffer
	if err := writeFrame(&buf, []byte(`{"id":"1"}`)); err != nil {
		t.Fatal(err)
	}
	got, err := readFrame(&buf)
	if err != nil || string(got) != `{"id":"1"}` {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestReadFrameEOF(t *testing.T) {
	if _, err := readFrame(bytes.NewReader(nil)); !errors.Is(err, io.EOF) {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestReadFrameShortHeader(t *testing.T) {
	if _, err := readFrame(strings.NewReader("\x01\x02")); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want unexpected EOF", err)
	}
}

func TestReadFrameZeroLength(t *testing.T) {
	_, err := readFrame(bytes.NewReader([]byte{0, 0, 0, 0}))
	if err == nil || !strings.Contains(err.Error(), "zero-length") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadFrameTooLarge(t *testing.T) {
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], maxMessageSize+1)
	_, err := readFrame(bytes.NewReader(hdr[:]))
	if err == nil || !strings.Contains(err.Error(), "frame too large") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadFrameShortBody(t *testing.T) {
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], 10)
	in := append(hdr[:], []byte("abc")...)
	if _, err := readFrame(bytes.NewReader(in)); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want unexpected EOF", err)
	}
}

func TestWriteFrameHeaderError(t *testing.T) {
	if err := writeFrame(&failWriter{}, []byte("x")); err == nil {
		t.Fatal("want header write error")
	}
}

func TestWriteFrameBodyError(t *testing.T) {
	if err := writeFrame(&failWriter{allow: 4}, []byte("x")); err == nil {
		t.Fatal("want body write error")
	}
}
