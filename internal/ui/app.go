package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aria2t/internal/checksum"
	"aria2t/internal/config"
	"aria2t/internal/daemon"
	"aria2t/internal/rpc"
	"aria2t/internal/sched"
)

type screen int

const (
	screenList screen = iota
	screenDetail
	screenStats
	screenSettings
	screenSeeding
	screenScheduler
)

type overlay int

const (
	overlayNone overlay = iota
	overlayAdd
	overlayThrottle
	overlayServers
	overlayPrompt
)

// verifyState tracks checksum verification of one stopped download.
type verifyState struct {
	Expected    string
	Running     bool
	Done, Total int64
	Finished    bool
	OK          bool
	Computed    string
	Err         error
}

// App is the root model.
type App struct {
	cfg     config.Config
	cfgPath string
	styles  Styles

	client    api
	version   string
	connected bool
	connErr   error
	daemon    *daemon.Daemon
	endpoint  string // "host:port" actually connected to (managed port is dynamic)

	screen  screen
	overlay overlay

	snap     snapshot
	downHist *ring
	upHist   *ring

	list      listModel
	detail    detailModel
	stats     statsModel
	settings  settingsModel
	seeding   seedingModel
	scheduler schedulerModel
	add       addModel
	throttle  throttleModel
	servers   serversModel
	prompt    promptModel

	verify map[string]*verifyState

	status    string
	statusErr bool
	statusSeq int

	lastSchedKey string

	width, height int

	// dial and spawn are swappable for tests.
	dial  func(srv config.Server) (api, string, error)
	spawn func(o daemon.Options) (*daemon.Daemon, error)
}

// NewApp builds the root model from configuration.
func NewApp(cfg config.Config, cfgPath string) *App {
	styles := NewStyles(PaletteByName(cfg.Theme))
	a := &App{
		cfg:      cfg,
		cfgPath:  cfgPath,
		styles:   styles,
		downHist: newRing(60),
		upHist:   newRing(60),
		verify:   map[string]*verifyState{},
		width:    120,
		height:   36,
		dial:     dialServer,
		spawn:    daemon.Start,
	}
	a.list = newListModel(a)
	a.detail = newDetailModel(a)
	a.stats = newStatsModel(a)
	a.settings = newSettingsModel(a)
	a.seeding = newSeedingModel(a)
	a.scheduler = newSchedulerModel(a)
	a.add = newAddModel(a)
	a.throttle = newThrottleModel(a)
	a.servers = newServersModel(a)
	return a
}

func dialServer(srv config.Server) (api, string, error) {
	var c *rpc.Client
	var err error
	switch srv.Protocol {
	case "http", "https":
		c = rpc.New(srv.URL(), srv.Secret)
	default:
		c, err = rpc.NewWS(srv.URL(), srv.Secret)
		if err != nil {
			return nil, "", err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	v, err := c.GetVersion(ctx)
	if err != nil {
		c.Close()
		return nil, "", err
	}
	return c, v, nil
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(a.connectCmd(), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, tickMsgAt)
}

// tickMsgAt wraps the tick time; named so tests can invoke it directly.
func tickMsgAt(t time.Time) tea.Msg { return tickMsg(t) }

func (a *App) connectCmd() tea.Cmd {
	srv := a.cfg.ActiveServer()
	if srv.Managed {
		return a.connectManagedCmd(srv)
	}
	dial := a.dial
	return func() tea.Msg {
		c, v, err := dial(srv)
		if err != nil {
			return connectErrMsg{err}
		}
		return connectedMsg{client: c, version: v, endpoint: fmt.Sprintf("%s:%d", srv.Host, srv.Port)}
	}
}

// connectManagedCmd spawns (or reuses) the built-in daemon, then dials it.
func (a *App) connectManagedCmd(srv config.Server) tea.Cmd {
	d := a.daemon // reuse a daemon spawned earlier this run
	spawn := a.spawn
	dial := a.dial
	dir := expandHome(a.cfg.Dir)
	dataDir := filepath.Join(filepath.Dir(a.cfgPath), "daemon")
	return func() tea.Msg {
		if d == nil {
			var err error
			d, err = spawn(daemon.Options{Dir: dir, DataDir: dataDir, Secret: srv.Secret, Port: srv.Port})
			if err != nil {
				return connectErrMsg{err}
			}
		}
		proxy := config.Server{Host: "localhost", Port: d.Port, Secret: d.Secret, Protocol: "ws"}
		c, v, err := dial(proxy)
		if err != nil {
			return connectErrMsg{fmt.Errorf("built-in daemon dial: %w", err)}
		}
		return connectedMsg{client: c, version: v, daemon: d, endpoint: fmt.Sprintf("localhost:%d", d.Port)}
	}
}

// Shutdown releases the RPC client and stops the managed daemon, if any.
// Called by main after the program loop exits.
func (a *App) Shutdown() {
	if a.client != nil {
		a.client.Close()
		a.client = nil
	}
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = a.daemon.Stop(ctx)
		a.daemon = nil
	}
}

// pollCmd gathers the full snapshot in one background round.
func (a *App) pollCmd() tea.Cmd {
	c := a.client
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var s snapshot
		var err error
		if s.Active, err = c.TellActive(ctx); err != nil {
			return pollMsg{err: err}
		}
		if s.Waiting, err = c.TellWaiting(ctx, 0, 1000); err != nil {
			return pollMsg{err: err}
		}
		if s.Stopped, err = c.TellStopped(ctx, 0, 1000); err != nil {
			return pollMsg{err: err}
		}
		if s.Stat, err = c.GetGlobalStat(ctx); err != nil {
			return pollMsg{err: err}
		}
		s.Taken = time.Now()
		return pollMsg{snap: s}
	}
}

// listenCmd waits for one push notification.
func (a *App) listenCmd() tea.Cmd {
	c := a.client
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		n, ok := <-c.Notifications()
		if !ok {
			return nil
		}
		return notifMsg(n)
	}
}

// rpcCmd runs an RPC action off the update loop and reports the outcome.
func (a *App) rpcCmd(okText string, fn func(ctx context.Context, c api) error) tea.Cmd {
	c := a.client
	if c == nil {
		return a.flash("not connected", true)
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return actionDoneMsg{text: okText, err: fn(ctx, c)}
	}
}

// flash shows a transient status-line message.
func (a *App) flash(text string, isErr bool) tea.Cmd {
	a.status, a.statusErr = text, isErr
	a.statusSeq++
	return tea.Tick(4*time.Second, clearStatusAt(a.statusSeq))
}

// clearStatusAt builds the tick payload that clears status message seq.
func clearStatusAt(seq int) func(time.Time) tea.Msg {
	return func(time.Time) tea.Msg { return clearStatusMsg{seq} }
}

func (a *App) saveConfig() {
	if err := config.Save(a.cfgPath, a.cfg); err != nil {
		a.status, a.statusErr = "config save failed: "+err.Error(), true
	}
}

// reconnect tears down the current client and dials the active server.
func (a *App) reconnect() tea.Cmd {
	if a.client != nil {
		a.client.Close()
		a.client = nil
	}
	a.connected = false
	return a.connectCmd()
}

func (a *App) setTheme(name string) {
	a.cfg.Theme = name
	a.styles = NewStyles(PaletteByName(name))
	a.saveConfig()
}

// applySchedule pushes the active rule's limits once per minute-key change.
func (a *App) applySchedule(now time.Time) tea.Cmd {
	if !a.cfg.SchedulerEnabled || a.client == nil {
		return nil
	}
	down, up, label := "0", "0", "unlimited"
	if r, ok := sched.Active(a.cfg.Rules, now); ok {
		down, up, label = r.Down, r.Up, r.Label
	}
	key := down + "/" + up
	if key == a.lastSchedKey {
		return nil
	}
	a.lastSchedKey = key
	opts := map[string]string{"max-overall-download-limit": down, "max-overall-upload-limit": up}
	return a.rpcCmd("scheduler: "+label+" ("+FmtLimit(down)+"/"+FmtLimit(up)+")", func(ctx context.Context, c api) error {
		return c.ChangeGlobalOption(ctx, opts)
	})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil

	case connectedMsg:
		a.client = msg.client
		a.version = msg.version
		a.connected = true
		a.connErr = nil
		a.endpoint = msg.endpoint
		if msg.daemon != nil {
			a.daemon = msg.daemon
		}
		a.lastSchedKey = "" // re-apply schedule on the new server
		return a, tea.Batch(a.pollCmd(), a.listenCmd())

	case connectErrMsg:
		a.connected = false
		a.connErr = msg.err
		return a, nil

	case tickMsg:
		cmds := []tea.Cmd{tickCmd()}
		if a.client != nil {
			cmds = append(cmds, a.pollCmd())
		} else if a.connErr != nil {
			cmds = append(cmds, a.connectCmd()) // simple 1s retry
		}
		if cmd := a.applySchedule(time.Time(msg)); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case pollMsg:
		if msg.err != nil {
			a.connected = false
			return a, nil
		}
		a.connected = true
		if a.list.reordering {
			msg.snap.Waiting = a.list.frozenWaiting(msg.snap.Waiting)
		}
		a.snap = msg.snap
		a.downHist.Push(msg.snap.Stat.DownSpeed())
		a.upHist.Push(msg.snap.Stat.UpSpeed())
		a.list.clampCursor()
		var cmd tea.Cmd
		if a.screen == screenDetail {
			cmd = a.detail.refreshCmd()
		}
		return a, cmd

	case notifMsg:
		cmds := []tea.Cmd{a.listenCmd()}
		if a.client != nil {
			cmds = append(cmds, a.pollCmd())
		}
		return a, tea.Batch(cmds...)

	case actionDoneMsg:
		if msg.err != nil {
			return a, a.flash(msg.err.Error(), true)
		}
		cmds := []tea.Cmd{a.pollCmd()}
		if msg.text != "" {
			cmds = append(cmds, a.flash(msg.text, false))
		}
		return a, tea.Batch(cmds...)

	case clearStatusMsg:
		if msg.seq == a.statusSeq {
			a.status = ""
		}
		return a, nil

	case verifyProgressMsg:
		if v, ok := a.verify[msg.gid]; ok {
			v.Done, v.Total = msg.done, msg.total
		}
		return a, nil

	case verifyDoneMsg:
		v, ok := a.verify[msg.gid]
		if !ok {
			v = &verifyState{}
			a.verify[msg.gid] = v
		}
		v.Running = false
		v.Finished = true
		v.OK = msg.ok
		v.Computed = msg.computed
		v.Err = msg.err
		if msg.err != nil {
			return a, a.flash("verify: "+msg.err.Error(), true)
		}
		if msg.ok {
			return a, a.flash("checksum verified", false)
		}
		return a, a.flash("checksum MISMATCH", true)

	case detailDataMsg:
		a.detail.absorb(msg)
		return a, nil

	case gidOptionsMsg:
		if msg.err == nil {
			a.seeding.absorbOptions(msg)
			a.throttle.absorbOptions(msg)
		}
		return a, nil

	case globalOptionsMsg:
		if msg.err == nil {
			a.settings.absorbGlobal(msg.opts)
			a.seeding.absorbGlobal(msg.opts)
		}
		return a, nil

	case latencyMsg:
		a.servers.absorbLatency(msg)
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return a, tea.Quit
	}
	// Overlays swallow all keys while open.
	if a.overlay != overlayNone {
		var cmd tea.Cmd
		switch a.overlay {
		case overlayAdd:
			a.add, cmd = a.add.update(msg)
		case overlayThrottle:
			a.throttle, cmd = a.throttle.update(msg)
		case overlayServers:
			a.servers, cmd = a.servers.update(msg)
		case overlayPrompt:
			a.prompt, cmd = a.prompt.update(msg)
		}
		return a, cmd
	}
	switch a.screen {
	case screenList:
		return a.handleListKey(msg)
	case screenDetail:
		var cmd tea.Cmd
		a.detail, cmd = a.detail.update(msg)
		return a, cmd
	case screenStats:
		if k := msg.String(); k == "esc" || k == "q" || k == "g" {
			a.screen = screenList
		}
		return a, nil
	case screenSettings:
		var cmd tea.Cmd
		a.settings, cmd = a.settings.update(msg)
		return a, cmd
	case screenSeeding:
		var cmd tea.Cmd
		a.seeding, cmd = a.seeding.update(msg)
		return a, cmd
	case screenScheduler:
		var cmd tea.Cmd
		a.scheduler, cmd = a.scheduler.update(msg)
		return a, cmd
	}
	return a, nil
}

func (a *App) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.list, cmd = a.list.update(msg)
	return a, cmd
}

// startVerify launches checksum verification for a stopped download.
func (a *App) startVerify(s rpc.Status) tea.Cmd {
	v := a.verify[s.GID]
	if v == nil || v.Expected == "" {
		return a.flash("no expected checksum — press c to paste one", true)
	}
	if v.Running {
		return a.flash("verification already running", true)
	}
	if len(s.Files) == 0 || s.Files[0].Path == "" {
		return a.flash("no local file path", true)
	}
	path := s.Files[0].Path
	v.Running, v.Finished = true, false
	gid, expected := s.GID, v.Expected
	// Progress is polled via shared state, not streamed per-chunk, to keep
	// the update loop quiet; a final message carries the outcome.
	return func() tea.Msg {
		ok, computed, err := checksum.Verify(path, expected, func(done, total int64) {
			v.Done, v.Total = done, total // written from one goroutine, read for display only
		})
		return verifyDoneMsg{gid: gid, ok: ok, computed: computed, err: err}
	}
}

// redownload re-queues a stopped download from its recorded URIs.
func (a *App) redownload(s rpc.Status) tea.Cmd {
	var uris []string
	seen := map[string]bool{}
	for _, f := range s.Files {
		for _, u := range f.URIs {
			if !seen[u.URI] {
				seen[u.URI] = true
				uris = append(uris, u.URI)
			}
		}
	}
	if len(uris) == 0 {
		return a.flash("no source URIs recorded — cannot re-download", true)
	}
	gid, dir := s.GID, s.Dir
	return a.rpcCmd("re-download queued", func(ctx context.Context, c api) error {
		_ = c.RemoveDownloadResult(ctx, gid) // best effort: clear the old result
		opts := map[string]string{}
		if dir != "" {
			opts["dir"] = dir
		}
		_, err := c.AddURI(ctx, uris, opts)
		return err
	})
}

// openDirBin is the file-manager launcher; a var so tests can substitute a
// harmless command.
var openDirBin = openBinFor(runtime.GOOS)

// openBinFor picks the platform's directory opener.
func openBinFor(goos string) string {
	if goos == "darwin" {
		return "open"
	}
	return "xdg-open"
}

// openDir reveals the download directory in the OS file manager.
func (a *App) openDir(dir string) tea.Cmd {
	if dir == "" {
		return a.flash("no directory", true)
	}
	bin := openDirBin
	return func() tea.Msg {
		if err := exec.Command(bin, dir).Start(); err != nil {
			return actionDoneMsg{err: fmt.Errorf("open %s: %w", dir, err)}
		}
		return actionDoneMsg{}
	}
}

// header renders the top bar shared by every screen.
func (a *App) header() string {
	st := a.styles
	srv := a.cfg.ActiveServer()
	conn := st.Green.Render("● connected")
	if !a.connected {
		conn = st.Red.Render("● disconnected")
	}
	endpoint := a.endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s:%d", srv.Host, srv.Port)
	}
	if srv.Managed {
		endpoint += " (built-in)"
	}
	left := lipgloss.JoinHorizontal(lipgloss.Center,
		st.Brand.Render("aria2t"), st.Faint.Render(" │ "),
		st.Dim.Render(endpoint+" "), conn)
	right := st.Cyan.Render("▼ "+FmtSpeed(a.snap.Stat.DownSpeed())) + " " +
		st.Magenta.Render("▲ "+FmtSpeed(a.snap.Stat.UpSpeed()))
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	return " " + left + lipgloss.NewStyle().Width(gap).Render("") + right
}

// statusLine renders the transient message, if any.
func (a *App) statusLine() string {
	if a.status == "" {
		return ""
	}
	if a.statusErr {
		return " " + a.styles.Red.Render(a.status)
	}
	return " " + a.styles.Green.Render(a.status)
}

func (a *App) View() string {
	var body string
	switch a.screen {
	case screenList:
		body = a.list.view()
	case screenDetail:
		body = a.detail.view()
	case screenStats:
		body = a.stats.view()
	case screenSettings:
		body = a.settings.view()
	case screenSeeding:
		body = a.seeding.view()
	case screenScheduler:
		body = a.scheduler.view()
	}
	if a.overlay != overlayNone {
		var modal string
		switch a.overlay {
		case overlayAdd:
			modal = a.add.view()
		case overlayThrottle:
			modal = a.throttle.view()
		case overlayServers:
			modal = a.servers.view()
		case overlayPrompt:
			modal = a.prompt.view()
		}
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, modal)
	}
	return body
}
