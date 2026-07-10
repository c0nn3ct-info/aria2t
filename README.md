[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<h1 align="center">aria2t</h1>

<p align="center"><strong>Terminal client for the aria2 download manager</strong></p>
<p align="center"><em>Manage downloads, torrents, magnet links, and Metalink from the terminal.</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License: Apache-2.0"></a>
</p>

<p align="center">
  <img alt="aria2t demo" src="./tui/docs/media/demo.gif" width="720">
</p>

> [!IMPORTANT]
> Downloads are performed by [aria2](https://aria2.github.io/); aria2t is its control panel. By default aria2t spawns and manages its own `aria2c` daemon, so no setup is required. Point it at an aria2 you already run and it connects as a regular RPC client. aria2t collects no analytics or telemetry and talks over the network only to the aria2 server you configure.

aria2t is a terminal client for the aria2 download manager, styled with the Tokyo Night palette. It talks to aria2 over JSON-RPC: either to a private daemon it spawns and manages itself, or to any aria2 already running on a seedbox, NAS, or remote machine. Every action is available from both the keyboard and the mouse, and the application builds into a single static Go binary.

## ✨ Features

- **Support for every aria2 source** — URL mirrors, `.torrent`, `.metalink`, magnet links, and aria2 input files.
- **No-setup start** — aria2t finds `aria2c`, spawns a private daemon, and manages its entire lifecycle; an external aria2 connects via the `--url` flag.
- **Download management** — pause and resume one or all, removal, queue reordering, speed limits per download or on a schedule.
- **Details for every download** — piece map, peers and mirror speeds, per-file progress, ratio, and BitTorrent seeding controls.
- **Integrity checking** — a finished file is checked against a sha-256 checksum and re-downloaded on mismatch; errors are described in plain language rather than codes.
- **State preserved across restarts** — finished and in-progress downloads are restored, as is a file-selection window closed without an answer.

## 📦 Supported sources

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2 input file`

aria2t can add anything aria2 accepts: plain URLs (multiple lines are treated as mirrors of the same file), magnet links and `.torrent` files with DHT and seeding support, `.metalink`, and aria2's `--input-file` batch format with separate options for each entry.

## 🧩 How it works

The aria2 daemon downloads the files; the interface controls it over JSON-RPC.

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

By default aria2t finds `aria2c` on your `PATH` and starts a private daemon on a random port with a random secret. The session is saved on exit and restored on the next launch, and the child process is stopped cleanly together with the program; a daemon left behind by a crash is shut down on the next start. If an external server is configured, the built-in daemon is not started and aria2t operates as a regular RPC client.

## 📥 Install

### Before you start

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — only for the built-in daemon; not needed to connect to an external server.
- A terminal with 256-color or truecolor support. A mouse is optional: every action is available from the keyboard.
- Go 1.25 or newer to build from source.

### Build and run

```sh
git clone https://github.com/c0nn3ct-info/aria2t.git
cd aria2t/tui && go build -o aria2t ./cmd/aria2t
./aria2t
```

On first launch aria2t starts the private daemon and opens an empty download list. The `a` key opens the add form, and `↵` adds a link from the clipboard.

### Connecting to an external aria2

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

Configuration is stored in `~/.config/aria2t/config.json` (the path can be overridden with `--config`); the managed daemon keeps its session under `~/.config/aria2t/daemon/`.

### Updating

Rebuild the binary and replace the old one. Configuration, scheduler rules, and the daemon session are stored under `~/.config/aria2t/` and survive the replacement.

### Uninstalling

1. Delete the binary.
2. Delete the data directory `~/.config/aria2t/` (configuration, daemon session, logs).

## ⌨️ Using aria2t

**Adding.** `a` opens the add form, prefilled from the clipboard. Several URIs, one per line, are treated as mirrors of the same file. `^t` switches the tabs (URL · `.torrent` · `.metalink` · input file), `^o` opens a file browser, and `^s` adds the download paused. On the empty first-run screen `↵` adds a link from the clipboard.

**Choosing files.** Multi-file torrents, Metalink, and magnet links open a file tree with checkboxes; a magnet link stays paused until the selection is confirmed. `space` marks a file or folder, `a` and `n` select all or none, `↵` confirms the selection. The selection can be changed later with `f`. If you quit before confirming, the window opens again on the next launch.

**The list.** `tab` or the `1`–`4` keys switch the All / Active / Waiting / Stopped tabs. `space` pauses and resumes the selected download, `P` and `U` do the same for all, `d` removes, `l` limits speed, `/` filters by name, `y` copies the source URL, `↵` opens the details. On the Waiting tab `J` and `K` move the selected item within the queue.

**Speed.** `l` limits the speed of the selected download; values are picked from presets. A global limit is set in the settings (`,`), saved, and re-applied after the daemon restarts. `S` opens the scheduler, where global limits are set by time of day — for example 5 MiB/s during work hours and unlimited at night.

**Integrity.** On the Stopped tab `c` stores the expected sha-256 checksum, `v` checks the local file against it, `R` re-downloads the file on mismatch, and `X` clears the list.

The full list of key bindings opens with **`?`**. Every hint in the bottom bar can also be clicked; in dialogs the green button confirms the action and the red one cancels.

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

A server with the `managed` field is the built-in daemon; its port and secret are chosen when it starts. The remaining entries describe external servers. `globalDown` and `globalUp` set the saved global speed limits, and `seedRatio` and `seedTime` the default seeding parameters; on connect all of these values are re-applied to the daemon. Scheduler rules (`S`) are stored in this file as well.

## ❓ FAQ

**What is aria2 and why does it need a separate interface?**
[aria2](https://aria2.github.io/) is a fast download engine supporting HTTP(S), FTP, SFTP, BitTorrent, and Metalink. It runs in the background, is controlled over JSON-RPC, and has no interface of its own beyond one-off commands. aria2t adds a real-time download list, file selection, queue management, speed limits, and integrity checking.

**How is this different from running `aria2c` directly?**
`aria2c` either downloads a file and exits, or runs as a daemon controlled by scripts. aria2t takes over managing that daemon: it starts it, saves and restores the session, stops it cleanly on exit, and shuts down a daemon left behind by a crash. On top of that aria2t provides an interactive interface.

**Does it work with an aria2 I already run?**
Yes. The `--url ws://host:6800/jsonrpc --secret …` flag or the built-in server switcher with latency probes connects aria2t to any aria2 over WebSocket or HTTP(S) — for example on a seedbox, a NAS, or in a Docker container. The built-in daemon is not started in that case.

**Are downloads preserved after quitting?**
Yes. The managed daemon saves its session on exit and restores it on the next launch: active, waiting, finished, and seeding downloads are restored, and already-downloaded files are recognized rather than downloaded again. A file-selection window closed without an answer also opens again.

**Can I choose which files of a torrent to download?**
Yes. Torrents, magnet links, and Metalink are added paused, and a file tree opens first; a magnet link is paused after its metadata arrives. Data transfer starts only after the selection is confirmed, and it can be changed later with `f`.

**Does it seed torrents?**
Yes. A finished torrent keeps seeding and its status changes to seeding. The global seeding parameters — ratio and time — are set in the settings and re-applied on every daemon restart.

**Keyboard or mouse?**
Both are fully supported. Every hint in the key bar can be clicked, every mouse action has a keyboard equivalent, and the full list of bindings opens with `?`.

**Does aria2t send any data anywhere?**
No. The program collects no analytics or telemetry, fetches no remote configuration, and talks over the network only to the aria2 server you configure.

## 🛠️ Development

Statement coverage in the module is **100%**, and this is checked automatically:

```sh
cd tui
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```

## 🙏 Acknowledgments

- **[aria2](https://github.com/aria2/aria2)** (GPL-2.0) — the download engine that does all the transfer, BitTorrent, and Metalink work; aria2t is its control panel.
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**, **[Bubbles](https://github.com/charmbracelet/bubbles)**, and **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** (MIT) — the Charm TUI stack aria2t is built on.
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — both palettes are taken unchanged from the Tokyo Night and Tokyo Night Day themes.
