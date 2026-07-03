package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"aria2t/internal/config"
)

// stubProgram swaps runProgram for a stub returning err and restores it on
// cleanup. It returns a pointer to the model the stub received.
func stubProgram(t *testing.T, err error) *tea.Model {
	t.Helper()
	var got tea.Model
	orig := runProgram
	runProgram = func(m tea.Model) error {
		got = m
		return err
	}
	t.Cleanup(func() { runProgram = orig })
	return &got
}

// missingConfig returns a path to a config file that does not exist, so
// config.Load yields the defaults without touching the user's real config.
func missingConfig(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "absent.json")
}

// quitModel terminates the program loop immediately.
type quitModel struct{}

func (quitModel) Init() tea.Cmd                         { return tea.Quit }
func (m quitModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (quitModel) View() string                          { return "" }

// TestRunProgramDefault executes the real runProgram closure against a model
// that quits at once. programOpts is swapped so the program neither renders
// to the terminal nor grabs a TTY for input.
func TestRunProgramDefault(t *testing.T) {
	origOpts := programOpts
	programOpts = []tea.ProgramOption{
		tea.WithInput(&bytes.Buffer{}),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
	}
	t.Cleanup(func() { programOpts = origOpts })

	if err := runProgram(quitModel{}); err != nil {
		t.Fatalf("runProgram(quitModel) = %v, want nil", err)
	}
}

func TestMainFunc(t *testing.T) {
	stubProgram(t, nil)

	var code = -1
	origExit := osExit
	osExit = func(c int) { code = c }
	origArgs := os.Args
	os.Args = []string{"aria2t", "--config", missingConfig(t)}
	t.Cleanup(func() {
		osExit = origExit
		os.Args = origArgs
	})

	main()
	if code != 0 {
		t.Fatalf("main exited with %d, want 0", code)
	}
}

func TestRunHelp(t *testing.T) {
	var buf bytes.Buffer
	if got := run([]string{"--help"}, &buf); got != 0 {
		t.Fatalf("run(--help) = %d, want 0", got)
	}
	if !strings.Contains(buf.String(), "Usage") {
		t.Fatalf("usage not printed, got %q", buf.String())
	}
}

func TestRunParseError(t *testing.T) {
	var buf bytes.Buffer
	if got := run([]string{"--definitely-not-a-flag"}, &buf); got != 2 {
		t.Fatalf("run(bad flag) = %d, want 2", got)
	}
}

func TestRunBadURL(t *testing.T) {
	var buf bytes.Buffer
	args := []string{"--config", missingConfig(t), "--url", "://bad"}
	if got := run(args, &buf); got != 1 {
		t.Fatalf("run(bad url) = %d, want 1", got)
	}
	if !strings.Contains(buf.String(), "bad --url") {
		t.Fatalf("error not printed, got %q", buf.String())
	}
}

func TestRunBadURLPort(t *testing.T) {
	var buf bytes.Buffer
	args := []string{"--config", missingConfig(t), "--url", "ws://h:99999999999999999999"}
	if got := run(args, &buf); got != 1 {
		t.Fatalf("run(bad port) = %d, want 1", got)
	}
	if !strings.Contains(buf.String(), "bad --url port") {
		t.Fatalf("port error not printed, got %q", buf.String())
	}
}

func TestRunURLSuccess(t *testing.T) {
	got := stubProgram(t, nil)
	var buf bytes.Buffer
	args := []string{"--config", missingConfig(t), "--url", "ws://example.com:7000/jsonrpc", "--secret", "s"}
	if code := run(args, &buf); code != 0 {
		t.Fatalf("run(--url) = %d, want 0; stderr %q", code, buf.String())
	}
	if *got == nil {
		t.Fatal("runProgram never received a model")
	}
}

func TestRunSecretOnActiveServer(t *testing.T) {
	stubProgram(t, nil)
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Default()
	cfg.Servers = []config.Server{{Name: "ext", Host: "h", Port: 1, Protocol: "ws"}}
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if code := run([]string{"--config", path, "--secret", "sst"}, &buf); code != 0 {
		t.Fatalf("run(--secret) = %d, want 0; stderr %q", code, buf.String())
	}
}

func TestRunSecretWithNoServers(t *testing.T) {
	stubProgram(t, nil)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"servers": []}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if code := run([]string{"--config", path, "--secret", "sst"}, &buf); code != 0 {
		t.Fatalf("run = %d, want 0; stderr %q", code, buf.String())
	}
}

func TestRunConfigLoadWarning(t *testing.T) {
	stubProgram(t, nil)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if code := run([]string{"--config", path}, &buf); code != 0 {
		t.Fatalf("run(corrupt config) = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "warning:") {
		t.Fatalf("warning not printed, got %q", buf.String())
	}
}

func TestRunProgramError(t *testing.T) {
	stubProgram(t, errors.New("boom"))
	var buf bytes.Buffer
	if code := run([]string{"--config", missingConfig(t)}, &buf); code != 1 {
		t.Fatalf("run(program error) = %d, want 1", code)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Fatalf("program error not printed, got %q", buf.String())
	}
}

func TestServerFromURL(t *testing.T) {
	tests := []struct {
		name, raw, secret string
		want              config.Server
		wantErr           string
	}{
		{
			name: "full url",
			raw:  "ws://example.com:7000/jsonrpc", secret: "s",
			want: config.Server{Name: "cli", Host: "example.com", Port: 7000, Secret: "s", Protocol: "ws"},
		},
		{
			name: "default port",
			raw:  "http://example.com",
			want: config.Server{Name: "cli", Host: "example.com", Port: 6800, Protocol: "http"},
		},
		{
			name: "bare host defaults scheme",
			raw:  "localhost",
			want: config.Server{Name: "cli", Host: "localhost", Port: 6800, Protocol: "ws"},
		},
		{
			name: "trailing slash trimmed",
			raw:  "localhost/",
			want: config.Server{Name: "cli", Host: "localhost", Port: 6800, Protocol: "ws"},
		},
		{name: "unparseable", raw: "://bad", wantErr: "bad --url"},
		{name: "overflowing port", raw: "ws://h:99999999999999999999", wantErr: "bad --url port"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := serverFromURL(tt.raw, tt.secret)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("serverFromURL(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}
