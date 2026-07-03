// Command aria2t is a terminal UI for the aria2 download manager. By
// default it spawns and manages a private aria2c daemon; --url (or servers
// in the config) connects to an external one instead.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
	"aria2t/internal/ui"
)

// osExit, programOpts and runProgram are indirections so run() and main()
// are testable.
var (
	osExit      = os.Exit
	programOpts = []tea.ProgramOption{tea.WithAltScreen()}
	runProgram  = func(m tea.Model) error {
		_, err := tea.NewProgram(m, programOpts...).Run()
		return err
	}
)

func main() { osExit(run(os.Args[1:], os.Stderr)) }

func run(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("aria2t", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", config.DefaultPath(), "path to config file")
	rpcURL := fs.String("url", "", "external aria2 RPC endpoint, e.g. ws://localhost:6800/jsonrpc (skips the built-in daemon)")
	secret := fs.String("secret", "", "aria2 RPC secret (overrides config)")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, "warning:", err)
	}
	if *rpcURL != "" {
		srv, err := serverFromURL(*rpcURL, *secret)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		cfg.Servers = append([]config.Server{srv}, cfg.Servers...)
		cfg.Active = 0
	} else if *secret != "" && len(cfg.Servers) > 0 {
		if cfg.Active < 0 || cfg.Active >= len(cfg.Servers) {
			cfg.Active = 0 // stale index in a hand-edited config must not panic
		}
		cfg.Servers[cfg.Active].Secret = *secret
	}

	app := ui.NewApp(cfg, *cfgPath)
	runErr := runProgram(app)
	app.Shutdown() // stops the managed daemon even when the loop errored
	if runErr != nil {
		fmt.Fprintln(stderr, runErr)
		return 1
	}
	return 0
}

// serverFromURL turns --url into a transient config server entry.
func serverFromURL(raw, secret string) (config.Server, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return config.Server{}, fmt.Errorf("bad --url: %w", err)
	}
	proto := u.Scheme
	if proto == "" {
		proto = "ws"
	}
	port := 6800 // aria2 default for bare host names
	switch {
	case u.Port() != "":
		if port, err = strconv.Atoi(u.Port()); err != nil {
			return config.Server{}, fmt.Errorf("bad --url port: %w", err)
		}
	case u.Scheme == "http" || u.Scheme == "ws":
		port = 80
	case u.Scheme == "https" || u.Scheme == "wss":
		port = 443
	}
	host := u.Hostname()
	path := ""
	if host == "" {
		// Bare "localhost" parses entirely into the path; treat it as a host.
		host = strings.TrimSuffix(raw, "/")
	} else if p := u.EscapedPath(); p != "" && p != "/" {
		path = p
	}
	return config.Server{Name: "cli", Host: host, Port: port, Secret: secret, Protocol: proto, Path: path, Transient: true}, nil
}
