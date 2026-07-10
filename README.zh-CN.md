[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<h1 align="center">aria2t</h1>

<p align="center"><strong>aria2 下载管理器的终端界面</strong></p>
<p align="center"><em>下载、种子、磁力链接、Metalink —— 一个键鼠皆宜的 TUI，零配置。</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
</p>

<p align="center">
  <img alt="aria2t 演示" src="./docs/media/demo.gif" width="720">
</p>

> [!IMPORTANT]
> aria2t 是一个操控面板 —— 下载由 [aria2](https://aria2.github.io/) 完成。默认情况下，它会为你启动并托管一个私有的 `aria2c` 守护进程（零配置）；指向你已经在运行的 aria2 时，它就成为一个纯 RPC 客户端。没有数据分析，没有遥测 —— 它只与你配置的 aria2 端点通信。

aria2t 是 aria2 下载管理器的终端界面，采用 Tokyo Night 配色。它通过 JSON-RPC 与 aria2 通信 —— 既可以连接由它自己启动并托管的私有守护进程，也可以连接你已经跑在盒子、NAS 或远程机器上的任意 aria2 —— 每个操作都同时支持键盘**和**鼠标。一个静态 Go 二进制文件。

## ✨ 功能

- **接受 aria2 支持的一切** — URL 镜像、`.torrent`、`.metalink`、磁力链接、aria2 输入文件 — 并让你**在任何下载开始之前选择要下载哪些文件**。
- **零配置** — 找到 `aria2c`，启动一个私有守护进程，并托管其完整生命周期。或用 `--url` 将它指向一个外部 aria2。
- **掌控每个下载** — 单个或全部暂停/恢复、删除、重排队列、按下载或按时段限速。
- **展示下载的内部** — 分片图、节点与镜像速度、单文件进度、上传分享率，以及 BitTorrent 做种控制。
- **帮你把关** — 用 sha-256 校验已完成的文件，不匹配时重新下载；用通俗的语言而非错误码解释失败原因。
- **重启不丢失** — 已完成和进行中的下载都会回来，你未作答就关闭的文件选择器也会重新打开。
- **不碍事** — 有内容完成时响铃，边输入边过滤，用延迟探测切换服务器，并可切换深色/浅色主题。

## 📦 支持的来源

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2 输入文件`

aria2 接受的一切，aria2t 都能添加：普通 URL —— 每行一个表示同一文件的镜像 —— 支持 DHT 和做种的磁力链接与 `.torrent` 文件、`.metalink`，以及 aria2 自己的 `--input-file` 批量格式（带每个下载的独立选项）。多文件种子、磁力链接和 metalink 会在下载开始**之前**打开一个复选框树 —— 绝不会拉取任何不需要的内容。

## 🖥️ 界面

| | |
|---|---|
| ![下载列表](./docs/media/list-active.png) 列表 — 实时进度、速度，以及彩色 STATUS 列 | ![文件选择器](./docs/media/files-picker.png) 在任何下载开始前先选择文件 |
| ![详情](./docs/media/detail.png) 详情 — 分片图、节点/镜像、单文件进度 | ![添加](./docs/media/add-overlay.png) 添加 — 剪贴板预填，绿色/红色按钮 |
| ![完整性](./docs/media/stopped-integrity.png) sha-256 校验 + 通俗易懂的错误说明 | ![调度器](./docs/media/scheduler.png) 按时段的带宽调度器 |

## 🧩 工作原理

TUI 本身从不下载任何东西 —— 下载由 aria2 守护进程完成，两者之间流动的只有 JSON-RPC。

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

默认情况下，aria2t 在你的 `PATH` 中找到 `aria2c`，并在一个随机端口上用随机密钥运行一个私有守护进程：完整的会话在退出时保存、下次启动时恢复，退出时子进程会被干净地停止（崩溃遗留的守护进程会在下次启动时被回收）。改为将它指向外部服务器时，内置守护进程完全不会启动 — aria2t 就成了一个纯 RPC 客户端。

## 📥 安装

### 开始之前

- [aria2](https://aria2.github.io/)（`brew install aria2` / `apt install aria2`）— 仅零配置模式需要；连接外部服务器时不需要。
- 支持 256 色或真彩色的终端。鼠标可选；每个操作都有按键。
- 从源码构建需要 Go 1.25+。

### 构建并运行

```sh
go build -o aria2t ./cmd/aria2t     # or: go install aria2t/cmd/aria2t
./aria2t
```

首次启动会拉起私有守护进程，并把你带入一个空列表，按 `a` 即可开始 —— 或按 `↵` 直接从剪贴板添加链接。

### 连接外部 aria2

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

配置持久化在 `~/.config/aria2t/config.json`（`--config` 可覆盖）；托管守护进程把它的会话保存在 `~/.config/aria2t/daemon/` 下。

### 更新

重新构建并替换二进制文件即可 —— 配置、调度规则和守护进程会话都保存在 `~/.config/aria2t/` 下，替换后不会丢失。

### 卸载

1. 删除二进制文件。
2. 删除数据目录：`~/.config/aria2t/`（配置、守护进程会话、日志）。

## ⌨️ 使用

**添加。** `a` 打开添加表单，并从剪贴板预填。每行一个 URI 表示同一文件的镜像；`^t` 循环切换标签页（URL · `.torrent` · `.metalink` · 输入文件），`^o` 浏览磁盘选择文件，`^s` 切换暂停启动。在空的首次运行界面，`↵` 直接从剪贴板添加链接。

**选择文件。** 多文件种子、metalink 和磁力链接会**在任何下载开始之前**打开一个复选框树（磁力链接在你完成选择之前保持暂停）：`space` 切换单个文件或文件夹，`a`/`n` 全选/全不选，`↵` 确认。之后按 `f` 可更改选择。在选择之前退出，下次启动时选择器会重新打开 — 绝不会拉取任何不需要的内容。

**列表操作。** `tab` 或 `1`–`4` 在 全部 / Active / Waiting / Stopped 之间切换。`space` 暂停/恢复所选行，`P`/`U` 对全部操作，`d` 删除，`l` 限速，`/` 边输入边过滤，`y` 复制来源 URL，`↵` 打开详情。在 Waiting 标签页，`J`/`K` 抓取并在队列中移动某一项。

**带宽。** `l` 用预设选项限速所选下载。在设置（`,`）中设定的全局上限会被保存，并在守护进程重启时重新应用。`S` 按时段调度全局限速（“工作时 5 MiB/s，夜间不限速”）。

**完整性。** 在 Stopped 标签页：`c` 保存期望的 sha-256，`v` 校验本地文件，`R` 在不匹配时重新下载，`X` 清空列表。

随时按 **`?`** 查看完整的按键映射。每个快捷键栏提示也都可点击，对话框使用绿色（继续）/ 红色（取消）按钮。

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

带 `managed` 的服务器是内置守护进程（端口和密钥在启动时决定）；其余都是外部端点。`globalDown`/`globalUp` 是保存的总体速度上限，`seedRatio`/`seedTime` 是做种默认值 — 全部在连接时重新应用到守护进程。调度器规则（`S`）也保存在这里。

## ❓ 常见问题

**什么是 aria2？为什么要给它做一个 TUI？**
[aria2](https://aria2.github.io/) 是一个快速的多协议下载引擎 —— HTTP(S)、FTP、SFTP、BitTorrent、Metalink —— 无界面运行，讲 JSON-RPC。除了一次性命令，它没有自己的界面。aria2t 就是那个界面：实时列表、文件选择、队列重排、限速和完整性校验，全部在一个全屏 TUI 里。

**这和直接运行 `aria2c` 有什么区别？**
`aria2c` 要么下载一个 URL 然后退出，要么作为守护进程运行、靠脚本操控。aria2t 替你管理这个守护进程 —— 启动、会话保存/恢复、干净关闭、回收崩溃遗留的守护进程 —— 并在其上提供全部交互操作。

**能用于我已经在运行的 aria2 吗？**
能。`--url ws://host:6800/jsonrpc --secret …`（或带延迟探测的应用内服务器切换器）可通过 WebSocket 或 HTTP(S) 连接任意 aria2 —— 盒子、NAS、Docker 容器。此时内置守护进程不会启动。

**退出后下载还在吗？**
在。托管守护进程在退出时保存完整会话，下次启动时恢复 —— 进行中、排队、已完成和做种中的下载全部回来，已完成的文件会被识别而不是重新下载。就连你未作答就关闭的文件选择器也会重新打开。

**能在种子下载前选择里面的文件吗？**
能 —— 种子、磁力链接和 metalink 以暂停状态添加，先打开一个复选框树（磁力链接等元数据到达后再暂停）。在你确认之前不会传输任何数据，之后按 `f` 还能重新打开选择。

**它做种吗？**
做。完成的种子会继续上传，状态变为独立的 *seeding*；全局分享率/时长默认值在设置中配置，并在守护进程每次重启时重新应用。

**键盘还是鼠标？**
都行，完整支持：每个快捷键栏提示都可点击，每次点击都有等价按键。`?` 显示完整映射。

**aria2t 会向外发送数据吗？**
不会。没有数据分析、没有遥测、没有远程配置 —— 唯一的网络通信对象就是你配置的 aria2 端点。

## 🛠️ 开发

整个模块保持 **100% 语句覆盖率**，并强制执行：

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```
## 🙏 致谢

- **[aria2](https://github.com/aria2/aria2)**（GPL-2.0）— 完成全部实际传输、BitTorrent 和 Metalink 工作的下载引擎。aria2t 是操控面板；干活的是 aria2。
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**、**[Bubbles](https://github.com/charmbracelet/bubbles)** 和 **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**（MIT）— aria2t 所构建于的 Charm TUI 栈。
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — 两套配色均原样取自 Tokyo Night / Tokyo Night Day。
