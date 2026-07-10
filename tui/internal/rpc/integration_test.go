package rpc

// Integration test against a live aria2 daemon. Skipped unless
// ARIA2T_SMOKE_URL is set, e.g.:
//
//	aria2c --enable-rpc --rpc-listen-port=6899 --rpc-secret=smoketest --daemon
//	ARIA2T_SMOKE_URL=localhost:6899 ARIA2T_SMOKE_SECRET=smoketest go test ./internal/rpc/ -run Integration -v

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestIntegrationLiveDaemon(t *testing.T) {
	host := os.Getenv("ARIA2T_SMOKE_URL")
	if host == "" {
		t.Skip("ARIA2T_SMOKE_URL not set")
	}
	secret := os.Getenv("ARIA2T_SMOKE_SECRET")
	ctx := context.Background()

	transports := map[string]func() (*Client, error){
		"http": func() (*Client, error) { return New("http://"+host+"/jsonrpc", secret), nil },
		"ws":   func() (*Client, error) { return NewWS("ws://"+host+"/jsonrpc", secret) },
	}
	for name, mk := range transports {
		t.Run(name, func(t *testing.T) {
			c, err := mk()
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer c.Close()

			v, err := c.GetVersion(ctx)
			if err != nil || v == "" {
				t.Fatalf("getVersion: %q %v", v, err)
			}

			gid, err := c.AddURI(ctx, []string{"http://localhost:1/never.bin"}, map[string]string{"pause": "true"})
			if err != nil {
				t.Fatalf("addUri: %v", err)
			}
			gid2, err := c.AddURI(ctx, []string{"http://localhost:1/never2.bin"}, map[string]string{"pause": "true"})
			if err != nil {
				t.Fatalf("addUri2: %v", err)
			}
			t.Cleanup(func() {
				for _, g := range []string{gid, gid2} {
					_ = c.Remove(ctx, g)
					_ = c.RemoveDownloadResult(ctx, g)
				}
			})

			waiting, err := c.TellWaiting(ctx, 0, 100)
			if err != nil || len(waiting) < 2 {
				t.Fatalf("tellWaiting: n=%d err=%v", len(waiting), err)
			}

			if _, err := c.ChangePosition(ctx, gid2, 0, "POS_SET"); err != nil {
				t.Fatalf("changePosition: %v", err)
			}

			if err := c.ChangeOption(ctx, gid, map[string]string{"max-download-limit": "5M"}); err != nil {
				t.Fatalf("changeOption: %v", err)
			}
			opts, err := c.GetOption(ctx, gid)
			if err != nil {
				t.Fatalf("getOption: %v", err)
			}
			if got := opts["max-download-limit"]; got != "5242880" && got != "5M" {
				t.Fatalf("limit roundtrip: %q", got)
			}

			if _, err := c.GetGlobalStat(ctx); err != nil {
				t.Fatalf("getGlobalStat: %v", err)
			}
			if err := c.ChangeGlobalOption(ctx, map[string]string{"max-overall-download-limit": "0"}); err != nil {
				t.Fatalf("changeGlobalOption: %v", err)
			}
			st, err := c.TellStatus(ctx, gid)
			if err != nil || st.Status != "paused" {
				t.Fatalf("tellStatus: %+v %v", st, err)
			}
			fmt.Println("live daemon", name, "ok, version", v)
		})
	}
}
