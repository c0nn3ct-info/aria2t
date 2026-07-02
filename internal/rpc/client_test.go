package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// rpcServer records the last request and answers with a canned result.
func rpcServer(t *testing.T, result string) (*httptest.Server, *request) {
	t.Helper()
	var last request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&last); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":"` + last.ID + `","result":` + result + `}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &last
}

func TestTokenInjection(t *testing.T) {
	srv, last := rpcServer(t, `"ok-gid"`)
	c := New(srv.URL, "s3cret")
	if _, err := c.AddURI(context.Background(), []string{"http://x/f.iso"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.Method != "aria2.addUri" {
		t.Fatalf("method = %q", last.Method)
	}
	if len(last.Params) < 1 || last.Params[0] != "token:s3cret" {
		t.Fatalf("token not first param: %v", last.Params)
	}
}

func TestNoTokenWhenSecretEmpty(t *testing.T) {
	srv, last := rpcServer(t, `[]`)
	c := New(srv.URL, "")
	if _, err := c.TellActive(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(last.Params) != 0 {
		t.Fatalf("expected no params, got %v", last.Params)
	}
}

func TestTellActiveDecodes(t *testing.T) {
	srv, _ := rpcServer(t, `[{"gid":"2089b05ecca3d829","status":"active","totalLength":"658505728","completedLength":"408273485","downloadSpeed":"13002342","uploadSpeed":"0","numPieces":"10240","files":[{"path":"/dl/debian-13.1.0-amd64-netinst.iso","length":"658505728","selected":"true"}]}]`)
	c := New(srv.URL, "")
	got, err := c.TellActive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	s := got[0]
	if s.GID != "2089b05ecca3d829" || s.Total() != 658505728 || s.DownSpeed() != 13002342 {
		t.Fatalf("bad decode: %+v", s)
	}
	if p := s.Progress(); p < 0.61 || p > 0.63 {
		t.Fatalf("progress = %f", p)
	}
}

func TestRPCErrorSurfaced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"jsonrpc":"2.0","id":"1","error":{"code":1,"message":"Unauthorized"}}`))
	}))
	defer srv.Close()
	c := New(srv.URL, "wrong")
	_, err := c.TellActive(context.Background())
	rpcErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("want *Error, got %T: %v", err, err)
	}
	if rpcErr.Code != 1 || rpcErr.Message != "Unauthorized" {
		t.Fatalf("bad error: %+v", rpcErr)
	}
}

func TestStatusNumericGettersDegrade(t *testing.T) {
	s := Status{TotalLength: "garbage", CompletedLength: ""}
	if s.Total() != 0 || s.Completed() != 0 || s.Progress() != 0 {
		t.Fatal("malformed numbers must degrade to zero")
	}
}

func TestStatusName(t *testing.T) {
	cases := []struct {
		name string
		in   Status
		want string
	}{
		{"torrent", Status{BitTorrent: &BTInfo{Info: struct {
			Name string `json:"name"`
		}{Name: "ubuntu.iso"}}}, "ubuntu.iso"},
		{"file path", Status{Files: []File{{Path: "/dl/a.tar.xz"}}}, "a.tar.xz"},
		{"uri fallback", Status{Files: []File{{URIs: []URI{{URI: "https://x.org/b.iso"}}}}}, "b.iso"},
		{"gid fallback", Status{GID: "abc123"}, "abc123"},
	}
	for _, tc := range cases {
		if got := tc.in.Name(); got != tc.want {
			t.Errorf("%s: Name() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestChangePosition(t *testing.T) {
	srv, last := rpcServer(t, `2`)
	c := New(srv.URL, "t")
	pos, err := c.ChangePosition(context.Background(), "g1", 2, "POS_SET")
	if err != nil {
		t.Fatal(err)
	}
	if pos != 2 {
		t.Fatalf("pos = %d", pos)
	}
	if last.Method != "aria2.changePosition" {
		t.Fatalf("method = %q", last.Method)
	}
	// params: token, gid, pos, how
	if len(last.Params) != 4 || last.Params[3] != "POS_SET" {
		t.Fatalf("params = %v", last.Params)
	}
}
