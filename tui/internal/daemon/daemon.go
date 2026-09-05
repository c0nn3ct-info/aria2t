// Package daemon spawns and supervises a private aria2c process so aria2t
// works out of the box without a user-managed server.
package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"aria2t/internal/control"
	"aria2t/internal/rpc"
)

// Options configures a spawned daemon. Zero values are filled with defaults.
type Options struct {
	Bin         string                              // aria2c path; discovered when empty
	Dir         string                              // download directory
	DataDir     string                              // holds session file and log
	Secret      string                              // RPC secret; random when empty
	Port        int                                 // RPC port; free ephemeral port when 0
	ReadyProbe  func(port int, secret string) error // test hook; default polls RPC getVersion
	ReadyWithin time.Duration                       // default 10s
	// Detach runs the child in its own session/process group so it survives
	// this process's exit (used by the native-messaging host, which exits
	// whenever the browser disconnects while the daemon keeps downloading).
	Detach bool
}

// Daemon is a running aria2c child process.
type Daemon struct {
	Port   int
	Secret string

	cmd       *exec.Cmd
	done      chan struct{} // closed once Wait returns
	stateFile string        // reap state path, removed on a clean Stop
}

// Alive reports whether the child process is still running.
func (d *Daemon) Alive() bool {
	select {
	case <-d.done:
		return false
	default:
		return true
	}
}

// commonPaths are checked when aria2c is not on PATH (Homebrew, MacPorts…).
var commonPaths = []string{
	"/opt/homebrew/bin/aria2c",
	"/usr/local/bin/aria2c",
	"/usr/bin/aria2c",
	"/opt/local/bin/aria2c",
}

// ErrBinaryNotFound is returned when no aria2c can be located. Callers match
// it with errors.Is rather than reading the prose: the native host turns it
// into a machine-readable code, and the extension used to sniff the message
// text instead — where the words "not found" collided with Chrome's own
// "Specified native messaging host not found" and a missing aria2c was
// reported to the user as a missing host.
var ErrBinaryNotFound = errors.New("aria2c not found — install it (brew install aria2) or connect to an external server with --url")

// aria2cName is the binary's filename on the given platform. Taken as a
// parameter so both branches run in a test on any host.
func aria2cName(goos string) string {
	if goos == "windows" {
		return "aria2c.exe"
	}
	return "aria2c"
}

// osExecutable is an indirection for tests over the running binary's path.
var osExecutable = os.Executable

// FindBinary locates aria2c: on PATH, then beside the aria2t binary, then in
// the usual install prefixes.
//
// The beside-the-binary probe is what makes the Windows installer's private
// aria2c.exe usable: it is dropped next to aria2t.exe, and neither the shell
// that ran the installer nor the browser that spawns the native host has
// re-read the user PATH yet.
func FindBinary() (string, error) {
	name := aria2cName(runtime.GOOS)
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	if exe, err := osExecutable(); err == nil {
		beside := filepath.Join(filepath.Dir(exe), name)
		if st, err := os.Stat(beside); err == nil && !st.IsDir() {
			return beside, nil
		}
	}
	for _, p := range commonPaths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", ErrBinaryNotFound
}

// netListen and randRead are indirections for tests.
var (
	netListen = net.Listen
	randRead  = rand.Read
	// filterSession drops session entries whose control file was cleaned; see
	// internal/control.
	filterSession = control.FilterSession
)

// daemonState* are indirections for tests over the reap state file.
var (
	daemonStateRead   = os.ReadFile
	daemonStateWrite  = os.WriteFile
	daemonStateRemove = os.Remove
)

// daemonState records a spawned daemon's coordinates so the next launch can
// reap it if this process died without stopping it (a crash or kill -9).
type daemonState struct {
	PID    int    `json:"pid"`
	Port   int    `json:"port"`
	Secret string `json:"secret"`
}

func stateFilePath(dataDir string) string { return filepath.Join(dataDir, "daemon.json") }

// State reports the daemon coordinates recorded in dataDir's state file
// (daemon.json), if any. ok is false when no state is recorded or the file
// is unreadable/malformed. It does not check that the process is alive —
// probe the RPC port for that.
func State(dataDir string) (pid, port int, secret string, ok bool) {
	raw, err := daemonStateRead(stateFilePath(dataDir))
	if err != nil {
		return 0, 0, "", false
	}
	var st daemonState
	if json.Unmarshal(raw, &st) != nil || st.Port == 0 {
		return 0, 0, "", false
	}
	return st.PID, st.Port, st.Secret, true
}

// reapStale cleanly shuts down a daemon left running by a previous aria2t. It
// only targets a process still answering on the recorded port with the recorded
// secret — unambiguously our own aria2c — so it can never hit an unrelated
// program that happens to have inherited a recycled PID. Best effort: a dead
// port simply fails the RPC and is ignored.
func reapStale(dataDir string) {
	_, port, secret, ok := State(dataDir)
	if !ok {
		return
	}
	c := rpc.New(fmt.Sprintf("http://localhost:%d/jsonrpc", port), secret)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.SaveSession(ctx)
	_ = c.Shutdown(ctx)
}

// writeState records the running daemon so a later launch can reap it.
func writeState(path string, st daemonState) {
	raw, _ := json.Marshal(st) // cannot fail for this type
	_ = daemonStateWrite(path, raw, 0o600)
}

// FreePort asks the kernel for an unused TCP port on localhost.
func FreePort() (int, error) {
	l, err := netListen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// randomSecret returns 32 hex chars from crypto/rand.
func randomSecret() (string, error) {
	buf := make([]byte, 16)
	if _, err := randRead(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// buildArgs assembles the aria2c command line for the given settings.
// The RPC secret travels via the conf file, not argv, so it never shows in
// the process list. The session file is passed as --input-file only when
// it already exists.
func buildArgs(dir, sessionFile, logFile, confFile string, port int) []string {
	args := []string{
		"--enable-rpc",
		fmt.Sprintf("--rpc-listen-port=%d", port),
		"--conf-path=" + confFile,
		"--rpc-listen-all=false",
		"--save-session=" + sessionFile,
		"--save-session-interval=30",
		// Persist completed (and seeding) downloads too, not just
		// unfinished/errored ones — otherwise the stopped list is empty after a
		// restart. Reloaded completes are recognised via their control file and
		// are not re-downloaded.
		"--force-save=true",
		// Save a magnet's resolved metadata as <infohash>.torrent and reload it
		// on the next launch. Without this, --save-session persists the magnet
		// URI (not the resolved torrent), so a restart re-downloads metadata
		// before the actual transfer resumes. With both, the saved .torrent is
		// read at startup and the download resumes straight away.
		"--bt-save-metadata=true",
		"--bt-load-saved-metadata=true",
		"--log=" + logFile,
		"--log-level=warn",
		"--quiet=true",
	}
	if dir != "" {
		args = append(args, "--dir="+dir)
	}
	if _, err := os.Stat(sessionFile); err == nil {
		args = append(args, "--input-file="+sessionFile)
	}
	return args
}

// defaultProbe checks the RPC endpoint answers getVersion.
func defaultProbe(port int, secret string) error {
	c := rpc.New(fmt.Sprintf("http://localhost:%d/jsonrpc", port), secret)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.GetVersion(ctx)
	return err
}

// Start launches aria2c and waits until its RPC interface answers.
func Start(opts Options) (*Daemon, error) {
	bin := opts.Bin
	if bin == "" {
		var err error
		if bin, err = FindBinary(); err != nil {
			return nil, err
		}
	}
	port := opts.Port
	if port == 0 {
		var err error
		if port, err = FreePort(); err != nil {
			return nil, err
		}
	}
	secret := opts.Secret
	if secret == "" {
		var err error
		if secret, err = randomSecret(); err != nil {
			return nil, err
		}
	}
	dataDir := opts.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(os.TempDir(), "aria2t")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	reapStale(dataDir) // clean up a daemon a previous crash left running
	sessionFile := filepath.Join(dataDir, "session.txt")
	// Downloads whose control file was deleted on completion must not be
	// reloaded — without it aria2 would download them again. This is the one
	// moment nothing else is writing the session. Best effort: a failure here
	// only means a stale entry, never a failed launch.
	_ = filterSession(sessionFile, dataDir)
	logFile := filepath.Join(dataDir, "aria2.log")
	confFile := filepath.Join(dataDir, "aria2t.conf")
	if err := os.WriteFile(confFile, []byte("rpc-secret="+secret+"\n"), 0o600); err != nil {
		return nil, err
	}

	cmd := exec.Command(bin, buildArgs(opts.Dir, sessionFile, logFile, confFile, port)...)
	var stderr strings.Builder
	if opts.Detach {
		// The child must outlive this process: give it its own
		// session/process group so signals aimed at us (terminal ^C, group
		// kills) never reach it, and capture no stderr — an in-process pipe
		// dies with us and a later aria2c write to it would SIGPIPE-kill
		// the daemon. Startup errors go to the log file instead.
		attr := detachTemplate // copy: one attr value per spawn
		cmd.SysProcAttr = &attr
	} else {
		cmd.Stderr = &stderr
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	d := &Daemon{Port: port, Secret: secret, cmd: cmd, done: make(chan struct{}), stateFile: stateFilePath(dataDir)}
	writeState(d.stateFile, daemonState{PID: cmd.Process.Pid, Port: port, Secret: secret})
	go func() {
		_ = cmd.Wait()
		close(d.done)
	}()

	probe := opts.ReadyProbe
	if probe == nil {
		probe = defaultProbe
	}
	deadline := opts.ReadyWithin
	if deadline == 0 {
		deadline = 10 * time.Second
	}
	start := time.Now()
	for {
		select {
		case <-d.done:
			return nil, fmt.Errorf("aria2c exited during startup: %s", startupDetail(stderr.String(), logFile))
		default:
		}
		if err := probe(port, secret); err == nil {
			return d, nil
		}
		if time.Since(start) > deadline {
			d.kill()
			return nil, fmt.Errorf("aria2c did not become ready within %s", deadline)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// startupDetail is what an "exited during startup" error can say about why.
//
// A supervised child wrote its complaint to the captured stderr. A detached one
// (the native host's case) has no pipe to us, so its only words are in the log
// file aria2c was told to write — without reading it the error was the bare
// "aria2c exited during startup: " that the extension then showed verbatim,
// which told the user nothing and a bug report even less.
func startupDetail(stderr, logFile string) string {
	if s := strings.TrimSpace(stderr); s != "" {
		return s
	}
	if tail := LogTail(logFile, 5); len(tail) > 0 {
		return strings.Join(tail, " | ")
	}
	return "no output (see " + logFile + ")"
}

// logTailBytes bounds how much of the log is read for a tail: the last lines
// are the ones that explain a failure, and a daemon that has been running for
// weeks has a log nobody needs whole.
const logTailBytes = 64 << 10

// LogTail returns the last n non-empty lines of the file at path, or nil when
// there is nothing to read. Best effort by design — it decorates an error and
// a problem report, and a missing log is itself a fact worth reporting.
func LogTail(path string, n int) []string {
	// Whole-file read, then a slice: aria2 logs at warn level here, so the file
	// stays small, and it keeps this to one branch a test can actually reach.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if len(raw) > logTailBytes {
		raw = raw[len(raw)-logTailBytes:]
	}
	var lines []string
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimRight(l, "\r"); strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// URL returns the websocket RPC endpoint of the daemon.
func (d *Daemon) URL() string { return fmt.Sprintf("ws://localhost:%d/jsonrpc", d.Port) }

// Stop shuts the daemon down: saveSession + shutdown over RPC, then
// SIGTERM, then SIGKILL — first thing that works wins.
func (d *Daemon) Stop(ctx context.Context) error {
	defer func() { _ = daemonStateRemove(d.stateFile) }() // clean exit leaves nothing to reap
	c := rpc.New(fmt.Sprintf("http://localhost:%d/jsonrpc", d.Port), d.Secret)
	_ = c.SaveSession(ctx)
	_ = c.Shutdown(ctx)
	select {
	case <-d.done:
		return nil
	case <-time.After(3 * time.Second):
	}
	_ = d.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-d.done:
		return nil
	case <-time.After(2 * time.Second):
	}
	d.kill()
	<-d.done
	return nil
}

func (d *Daemon) kill() { _ = d.cmd.Process.Kill() }
