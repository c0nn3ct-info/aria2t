[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./tui/docs/media/logo-dark.svg">
    <img alt="" src="./tui/docs/media/logo-light.svg" width="128">
  </picture>
</p>

<h1 align="center">aria2t</h1>

<p align="center"><strong>ダウンロードマネージャー aria2 のターミナルクライアント</strong></p>
<p align="center"><em>ダウンロード・トレント・マグネットリンク・Metalink をターミナルから管理。</em></p>

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
    <img alt="aria2t デモ" src="./tui/docs/media/demo-light.gif" width="720">
  </picture>
</p>

> [!IMPORTANT]
> ダウンロードを実行するのは [aria2](https://aria2.github.io/) であり、aria2t はその操作パネルです。デフォルトでは aria2t が専用の `aria2c` デーモンを起動・管理するため、設定は不要です。既に稼働中の aria2 を指定すれば、通常の RPC クライアントとして接続します。aria2t はアナリティクスもテレメトリも収集せず、ネットワーク通信の相手は設定された aria2 サーバーのみです。

aria2t は、ダウンロードマネージャー aria2 のターミナルクライアントで、Tokyo Night カラーパレットを採用しています。aria2 とは JSON-RPC で通信します。相手は、自ら起動して管理する専用デーモン、またはシードボックス・NAS・リモートマシンで既に稼働している任意の aria2 です。すべての操作はキーボードとマウスの両方から行え、アプリケーションは単一の静的 Go バイナリにビルドされます。

## ✨ 機能

- **aria2 の全ソースに対応** — URL ミラー、`.torrent`、`.metalink`、マグネットリンク、aria2 入力ファイル。
- **設定不要の起動** — aria2t が `aria2c` を見つけ、専用デーモンを起動してそのライフサイクル全体を管理します。外部の aria2 へは `--url` オプションで接続します。
- **ダウンロード管理** — 個別または一括の一時停止と再開、削除、キューの並べ替え、ダウンロード単位またはスケジュールによる速度制限。
- **各ダウンロードの詳細** — ピースマップ、ピアとミラーの速度、ファイルごとの進捗、レシオ、BitTorrent シードの制御。
- **整合性の検証** — 完了したファイルを sha-256 チェックサムと照合し、不一致なら再ダウンロードします。エラーはコードではなく平易な言葉で説明されます。
- **再起動をまたぐ状態の保持** — 完了・進行中のダウンロードが復元され、未回答のまま閉じたファイル選択ウィンドウも再び開きます。

## 📦 対応ソース

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2 入力ファイル`

aria2t には aria2 が受け付けるものをすべて追加できます。通常の URL（複数行は同じファイルのミラーとして扱われます）、DHT とシードに対応したマグネットリンクと `.torrent`、`.metalink`、そしてエントリごとに個別オプションを持つ aria2 の `--input-file` バッチ形式です。

## 🧩 仕組み

ファイルをダウンロードするのは aria2 デーモンで、インターフェースはそれを JSON-RPC で制御します。

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

デフォルトでは、aria2t は `PATH` から `aria2c` を見つけ、ランダムなポートとランダムな secret で専用デーモンを起動します。セッションは終了時に保存され、次回起動時に復元されます。子プロセスはプログラムとともにクリーンに停止し、クラッシュで残ったデーモンは次回起動時に終了されます。外部サーバーが設定されている場合、内蔵デーモンは起動せず、aria2t は通常の RPC クライアントとして動作します。

## 📥 インストール

### はじめに

- [aria2](https://aria2.github.io/)（`brew install aria2` / `apt install aria2`）— 内蔵デーモンにのみ必要です。外部サーバーへの接続には不要です。
- 256 色または truecolor 対応のターミナル。マウスは任意です。すべての操作はキーボードから行えます。
- ソースからビルドする場合は Go 1.25 以降。

### ビルドと実行

```sh
git clone https://github.com/c0nn3ct-info/aria2t.git
cd aria2t/tui && go build -o aria2t ./cmd/aria2t
./aria2t
```

初回起動時に aria2t は専用デーモンを起動し、空のダウンロード一覧を開きます。`a` キーで追加フォームが開き、`↵` でクリップボードのリンクが追加されます。

### 外部の aria2 への接続

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

設定は `~/.config/aria2t/config.json` に保存されます（パスは `--config` で変更できます）。管理されるデーモンはセッションを `~/.config/aria2t/daemon/` 以下に保持します。

### 更新

バイナリを再ビルドして古いものと置き換えます。設定・スケジューラのルール・デーモンのセッションは `~/.config/aria2t/` 以下に保存されており、置き換え後も保持されます。

### アンインストール

1. バイナリを削除します。
2. データディレクトリ `~/.config/aria2t/`（設定、デーモンのセッション、ログ）を削除します。

## ⌨️ 使い方

**追加。** `a` で追加フォームが開き、クリップボードから自動入力されます。1 行に 1 つずつ並べた複数の URI は、同じファイルのミラーとして扱われます。`^t` でタブを切り替え（URL · `.torrent` · `.metalink` · 入力ファイル）、`^o` でファイルブラウザを開き、`^s` でダウンロードを一時停止状態で追加します。初回起動の何もない画面では、`↵` でクリップボードのリンクを追加できます。

**ファイルの選択。** 複数ファイルのトレント、Metalink、マグネットリンクは、チェックボックス付きのファイルツリーを開きます。マグネットリンクは選択が確定するまで一時停止のままです。`space` でファイルまたはフォルダを選択し、`a` と `n` で全選択・全解除、`↵` で選択を確定します。選択はあとから `f` で変更できます。確定前に終了した場合、ウィンドウは次回起動時に再び開きます。

**一覧。** `tab` または `1`–`4` キーで All / Active / Waiting / Stopped のタブを切り替えます。`space` は選択中のダウンロードを一時停止・再開し、`P` と `U` は全体に同じ操作を行い、`d` は削除、`l` は速度制限、`/` は名前での絞り込み、`y` はソース URL のコピー、`↵` は詳細を開きます。Waiting タブでは `J` と `K` で選択中の項目をキュー内で移動します。

**速度。** `l` は選択中のダウンロードの速度を制限します。値はプリセットから選びます。全体の上限は設定（`,`）で指定され、保存されてデーモンの再起動後にも再適用されます。`S` はスケジューラを開き、時間帯ごとに全体の制限を設定できます。たとえば勤務時間中は 5 MiB/s、夜間は無制限といった形です。

**整合性。** Stopped タブでは、`c` で期待する sha-256 チェックサムを保存し、`v` でローカルファイルと照合し、`R` で不一致時にファイルを再ダウンロードし、`X` で一覧をクリアします。

キー割り当ての全一覧は **`?`** で開きます。下部バーのヒントはすべてマウスクリックにも反応します。ダイアログでは緑のボタンが操作を確定し、赤のボタンがキャンセルします。

## ⚙️ 設定

`~/.config/aria2t/config.json`（省略した項目はデフォルト値を保ちます）:

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

`managed` フィールドを持つサーバーが内蔵デーモンです。ポートと secret はその起動時に選ばれます。残りのエントリは外部サーバーを表します。`globalDown` と `globalUp` は保存される全体の速度制限を、`seedRatio` と `seedTime` はシードのデフォルトパラメータを定めます。接続時にこれらの値はすべてデーモンへ再適用されます。スケジューラのルール（`S`）もこのファイルに保存されます。

## ❓ FAQ

**aria2 とは何ですか。なぜ別のインターフェースが必要なのですか。**
[aria2](https://aria2.github.io/) は HTTP(S)、FTP、SFTP、BitTorrent、Metalink に対応した高速なダウンロードエンジンです。バックグラウンドで動作し、JSON-RPC で制御され、単発コマンド以外に独自のインターフェースを持ちません。aria2t はそこに、リアルタイムのダウンロード一覧、ファイル選択、キュー管理、速度制限、整合性検証を加えます。

**`aria2c` を直接実行するのと何が違いますか。**
`aria2c` は、ファイルを 1 つダウンロードして終了するか、スクリプトで制御するデーモンとして動作するかのいずれかです。aria2t はそのデーモンの管理を引き受けます。起動し、セッションを保存・復元し、終了時にクリーンに停止し、クラッシュで残ったデーモンを終了します。その上で aria2t は対話的なインターフェースを提供します。

**既に稼働中の aria2 でも使えますか。**
はい。`--url ws://host:6800/jsonrpc --secret …` オプション、またはレイテンシ計測付きの内蔵サーバー切り替えで、WebSocket や HTTP(S) 経由の任意の aria2 に接続できます。たとえばシードボックス、NAS、Docker コンテナ内の aria2 です。この場合、内蔵デーモンは起動しません。

**終了後もダウンロードは保持されますか。**
はい。管理されるデーモンは終了時にセッションを保存し、次回起動時に復元します。アクティブ・待機中・完了・シード中のダウンロードが復元され、ダウンロード済みのファイルは認識されて再取得されません。未回答のまま閉じたファイル選択ウィンドウも再び開きます。

**トレントのどのファイルをダウンロードするか選べますか。**
はい。トレント、マグネットリンク、Metalink は一時停止状態で追加され、まずファイルツリーが開きます。マグネットリンクはメタデータの受信後に一時停止します。データ転送は選択の確定後にのみ始まり、選択はあとから `f` で変更できます。

**トレントのシードはしますか。**
はい。完了したトレントはシードを続け、ステータスが seeding に変わります。全体のシードパラメータ — レシオと時間 — は設定で指定され、デーモンの再起動のたびに再適用されます。

**キーボードとマウスのどちらで操作しますか。**
どちらも完全に対応しています。キーバーのヒントはすべてクリックでき、マウスの各操作にはキーボードの等価操作があり、キー割り当ての全一覧は `?` で開きます。

**aria2t は外部にデータを送信しますか。**
いいえ。プログラムはアナリティクスもテレメトリも収集せず、リモート設定も取得せず、ネットワーク通信の相手は設定された aria2 サーバーのみです。

## 🛠️ 開発

モジュールのステートメントカバレッジは **100%** で、これは自動的に検証されます:

```sh
cd tui
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```

## 🙏 謝辞

- **[aria2](https://github.com/aria2/aria2)**（GPL-2.0）— 転送・BitTorrent・Metalink のすべての作業を担うダウンロードエンジン。aria2t はその操作パネルです。
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**、**[Bubbles](https://github.com/charmbracelet/bubbles)**、**[Lip Gloss](https://github.com/charmbracelet/lipgloss)**（MIT）— aria2t の土台である Charm の TUI スタック。
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — 両テーマのパレットは Tokyo Night および Tokyo Night Day から変更なしで採用しています。
