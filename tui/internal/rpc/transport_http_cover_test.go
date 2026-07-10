package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPTransportClose(t *testing.T) {
	if err := New("http://unused", "").Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
}

func TestHTTPCallMarshalError(t *testing.T) {
	tr := newHTTPTransport("http://unused")
	// A channel cannot be marshaled to JSON.
	if err := tr.Call(context.Background(), "m", []any{make(chan int)}, nil); err == nil {
		t.Fatal("want marshal error")
	}
}

func TestHTTPCallBadURL(t *testing.T) {
	tr := newHTTPTransport("://missing-scheme")
	if err := tr.Call(context.Background(), "m", nil, nil); err == nil {
		t.Fatal("want request creation error")
	}
}

func TestHTTPCallUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // nothing listens anymore
	tr := newHTTPTransport(srv.URL)
	if err := tr.Call(context.Background(), "m", nil, nil); err == nil {
		t.Fatal("want connection error")
	}
}

func TestHTTPCallBodyReadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("hijacking not supported")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		// Declare more body than we send, then slam the connection shut.
		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 1000\r\n\r\npartial"))
		conn.Close()
	}))
	defer srv.Close()
	tr := newHTTPTransport(srv.URL)
	if err := tr.Call(context.Background(), "m", nil, nil); err == nil {
		t.Fatal("want body read error")
	}
}

func TestHTTPCallNonJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>gateway sadness</html>"))
	}))
	defer srv.Close()
	tr := newHTTPTransport(srv.URL)
	err := tr.Call(context.Background(), "m", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "bad response") {
		t.Fatalf("err = %v", err)
	}
}
