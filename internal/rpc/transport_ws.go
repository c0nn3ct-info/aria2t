package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsTransport multiplexes JSON-RPC calls over one websocket connection and
// forwards aria2 notifications (frames without an id) to notify.
type wsTransport struct {
	conn   *websocket.Conn
	notify chan<- Notification

	mu      sync.Mutex // guards writes and pending
	pending map[string]chan response
	closed  bool
	done    chan struct{}
}

// NewWS dials url (ws://host:port/jsonrpc) and returns a client whose
// Notifications channel receives aria2 push events.
func NewWS(url, secret string) (*Client, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	notes := make(chan Notification, 16)
	t := &wsTransport{
		conn:    conn,
		notify:  notes,
		pending: make(map[string]chan response),
		done:    make(chan struct{}),
	}
	go t.readLoop()
	return &Client{t: t, secret: secret, notes: notes}, nil
}

func (t *wsTransport) readLoop() {
	defer close(t.done)
	for {
		_, raw, err := t.conn.ReadMessage()
		if err != nil {
			t.failAll(err)
			return
		}
		var r response
		if err := json.Unmarshal(raw, &r); err != nil {
			continue
		}
		if len(r.ID) == 0 || string(r.ID) == "null" { // notification
			if r.Method != "" {
				n := Notification{Method: r.Method, GIDs: gidsFromParams(r.Params)}
				select {
				case t.notify <- n:
				default: // never block the read loop on a slow consumer
				}
			}
			continue
		}
		var id string
		if err := json.Unmarshal(r.ID, &id); err != nil {
			continue
		}
		t.mu.Lock()
		ch, ok := t.pending[id]
		delete(t.pending, id)
		t.mu.Unlock()
		if ok {
			ch <- r
		}
	}
}

func (t *wsTransport) failAll(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	for id, ch := range t.pending {
		delete(t.pending, id)
		close(ch)
	}
}

func (t *wsTransport) Call(ctx context.Context, method string, params []any, result any) error {
	id := nextID()
	body, err := json.Marshal(request{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return err
	}
	ch := make(chan response, 1)

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return errors.New("rpc: websocket closed")
	}
	t.pending[id] = ch
	err = t.conn.WriteMessage(websocket.TextMessage, body)
	t.mu.Unlock()
	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return err
	}

	select {
	case r, ok := <-ch:
		if !ok {
			return errors.New("rpc: connection lost")
		}
		if r.Error != nil {
			return r.Error
		}
		if result == nil {
			return nil
		}
		return json.Unmarshal(r.Result, result)
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return ctx.Err()
	}
}

func (t *wsTransport) Close() error {
	err := t.conn.Close()
	<-t.done
	return err
}
