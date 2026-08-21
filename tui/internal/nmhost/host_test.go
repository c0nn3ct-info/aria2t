package nmhost

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"aria2t/internal/daemon"
)

// stubState swaps readState for a stub and restores it on cleanup.
func stubState(t *testing.T, port int, secret string, ok bool) {
	t.Helper()
	orig := readState
	readState = func(string) (int, int, string, bool) { return 1, port, secret, ok }
	t.Cleanup(func() { readState = orig })
}

// stubProbe swaps probeRPC for a stub returning err.
func stubProbe(t *testing.T, err error) {
	t.Helper()
	orig := probeRPC
	probeRPC = func(int, string) error { return err }
	t.Cleanup(func() { probeRPC = orig })
}

// stubGoos swaps the reveal platform switch.
func stubGoos(t *testing.T, g string) {
	t.Helper()
	orig := goos
	goos = g
	t.Cleanup(func() { goos = orig })
}

type openCall struct {
	bin  string
	args []string
}

// stubOpen records the execOpen invocation and returns err from it.
func stubOpen(t *testing.T, err error) *openCall {
	t.Helper()
	rec := &openCall{}
	orig := execOpen
	execOpen = func(bin string, args ...string) error {
		rec.bin = bin
		rec.args = args
		return err
	}
	t.Cleanup(func() { execOpen = orig })
	return rec
}

// decodeAcks reads every frame out of buf as a response.
func decodeAcks(t *testing.T, buf *bytes.Buffer) []response {
	t.Helper()
	var acks []response
	for buf.Len() > 0 {
		raw, err := readFrame(buf)
		if err != nil {
			t.Fatal(err)
		}
		var r response
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatal(err)
		}
		acks = append(acks, r)
	}
	return acks
}

func TestRunEndToEnd(t *testing.T) {
	origCfg := configPath
	configPath = func() string { return filepath.Join(t.TempDir(), "config.json") }
	t.Cleanup(func() { configPath = origCfg })

	var in bytes.Buffer
	frame(&in, `{"id":"1","type":"hello"}`)
	frame(&in, `{"id":"2","type":"bogus"}`)
	var out bytes.Buffer
	if err := Run(&in, &out, "v9.9"); err != nil {
		t.Fatal(err)
	}
	acks := decodeAcks(t, &out)
	if len(acks) != 2 {
		t.Fatalf("got %d acks, want 2", len(acks))
	}
	if acks[0].ID != "1" || acks[0].Type != "ack" || !acks[0].OK {
		t.Fatalf("hello ack = %+v", acks[0])
	}
	data := acks[0].Data.(map[string]any)
	if data["version"] != "v9.9" || data["platform"] != runtime.GOOS+"-"+runtime.GOARCH {
		t.Fatalf("hello data = %v", data)
	}
	if acks[1].OK || acks[1].Error != "unknown type: bogus" {
		t.Fatalf("unknown ack = %+v", acks[1])
	}
}

func TestServeBadRequestJSON(t *testing.T) {
	var in bytes.Buffer
	frame(&in, `{nope`)
	var out bytes.Buffer
	h := &host{out: bufio.NewWriter(&out)}
	if err := h.serve(&in); err != nil {
		t.Fatal(err)
	}
	acks := decodeAcks(t, &out)
	if len(acks) != 1 || acks[0].OK || !strings.Contains(acks[0].Error, "bad request") {
		t.Fatalf("acks = %+v", acks)
	}
}

func TestServeReadError(t *testing.T) {
	// A header claiming a frame beyond the cap is a fatal protocol error.
	var hdr [4]byte
	hdr[3] = 0xff
	h := &host{out: bufio.NewWriter(io.Discard)}
	err := h.serve(bytes.NewReader(hdr[:]))
	if err == nil || !strings.Contains(err.Error(), "frame too large") {
		t.Fatalf("err = %v", err)
	}
}

func TestServeWriteError(t *testing.T) {
	var in bytes.Buffer
	frame(&in, `{"id":"1","type":"hello"}`)
	// A tiny buffer forces the ack write through to the failing sink.
	h := &host{out: bufio.NewWriterSize(&failWriter{}, 8)}
	if err := h.serve(&in); err == nil {
		t.Fatal("want write error")
	}
}

func TestWriteFlushError(t *testing.T) {
	// A large buffer accepts the whole frame; only Flush hits the sink.
	h := &host{out: bufio.NewWriterSize(&failWriter{}, 1<<16)}
	if err := h.write(response{ID: "1", Type: "ack", OK: true}); err == nil {
		t.Fatal("want flush error")
	}
}

func TestHandleStatusNoState(t *testing.T) {
	stubState(t, 0, "", false)
	h := &host{}
	resp := h.handle(request{ID: "s", Type: "status"})
	if !resp.OK || resp.Data.(daemonData) != (daemonData{}) {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestHandleStatusProbeFails(t *testing.T) {
	stubState(t, 6800, "sec", true)
	stubProbe(t, errors.New("connection refused"))
	h := &host{}
	if d := h.handle(request{Type: "status"}).Data.(daemonData); d.Running {
		t.Fatalf("dead daemon reported running: %+v", d)
	}
}

func TestHandleStatusRunning(t *testing.T) {
	stubState(t, 6800, "sec", true)
	stubProbe(t, nil)
	h := &host{}
	d := h.handle(request{Type: "status"}).Data.(daemonData)
	if !d.Running || d.Port != 6800 || d.Secret != "sec" {
		t.Fatalf("d = %+v", d)
	}
}

func TestEnsureAlreadyRunning(t *testing.T) {
	stubState(t, 6800, "sec", true)
	stubProbe(t, nil)
	orig := startDaemon
	startDaemon = func(daemon.Options) (*daemon.Daemon, error) {
		t.Fatal("ensure must not spawn when the daemon answers")
		return nil, nil
	}
	t.Cleanup(func() { startDaemon = orig })
	h := &host{}
	d := h.handle(request{Type: "ensure"}).Data.(daemonData)
	if !d.Running || d.Port != 6800 || d.Secret != "sec" {
		t.Fatalf("d = %+v", d)
	}
	if h.daemon != nil {
		t.Fatal("no handle must be stored when nothing was spawned")
	}
}

func TestEnsureSpawns(t *testing.T) {
	stubState(t, 0, "", false)
	want := &daemon.Daemon{Port: 42, Secret: "z"}
	var got daemon.Options
	orig := startDaemon
	startDaemon = func(o daemon.Options) (*daemon.Daemon, error) {
		got = o
		return want, nil
	}
	t.Cleanup(func() { startDaemon = orig })
	h := &host{dataDir: "/dd", dir: "/dl"}
	resp := h.handle(request{ID: "e", Type: "ensure"})
	if !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	if d := resp.Data.(daemonData); !d.Running || d.Port != 42 || d.Secret != "z" {
		t.Fatalf("d = %+v", d)
	}
	if !got.Detach || got.DataDir != "/dd" || got.Dir != "/dl" {
		t.Fatalf("spawn opts = %+v", got)
	}
	if h.daemon != want {
		t.Fatal("ensure must keep the spawned handle for the host's lifetime")
	}
}

func TestEnsureSpawnError(t *testing.T) {
	stubState(t, 0, "", false)
	orig := startDaemon
	startDaemon = func(daemon.Options) (*daemon.Daemon, error) {
		return nil, errors.New("aria2c not found")
	}
	t.Cleanup(func() { startDaemon = orig })
	h := &host{}
	resp := h.handle(request{Type: "ensure"})
	if resp.OK || !strings.Contains(resp.Error, "aria2c not found") {
		t.Fatalf("resp = %+v", resp)
	}
}

// existingFile returns the path of a real file, so reveal takes its
// select-the-file branch instead of the missing-target fallback.
func existingFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "file.iso")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRevealDarwin(t *testing.T) {
	stubGoos(t, "darwin")
	rec := stubOpen(t, nil)
	path := existingFile(t)
	h := &host{}
	if resp := h.handle(request{Type: "reveal", Path: path}); !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.bin != "open" || len(rec.args) != 2 || rec.args[0] != "-R" || rec.args[1] != path {
		t.Fatalf("call = %+v", rec)
	}
}

func TestRevealWindows(t *testing.T) {
	stubGoos(t, "windows")
	rec := stubOpen(t, nil)
	path := existingFile(t)
	if err := reveal(path); err != nil {
		t.Fatal(err)
	}
	if rec.bin != "explorer" || len(rec.args) != 1 || rec.args[0] != "/select,"+path {
		t.Fatalf("call = %+v", rec)
	}
}

func TestRevealLinux(t *testing.T) {
	stubGoos(t, "linux")
	rec := stubOpen(t, nil)
	path := existingFile(t)
	if err := reveal(path); err != nil {
		t.Fatal(err)
	}
	if rec.bin != "xdg-open" || len(rec.args) != 1 || rec.args[0] != filepath.Dir(path) {
		t.Fatalf("call = %+v", rec)
	}
}

func TestRevealError(t *testing.T) {
	stubGoos(t, "darwin")
	stubOpen(t, errors.New("no such app"))
	h := &host{}
	resp := h.handle(request{Type: "reveal", Path: existingFile(t)})
	if resp.OK || !strings.Contains(resp.Error, "no such app") {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestRevealMissingPath(t *testing.T) {
	rec := stubOpen(t, nil)
	h := &host{}
	resp := h.handle(request{Type: "reveal"})
	if resp.OK || !strings.Contains(resp.Error, "missing path") {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.bin != "" {
		t.Fatal("nothing must be opened without a path")
	}
}

// A finished download whose file was moved or deleted still opens the folder
// it was downloaded into, rather than failing with the launcher's exit status.
func TestRevealFallsBackToNearestExistingDir(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(dir, "sub", "deeper", "gone.iso")
	for _, tc := range []struct{ goos, bin string }{
		{"darwin", "open"},
		{"windows", "explorer"},
		{"linux", "xdg-open"},
	} {
		t.Run(tc.goos, func(t *testing.T) {
			stubGoos(t, tc.goos)
			rec := stubOpen(t, nil)
			if err := reveal(gone); err != nil {
				t.Fatal(err)
			}
			if rec.bin != tc.bin || len(rec.args) != 1 || rec.args[0] != dir {
				t.Fatalf("call = %+v, want %s %s", rec, tc.bin, dir)
			}
		})
	}
}

func TestRevealWithNoSurvivingAncestor(t *testing.T) {
	orig := statPath
	statPath = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	t.Cleanup(func() { statPath = orig })
	rec := stubOpen(t, nil)

	err := reveal("/dl/gone.iso")
	if err == nil || !strings.Contains(err.Error(), "no longer on disk") {
		t.Fatalf("err = %v, want no longer on disk", err)
	}
	if rec.bin != "" {
		t.Fatal("nothing must be opened when no ancestor exists")
	}
}

// The root always exists, so an unbounded walk up would "succeed" by opening a
// window on "/" — useless, and it must stay an error.
func TestRevealNeverFallsBackToTheRoot(t *testing.T) {
	stubGoos(t, "darwin")
	rec := stubOpen(t, nil)
	err := reveal("/nonexistent-root-xyz/a/b.iso")
	if err == nil || !strings.Contains(err.Error(), "no longer on disk") {
		t.Fatalf("err = %v, want no longer on disk", err)
	}
	if rec.bin != "" {
		t.Fatalf("opened %+v, want nothing", rec)
	}
}

// runLauncher must surface what the launcher printed — "exit status 1" alone
// tells the extension nothing.
func TestRunLauncherIncludesOutput(t *testing.T) {
	err := runLauncher("sh", "-c", "echo the file does not exist >&2; exit 1")
	if err == nil || !strings.Contains(err.Error(), "the file does not exist") {
		t.Fatalf("err = %v, want the launcher's message", err)
	}
	if err := runLauncher("true"); err != nil {
		t.Fatalf("runLauncher(true) = %v", err)
	}
	if err := runLauncher("sh", "-c", "exit 3"); err == nil ||
		!strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("err = %v, want the bare exit status", err)
	}
}

type cleanCall struct {
	dataDir string
	gid     string
	files   []string
	called  bool
}

// stubClean records the control.Clean invocation and returns err from it.
func stubClean(t *testing.T, err error) *cleanCall {
	t.Helper()
	rec := &cleanCall{}
	orig := cleanControlFiles
	cleanControlFiles = func(dataDir, gid string, files []string) error {
		rec.dataDir, rec.gid, rec.files, rec.called = dataDir, gid, files, true
		return err
	}
	t.Cleanup(func() { cleanControlFiles = orig })
	return rec
}

func TestCleanControl(t *testing.T) {
	rec := stubClean(t, nil)
	h := &host{dataDir: "/state"}
	resp := h.handle(request{
		Type:  "clean-control",
		GID:   "abc",
		Dir:   "/dl",
		Name:  "Movie",
		Files: []string{"/dl/Movie/a.mkv"},
	})
	if !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.dataDir != "/state" || rec.gid != "abc" {
		t.Fatalf("call = %+v", rec)
	}
	want := []string{"/dl/Movie/a.mkv.aria2", "/dl/Movie.aria2"}
	if len(rec.files) != len(want) {
		t.Fatalf("files = %v, want %v", rec.files, want)
	}
	for i := range want {
		if rec.files[i] != want[i] {
			t.Fatalf("files = %v, want %v", rec.files, want)
		}
	}
}

func TestCleanControlMissingGID(t *testing.T) {
	rec := stubClean(t, nil)
	h := &host{}
	resp := h.handle(request{Type: "clean-control", Files: []string{"/dl/a.iso"}})
	if resp.OK || !strings.Contains(resp.Error, "missing gid") {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.called {
		t.Fatal("nothing must be deleted without a gid")
	}
}

func TestCleanControlOptedOut(t *testing.T) {
	rec := stubClean(t, nil)
	h := &host{keepControl: true}
	if resp := h.handle(request{Type: "clean-control", GID: "abc"}); !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.called {
		t.Fatal("keepControl must suppress the deletion")
	}
}

func TestCleanControlError(t *testing.T) {
	stubClean(t, errors.New("permission denied"))
	h := &host{}
	resp := h.handle(request{Type: "clean-control", GID: "abc", Files: []string{"/dl/a.iso"}})
	if resp.OK || !strings.Contains(resp.Error, "permission denied") {
		t.Fatalf("resp = %+v", resp)
	}
}

// TestExecOpenDefault exercises the real execOpen seam with a no-op binary.
func TestExecOpenDefault(t *testing.T) {
	if err := execOpen("true"); err != nil {
		t.Fatalf("execOpen(true) = %v", err)
	}
}

// portOf extracts the TCP port from an httptest URL.
func portOf(t *testing.T, url string) int {
	t.Helper()
	port, err := strconv.Atoi(url[strings.LastIndex(url, ":")+1:])
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func TestDefaultProbeSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"version":"1.37.0"}}`))
	}))
	defer srv.Close()
	if err := defaultProbe(portOf(t, srv.URL), "s"); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultProbeFailure(t *testing.T) {
	// Grab a port and close it so nothing listens there.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	if err := defaultProbe(port, "s"); err == nil {
		t.Fatal("want error probing a closed port")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got := expandHome("~/Downloads"); got != filepath.Join(home, "Downloads") {
		t.Fatalf("got %q", got)
	}
	if got := expandHome("/abs/path"); got != "/abs/path" {
		t.Fatalf("got %q", got)
	}
	orig := userHome
	userHome = func() (string, error) { return "", errors.New("no home") }
	t.Cleanup(func() { userHome = orig })
	if got := expandHome("~/x"); got != "~/x" {
		t.Fatalf("got %q", got)
	}
}

// deleteCall records what the delete seams were asked to do.
type deleteCall struct {
	dataDir, gid, root string
	targets            []string
	called             bool
}

// stubDelete swaps both delete seams: Targets keeps its real containment rules
// (that is the part worth exercising through the verb), while Delete only
// records, so no test can reach the disk.
func stubDelete(t *testing.T, err error) *deleteCall {
	t.Helper()
	rec := &deleteCall{}
	orig := deleteDownloadFiles
	deleteDownloadFiles = func(dataDir, gid, root string, targets []string) error {
		rec.dataDir, rec.gid, rec.root, rec.targets, rec.called = dataDir, gid, root, targets, true
		return err
	}
	t.Cleanup(func() { deleteDownloadFiles = orig })
	return rec
}

func TestDeleteFiles(t *testing.T) {
	rec := stubDelete(t, nil)
	h := &host{dataDir: "/state", dir: "/dl"}
	resp := h.handle(request{
		Type:  "delete-files",
		GID:   "abc",
		Dir:   "/dl/Movie",
		Files: []string{"/dl/Movie/a.mkv", "[METADATA]x"},
	})
	if !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.dataDir != "/state" || rec.gid != "abc" || rec.root != "/dl" {
		t.Fatalf("call = %+v", rec)
	}
	// The placeholder is dropped, the real file kept — and its control file with
	// it: aria2 never lists that as a file of the download, so deleting "the
	// download's files" used to leave it behind.
	want := []string{"/dl/Movie/a.mkv", "/dl/Movie/a.mkv.aria2"}
	if !reflect.DeepEqual(rec.targets, want) {
		t.Fatalf("targets = %v, want %v", rec.targets, want)
	}
}

// A torrent leaves two kinds of bookkeeping behind, and the user reported both
// still sitting in the download folder after a removal: the control file named
// after the torrent, and the <infohash>.torrent the daemon saves because it runs
// with --bt-save-metadata.
func TestDeleteFilesTakesTheTorrentBookkeepingWithIt(t *testing.T) {
	rec := stubDelete(t, nil)
	h := &host{dataDir: "/state", dir: "/dl"}
	resp := h.handle(request{
		Type:     "delete-files",
		GID:      "abc",
		Dir:      "/dl",
		Name:     "Show",
		InfoHash: "1E838BF16C32054F7D7A02197A20DEB0545E74B7",
		Files:    []string{"/dl/Show/a.mkv"},
	})
	if !resp.OK {
		t.Fatalf("resp = %+v", resp)
	}
	want := []string{
		"/dl/Show/a.mkv",
		"/dl/Show/a.mkv.aria2",
		"/dl/Show.aria2",
		// aria2 writes the hash lower-cased whatever case it reports.
		"/dl/1e838bf16c32054f7d7a02197a20deb0545e74b7.torrent",
	}
	if !reflect.DeepEqual(rec.targets, want) {
		t.Fatalf("targets = %v, want %v", rec.targets, want)
	}
}

// The containment rule covers the derived paths too, not just the listed files.
func TestDeleteFilesRefusesBookkeepingOutsideTheDownloadDir(t *testing.T) {
	rec := stubDelete(t, nil)
	h := &host{dataDir: "/state", dir: "/dl"}
	resp := h.handle(request{
		Type: "delete-files",
		GID:  "abc",
		Dir:  "/dl",
		Name: "../../etc/shadow",
	})
	if resp.OK || !strings.Contains(resp.Error, "outside") {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.called {
		t.Fatal("nothing must be deleted once a path escapes")
	}
}

func TestDeleteFilesMissingGID(t *testing.T) {
	rec := stubDelete(t, nil)
	h := &host{dir: "/dl"}
	resp := h.handle(request{Type: "delete-files", Dir: "/dl", Files: []string{"/dl/a.iso"}})
	if resp.OK || !strings.Contains(resp.Error, "missing gid") {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.called {
		t.Fatal("nothing must be deleted without a gid")
	}
}

// The refusal that matters: a path the host cannot place inside the download
// directory it was configured with never reaches the disk.
func TestDeleteFilesRefusesOutsideTheDownloadDir(t *testing.T) {
	rec := stubDelete(t, nil)
	h := &host{dataDir: "/state", dir: "/dl"}
	resp := h.handle(request{
		Type:  "delete-files",
		GID:   "abc",
		Dir:   "/dl",
		Files: []string{"/dl/../etc/passwd"},
	})
	if resp.OK || !strings.Contains(resp.Error, "outside") {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.called {
		t.Fatal("a refused path must not reach the deletion")
	}
}

func TestDeleteFilesRefusesADirOutsideTheRoot(t *testing.T) {
	rec := stubDelete(t, nil)
	h := &host{dataDir: "/state", dir: "/dl"}
	resp := h.handle(request{Type: "delete-files", GID: "abc", Dir: "/somewhere", Files: nil})
	if resp.OK || !strings.Contains(resp.Error, "outside") {
		t.Fatalf("resp = %+v", resp)
	}
	if rec.called {
		t.Fatal("a refused dir must not reach the deletion")
	}
}

func TestDeleteFilesError(t *testing.T) {
	stubDelete(t, errors.New("permission denied"))
	h := &host{dataDir: "/state", dir: "/dl"}
	resp := h.handle(request{
		Type:  "delete-files",
		GID:   "abc",
		Dir:   "/dl",
		Files: []string{"/dl/a.iso"},
	})
	if resp.OK || !strings.Contains(resp.Error, "permission denied") {
		t.Fatalf("resp = %+v", resp)
	}
}
