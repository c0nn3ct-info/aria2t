package rpc

import (
	"context"
	"encoding/json"
	"strconv"
	"sync/atomic"
)

// Transport carries one JSON-RPC call and decodes its result.
type Transport interface {
	Call(ctx context.Context, method string, params []any, result any) error
	Close() error
}

type request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params,omitempty"`
}

type response struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"` // set on notifications only
	Result json.RawMessage `json:"result"`
	Params json.RawMessage `json:"params"`
	Error  *Error          `json:"error"`
}

var reqID atomic.Int64

func nextID() string { return "aria2t-" + strconv.FormatInt(reqID.Add(1), 10) }

// Client is a typed aria2 JSON-RPC client.
type Client struct {
	t      Transport
	secret string
	notes  chan Notification
}

// New returns a client over HTTP POST. url is the full endpoint,
// e.g. http://localhost:6800/jsonrpc.
func New(url, secret string) *Client {
	return &Client{t: newHTTPTransport(url), secret: secret, notes: make(chan Notification, 16)}
}

// Notifications delivers aria2 push events. Only the websocket transport
// produces them; over HTTP the channel stays silent.
func (c *Client) Notifications() <-chan Notification { return c.notes }

func (c *Client) Close() error { return c.t.Close() }

// call injects the token parameter and dispatches through the transport.
func (c *Client) call(ctx context.Context, method string, result any, params ...any) error {
	if c.secret != "" {
		params = append([]any{"token:" + c.secret}, params...)
	}
	return c.t.Call(ctx, method, params, result)
}

func (c *Client) TellActive(ctx context.Context) ([]Status, error) {
	var out []Status
	err := c.call(ctx, "aria2.tellActive", &out)
	return out, err
}

func (c *Client) TellWaiting(ctx context.Context, offset, num int) ([]Status, error) {
	var out []Status
	err := c.call(ctx, "aria2.tellWaiting", &out, offset, num)
	return out, err
}

func (c *Client) TellStopped(ctx context.Context, offset, num int) ([]Status, error) {
	var out []Status
	err := c.call(ctx, "aria2.tellStopped", &out, offset, num)
	return out, err
}

func (c *Client) TellStatus(ctx context.Context, gid string) (Status, error) {
	var out Status
	err := c.call(ctx, "aria2.tellStatus", &out, gid)
	return out, err
}

func (c *Client) AddURI(ctx context.Context, uris []string, opts map[string]string) (string, error) {
	var gid string
	err := c.call(ctx, "aria2.addUri", &gid, uris, orEmpty(opts))
	return gid, err
}

// AddTorrent takes base64-encoded .torrent contents.
func (c *Client) AddTorrent(ctx context.Context, b64 string, opts map[string]string) (string, error) {
	var gid string
	err := c.call(ctx, "aria2.addTorrent", &gid, b64, []string{}, orEmpty(opts))
	return gid, err
}

// AddMetalink takes base64-encoded .metalink contents.
func (c *Client) AddMetalink(ctx context.Context, b64 string, opts map[string]string) ([]string, error) {
	var gids []string
	err := c.call(ctx, "aria2.addMetalink", &gids, b64, orEmpty(opts))
	return gids, err
}

func (c *Client) Pause(ctx context.Context, gid string) error {
	return c.call(ctx, "aria2.pause", nil, gid)
}

func (c *Client) Unpause(ctx context.Context, gid string) error {
	return c.call(ctx, "aria2.unpause", nil, gid)
}

func (c *Client) Remove(ctx context.Context, gid string) error {
	return c.call(ctx, "aria2.remove", nil, gid)
}

func (c *Client) RemoveDownloadResult(ctx context.Context, gid string) error {
	return c.call(ctx, "aria2.removeDownloadResult", nil, gid)
}

// ChangePosition moves a waiting download. how is POS_SET, POS_CUR or POS_END.
func (c *Client) ChangePosition(ctx context.Context, gid string, pos int, how string) (int, error) {
	var newPos int
	err := c.call(ctx, "aria2.changePosition", &newPos, gid, pos, how)
	return newPos, err
}

func (c *Client) ChangeOption(ctx context.Context, gid string, opts map[string]string) error {
	return c.call(ctx, "aria2.changeOption", nil, gid, opts)
}

func (c *Client) GetOption(ctx context.Context, gid string) (map[string]string, error) {
	var out map[string]string
	err := c.call(ctx, "aria2.getOption", &out, gid)
	return out, err
}

func (c *Client) ChangeGlobalOption(ctx context.Context, opts map[string]string) error {
	return c.call(ctx, "aria2.changeGlobalOption", nil, opts)
}

func (c *Client) GetGlobalOption(ctx context.Context) (map[string]string, error) {
	var out map[string]string
	err := c.call(ctx, "aria2.getGlobalOption", &out)
	return out, err
}

func (c *Client) GetGlobalStat(ctx context.Context) (GlobalStat, error) {
	var out GlobalStat
	err := c.call(ctx, "aria2.getGlobalStat", &out)
	return out, err
}

func (c *Client) GetPeers(ctx context.Context, gid string) ([]Peer, error) {
	var out []Peer
	err := c.call(ctx, "aria2.getPeers", &out, gid)
	return out, err
}

// SaveSession asks aria2 to persist current downloads to its session file.
func (c *Client) SaveSession(ctx context.Context) error {
	return c.call(ctx, "aria2.saveSession", nil)
}

// Shutdown asks aria2 to exit gracefully.
func (c *Client) Shutdown(ctx context.Context) error {
	return c.call(ctx, "aria2.shutdown", nil)
}

func (c *Client) GetVersion(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	err := c.call(ctx, "aria2.getVersion", &out)
	return out.Version, err
}

func orEmpty(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
