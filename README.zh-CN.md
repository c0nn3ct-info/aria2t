[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./tui/docs/media/logo-dark.svg">
    <img alt="" src="./tui/docs/media/logo-light.svg" width="128">
  </picture>
</p>

<h1 align="center">Aria2t</h1>

<p align="center"><strong>aria2 下载管理器</strong></p>
<p align="center"><em>从终端或浏览器管理 aria2。</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License: Apache-2.0"></a>
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./tui/docs/media/demo.gif">
    <img alt="Aria2t 演示" src="./tui/docs/media/demo-light.gif" width="720">
  </picture>
</p>

> [!IMPORTANT]
> 下载由 [aria2](https://aria2.github.io/) 执行；Aria2t 是它的控制面板。默认情况下，Aria2t 会启动并管理自己的 `aria2c` 守护进程，因此无需任何配置。若指定一个已在运行的 aria2，它将作为普通 RPC 客户端接入。Aria2t 不收集数据分析和遥测，网络通信对象仅限于所配置的 aria2 服务器。

Aria2t 是支持终端和浏览器界面的 aria2 下载管理器。它可以启动专用守护进程，也能通过 JSON-RPC 连接运行在盒子、NAS 或远程机器上的 aria2。

## ✨ 功能

- **支持 aria2 的全部来源** — URL 镜像、`.torrent`、`.metalink`、磁力链接和 aria2 输入文件。
- **浏览器扩展** — Chrome 扩展把浏览器下载和磁力链接交给同一个守护进程，并可按大小、域名或文件类型过滤。安装说明见 <https://aria2t.c0nn3ct.info/extension>。
- **免配置启动** — Aria2t 找到 `aria2c`，启动私有守护进程并管理其完整生命周期；外部 aria2 通过 `--url` 选项接入。
- **下载管理** — 单个或全部的暂停与恢复、删除、队列重排、按下载或按计划的速度限制。
- **每个下载的详细信息** — 分片图、节点与镜像速度、单文件进度、分享率，以及 BitTorrent 做种控制。
- **完整性校验** — 已完成的文件与 sha-256 校验和比对，不匹配时重新下载；错误以通俗语言而非错误码描述。
- **跨重启的状态保持** — 已完成和进行中的下载得到恢复，未作答就关闭的文件选择窗口同样会重新打开。

## 📦 支持的来源

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2 输入文件`

aria2 接受的一切都可以添加到 Aria2t 中：普通 URL（多行视为同一文件的镜像）、支持 DHT 和做种的磁力链接与 `.torrent` 文件、`.metalink`，以及每条记录可带独立选项的 aria2 `--input-file` 批量格式。

## 🧩 工作原理

文件由 aria2 守护进程下载，前端通过 JSON-RPC 控制它。终端客户端和浏览器扩展就是这样的两个前端，它们操作的是同一个守护进程。

```
  Front ends                                Your machine
  ┌──────────────────┐                      ┌──────────────────┐
  │ terminal client  │──┐  JSON-RPC ws://   │  managed aria2c  │
  │  Bubble Tea TUI  │  ├──────────────────▶│ spawned · reaped │
  └──────────────────┘  │   push + poll     └──────────────────┘
  ┌──────────────────┐  │                            │ downloads · seeds
  │ Chrome extension │──┘                            │
  │ capture · picker │                               ▼
  └────────┬─────────┘                      ┌──────────────────┐
           │  ws:// or http(s)://           │   HTTP mirrors   │
           ▼                                │ BitTorrent · DHT │
  ┌──────────────────┐      downloads       │     Metalink     │
  │  external aria2  │─────────────────────▶│                  │
  │  seedbox · NAS   │                      └──────────────────┘
  └──────────────────┘
```

默认情况下，Aria2t 在 `PATH` 中找到 `aria2c`，并在随机端口上以随机密钥启动私有守护进程。会话在退出时保存、下次启动时恢复；子进程随程序一同干净地停止，崩溃遗留的守护进程会在下次启动时被终止。若配置了外部服务器，内置守护进程不会启动，Aria2t 作为普通 RPC 客户端工作。

## 📥 安装

### 开始之前

- [aria2](https://aria2.github.io/)（`brew install aria2` / `apt install aria2`）— 仅内置守护进程需要；连接外部服务器时不需要。
- 支持 256 色或真彩色的终端。鼠标可选：所有操作均可通过键盘完成。
- 从源码构建需要 Go 1.25 或更新版本。

### 构建并运行

```sh
git clone https://github.com/c0nn3ct-info/aria2t.git
cd aria2t/tui && go build -o aria2t ./cmd/aria2t
./aria2t
```

首次启动时，Aria2t 会启动私有守护进程并打开一个空的下载列表。按 `a` 打开添加表单，按 `↵` 添加剪贴板中的链接。

### 连接外部 aria2

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

配置保存在 `~/.config/aria2t/config.json`（路径可用 `--config` 覆盖）；托管守护进程的会话保存在 `~/.config/aria2t/daemon/` 下。

### 更新

重新构建二进制文件并替换旧的。配置、调度规则和守护进程会话保存在 `~/.config/aria2t/` 下，替换后不会丢失。

### 卸载

1. 删除二进制文件。
2. 删除数据目录 `~/.config/aria2t/`（配置、守护进程会话、日志）。

## ⌨️ 使用

**添加。** `a` 打开添加表单，并从剪贴板预填。多个 URI 每行一个，视为同一文件的镜像。`^t` 切换标签页（URL · `.torrent` · `.metalink` · 输入文件），`^o` 打开文件浏览器，`^s` 以暂停状态添加下载。在首次运行的空界面上，`↵` 添加剪贴板中的链接。

**选择文件。** 多文件种子、Metalink 和磁力链接会打开带复选框的文件树；磁力链接在选择确认之前保持暂停。`space` 勾选文件或文件夹，`a` 和 `n` 全选或全不选，`↵` 确认选择。之后可用 `f` 更改选择。若在确认前退出程序，窗口会在下次启动时重新打开。

**列表。** `tab` 或 `1`–`4` 键切换 All / Active / Waiting / Stopped 标签页。`space` 暂停和恢复所选下载，`P` 和 `U` 对全部执行相同操作，`d` 删除，`l` 限速，`/` 按名称过滤，`y` 复制来源 URL，`↵` 打开详情。在 Waiting 标签页，`J` 和 `K` 在队列内移动所选项。

**速度。** `l` 限制所选下载的速度；数值从预设中选取。全局上限在设置（`,`）中指定，会被保存并在守护进程重启后重新应用。`S` 打开调度器，按时段设定全局限制，例如工作时间 5 MiB/s、夜间不限速。

**完整性。** 在 Stopped 标签页，`c` 保存期望的 sha-256 校验和，`v` 用它比对本地文件，`R` 在不匹配时重新下载文件，`X` 清空列表。

按 **`?`** 打开完整的按键列表。底栏的所有提示同样可以点击；对话框中绿色按钮确认操作，红色按钮取消。

## ⚙️ 配置

`~/.config/aria2t/config.json`（省略的字段保持默认值）：

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

带 `managed` 字段的服务器是内置守护进程；其端口和密钥在启动时确定。其余条目描述外部服务器。`globalDown` 和 `globalUp` 设定保存的全局速度限制，`seedRatio` 和 `seedTime` 设定默认做种参数；连接时这些值都会重新应用到守护进程。调度规则（`S`）也保存在此文件中。

## ❓ 常见问题

**什么是 aria2？为什么它需要一个独立的界面？**
[aria2](https://aria2.github.io/) 是一个快速的下载引擎，支持 HTTP(S)、FTP、SFTP、BitTorrent 和 Metalink。它在后台运行，通过 JSON-RPC 控制，除一次性命令外没有自己的界面。Aria2t 为它补充了实时下载列表、文件选择、队列管理、速度限制和完整性校验。

**这与直接运行 `aria2c` 有什么区别？**
`aria2c` 要么下载一个文件后退出，要么作为由脚本控制的守护进程运行。Aria2t 接管了该守护进程的管理：启动它、保存和恢复会话、退出时干净地停止它、终止崩溃遗留的守护进程。在此之上，Aria2t 提供交互式界面。

**能用于我已经在运行的 aria2 吗？**
能。`--url ws://host:6800/jsonrpc --secret …` 选项或带延迟探测的内置服务器切换器，可将 Aria2t 通过 WebSocket 或 HTTP(S) 接入任意 aria2，例如运行在盒子、NAS 或 Docker 容器中的实例。此时内置守护进程不会启动。

**退出程序后下载还在吗？**
在。托管守护进程在退出时保存会话，并在下次启动时恢复：活动、等待、已完成和做种中的下载都会恢复，已下载完成的文件会被识别而不会重新下载。未作答就关闭的文件选择窗口也会重新打开。

**可以选择下载种子中的哪些文件吗？**
可以。种子、磁力链接和 Metalink 以暂停状态添加，并先打开文件树；磁力链接在收到元数据后暂停。数据传输仅在确认选择后开始，之后可用 `f` 更改选择。

**它会做种吗？**
会。已完成的种子继续做种，其状态变为 seeding。全局做种参数 — 分享率和时长 — 在设置中指定，并在守护进程每次重启时重新应用。

**键盘还是鼠标？**
两者都得到完整支持。按键栏的所有提示都可以点击，每个鼠标操作都有对应按键，完整的按键列表通过 `?` 打开。

**Aria2t 会向外发送任何数据吗？**
不会。程序不收集数据分析和遥测，不获取远程配置，网络通信对象仅限于所配置的 aria2 服务器。

## 🛠️ 开发

模块的语句覆盖率为 **100%**，且由自动检查保证：

```sh
cd tui
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```

## 🙏 致谢

- **[aria2](https://github.com/aria2/aria2)**（GPL-2.0）— 承担全部传输、BitTorrent 和 Metalink 工作的下载引擎；Aria2t 是它的控制面板。
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**、**[Bubbles](https://github.com/charmbracelet/bubbles)** 和 **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**（MIT）— Aria2t 构建于其上的 Charm TUI 栈。
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — 两套配色均原样取自 Tokyo Night 与 Tokyo Night Day 主题。
