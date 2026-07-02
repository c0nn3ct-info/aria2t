package rpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestNewWSDialError(t *testing.T) {
	// A plain HTTP server refuses the websocket upgrade.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	if _, err := NewWS("ws"+strings.TrimPrefix(srv.URL, "http"), ""); err == nil {
		t.Fatal("want handshake error")
	}
}

// TestWSReadLoopSkipsGarbage feeds the read loop frames it must ignore
// (malformed JSON, a non-string id, an id-less frame without a method) and
// then verifies a real call still succeeds. The call passes a nil result to
// hit the ws Call result short-circuit.
func TestWSReadLoopSkipsGarbage(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte(`{{{not json`))
		conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":123,"result":"x"}`))
		conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","id":null,"params":[]}`))
		for {
			var req request
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "OK"})
		}
	})
	c, err := NewWS(url, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.SaveSession(context.Background()); err != nil {
		t.Fatalf("call after garbage frames: %v", err)
	}
}

func TestWSErrorResponse(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		for {
			var req request
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			conn.WriteJSON(map[string]any{
				"jsonrpc": "2.0", "id": req.ID,
				"error": map[string]any{"code": 1, "message": "nope"},
			})
		}
	})
	c, err := NewWS(url, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.GetVersion(context.Background())
	var rpcErr *Error
	if !errors.As(err, &rpcErr) || rpcErr.Code != 1 || rpcErr.Message != "nope" {
		t.Fatalf("err = %v", err)
	}
}

// TestWSConnectionLost covers failAll draining pending calls and both Call
// failure modes: an in-flight call sees its channel closed, and a later call
// is rejected because the transport is marked closed.
func TestWSConnectionLost(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		conn.ReadMessage() // swallow one request, then drop the connection
	})
	c, err := NewWS(url, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.GetVersion(context.Background()); err == nil || !strings.Contains(err.Error(), "connection lost") {
		t.Fatalf("in-flight err = %v", err)
	}
	if _, err := c.GetVersion(context.Background()); err == nil || !strings.Contains(err.Error(), "websocket closed") {
		t.Fatalf("write-after-close err = %v", err)
	}
}

func TestWSCallMarshalError(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	c, err := NewWS(url, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.t.Call(context.Background(), "m", []any{make(chan int)}, nil); err == nil {
		t.Fatal("want marshal error")
	}
}

// TestWSWriteError hits the WriteMessage failure branch while the transport
// is not yet marked closed: the underlying conn is closed locally and no
// readLoop runs, so t.closed stays false.
func TestWSWriteError(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	d := websocket.Dialer{}
	conn, _, err := d.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	tr := &wsTransport{
		conn:    conn,
		notify:  make(chan Notification, 1),
		pending: make(map[string]chan response),
		done:    make(chan struct{}),
	}
	if err := tr.Call(context.Background(), "m", nil, nil); err == nil {
		t.Fatal("want write error")
	}
	tr.mu.Lock()
	n := len(tr.pending)
	tr.mu.Unlock()
	if n != 0 {
		t.Fatalf("pending not cleaned up: %d entries", n)
	}
}

func TestClientNotificationsAccessor(t *testing.T) {
	c := New("http://unused", "")
	if c.Notifications() == nil {
		t.Fatal("Notifications() = nil")
	}
}
