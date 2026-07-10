package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type httpTransport struct {
	url    string
	client *http.Client
}

func newHTTPTransport(url string) *httpTransport {
	return &httpTransport{url: url, client: &http.Client{Timeout: 10 * time.Second}}
}

func (t *httpTransport) Call(ctx context.Context, method string, params []any, result any) error {
	body, err := json.Marshal(request{JSONRPC: "2.0", ID: nextID(), Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	var r response
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("rpc: bad response (%s): %w", resp.Status, err)
	}
	if r.Error != nil {
		return r.Error
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(r.Result, result)
}

func (t *httpTransport) Close() error { return nil }
