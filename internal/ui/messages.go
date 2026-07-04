package ui

import (
	"context"
	"time"

	"aria2t/internal/daemon"
	"aria2t/internal/rpc"
)

// api is the slice of rpc.Client the UI consumes; tests substitute fakes.
type api interface {
	TellActive(ctx context.Context) ([]rpc.Status, error)
	TellWaiting(ctx context.Context, offset, num int) ([]rpc.Status, error)
	TellStopped(ctx context.Context, offset, num int) ([]rpc.Status, error)
	TellStatus(ctx context.Context, gid string) (rpc.Status, error)
	AddURI(ctx context.Context, uris []string, opts map[string]string) (string, error)
	AddTorrent(ctx context.Context, b64 string, opts map[string]string) (string, error)
	AddMetalink(ctx context.Context, b64 string, opts map[string]string) ([]string, error)
	Pause(ctx context.Context, gid string) error
	PauseAll(ctx context.Context) error
	Unpause(ctx context.Context, gid string) error
	UnpauseAll(ctx context.Context) error
	Remove(ctx context.Context, gid string) error
	RemoveDownloadResult(ctx context.Context, gid string) error
	PurgeDownloadResult(ctx context.Context) error
	ChangePosition(ctx context.Context, gid string, pos int, how string) (int, error)
	ChangeOption(ctx context.Context, gid string, opts map[string]string) error
	GetOption(ctx context.Context, gid string) (map[string]string, error)
	ChangeGlobalOption(ctx context.Context, opts map[string]string) error
	GetGlobalOption(ctx context.Context) (map[string]string, error)
	GetGlobalStat(ctx context.Context) (rpc.GlobalStat, error)
	GetPeers(ctx context.Context, gid string) ([]rpc.Peer, error)
	GetVersion(ctx context.Context) (string, error)
	Notifications() <-chan rpc.Notification
	Close() error
}

type tickMsg time.Time

// snapshot is one polling round of everything the list and stats need.
type snapshot struct {
	Active  []rpc.Status
	Waiting []rpc.Status
	Stopped []rpc.Status
	Stat    rpc.GlobalStat
	Taken   time.Time
}

type pollMsg struct {
	snap snapshot
	err  error
}

type connectedMsg struct {
	client   api
	version  string
	endpoint string         // host:port actually reached
	daemon   *daemon.Daemon // non-nil when a managed daemon was spawned
}

type connectErrMsg struct {
	err    error
	daemon *daemon.Daemon // daemon spawned before the failure, so it can be reused/stopped
}

type notifMsg rpc.Notification

// statusMsg is a transient status-line message.
type statusMsg struct {
	text  string
	isErr bool
}

type clearStatusMsg struct{ seq int }

// actionDoneMsg reports an RPC action; empty text means silent success.
type actionDoneMsg struct {
	text string
	err  error
}

type detailDataMsg struct {
	status rpc.Status
	peers  []rpc.Peer
	err    error
}

type gidOptionsMsg struct {
	gid  string
	opts map[string]string
	err  error
}

type globalOptionsMsg struct {
	opts map[string]string
	err  error
}

type verifyDoneMsg struct {
	gid      string
	ok       bool
	computed string
	err      error
}

type latencyMsg struct {
	index int
	d     time.Duration
	err   error
}
