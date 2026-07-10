package rpc

import (
	"context"
	"testing"
)

// TestTypedMethodsCover drives every typed client method that lacked coverage
// against the canned rpcServer, verifying the wire method name and decoding.
func TestTypedMethodsCover(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		result string
		method string
		do     func(t *testing.T, c *Client)
	}{
		{"TellWaiting", `[{"gid":"w1"}]`, "aria2.tellWaiting", func(t *testing.T, c *Client) {
			out, err := c.TellWaiting(ctx, 0, 5)
			if err != nil || len(out) != 1 || out[0].GID != "w1" {
				t.Fatalf("out=%v err=%v", out, err)
			}
		}},
		{"TellStopped", `[{"gid":"s1"}]`, "aria2.tellStopped", func(t *testing.T, c *Client) {
			out, err := c.TellStopped(ctx, 0, 5)
			if err != nil || len(out) != 1 || out[0].GID != "s1" {
				t.Fatalf("out=%v err=%v", out, err)
			}
		}},
		{"TellStatus", `{"gid":"g1","status":"active"}`, "aria2.tellStatus", func(t *testing.T, c *Client) {
			out, err := c.TellStatus(ctx, "g1")
			if err != nil || out.GID != "g1" {
				t.Fatalf("out=%+v err=%v", out, err)
			}
		}},
		{"AddTorrent", `"tg1"`, "aria2.addTorrent", func(t *testing.T, c *Client) {
			gid, err := c.AddTorrent(ctx, "dG9ycmVudA==", map[string]string{"dir": "/dl"})
			if err != nil || gid != "tg1" {
				t.Fatalf("gid=%q err=%v", gid, err)
			}
		}},
		{"AddMetalink", `["m1","m2"]`, "aria2.addMetalink", func(t *testing.T, c *Client) {
			gids, err := c.AddMetalink(ctx, "bWV0YQ==", nil)
			if err != nil || len(gids) != 2 || gids[0] != "m1" {
				t.Fatalf("gids=%v err=%v", gids, err)
			}
		}},
		{"Pause", `"g1"`, "aria2.pause", func(t *testing.T, c *Client) {
			if err := c.Pause(ctx, "g1"); err != nil {
				t.Fatal(err)
			}
		}},
		{"Unpause", `"g1"`, "aria2.unpause", func(t *testing.T, c *Client) {
			if err := c.Unpause(ctx, "g1"); err != nil {
				t.Fatal(err)
			}
		}},
		{"Remove", `"g1"`, "aria2.remove", func(t *testing.T, c *Client) {
			if err := c.Remove(ctx, "g1"); err != nil {
				t.Fatal(err)
			}
		}},
		{"RemoveDownloadResult", `"OK"`, "aria2.removeDownloadResult", func(t *testing.T, c *Client) {
			if err := c.RemoveDownloadResult(ctx, "g1"); err != nil {
				t.Fatal(err)
			}
		}},
		{"ChangeOption", `"OK"`, "aria2.changeOption", func(t *testing.T, c *Client) {
			if err := c.ChangeOption(ctx, "g1", map[string]string{"max-download-limit": "1K"}); err != nil {
				t.Fatal(err)
			}
		}},
		{"GetOption", `{"dir":"/dl"}`, "aria2.getOption", func(t *testing.T, c *Client) {
			out, err := c.GetOption(ctx, "g1")
			if err != nil || out["dir"] != "/dl" {
				t.Fatalf("out=%v err=%v", out, err)
			}
		}},
		{"ChangeGlobalOption", `"OK"`, "aria2.changeGlobalOption", func(t *testing.T, c *Client) {
			if err := c.ChangeGlobalOption(ctx, map[string]string{"max-overall-download-limit": "0"}); err != nil {
				t.Fatal(err)
			}
		}},
		{"GetGlobalOption", `{"dir":"/dl"}`, "aria2.getGlobalOption", func(t *testing.T, c *Client) {
			out, err := c.GetGlobalOption(ctx)
			if err != nil || out["dir"] != "/dl" {
				t.Fatalf("out=%v err=%v", out, err)
			}
		}},
		{"GetGlobalStat", `{"downloadSpeed":"1024","numActive":"2"}`, "aria2.getGlobalStat", func(t *testing.T, c *Client) {
			out, err := c.GetGlobalStat(ctx)
			if err != nil || out.DownSpeed() != 1024 || out.Active() != 2 {
				t.Fatalf("out=%+v err=%v", out, err)
			}
		}},
		{"GetPeers", `[{"ip":"1.2.3.4","downloadSpeed":"7"}]`, "aria2.getPeers", func(t *testing.T, c *Client) {
			out, err := c.GetPeers(ctx, "g1")
			if err != nil || len(out) != 1 || out[0].IP != "1.2.3.4" {
				t.Fatalf("out=%v err=%v", out, err)
			}
		}},
		{"SaveSession", `"OK"`, "aria2.saveSession", func(t *testing.T, c *Client) {
			if err := c.SaveSession(ctx); err != nil {
				t.Fatal(err)
			}
		}},
		{"Shutdown", `"OK"`, "aria2.shutdown", func(t *testing.T, c *Client) {
			if err := c.Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, last := rpcServer(t, tc.result)
			c := New(srv.URL, "s")
			tc.do(t, c)
			if last.Method != tc.method {
				t.Fatalf("method = %q, want %q", last.Method, tc.method)
			}
		})
	}
}

func TestOrEmpty(t *testing.T) {
	if m := orEmpty(nil); m == nil || len(m) != 0 {
		t.Fatalf("orEmpty(nil) = %v", m)
	}
	in := map[string]string{"k": "v"}
	if m := orEmpty(in); len(m) != 1 || m["k"] != "v" {
		t.Fatalf("orEmpty(non-nil) = %v", m)
	}
}
