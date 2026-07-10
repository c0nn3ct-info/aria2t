[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<h1 align="center">aria2t</h1>

<p align="center"><strong>Terminal UI for the aria2 download manager</strong></p>
<p align="center"><em>Downloads, torrents, magnets, metalinks — one keyboard-and-mouse TUI, zero setup.</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
</p>

<p align="center">
  <img alt="aria2t demo" src="./docs/media/demo.gif" width="720">
</p>

> [!IMPORTANT]
> aria2t is a control surface — [aria2](https://aria2.github.io/) does the downloading. By default it spawns and manages a private `aria2c` daemon for you (zero setup); pointed at an aria2 you already run, it becomes a pure RPC client. No analytics, no telemetry — it talks only to the aria2 endpoint you configure.

aria2t is a terminal UI for the aria2 download manager, in the Tokyo Night palette. It talks to aria2 over JSON-RPC — to a private daemon it spawns and manages for you, or to any aria2 you already run on a seedbox, NAS, or remote box — and every action works by keyboard **and** mouse. One static Go binary.

## ✨ Features

- **Adds anything aria2 takes** — URL mirrors, `.torrent`, `.metalink`, magnet links, aria2 input files — and lets you **choose which files to download before anything starts**.
- **Zero setup** — finds `aria2c`, spawns a private daemon, and manages its whole lifecycle. Or point it at an external aria2 with `--url`.
- **Drives every download** — pause/resume one or all, remove, reorder the queue, throttle per download or on a time-of-day schedule.
- **Shows the inside of a download** — per-piece map, peers and mirror speeds, per-file progress, upload ratio, and BitTorrent seeding controls.
- **Keeps you honest** — verifies a finished file against a sha-256 and re-downloads on mismatch; explains failures in plain English instead of error codes.
- **Survives restarts** — completed and in-progress downloads come back, and a file picker you closed without answering reopens.
- **Stays out of the way** — rings the bell when something finishes, filters as you type, switches servers with latency probes, and toggles a dark/light theme.

## 📦 Supported sources

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2 input file`

Anything aria2 accepts, aria2t adds: plain URLs — one per line means mirrors of the same file — magnet links and `.torrent` files with DHT and seeding, `.metalink`, and aria2's own `--input-file` batch format with per-download options. Multi-file torrents, magnets, and metalinks open a checkbox tree **before** the download starts, so nothing unwanted is ever fetched.

## 🖥️ Screens

| | |
|---|---|
| ![Download list](./docs/media/list-active.png) The list — live progress, speeds, a colored STATUS column | ![File picker](./docs/media/files-picker.png) Pick files before anything downloads |
| ![Detail](./docs/media/detail.png) Detail — piece map, peers/mirrors, per-file progress | ![Add](./docs/media/add-overlay.png) Add — clipboard-prefilled, green/red buttons |
| ![Integrity](./docs/media/stopped-integrity.png) sha-256 verify + plain-English errors | ![Scheduler](./docs/media/scheduler.png) Time-of-day bandwidth scheduler |

## 🧩 How it works

The TUI never downloads anything itself — an aria2 daemon does, and JSON-RPC is the only thing that flows between them.

```
  Terminal                                  Your machine
  ┌──────────────────┐    JSON-RPC ws://    ┌──────────────────┐
  │      aria2t      │ ◀──────────────────▶ │  managed aria2c  │
  │  Bubble Tea TUI  │     push + poll      │ spawned · reaped │
  └────────┬─────────┘                      └────────┬─────────┘
           │                                         │ downloads · seeds
           │  ws:// or http(s)://                    ▼
           ▼                                ┌──────────────────┐
  ┌──────────────────┐                      │   HTTP mirrors   │
  │  external aria2  │─────────────────────▶│ BitTorrent · DHT │
  │  seedbox · NAS   │      downloads       │     Metalink     │
  └──────────────────┘                      └──────────────────┘
```

By default aria2t finds `aria2c` on your `PATH` and runs a private daemon on a random port with a random secret: the full session is saved on exit and restored next launch, and the child is stopped cleanly when you quit (a crash-orphaned daemon is reaped on the next start). Point it at an external server instead and the built-in daemon never starts — aria2t becomes a pure RPC client.

## 📥 Install

### Before you start

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — only for the zero-setup mode; not needed to connect to an external server.
- A terminal with 256-color or truecolor support. Mouse is optional; every action has a key.
- Go 1.25+ to build from source.

### Build and run

```sh
go build -o aria2t ./cmd/aria2t     # or: go install aria2t/cmd/aria2t
./aria2t
```

The first launch spawns the private daemon and drops you in an empty list, ready for `a` — or `↵` to add a link straight from the clipboard.

### Connect to an external aria2

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

Config persists to `~/.config/aria2t/config.json` (`--config` overrides); the managed daemon keeps its session under `~/.config/aria2t/daemon/`.

### Updating

Rebuild and replace the binary — config, scheduler rules, and the daemon session all live under `~/.config/aria2t/` and survive the swap.

### Uninstalling

1. Delete the binary.
2. Delete the data directory: `~/.config/aria2t/` (config, daemon session, logs).

## ⌨️ Using aria2t

**Add.** `a` opens the add form, pre-filled from your clipboard. One URI per line means mirrors of the same file; `^t` cycles the tabs (URL · `.torrent` · `.metalink` · input file), `^o` browses the disk for a file, `^s` toggles start-paused. On the empty first-run screen, `↵` adds a link straight from the clipboard.

**Choose files.** Multi-file torrents, metalinks, and magnets open a checkbox tree **before anything downloads** (a magnet stays paused until you pick): `space` toggles a file or folder, `a`/`n` select all/none, `↵` confirms. Press `f` to change the selection later. Quit before choosing and the picker reopens next launch — nothing unwanted is ever fetched.

**Drive the list.** `tab` or `1`–`4` switch All / Active / Waiting / Stopped. `space` pauses/resumes the selected row, `P`/`U` do all, `d` removes, `l` limits, `/` filters as you type, `y` copies the source URL, `↵` opens details. On the Waiting tab, `J`/`K` grab and move an item in the queue.

**Bandwidth.** `l` throttles the selected download with preset chips. A global cap set in settings (`,`) is saved and re-applied when the daemon restarts. `S` schedules global limits by time of day ("5 MiB/s at work, unlimited at night").

**Integrity.** On the Stopped tab: `c` stores an expected sha-256, `v` verifies the local file, `R` re-downloads on mismatch, `X` clears the list.

Press **`?`** anytime for the full key map. Every key-bar hint is also clickable, and dialogs use green (proceed) / red (cancel) buttons.

## ⚙️ Configuration

`~/.config/aria2t/config.json` (fields you omit keep their defaults):

```json
{
  "servers": [
    { "name": "built-in", "host": "localhost", "protocol": "ws", "managed": true },
    { "name": "seedbox", "host": "sb.example.net", "port": 6800, "secret": "…", "protocol": "wss" }
  ],
  "active": 0,
  "theme": "dark",
  "dir": "~/Downloads",
  "split": 16,
  "globalDown": "5M", "globalUp": "512K",
  "seedRatio": "1.5", "seedTime": "0"
}
```

A `managed` server is the built-in daemon (port and secret are picked at spawn); anything else is an external endpoint. `globalDown`/`globalUp` are saved overall speed caps and `seedRatio`/`seedTime` the seeding defaults — all re-applied to the daemon on connect. Scheduler rules (`S`) are stored here too.

## ❓ FAQ

**What is aria2 and why put a TUI on it?**
[aria2](https://aria2.github.io/) is a fast, multi-protocol download engine — HTTP(S), FTP, SFTP, BitTorrent, Metalink — that runs headless and speaks JSON-RPC. It has no interface of its own beyond one-shot commands. aria2t is that interface: a live list, file picking, queue reordering, throttling, and integrity checks in one full-screen TUI.

**How is this different from just running `aria2c`?**
`aria2c` either downloads one URL and exits, or runs as a daemon you script against. aria2t manages that daemon for you — spawn, session save/restore, clean shutdown, reaping a crash-orphaned daemon — and puts every interactive action on top.

**Does it work with an aria2 I already run?**
Yes. `--url ws://host:6800/jsonrpc --secret …` (or the in-app server switcher, with latency probes) connects to any aria2 over WebSocket or HTTP(S) — seedbox, NAS, Docker container. The built-in daemon never starts.

**Do downloads survive quitting?**
Yes. The managed daemon saves its full session on exit and restores it on the next launch — active, queued, completed, and seeding downloads all come back, and finished files are recognised rather than re-downloaded. Even a file picker you quit without answering reopens.

**Can I choose files inside a torrent before it downloads?**
Yes — torrents, magnets, and metalinks are added paused and a checkbox tree opens first (a magnet waits for its metadata, then pauses). Nothing transfers until you confirm, and `f` reopens the selection later.

**Does it seed?**
Yes. A finished torrent keeps uploading and its status flips to a distinct *seeding*; global ratio/time defaults live in Settings and are re-applied whenever the daemon restarts.

**Keyboard or mouse?**
Both, fully: every key-bar hint is clickable and every click has a key equivalent. `?` shows the whole map.

**Does aria2t send anything anywhere?**
No. No analytics, no telemetry, no remote config — the only network peer is the aria2 endpoint you configure.

## 🛠️ Development

The module holds **100% statement coverage**, enforced:

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```


## 🙏 Acknowledgments

- **[aria2](https://github.com/aria2/aria2)** (GPL-2.0) — the download engine that does all the actual transfer, BitTorrent, and Metalink work. aria2t is a control surface; aria2 does the work.
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**, **[Bubbles](https://github.com/charmbracelet/bubbles)**, and **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** (MIT) — the Charm TUI stack aria2t is built on.
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — both palettes are taken verbatim from Tokyo Night / Tokyo Night Day.
