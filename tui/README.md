# Aria2t (application)

The terminal front end of Aria2t, a download manager for [aria2](https://aria2.github.io/) — a Bubble Tea TUI that manages downloads, torrents, magnet links, and Metalink. It is one of two front ends onto the same aria2 daemon; the other is the browser extension, and this binary doubles as its native-messaging host. See the [root README](../README.md) for the full feature tour and the site at <https://aria2t.c0nn3ct.info>.

This is the application source, published under `tui/` in the public [c0nn3ct-info/aria2t](https://github.com/c0nn3ct-info/aria2t) repo — a snapshot synced from the private development repo. Issues and pull requests welcome on the public repo.

## Layout

- `cmd/aria2t` — entry point and flag handling.
- `internal/rpc` — typed aria2 JSON-RPC client (HTTP and WebSocket transports).
- `internal/ui` — the Bubble Tea application: screens, overlays, mouse hitmap, themes.
- `internal/daemon` — the managed `aria2c` child: spawn, session persistence, clean shutdown, stale-daemon reap.
- `internal/config` — JSON config under `~/.config/aria2t/`.
- `internal/sched` — speed-limit schedule resolution.
- `internal/checksum` — streaming sha-256 verification.

## Build

```bash
go build -o aria2t ./cmd/aria2t
```

Go 1.25 or newer. Stamp a version with `-ldflags "-X main.appVersion=v1.2.3"`. The managed daemon needs `aria2c` on `PATH` at runtime; connecting to an external aria2 does not.

## Terminal interaction

The full interface is keyboard-operable. Press `?` for contextual key help or
`Ctrl+P` to search and run commands. The supported functional minimum is
80 columns by 24 rows; below it Aria2t shows a resize message without changing
the current selection, focus, or unfinished form values.

Use `aria2t --accessible` for a keyboard-only, colour-free ASCII presentation.
It avoids the alternate screen and mouse capture, reduces idle refreshes, and
keeps recent status changes in a linear activity log for assistive technology.

## Test

```bash
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```

The module holds 100% statement coverage; any new statement needs a test in the same change.

## License

Apache-2.0 — see [`LICENSE`](../LICENSE). aria2 (GPL-2.0) runs as a separate process and is only spoken to over JSON-RPC.
