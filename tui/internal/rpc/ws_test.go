package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// wsServer upgrades and passes the connection to handle.
func wsServer(t *testing.T, handle func(*websocket.Conn)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		handle(conn)
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestWSCallRoundtrip(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		for {
			var req request
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			if req.Method != "aria2.getVersion" {
				t.Errorf("method = %q", req.Method)
			}
			conn.WriteJSON(map[string]any{
				"jsonrpc": "2.0", "id": req.ID,
				"result": map[string]string{"version": "1.37.0"},
			})
		}
	})
	c, err := NewWS(url, "s")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	v, err := c.GetVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v != "1.37.0" {
		t.Fatalf("version = %q", v)
	}
}

func TestWSNotificationRouted(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte(
			`{"jsonrpc":"2.0","method":"aria2.onDownloadComplete","params":[{"gid":"g42"}]}`))
		// keep connection open until client closes
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
	select {
	case n := <-c.Notifications():
		if n.Method != "aria2.onDownloadComplete" || len(n.GIDs) != 1 || n.GIDs[0] != "g42" {
			t.Fatalf("bad notification: %+v", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("notification not delivered")
	}
}

func TestWSContextCancel(t *testing.T) {
	url := wsServer(t, func(conn *websocket.Conn) {
		// swallow requests, never answer
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
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.GetVersion(ctx); err == nil {
		t.Fatal("want context error")
	}
}

func TestGidsFromParams(t *testing.T) {
	got := gidsFromParams(json.RawMessage(`[{"gid":"a"},{"gid":"b"}]`))
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
	if gidsFromParams(json.RawMessage(`not json`)) != nil {
		t.Fatal("malformed params must yield nil")
	}
}
