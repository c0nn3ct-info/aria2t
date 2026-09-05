package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"aria2t/internal/checksum"
	"aria2t/internal/config"
	"aria2t/internal/control"
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
	overlayConfirm
	overlayHelp
	overlayFiles
	overlayBrowse
	overlayCommands
)

// verifyState tracks checksum verification of one stopped download.
// Done and Total are atomic because the hashing goroutine updates them
// while the render loop reads them.
type verifyState struct {
	Expected    string
	Running     bool
	Done, Total atomic.Int64
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
	// spawned tracks the most recent child from the connect goroutine so
	// Shutdown can stop it even when the user quits before the connect
	// message is delivered.
	spawned  atomic.Pointer[daemon.Daemon]
	endpoint string // "host:port" actually connected to (managed port is dynamic)

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
	confirm   confirmModel
	help      helpModel
	files     filesModel
	browse    browseModel
	commands  commandModel

	hits hitmap

	verify map[string]*verifyState

	// knownStopped tracks which stopped GIDs have been seen, so a poll can
	// tell a fresh completion/failure from history; seeded on first poll.
	knownStopped  map[string]bool
	stoppedSeeded bool
	// metaCleaned tracks magnet-metadata leftovers already purged, so the
	// removal fires once per gid.
	metaCleaned map[string]bool
	// pendingMagnets maps a magnet's metadata gid → whether to start the real
	// torrent once files are chosen. Watched each poll for its followedBy.
	pendingMagnets map[string]bool
	// magnetQueue holds resolved magnets (their real torrent is paused) waiting
	// to be presented one picker at a time, so several finishing at once don't
	// stack modals or get lost.
	magnetQueue []magnetReady
	// picks persists downloads added paused and awaiting file selection, so a
	// picker left unanswered at quit is reopened next launch. picksReconciled
	// gates the one-shot restore against the first snapshot after connect.
	picks           []pendingPick
	picksReconciled bool

	status     string
	statusErr  bool
	statusSeq  int
	pollSeq    uint64
	polling    bool
	accessible bool
	events     []string

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
		cfg:            cfg,
		cfgPath:        cfgPath,
		styles:         styles,
		downHist:       newRing(60),
		upHist:         newRing(60),
		verify:         map[string]*verifyState{},
		metaCleaned:    map[string]bool{},
		pendingMagnets: map[string]bool{},
		width:          120,
		height:         36,
		dial:           dialServer,
		spawn:          daemon.Start,
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
	a.help = newHelpModel(a)
	a.files = newFilesModel(a)
	a.commands = newCommandModel(a)
	return a
}

// SetAccessible enables the keyboard-only, colour-free, ASCII presentation.
// It is intended for screen readers and terminals with limited capabilities.
func (a *App) SetAccessible(on bool) { a.accessible = on }

// keyHint is one clickable hint in a screen's key-bar: token is the key the
// click synthesizes (empty = display-only), key/label are shown.
type keyHint struct{ token, key, label string }

// magnetReady is a magnet whose metadata resolved: its (paused) torrent gid and
// whether to start it once files are chosen. parent is the persisted-pick key
// (the metadata gid for a magnet, the torrent gid for a restored torrent pick),
// cleared once the picker is answered.
type magnetReady struct {
	gid     string
	unpause bool
	parent  string
}

// overlayOffset computes where a centered modal lands on screen, so its
// clickable regions can be registered in absolute coordinates.
func (a *App) overlayOffset(modal string) (x, y int) {
	x = (a.width - lipgloss.Width(modal)) / 2
	y = (a.height - lipgloss.Height(modal)) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// handleMouse routes wheel and click events through the hitmap.
func (a *App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Wheel behaves like j/k, but only in contexts where those keys
	// navigate — never where they would type into a text input.
	if msg.Action == tea.MouseActionPress &&
		(msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
		if !a.wheelNavigates() {
			return a, nil
		}
		if msg.Button == tea.MouseButtonWheelUp {
			return a.handleKey(key_("k"))
		}
		return a.handleKey(key_("j"))
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return a, nil
	}
	id, ok := a.hits.hit(msg.X, msg.Y)
	if !ok {
		return a, nil
	}
	var cmd tea.Cmd
	switch a.overlay {
	case overlayAdd:
		a.add, cmd = a.add.mouse(id)
	case overlayThrottle:
		a.throttle, cmd = a.throttle.mouse(id)
	case overlayServers:
		a.servers, cmd = a.servers.mouse(id)
	case overlayConfirm:
		a.confirm, cmd = a.confirm.mouse(id)
	case overlayFiles:
		a.files, cmd = a.files.mouse(id)
	case overlayBrowse:
		a.browse, cmd = a.browse.mouse(id)
	case overlayHelp:
		a.overlay = overlayNone
	case overlayCommands:
		a.commands, cmd = a.commands.mouse(id)
	case overlayPrompt:
		a.prompt, cmd = a.prompt.mouse(id)
	default:
		return a.screenMouse(id)
	}
	return a, cmd
}

// screenMouse dispatches clicks on the current screen.
func (a *App) screenMouse(id string) (tea.Model, tea.Cmd) {
	if id == "back" && a.screen != screenList {
		a.screen = screenList
		return a, nil
	}
	var cmd tea.Cmd
	switch a.screen {
	case screenList:
		a.list, cmd = a.list.mouse(id)
	case screenDetail:
		a.detail, cmd = a.detail.mouse(id)
	case screenStats:
		a.stats, cmd = a.stats.mouse(id)
	case screenSettings:
		a.settings, cmd = a.settings.mouse(id)
	case screenSeeding:
		a.seeding, cmd = a.seeding.mouse(id)
	case screenScheduler:
		a.scheduler, cmd = a.scheduler.mouse(id)
	}
	return a, cmd
}

// hintbar renders a clickable key-hint line at row y: each hint with a
// non-empty token registers a "key:<token>" region so mouse users can trigger
// the same action as the key. Screens route "key" clicks back through update.
func (a *App) hintbar(y int, hints []keyHint) string {
	return a.hintbarEx(y, hints, a.styles.Key, "")
}

// hintbarEx is the one shared key-bar renderer. keyStyle colours the key glyph
// (Key normally; Magenta in the list's reorder mode). trailer, when set, is
// right-aligned on the same row (the list's cursor/total counter).
func (a *App) hintbarEx(y int, hints []keyHint, keyStyle lipgloss.Style, trailer string) string {
	st := a.styles
	budget := a.width // width available to hints (reserve room for the trailer)
	if trailer != "" {
		budget -= lipgloss.Width(trailer) + 2
	}
	var parts []string
	x := 1
	for _, h := range hints {
		part := keyStyle.Render(h.key) + " " + st.Dim.Render(h.label)
		w := lipgloss.Width(part)
		// Width-adaptive: drop the lowest-priority (rightmost) hints rather than
		// wrap to a second line (a wrapped bar breaks the screenFrame/bottomBar
		// line count). Always keep at least the first hint.
		if len(parts) > 0 && x+w-1 >= budget {
			break
		}
		parts = append(parts, part)
		if h.token != "" && x+w-1 < a.width {
			a.hits.add("key:"+h.token, x, y, x+w-1, y)
		}
		x += w + 2
	}
	line := " " + strings.Join(parts, "  ")
	if trailer != "" {
		gap := a.width - lipgloss.Width(line) - lipgloss.Width(trailer) - 1
		if gap < 1 {
			gap = 1
		}
		line += strings.Repeat(" ", gap) + trailer
	}
	return line
}

// checkbox renders the shared [x]/[ ] indicator: Green when on, Dim off. ASCII,
// 3 cells — safe in column-aligned rows.
func (a *App) checkbox(on bool) string {
	if on {
		return a.styles.Green.Render("[x]")
	}
	return a.styles.Dim.Render("[ ]")
}

// tab renders a tab/segment chip: a filled Accent badge when active, bracketed
// dim text when idle. Shared by the list tabs, add source tabs, and the servers
// ws/http toggle.
func (a *App) tab(label string, active bool) string {
	if active {
		return a.styles.Badge.Render(label)
	}
	return a.styles.TabIdle.Render("[ " + label + " ]")
}

// modalCard is the shared modal-border rule: Red for a destructive dialog,
// Accent otherwise. No background (see the Modal style).
func (a *App) modalCard(destructive bool) lipgloss.Style {
	if destructive {
		return a.styles.Modal.BorderForeground(a.styles.P.Red)
	}
	return a.styles.Modal
}

// screenFrame pins a screen's key-bar (and the transient status line) to the
// terminal bottom: it clips/pads the body so the frame is exactly a.height lines
// and the bar is always the last visible row — the fix for hints scrolling off a
// short terminal. The bar is rendered at its final y so its click regions match.
// (list.go keeps its own bottomBar: it has the position trailer + filter modes.)
func (a *App) screenFrame(body string, hints []keyHint) string {
	status := a.statusLine() // "" or "\n <flash>"
	top := a.height - 1 - strings.Count(status, "\n")
	if top < 1 {
		top = 1
	}
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(lines) > top {
		lines = lines[:top]
	}
	for len(lines) < top {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n" + a.hintbar(top, hints) + status
}

// wheelNavigates reports whether j/k currently move a selection instead
// of typing into a focused text input.
func (a *App) wheelNavigates() bool {
	switch a.overlay {
	case overlayNone:
	case overlayServers:
		return !a.servers.editing
	case overlayFiles:
		return true // j/k scroll the tree; no text input
	case overlayBrowse:
		return true // j/k scroll the file list
	default:
		return false
	}
	switch a.screen {
	case screenList:
		return !a.list.filtering // wheel must not type j/k into the filter
	case screenDetail:
		return true // j/k only move the file cursor; no inputs
	case screenScheduler:
		return !a.scheduler.editing
	case screenSeeding:
		return a.seeding.focus >= a.seeding.trackersStart()
	default:
		return false // settings and stats have inputs / nothing to scroll
	}
}

// key_ builds a rune key message (helper for synthesized input).
func key_(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// confirmRemove opens the shared confirmation modal for removals.
func (a *App) confirmRemove(name string, onYes func() tea.Cmd) tea.Cmd {
	a.confirm = newConfirmModel(a, "Remove download?", name, onYes)
	a.overlay = overlayConfirm
	return nil
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
	return tickCmdAfter(time.Second)
}

func tickCmdAfter(d time.Duration) tea.Cmd { return tea.Tick(d, tickMsgAt) }

func (a *App) pollInterval() time.Duration {
	for _, v := range a.verify {
		if v.Running {
			return time.Second
		}
	}
	if len(a.snap.Active) > 0 || len(a.snap.Waiting) > 0 || a.connErr != nil {
		return time.Second
	}
	if a.accessible {
		return 10 * time.Second
	}
	return 5 * time.Second
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
			return connectErrMsg{err: err}
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
		if d != nil && !d.Alive() {
			d = nil // the previous child died; spawn a fresh one
		}
		if d == nil {
			var err error
			d, err = spawn(daemon.Options{Dir: dir, DataDir: dataDir, Secret: srv.Secret, Port: srv.Port})
			if err != nil {
				return connectErrMsg{err: err}
			}
		}
		a.spawned.Store(d) // visible to Shutdown even if this msg is never delivered
		proxy := config.Server{Host: "localhost", Port: d.Port, Secret: d.Secret, Protocol: "ws"}
		c, v, err := dial(proxy)
		if err != nil {
			// Keep the daemon handle: the retry reuses it instead of
			// spawning a duplicate, and Shutdown can still stop it.
			return connectErrMsg{err: fmt.Errorf("built-in daemon dial: %w", err), daemon: d}
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
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if a.daemon != nil {
		_ = a.daemon.Stop(ctx)
	}
	// A connect could still be in flight when the user quit; its daemon
	// never reached a.daemon but is recorded in the spawned slot.
	if sp := a.spawned.Load(); sp != nil && sp != a.daemon {
		_ = sp.Stop(ctx)
	}
	a.daemon = nil
}

// pollCmd is the unguarded polling command used by focused unit tests.
func (a *App) pollCmd() tea.Cmd { return a.pollCmdSeq(0) }

// requestPoll starts at most one production poll at a time. Its generation
// prevents an old connection's response from replacing a newer snapshot.
func (a *App) requestPoll() tea.Cmd {
	if a.client == nil || a.polling {
		return nil
	}
	a.polling = true
	a.pollSeq++
	return a.pollCmdSeq(a.pollSeq)
}

// pollCmdSeq gathers the full snapshot in one background round.
func (a *App) pollCmdSeq(seq uint64) tea.Cmd {
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
			return pollMsg{seq: seq, err: err}
		}
		if s.Waiting, err = c.TellWaiting(ctx, 0, 1000); err != nil {
			return pollMsg{seq: seq, err: err}
		}
		if s.Stopped, err = c.TellStopped(ctx, 0, 1000); err != nil {
			return pollMsg{seq: seq, err: err}
		}
		if s.Stat, err = c.GetGlobalStat(ctx); err != nil {
			return pollMsg{seq: seq, err: err}
		}
		s.Taken = time.Now()
		return pollMsg{seq: seq, snap: s}
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
	a.status = "Working: " + strings.TrimSpace(okText) + "…"
	if strings.TrimSpace(okText) == "" {
		a.status = "Working…"
	}
	a.statusErr = false
	a.statusSeq++
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return actionDoneMsg{text: okText, err: fn(ctx, c)}
	}
}

// removePurgeInterval is how long removeFn waits between purge retries.
var removePurgeInterval = 200 * time.Millisecond

// removeFn builds the rpcCmd body that removes a download, purges its result so
// --force-save cannot resurrect it on the next launch, and takes aria2's
// leftovers off the disk with it.
//
// A stopped download is already a result, so a single purge deletes it. An
// active/seeding one must first be stopped by aria2.remove — which is
// asynchronous: aria2 only moves the group to the results list after it has
// torn down peer connections and told the tracker it stopped, which for a
// seeding torrent with live peers takes a beat. The immediate purge therefore
// often runs before the result exists and fails silently, leaving the download
// parked as a "complete" result instead of being deleted. Retry the purge until
// it lands (bounded by the caller's context), so a seeding torrent is actually
// removed the moment its teardown completes.
//
// The leftover paths are worked out here rather than in the body: this runs on
// the update goroutine, the body does not, and s is gone from aria2 by the time
// it would be asked for them.
func (a *App) removeFn(gid string, s rpc.Status) func(context.Context, api) error {
	stopped := isStopped(s.Status)
	leftovers := a.leftovers(s)
	dataDir := a.daemonDir()
	clean := func() {
		if len(leftovers) > 0 {
			_ = cleanControlFiles(dataDir, gid, leftovers)
		}
	}
	return func(ctx context.Context, c api) error {
		if stopped {
			if err := c.RemoveDownloadResult(ctx, gid); err != nil {
				return err
			}
			clean()
			return nil
		}
		if err := c.Remove(ctx, gid); err != nil {
			return err
		}
		// Deferred, so a purge that never lands still leaves nothing behind:
		// the download is stopped either way, and its bookkeeping is dead.
		defer clean()
		for {
			if err := c.RemoveDownloadResult(ctx, gid); err == nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return nil // best effort; give up rather than block forever
			case <-time.After(removePurgeInterval):
			}
		}
	}
}

// flash shows a transient status-line message.
func (a *App) flash(text string, isErr bool) tea.Cmd {
	a.status, a.statusErr = safeText(text), isErr
	prefix := "INFO: "
	if isErr {
		prefix = "ERROR: "
	}
	a.events = append(a.events, prefix+a.status)
	if len(a.events) > 50 {
		a.events = append([]string(nil), a.events[len(a.events)-50:]...)
	}
	a.statusSeq++
	return tea.Tick(4*time.Second, clearStatusAt(a.statusSeq))
}

// clearStatusAt builds the tick payload that clears status message seq.
func clearStatusAt(seq int) func(time.Time) tea.Msg {
	return func(time.Time) tea.Msg { return clearStatusMsg{seq} }
}

// saveConfig persists the config, surfacing a failure on the status line
// and returning it so callers can avoid masking it with a success flash.
func (a *App) saveConfig() error {
	err := config.Save(a.cfgPath, a.cfg)
	if err != nil {
		a.status, a.statusErr = "config save failed: "+err.Error(), true
	}
	return err
}

// reconnect tears down the current client and dials the active server.
func (a *App) reconnect() tea.Cmd {
	if a.client != nil {
		a.client.Close()
		a.client = nil
	}
	a.connected = false
	a.polling = false
	return a.connectCmd()
}

func (a *App) setTheme(name string) error {
	a.cfg.Theme = name
	a.styles = NewStyles(PaletteByName(name))
	return a.saveConfig()
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

// applySavedLimitsCmd re-applies the persisted global settings on connect, so
// they survive a managed-daemon restart. The speed caps are skipped when the
// scheduler is enabled (it owns the global limits); the seed defaults are
// independent of the scheduler and always applied when stored.
func (a *App) applySavedLimitsCmd() tea.Cmd {
	opts := map[string]string{}
	if !a.cfg.SchedulerEnabled {
		if a.cfg.GlobalDown != "" {
			opts["max-overall-download-limit"] = a.cfg.GlobalDown
		}
		if a.cfg.GlobalUp != "" {
			opts["max-overall-upload-limit"] = a.cfg.GlobalUp
		}
	}
	if a.cfg.SeedRatio != "" {
		opts["seed-ratio"] = a.cfg.SeedRatio
	}
	if a.cfg.SeedTime != "" {
		opts["seed-time"] = a.cfg.SeedTime
	}
	if len(opts) == 0 {
		return nil
	}
	return a.rpcCmd("settings restored", func(ctx context.Context, c api) error {
		return c.ChangeGlobalOption(ctx, opts)
	})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.list.clampCursor()
		if a.files.a != nil {
			a.files.clamp()
		}
		if a.browse.a != nil {
			a.browse.clamp()
		}
		if len(a.cfg.Rules) == 0 {
			a.scheduler.cursor = 0
		} else if a.scheduler.cursor >= len(a.cfg.Rules) {
			a.scheduler.cursor = len(a.cfg.Rules) - 1
		}
		if len(a.seeding.trackers) == 0 {
			a.seeding.tCursor = 0
		} else if a.seeding.tCursor >= len(a.seeding.trackers) {
			a.seeding.tCursor = len(a.seeding.trackers) - 1
		}
		return a, nil

	case connectedMsg:
		if a.client != nil && a.client != msg.client {
			a.client.Close() // overlapping connects must not leak the old client
		}
		a.client = msg.client
		a.polling = false
		a.version = msg.version
		a.connected = true
		a.connErr = nil
		a.endpoint = msg.endpoint
		if msg.daemon != nil {
			if a.daemon != nil && a.daemon != msg.daemon {
				stale := a.daemon
				go func() { // Stop blocks for seconds; keep the loop responsive
					ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
					defer cancel()
					_ = stale.Stop(ctx)
				}()
			}
			a.daemon = msg.daemon
		}
		a.lastSchedKey = ""     // re-apply schedule on the new server
		a.stoppedSeeded = false // reseed: the new server's history is not news
		a.knownStopped = nil
		a.metaCleaned = map[string]bool{}
		a.pendingMagnets = map[string]bool{}
		a.magnetQueue = nil
		a.picks = a.loadPicks() // restore unanswered file pickers
		a.picksReconciled = false
		return a, tea.Batch(a.requestPoll(), a.listenCmd(), a.applySavedLimitsCmd())

	case connectErrMsg:
		a.connected = false
		a.polling = false
		a.connErr = msg.err
		if msg.daemon != nil {
			a.daemon = msg.daemon // keep the spawned child for reuse and Shutdown
		}
		return a, nil

	case tickMsg:
		cmds := []tea.Cmd{tickCmdAfter(a.pollInterval())}
		if a.client != nil {
			cmds = append(cmds, a.requestPoll())
		} else if a.connErr != nil {
			cmds = append(cmds, a.connectCmd()) // simple 1s retry
		}
		if cmd := a.applySchedule(time.Time(msg)); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case pollMsg:
		if msg.seq != 0 && msg.seq != a.pollSeq {
			return a, nil
		}
		a.polling = false
		if msg.err != nil {
			// Treat a failed poll as a lost connection: drop the client so
			// the tick loop redials (and respawns the daemon if it died).
			a.connected = false
			a.connErr = msg.err
			if a.client != nil {
				a.client.Close()
				a.client = nil
			}
			return a, nil
		}
		a.connected = true
		msg.snap = safeSnapshot(msg.snap)
		if a.list.reordering {
			msg.snap.Waiting = a.list.frozenWaiting(msg.snap.Waiting)
		}
		// Restore any file pickers left unanswered before a previous quit
		// (once, against the first snapshot after connect).
		a.reconcilePicks(msg.snap)
		// A magnet whose metadata just resolved: open the picker on the real
		// (paused) torrent so files are chosen before anything downloads. Read
		// followedBy from the raw snapshot, before metadata is stripped below.
		magnetCmd := a.resolveMagnets(msg.snap)
		// Drop finished magnet-metadata leftovers: they'd otherwise sit in the
		// stopped list as a bare hash. Hide them and purge them from aria2.
		kept, metaGone := a.stripMetadata(msg.snap.Stopped)
		msg.snap.Stopped = kept
		notice := a.noticeStopped(msg.snap.Stopped)
		a.snap = msg.snap
		a.downHist.Push(msg.snap.Stat.DownSpeed())
		a.upHist.Push(msg.snap.Stat.UpSpeed())
		a.list.clampCursor()
		var cmd tea.Cmd
		if a.screen == screenDetail {
			cmd = a.detail.refreshCmd()
		}
		// Present the next queued magnet if the list is idle now (also drains
		// the queue after the user closes a previous picker).
		return a, tea.Batch(cmd, notice, magnetCmd, a.presentNextMagnet(), a.removeResultsCmd(metaGone))

	case notifMsg:
		cmds := []tea.Cmd{a.listenCmd()}
		if a.client != nil {
			cmds = append(cmds, a.requestPoll())
		}
		return a, tea.Batch(cmds...)

	case actionDoneMsg:
		if msg.err != nil {
			// A failed scheduler push must retry next tick, not wait for
			// the next rule change.
			a.lastSchedKey = ""
			return a, a.flash(msg.err.Error(), true)
		}
		cmds := []tea.Cmd{a.requestPoll()}
		if msg.text != "" {
			cmds = append(cmds, a.flash(msg.text, false))
		}
		return a, tea.Batch(cmds...)

	case addBatchDoneMsg:
		a.add.submitting = false
		if msg.err != nil {
			return a, a.flash(msg.err.Error(), true)
		}
		a.overlay = overlayNone
		return a, tea.Batch(a.requestPoll(), a.flash(msg.text, false))

	case clearStatusMsg:
		if msg.seq == a.statusSeq {
			a.status = ""
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
		a.detail.absorb(safeDetailData(msg))
		return a, nil

	case torrentAddedMsg:
		a.add.submitting = false
		if msg.err != nil {
			return a, a.flash(msg.err.Error(), true)
		}
		a.files = newFilesModel(a)
		a.files.gid = msg.gid
		a.files.fromAdd = true
		a.files.unpauseAfter = msg.unpause
		a.files.pickKey = msg.gid
		a.overlay = overlayFiles
		a.addPick(pendingPick{GID: msg.gid, Kind: "torrent", Unpause: msg.unpause})
		return a, tea.Batch(a.files.loadCmd(), a.flash("added — pick files", false))

	case magnetAddedMsg:
		a.add.submitting = false
		if msg.err != nil {
			return a, a.flash(msg.err.Error(), true)
		}
		a.overlay = overlayNone
		a.pendingMagnets[msg.gid] = msg.unpause
		a.addPick(pendingPick{GID: msg.gid, Kind: "magnet", Unpause: msg.unpause})
		return a, tea.Batch(a.requestPoll(), a.flash("magnet added — fetching metadata…", false))

	case metalinkAddedMsg:
		a.add.submitting = false
		if msg.err != nil {
			return a, a.flash(msg.err.Error(), true)
		}
		if len(msg.gids) <= 1 { // single (or empty) download: nothing to pick
			a.overlay = overlayNone
			if msg.unpause && len(msg.gids) == 1 {
				gid := msg.gids[0]
				return a, a.rpcCmd("added", func(ctx context.Context, c api) error {
					return c.Unpause(ctx, gid)
				})
			}
			return a, a.flash("added (paused)", false)
		}
		a.files = newFilesModel(a)
		a.files.gids = msg.gids
		a.files.name = msg.name
		a.files.fromAdd = true
		a.files.unpauseAfter = msg.unpause
		a.overlay = overlayFiles
		return a, tea.Batch(a.files.loadCmd(), a.flash("added — pick files", false))

	case filesMultiMsg:
		if a.overlay == overlayFiles {
			for i := range msg.statuses {
				msg.statuses[i] = safeStatus(msg.statuses[i])
			}
			var cmd tea.Cmd
			a.files, cmd = a.files.absorbMulti(msg)
			return a, cmd
		}
		return a, nil

	case browseDataMsg:
		if a.overlay == overlayBrowse && filepath.Clean(msg.dir) == filepath.Clean(a.browse.dir) {
			a.browse.absorb(msg)
		}
		return a, nil

	case filesDataMsg:
		if a.overlay == overlayFiles {
			msg.dir = safeText(msg.dir)
			for i := range msg.files {
				msg.files[i].Path = safeText(msg.files[i].Path)
			}
			var cmd tea.Cmd
			a.files, cmd = a.files.absorb(msg)
			return a, cmd
		}
		return a, nil

	case filesRetryMsg:
		if a.overlay == overlayFiles && a.files.gid == msg.gid {
			return a, a.files.loadCmd()
		}
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

	case tea.MouseMsg:
		return a.handleMouse(msg)

	case tea.KeyMsg:
		return a.handleKey(safeKey(msg))
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return a, tea.Quit
	}
	if a.tooSmall() {
		if msg.String() == "q" {
			return a, tea.Quit
		}
		return a, nil
	}
	if msg.String() == "ctrl+p" && a.overlay == overlayNone {
		a.commands = newCommandModel(a)
		a.overlay = overlayCommands
		return a, a.commands.focusCmd()
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
		case overlayConfirm:
			a.confirm, cmd = a.confirm.update(msg)
		case overlayFiles:
			a.files, cmd = a.files.update(msg)
		case overlayBrowse:
			a.browse, cmd = a.browse.update(msg)
		case overlayHelp:
			a.help, cmd = a.help.update(msg)
		case overlayCommands:
			a.commands, cmd = a.commands.update(msg)
		}
		return a, cmd
	}
	// "?" opens help everywhere except while it would be typed into the
	// list filter.
	if msg.String() == "?" && !(a.screen == screenList && a.list.filtering) {
		a.overlay = overlayHelp
		return a, nil
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
	// Progress lands in atomics polled by the 1s render tick, keeping the
	// update loop quiet; a final message carries the outcome.
	return func() tea.Msg {
		ok, computed, err := checksum.Verify(path, expected, func(done, total int64) {
			v.Done.Store(done)
			v.Total.Store(total)
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

// yank copies the selected download's source — its first recorded URI, or a
// magnet link built from the info hash — to the system clipboard.
func (a *App) yank(s rpc.Status) tea.Cmd {
	src := ""
	for _, f := range s.Files {
		if len(f.URIs) > 0 {
			src = f.URIs[0].URI
			break
		}
	}
	if src == "" && s.InfoHash != "" {
		src = "magnet:?xt=urn:btih:" + s.InfoHash
	}
	if src == "" {
		return a.flash("no source URI recorded", true)
	}
	return func() tea.Msg {
		if err := clipboardWrite(src); err != nil {
			return actionDoneMsg{err: fmt.Errorf("clipboard: %w", err)}
		}
		return actionDoneMsg{text: "source copied"}
	}
}

// addURICmd adds uris. A lone magnet uses the pause-metadata → pick-before-start
// flow (so files are chosen before anything downloads); anything else is a
// plain add. Shared by the add overlay and the empty-screen quick-add.
func (a *App) addURICmd(uris []string, opts map[string]string, unpause bool) tea.Cmd {
	c := a.client
	if c == nil {
		return a.flash("not connected", true)
	}
	if len(uris) == 1 && strings.HasPrefix(uris[0], "magnet:") {
		magnet := uris[0]
		o := mergeOpts(opts, map[string]string{"pause-metadata": "true"})
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			gid, err := c.AddURI(ctx, []string{magnet}, o)
			return magnetAddedMsg{gid: gid, unpause: unpause, err: err}
		}
	}
	return a.rpcCmd("added", func(ctx context.Context, c api) error {
		_, err := c.AddURI(ctx, uris, opts)
		return err
	})
}

// pauseGidCmd pauses one download (best effort); used to guarantee a magnet's
// resolved torrent is paused before the picker opens, even if the server did
// not honour pause-metadata.
func (a *App) pauseGidCmd(gid string) tea.Cmd {
	c := a.client
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = c.Pause(ctx, gid)
		return actionDoneMsg{}
	}
}

// resolveMagnets watches pending magnets for their metadata to finish. Once a
// magnet's real (paused) torrent appears in followedBy, the picker opens on it
// so files are chosen before the download starts; if the UI is busy, it flashes
// instead and leaves the torrent paused.
func (a *App) resolveMagnets(snap snapshot) tea.Cmd {
	if len(a.pendingMagnets) == 0 {
		return nil
	}
	byGID := map[string]rpc.Status{}
	for _, lst := range [][]rpc.Status{snap.Active, snap.Waiting, snap.Stopped} {
		for _, s := range lst {
			byGID[s.GID] = s
		}
	}
	var cmds []tea.Cmd
	for gid, unpause := range a.pendingMagnets {
		s, ok := byGID[gid]
		if !ok {
			continue // metadata download not in this poll yet
		}
		if s.Status == "error" {
			delete(a.pendingMagnets, gid)
			cmds = append(cmds, a.flash("magnet metadata failed", true))
			continue
		}
		// Metadata already resolved into this very download rather than a
		// followedBy child: aria2 loaded a saved .torrent (--bt-load-saved-metadata)
		// at startup or on a re-add, so the pending gid *is* the paused torrent
		// now, with no child to wait for. Present the picker on it directly.
		if len(s.FollowedBy) == 0 && s.IsTorrent() && !s.IsMetadata() && s.Status == "paused" {
			delete(a.pendingMagnets, gid)
			a.magnetQueue = append(a.magnetQueue, magnetReady{gid: gid, unpause: unpause, parent: gid})
			continue
		}
		if len(s.FollowedBy) == 0 {
			continue // metadata still downloading
		}
		delete(a.pendingMagnets, gid)
		child := s.FollowedBy[0]
		// Guarantee the torrent is paused before we present it, even if the
		// server ignored pause-metadata, then queue it for the picker.
		cmds = append(cmds, a.pauseGidCmd(child))
		a.magnetQueue = append(a.magnetQueue, magnetReady{gid: child, unpause: unpause, parent: gid})
	}
	return tea.Batch(cmds...)
}

// presentNextMagnet opens the picker for the next resolved magnet, but only
// when the list is idle — so magnets that finish while an earlier picker is
// still open (or while the user is elsewhere) wait their turn, paused, and are
// shown one at a time rather than stacking or being dropped.
func (a *App) presentNextMagnet() tea.Cmd {
	if len(a.magnetQueue) == 0 || a.overlay != overlayNone || a.screen != screenList {
		return nil
	}
	next := a.magnetQueue[0]
	a.magnetQueue = a.magnetQueue[1:]
	a.files = newFilesModel(a)
	a.files.gid = next.gid
	a.files.fromAdd = true
	a.files.unpauseAfter = next.unpause
	a.files.pickKey = next.parent
	a.files.moreQueued = len(a.magnetQueue)
	a.overlay = overlayFiles
	return a.files.loadCmd()
}

// stripMetadata removes finished magnet-metadata placeholders from a stopped
// list, returning the kept entries and the gids to purge (each only once).
func (a *App) stripMetadata(stopped []rpc.Status) (kept []rpc.Status, purge []string) {
	for _, s := range stopped {
		if s.IsMetadata() && s.Status == "complete" {
			if !a.metaCleaned[s.GID] {
				a.metaCleaned[s.GID] = true
				purge = append(purge, s.GID)
			}
			continue
		}
		kept = append(kept, s)
	}
	return kept, purge
}

// removeResultsCmd clears the given result gids from aria2 (best effort).
func (a *App) removeResultsCmd(gids []string) tea.Cmd {
	if len(gids) == 0 {
		return nil
	}
	c := a.client
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, g := range gids {
			_ = c.RemoveDownloadResult(ctx, g)
		}
		return actionDoneMsg{}
	}
}

// noticeStopped diffs the stopped list against the previous poll and turns
// newly finished or failed downloads into a flash + terminal bell. The first
// poll after a (re)connect seeds the set silently, so history is not
// replayed. Works over both transports — ws push events merely make the next
// poll happen sooner.
func (a *App) noticeStopped(stopped []rpc.Status) tea.Cmd {
	if !a.stoppedSeeded {
		a.stoppedSeeded = true
		a.knownStopped = make(map[string]bool, len(stopped))
		for _, s := range stopped {
			a.knownStopped[s.GID] = true
		}
		return nil
	}
	var done, failed []rpc.Status
	for _, s := range stopped {
		if a.knownStopped[s.GID] {
			continue
		}
		a.knownStopped[s.GID] = true
		switch s.Status {
		case "complete":
			done = append(done, s)
			a.cleanControl(s)
		case "error":
			failed = append(failed, s)
		}
	}
	if len(done)+len(failed) == 0 {
		return nil
	}
	bell()
	extra := len(done) + len(failed) - 1
	more := ""
	if extra > 0 {
		more = fmt.Sprintf(" (+%d more)", extra)
	}
	if len(failed) > 0 {
		text := "✗ " + failed[0].Name() + " failed: " + friendlyError(failed[0].ErrorCode, failed[0].ErrorMessage)
		return a.flash(text+more, true)
	}
	return a.flash("✓ "+done[0].Name()+" finished"+more, false)
}

// cleanControlFiles is an indirection for tests over the disk deletion.
var cleanControlFiles = control.Clean

// daemonDir is where the managed daemon keeps its session, conf and the record
// of gids whose control file was deleted.
func (a *App) daemonDir() string {
	return filepath.Join(filepath.Dir(a.cfgPath), "daemon")
}

// leftovers is what aria2 keeps beside a download without ever listing it as a
// file of one: the .aria2 control file, and the <infohash>.torrent saved because
// the daemon runs with --bt-save-metadata. Nothing that acts on the download's
// files touches either, so they outlive it in the download folder. Empty when
// the user opted out of control-file cleanup.
func (a *App) leftovers(s rpc.Status) []string {
	if !a.cfg.CleanControl() {
		return nil
	}
	paths := make([]string, 0, len(s.Files))
	for _, f := range s.Files {
		paths = append(paths, f.Path)
	}
	name := ""
	if s.BitTorrent != nil {
		name = s.BitTorrent.Info.Name
	}
	return control.Leftovers(s.Dir, name, s.InfoHash, paths)
}

// cleanControl removes what a finished download left behind, unless the user
// opted out. A torrent only reaches the stopped list once it has also finished
// seeding, so this never strips a seeding torrent's piece state. Failures are
// silent: a leftover control file is not worth interrupting the completion
// flash with.
func (a *App) cleanControl(s rpc.Status) {
	paths := a.leftovers(s)
	if len(paths) == 0 {
		return
	}
	_ = cleanControlFiles(a.daemonDir(), s.GID, paths)
}

// openDirBin is the file-manager launcher; a var so tests can substitute a
// harmless command.
var openDirBin = openBinFor(runtime.GOOS)
var absolutePath = filepath.Abs

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
	abs, err := absolutePath(dir)
	if err != nil {
		return a.flash("open directory: "+err.Error(), true)
	}
	bin := openDirBin
	return func() tea.Msg {
		if err := exec.Command(bin, abs).Start(); err != nil {
			return actionDoneMsg{err: fmt.Errorf("open %s: %w", abs, err)}
		}
		return actionDoneMsg{}
	}
}

// header renders the top bar shared by every screen.
func (a *App) header() string {
	st := a.styles
	srv := a.cfg.ActiveServer()
	conn := st.Green.Render("▪ connected")
	if !a.connected {
		conn = st.Red.Render("▪ disconnected")
	}
	endpoint := a.endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s:%d", srv.Host, srv.Port)
	}
	if srv.Managed {
		endpoint += " (built-in)"
	}
	left := lipgloss.JoinHorizontal(lipgloss.Center,
		st.Brand.Render("Aria2t"), st.Faint.Render(" │ "),
		st.Dim.Render(safeText(endpoint)+" "), conn)
	right := st.Cyan.Render("▼ "+FmtSpeed(a.snap.Stat.DownSpeed())) + " " +
		st.Magenta.Render("▲ "+FmtSpeed(a.snap.Stat.UpSpeed()))
	gap := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}
	return " " + left + lipgloss.NewStyle().Width(gap).Render("") + right
}

// statusLine renders the transient message, if any, on its own line.
// Callers append it after their keybar; without the leading newline the
// message would extend a full-width keybar past the terminal edge and be
// clipped invisibly.
func (a *App) statusLine() string {
	if a.status == "" {
		return ""
	}
	if a.statusErr {
		return "\n " + a.styles.Red.Render(a.status)
	}
	return "\n " + a.styles.Green.Render(a.status)
}

func (a *App) View() string {
	a.hits.reset()
	if a.tooSmall() {
		return a.tooSmallView()
	}
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
		a.hits.reset() // only the modal is interactive while it is open
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
		case overlayConfirm:
			modal = a.confirm.view()
		case overlayFiles:
			modal = a.files.view()
		case overlayBrowse:
			modal = a.browse.view()
		case overlayHelp:
			modal = a.help.view()
		case overlayCommands:
			modal = a.commands.view()
		}
		return a.outputView(a.composite(body, modal))
	}
	return a.outputView(body)
}

func (a *App) outputView(view string) string {
	if !a.accessible {
		return view
	}
	view = asciiText(ansi.Strip(view))
	if len(a.events) > 0 {
		start := max(0, len(a.events)-5)
		view += "\n\nActivity log\n" + strings.Join(a.events[start:], "\n")
	}
	return view
}

// composite centers the modal over the current screen, dimmed — the design's
// "modal over dimmed backdrop". Placement mirrors overlayOffset exactly, so the
// modal's registered click regions match what is drawn. The modal is
// transparent (no background fill) so it shows the terminal's own colours like
// the rest of the UI — "normal colours", not an opaque card that reads as too
// dark (Bg) or a clashing surface (Surface downsamples to blue). The backdrop
// is dimmed to a faint foreground so the bordered modal stands out.
func (a *App) composite(body, modal string) string {
	offX, offY := a.overlayOffset(modal)
	faint := a.styles.Faint
	rows := make([]string, a.height)
	bodyLines := strings.Split(body, "\n")
	for y := 0; y < a.height; y++ {
		line := ""
		if y < len(bodyLines) {
			line = ansi.Strip(bodyLines[y])
		}
		if cellWidth(line) > a.width {
			line = ansi.Truncate(line, a.width, "")
		}
		rows[y] = line
	}
	modalLines := strings.Split(modal, "\n")
	for i, ml := range modalLines {
		y := offY + i
		if y < 0 || y >= a.height {
			continue
		}
		left, right := "", ""
		plainWidth := cellWidth(rows[y])
		if offX <= plainWidth {
			left = ansi.Cut(rows[y], 0, offX)
		} else {
			left = rows[y] + strings.Repeat(" ", offX-plainWidth)
		}
		if end := offX + lipgloss.Width(ml); end < plainWidth {
			right = ansi.Cut(rows[y], end, plainWidth)
		}
		rows[y] = faint.Render(left) + ml + faint.Render(right)
	}
	for y := range rows {
		if offY <= y && y < offY+len(modalLines) {
			continue
		}
		rows[y] = faint.Render(rows[y])
	}
	return strings.Join(rows, "\n")
}

const minTermWidth, minTermHeight = 80, 24

func (a *App) tooSmall() bool {
	return a.width < minTermWidth || a.height < minTermHeight
}

func (a *App) tooSmallView() string {
	lines := []string{
		"Terminal too small",
		"",
		fmt.Sprintf("Required: at least %d × %d", minTermWidth, minTermHeight),
		fmt.Sprintf("Current:  %d × %d", a.width, a.height),
		"",
		"Resize the terminal to continue.",
		"Press q to quit.",
	}
	if a.height <= 0 || a.width <= 0 {
		return ""
	}
	if len(lines) > a.height {
		lines = lines[:a.height]
	}
	for len(lines) < a.height {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], a.width, "")
	}
	return strings.Join(lines, "\n")
}
