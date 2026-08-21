package nmhost

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"aria2t/internal/config"
	"aria2t/internal/control"
	"aria2t/internal/daemon"
	"aria2t/internal/rpc"
)

// request is one extension→host message. Verb-specific fields are inlined:
// reveal carries a path, clean-control a gid plus the download's file paths.
type request struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	GID  string `json:"gid,omitempty"`
	Dir  string `json:"dir,omitempty"`
	Name string `json:"name,omitempty"` // torrent name, for its control file
	// InfoHash names the <infohash>.torrent aria2 saves under --bt-save-metadata.
	InfoHash string   `json:"infoHash,omitempty"`
	Files    []string `json:"files,omitempty"`
}

// response is the ack for one request.
type response struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // always "ack"
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Data  any    `json:"data,omitempty"`
}

// helloData answers hello: the host build version and its platform.
type helloData struct {
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

// daemonData answers status/ensure: the RPC coordinates of the managed
// daemon, or running:false with zero values when none answers.
type daemonData struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	Secret  string `json:"secret"`
}

// Injection seams for tests (the repo's standard pattern over direct
// OS/daemon calls).
var (
	configPath  = config.DefaultPath
	startDaemon = daemon.Start
	readState   = daemon.State
	probeRPC    = defaultProbe
	execOpen    = runLauncher
	userHome    = os.UserHomeDir
	statPath    = os.Stat
	goos        = runtime.GOOS

	cleanControlFiles   = control.Clean
	deleteTargets       = control.Targets
	deleteDownloadFiles = control.Delete
)

// runLauncher runs a file-manager command, folding whatever it printed into
// the error. Without this the extension only ever sees "exit status 1", which
// says nothing about why the launcher refused.
func runLauncher(bin string, args ...string) error {
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err == nil {
		return nil
	}
	if msg := strings.TrimSpace(string(out)); msg != "" {
		return fmt.Errorf("%s: %w: %s", bin, err, msg)
	}
	return fmt.Errorf("%s: %w", bin, err)
}

// defaultProbe checks a daemon answers getVersion on the recorded port with
// the recorded secret.
func defaultProbe(port int, secret string) error {
	c := rpc.New(fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", port), secret)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := c.GetVersion(ctx)
	return err
}

// host serves one extension connection.
type host struct {
	version     string
	dataDir     string // the TUI's stable daemon dir (session.txt, daemon.json)
	dir         string // download dir from the user's config
	keepControl bool   // user opted out of deleting .aria2 control files
	mu          sync.Mutex
	out         *bufio.Writer
	// daemon holds the handle of a child spawned by ensure for the host's
	// lifetime. It is never stopped: the daemon is detached and must outlive
	// the host, which exits whenever the browser disconnects.
	daemon *daemon.Daemon
}

// Run serves native-messaging requests from r, acking to w, until EOF.
// version is the build-stamped host version reported by hello.
func Run(r io.Reader, w io.Writer, version string) error {
	cfgPath := configPath()
	cfg, _ := config.Load(cfgPath) // a broken config falls back to defaults
	h := &host{
		version:     version,
		dataDir:     filepath.Join(filepath.Dir(cfgPath), "daemon"), // same stable dir the TUI uses
		dir:         expandHome(cfg.Dir),
		keepControl: !cfg.CleanControl(),
		out:         bufio.NewWriter(w),
	}
	return h.serve(bufio.NewReader(r))
}

// serve is the read → dispatch → ack loop. EOF (the browser closed the
// pipe) returns nil; any other framing or write failure is fatal.
func (h *host) serve(r io.Reader) error {
	for {
		raw, err := readFrame(r)
		if errors.Is(err, io.EOF) {
			return nil // extension disconnected; the daemon keeps running
		}
		if err != nil {
			return err
		}
		var resp response
		var req request
		if jerr := json.Unmarshal(raw, &req); jerr != nil {
			resp = response{Type: "ack", OK: false, Error: "bad request: " + jerr.Error()}
		} else {
			resp = h.handle(req)
		}
		if err := h.write(resp); err != nil {
			return err
		}
	}
}

// write marshals and frames one ack; the mutex serializes writers.
func (h *host) write(resp response) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	raw, _ := json.Marshal(resp) // cannot fail for these types
	if err := writeFrame(h.out, raw); err != nil {
		return err
	}
	return h.out.Flush()
}

// handle dispatches one request to its verb.
func (h *host) handle(req request) response {
	ack := response{ID: req.ID, Type: "ack", OK: true}
	switch req.Type {
	case "hello":
		ack.Data = helloData{Version: h.version, Platform: runtime.GOOS + "-" + runtime.GOARCH}
	case "status":
		ack.Data = h.status()
	case "ensure":
		st, err := h.ensure()
		if err != nil {
			return response{ID: req.ID, Type: "ack", OK: false, Error: err.Error()}
		}
		ack.Data = st
	case "reveal":
		if err := reveal(req.Path); err != nil {
			return response{ID: req.ID, Type: "ack", OK: false, Error: err.Error()}
		}
	case "clean-control":
		if err := h.cleanControl(req); err != nil {
			return response{ID: req.ID, Type: "ack", OK: false, Error: err.Error()}
		}
	case "delete-files":
		if err := h.deleteFiles(req); err != nil {
			return response{ID: req.ID, Type: "ack", OK: false, Error: err.Error()}
		}
	default:
		return response{ID: req.ID, Type: "ack", OK: false, Error: "unknown type: " + req.Type}
	}
	return ack
}

// status probes the daemon recorded in daemon.json; a missing state file or
// a dead RPC port is running:false with zero values.
func (h *host) status() daemonData {
	_, port, secret, ok := readState(h.dataDir)
	if !ok || probeRPC(port, secret) != nil {
		return daemonData{}
	}
	return daemonData{Running: true, Port: port, Secret: secret}
}

// ensure returns a live daemon's coordinates, spawning one when none
// answers. Start reaps stale daemons and resumes the saved session; Detach
// puts the child in its own session so it survives this process's exit.
func (h *host) ensure() (daemonData, error) {
	if st := h.status(); st.Running {
		return st, nil
	}
	d, err := startDaemon(daemon.Options{Dir: h.dir, DataDir: h.dataDir, Detach: true})
	if err != nil {
		return daemonData{}, err
	}
	h.daemon = d // keep the handle alive; never stopped — see host.daemon
	return daemonData{Running: true, Port: d.Port, Secret: d.Secret}, nil
}

// cleanControl deletes the .aria2 control file a finished download left
// behind and records the gid so the next daemon start drops it from the saved
// session (see internal/control). The extension gates this on its own
// setting, but the host still honours the TUI's opt-out so one config can't
// be silently overridden by the other surface.
func (h *host) cleanControl(req request) error {
	if req.GID == "" {
		return fmt.Errorf("clean-control: missing gid")
	}
	if !h.keepControl {
		return cleanControlFiles(h.dataDir, req.GID, leftovers(req))
	}
	return nil
}

// leftovers is what aria2 keeps beside a download without ever listing it as
// one of its files: the .aria2 control file, and the <infohash>.torrent written
// because the daemon runs with --bt-save-metadata. Acting on the download's
// files alone left both sitting in the download folder.
func leftovers(req request) []string {
	return control.Leftovers(req.Dir, req.Name, req.InfoHash, req.Files)
}

// deleteFiles removes a download's files from disk, at the user's explicit
// request — aria2's own remove only forgets the download.
//
// Unlike clean-control this is not gated on a config setting: nothing about it is
// automatic, so there is no background behaviour for a user to opt out of. It is
// gated on containment instead, in control.Targets, which refuses anything it
// cannot place inside the configured download directory. A per-download --dir
// pointing somewhere else is therefore refused rather than followed, which is
// the trade this makes deliberately: the host will not delete outside the one
// directory it was configured with.
func (h *host) deleteFiles(req request) error {
	if req.GID == "" {
		return fmt.Errorf("delete-files: missing gid")
	}
	targets, err := deleteTargets(h.dir, req.Dir, req.Files)
	if err != nil {
		return fmt.Errorf("delete-files: %w", err)
	}
	// Vetted the same way as the data, so the containment rule above covers the
	// bookkeeping too rather than trusting paths derived from it.
	extra, err := deleteTargets(h.dir, req.Dir, leftovers(req))
	if err != nil {
		return fmt.Errorf("delete-files: %w", err)
	}
	return deleteDownloadFiles(h.dataDir, req.GID, h.dir, append(targets, extra...))
}

// reveal opens the containing folder of path in the OS file manager, with the
// file itself selected where the platform supports it.
//
// A finished download's file is often no longer there — moved, renamed or
// deleted after the fact — and every platform launcher just fails on a missing
// path ("open -R" exits 1). So a missing target falls back to the nearest
// existing ancestor directory, which is still where the user wanted to go, and
// only a path with no surviving ancestor is an error.
func reveal(path string) error {
	if path == "" {
		return fmt.Errorf("reveal: missing path")
	}
	if _, err := statPath(path); err == nil {
		switch goos {
		case "darwin":
			return execOpen("open", "-R", path)
		case "windows":
			return execOpen("explorer", "/select,"+path)
		default:
			return execOpen("xdg-open", filepath.Dir(path))
		}
	}
	dir, ok := nearestDir(filepath.Dir(path))
	if !ok {
		return fmt.Errorf("reveal: no longer on disk: %s", path)
	}
	switch goos {
	case "darwin":
		return execOpen("open", dir)
	case "windows":
		return execOpen("explorer", dir)
	default:
		return execOpen("xdg-open", dir)
	}
}

// nearestDir walks up from dir to the first directory that exists, stopping
// short of the filesystem root: opening a Finder/Explorer window on "/" is not
// what anyone meant by "show this download", so that case is an error instead.
func nearestDir(dir string) (string, bool) {
	for {
		if parent := filepath.Dir(dir); parent == dir {
			return "", false // dir is the root itself
		}
		if fi, err := statPath(dir); err == nil && fi.IsDir() {
			return dir, true
		}
		dir = filepath.Dir(dir)
	}
}

// expandHome resolves a leading ~/ against the user's home directory.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := userHome(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
