# aria2t

A modern terminal UI for the [aria2](https://aria2.github.io/) download manager, in the Tokyo Night palette. Talks to aria2's JSON-RPC interface over WebSocket or HTTP.

## Quick start

```sh
go build -o aria2t ./cmd/aria2t
./aria2t
```

That's it — aria2t finds `aria2c`, spawns a private daemon on a free port with a random secret, and shuts it down (saving the session for next launch) when you quit. Install aria2 once with `brew install aria2` if you don't have it.

Connecting to an **external** server is the optional path:

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

or add servers in the switcher (`s` → `+`). Configuration persists to `~/.config/aria2t/config.json` (override with `--config`); the managed daemon keeps its session and log under `~/.config/aria2t/daemon/`.

## Features

- **Zero-setup**: built-in managed aria2c daemon — spawned on demand, session resumed between runs, stopped cleanly on quit
- Download list with **Active / Waiting / Stopped** tabs, live progress, speeds and ETAs
- Add downloads: URL mirrors, `.torrent` and `.metalink` files, target dir, connection count, start-paused
- Detail view: pieces map, BitTorrent peers, file selection, announce list
- Global stats: 60-second speed sparkline, session totals, bandwidth per download
- **Queue reorder** — grab a waiting download with `J`/`K`, drop with `↵`
- **Per-download throttle** popup with preset chips (`∞`/1M/5M/10M/custom)
- **Seeding control** per torrent: stop ratio, seed time, DHT/PEX/LPD/encryption, tracker editor
- **Checksum verification** for finished downloads (paste an expected sha-256, verify locally, re-download on mismatch)
- **Multi-server switcher** with latency probes (local box, seedbox, NAS…)
- **Bandwidth scheduler** — time-of-day limit rules enforced via `changeGlobalOption`
- Dark and light **Tokyo Night** themes (`T` to toggle)
- **Mouse support**: click tabs/rows/buttons/chips, double-click opens details, wheel scrolls, click hints in the key bar
- Long lists scroll with the cursor; `space` smart-toggles pause/resume; removals ask for confirmation; `?` shows a help overlay

## Key bindings

| Context | Keys |
|---|---|
| List | `a` add · `space` pause/resume · `d` remove (confirms) · `↵` details · `g` stats · `l` limit · `s` servers · `S` scheduler · `t` seeding · `,` settings · `T` theme · `tab`/`1‑3` tabs · `?` help · `q` quit |
| Mouse | click = select/focus · double-click = open/connect · wheel = move · click tabs, chips, buttons, key-bar hints |
| Waiting tab | `J`/`K` grab + move · `gg`/`G` top/bottom · `↵` drop · `esc` cancel |
| Stopped tab | `c` paste checksum · `v` verify · `R` re-download · `o` open dir |
| Detail | `p` pause/resume · `d` remove · `f` file selection · `t` trackers · `o` open dir |
| Forms | `tab` next field · `space` toggle · `^s` save · `esc` back |

## Config example

```json
{
  "servers": [
    { "name": "local",   "host": "localhost",      "port": 6800, "secret": "…", "protocol": "ws" },
    { "name": "seedbox", "host": "sb.example.net", "port": 6800, "secret": "…", "protocol": "wss" }
  ],
  "active": 0,
  "theme": "dark",
  "schedulerEnabled": true,
  "rules": [
    { "start": "09:00", "end": "18:00", "days": [false,true,true,true,true,true,false],
      "label": "Working hours", "down": "5M", "up": "256K" }
  ]
}
```

## Development

The test suite holds **100% statement coverage** module-wide; keep it there:

```sh
go test ./...           # unit tests
go test ./... -coverprofile=cover.out -coverpkg=./... && go tool cover -func=cover.out | tail -1
go vet ./... && gofmt -l .

# integration test against a live daemon
aria2c --enable-rpc --rpc-listen-port=6899 --rpc-secret=smoketest --daemon
ARIA2T_SMOKE_URL=localhost:6899 ARIA2T_SMOKE_SECRET=smoketest \
  go test ./internal/rpc/ -run Integration -v
```

