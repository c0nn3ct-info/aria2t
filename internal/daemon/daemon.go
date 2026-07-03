// Package daemon spawns and supervises a private aria2c process so aria2t
// works out of the box without a user-managed server.
package daemon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
}

// Daemon is a running aria2c child process.
type Daemon struct {
	Port   int
	Secret string

	cmd  *exec.Cmd
	done chan struct{} // closed once Wait returns
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

// FindBinary locates aria2c.
func FindBinary() (string, error) {
	if p, err := exec.LookPath("aria2c"); err == nil {
		return p, nil
	}
	for _, p := range commonPaths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", errors.New("aria2c not found — install it (brew install aria2) or connect to an external server with --url")
}

// netListen and randRead are indirections for tests.
var (
	netListen = net.Listen
	randRead  = rand.Read
)

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
	sessionFile := filepath.Join(dataDir, "session.txt")
	logFile := filepath.Join(dataDir, "aria2.log")
	confFile := filepath.Join(dataDir, "aria2t.conf")
	if err := os.WriteFile(confFile, []byte("rpc-secret="+secret+"\n"), 0o600); err != nil {
		return nil, err
	}

	cmd := exec.Command(bin, buildArgs(opts.Dir, sessionFile, logFile, confFile, port)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	d := &Daemon{Port: port, Secret: secret, cmd: cmd, done: make(chan struct{})}
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
			return nil, fmt.Errorf("aria2c exited during startup: %s", strings.TrimSpace(stderr.String()))
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

// URL returns the websocket RPC endpoint of the daemon.
func (d *Daemon) URL() string { return fmt.Sprintf("ws://localhost:%d/jsonrpc", d.Port) }

// Stop shuts the daemon down: saveSession + shutdown over RPC, then
// SIGTERM, then SIGKILL — first thing that works wins.
func (d *Daemon) Stop(ctx context.Context) error {
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
