[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<h1 align="center">aria2t</h1>

<p align="center"><strong>ダウンロードマネージャー aria2 のターミナル UI</strong></p>
<p align="center"><em>ダウンロード・トレント・マグネット・Metalink — キーボードとマウスで操る 1 つの TUI、設定不要。</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
</p>

<p align="center">
  <img alt="aria2t デモ" src="./docs/media/demo.gif" width="720">
</p>

> [!IMPORTANT]
> aria2t は操作パネルであり、ダウンロード自体は [aria2](https://aria2.github.io/) が行います。デフォルトでは専用の `aria2c` デーモンを自動で起動・管理し（設定不要）、既に動かしている aria2 を指定すれば純粋な RPC クライアントになります。アナリティクスもテレメトリもなし — 通信相手はあなたが設定した aria2 エンドポイントだけです。

aria2t は、ダウンロードマネージャー aria2 のターミナル UI で、Tokyo Night カラーパレットを採用しています。aria2 と JSON-RPC で通信します — 自分で起動して管理する専用デーモンにも、シードボックス・NAS・リモートマシンで既に動かしている任意の aria2 にも接続でき、すべての操作がキーボード **と** マウスの両方で行えます。単一の静的 Go バイナリです。

## ✨ 機能

- **aria2 が扱えるものは何でも追加** — URL ミラー、`.torrent`、`.metalink`、マグネットリンク、aria2 入力ファイル — そして **何かが始まる前に、ダウンロードするファイルを選べます**。
- **設定不要** — `aria2c` を見つけ、専用デーモンを起動し、そのライフサイクル全体を管理します。または `--url` で外部の aria2 を指定します。
- **あらゆるダウンロードを操作** — 1 件ずつまたは一括で一時停止／再開、削除、キューの並べ替え、ダウンロードごと・時間帯ごとの速度制限。
- **ダウンロードの内側を表示** — ピースマップ、ピアとミラーの速度、ファイルごとの進捗、アップロードレシオ、BitTorrent のシード制御。
- **確実さを保証** — 完了したファイルを sha-256 と照合し、不一致なら再ダウンロード。失敗はエラーコードではなく平易な言葉で説明します。
- **再起動しても失われない** — 完了したダウンロードも進行中のダウンロードも戻ってきて、答えずに閉じたファイルピッカーは再び開きます。
- **邪魔をしない** — 何かが完了するとベルを鳴らし、入力しながら絞り込み、レイテンシ計測付きでサーバーを切り替え、ダーク／ライトテーマを切り替えます。

## 📦 対応ソース

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2 入力ファイル`

aria2 が受け付けるものはすべて aria2t で追加できます: 素の URL（1 行 1 URI は同じファイルのミラー）、DHT とシード対応のマグネットリンクと `.torrent`、`.metalink`、そしてダウンロードごとのオプション付きの aria2 独自の `--input-file` バッチ形式。複数ファイルの torrent・マグネット・metalink は、ダウンロードが始まる **前に** チェックボックスツリーを開きます — 不要なものが取得されることはありません。

## 🖥️ 画面

| | |
|---|---|
| ![ダウンロード一覧](./docs/media/list-active.png) 一覧 — リアルタイムの進捗、速度、色付きの STATUS 列 | ![ファイル選択](./docs/media/files-picker.png) 何かがダウンロードされる前にファイルを選択 |
| ![詳細](./docs/media/detail.png) 詳細 — ピースマップ、ピア／ミラー、ファイルごとの進捗 | ![追加](./docs/media/add-overlay.png) 追加 — クリップボードから自動入力、緑／赤のボタン |
| ![整合性](./docs/media/stopped-integrity.png) sha-256 検証と平易な言葉のエラー | ![スケジューラ](./docs/media/scheduler.png) 時間帯別の帯域スケジューラ |

## 🧩 仕組み

TUI 自身は何もダウンロードしません — それは aria2 デーモンの仕事で、両者の間を流れるのは JSON-RPC だけです。

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

デフォルトでは、aria2t は `PATH` から `aria2c` を見つけ、ランダムなポートとランダムな secret で専用デーモンを実行します。セッション全体は終了時に保存され次回起動時に復元され、終了時に子プロセスはクリーンに停止します（クラッシュで取り残されたデーモンは次回起動時に回収されます）。代わりに外部サーバーを指定すると、内蔵デーモンは一切起動せず — aria2t は純粋な RPC クライアントになります。

## 📥 インストール

### はじめに

- [aria2](https://aria2.github.io/)（`brew install aria2` / `apt install aria2`）— 設定不要モードでのみ必要。外部サーバーに接続する場合は不要。
- 256 色または truecolor 対応のターミナル。マウスは任意で、すべての操作にキーがあります。
- ソースからビルドする場合は Go 1.25 以上。

### ビルドと実行

```sh
go build -o aria2t ./cmd/aria2t     # or: go install aria2t/cmd/aria2t
./aria2t
```

初回起動で専用デーモンが立ち上がり、`a` を押せばすぐ追加できる空の一覧が表示されます — `↵` ならクリップボードのリンクをそのまま追加できます。

### 外部の aria2 に接続

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

設定は `~/.config/aria2t/config.json` に保存されます（`--config` で上書き可能）。管理されるデーモンはセッションを `~/.config/aria2t/daemon/` 以下に保持します。

### 更新

バイナリを再ビルドして置き換えるだけです — 設定・スケジューラのルール・デーモンのセッションはすべて `~/.config/aria2t/` にあり、入れ替えても失われません。

### アンインストール

1. バイナリを削除します。
2. データディレクトリを削除します: `~/.config/aria2t/`（設定・デーモンのセッション・ログ）。

## ⌨️ 使い方

**追加。** `a` で追加フォームが開き、クリップボードから自動入力されます。1 行 1 URI は同じファイルのミラーを意味します。`^t` でタブを切り替え（URL · `.torrent` · `.metalink` · 入力ファイル）、`^o` でディスクからファイルを参照、`^s` で一時停止状態での開始を切り替えます。何もない初回起動画面では、`↵` でクリップボードのリンクをそのまま追加できます。

**ファイルを選ぶ。** 複数ファイルの torrent、metalink、マグネットは、**何かがダウンロードされる前に** チェックボックスツリーを開きます（マグネットは選択するまで一時停止のままです）: `space` でファイルまたはフォルダを選択／解除、`a`/`n` で全選択／全解除、`↵` で確定します。あとから `f` で選択を変更できます。選択する前に終了すると、次回起動時にピッカーが再び開きます — 不要なものが取得されることはありません。

**一覧の操作。** `tab` または `1`–`4` で すべて／Active／Waiting／Stopped を切り替えます。`space` は選択行を一時停止／再開、`P`/`U` はすべてに対して実行、`d` は削除、`l` は制限、`/` は入力しながら絞り込み、`y` はソース URL をコピー、`↵` は詳細を開きます。Waiting タブでは、`J`/`K` でキュー内の項目をつかんで移動します。

**帯域。** `l` は選択中のダウンロードをプリセットチップで制限します。設定（`,`）で設定した全体の上限は保存され、デーモンの再起動時に再適用されます。`S` は時間帯ごとに全体の制限をスケジュールします（「勤務時間は 5 MiB/s、夜間は無制限」）。

**整合性。** Stopped タブでは: `c` で期待する sha-256 を保存、`v` でローカルファイルを検証、`R` は不一致時に再ダウンロード、`X` は一覧をクリアします。

いつでも **`?`** で全キーマップを表示できます。キーバーのヒントはすべてクリック可能で、ダイアログは緑（続行）／赤（キャンセル）のボタンを使います。

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

`managed` のサーバーは内蔵デーモンです（ポートと secret は起動時に選ばれます）。それ以外は外部エンドポイントです。`globalDown`／`globalUp` は保存された全体の速度上限、`seedRatio`／`seedTime` はシードのデフォルトで — いずれも接続時にデーモンへ再適用されます。スケジューラのルール（`S`）もここに保存されます。

## ❓ FAQ

**aria2 とは何で、なぜ TUI を付けるのですか？**
[aria2](https://aria2.github.io/) は高速なマルチプロトコル・ダウンロードエンジンです — HTTP(S)、FTP、SFTP、BitTorrent、Metalink — ヘッドレスで動き、JSON-RPC を話します。単発コマンド以外に自前のインターフェースを持ちません。aria2t がそのインターフェースです: リアルタイムの一覧、ファイル選択、キューの並べ替え、速度制限、整合性チェックを 1 つのフルスクリーン TUI に。

**`aria2c` を直接実行するのと何が違いますか？**
`aria2c` は 1 つの URL をダウンロードして終了するか、スクリプトで操作するデーモンとして動くかのどちらかです。aria2t はそのデーモンを代わりに管理し — 起動、セッションの保存／復元、クリーンな停止、クラッシュで取り残されたデーモンの回収 — その上にすべての対話的操作を載せます。

**既に動かしている aria2 でも使えますか？**
はい。`--url ws://host:6800/jsonrpc --secret …`（またはレイテンシ計測付きのアプリ内サーバー切り替え）で、WebSocket や HTTP(S) 経由の任意の aria2 に接続できます — シードボックス、NAS、Docker コンテナ。その場合、内蔵デーモンは起動しません。

**終了してもダウンロードは残りますか？**
はい。管理されるデーモンは終了時にセッション全体を保存し、次回起動時に復元します — 進行中・待機中・完了・シード中のダウンロードがすべて戻り、完了済みファイルは認識され再ダウンロードされません。答えずに閉じたファイルピッカーさえ再び開きます。

**トレントのダウンロード前に中のファイルを選べますか？**
はい — torrent・マグネット・metalink は一時停止状態で追加され、まずチェックボックスツリーが開きます（マグネットはメタデータを待ってから一時停止します）。確定するまで何も転送されず、`f` であとから選択をやり直せます。

**シードはしますか？**
はい。完了した torrent はアップロードを続け、ステータスは独立した *seeding* に変わります。全体のレシオ／時間のデフォルトは設定にあり、デーモンが再起動するたびに再適用されます。

**キーボードとマウス、どちらで操作しますか？**
両方、完全に対応します: キーバーのヒントはすべてクリック可能で、すべてのクリックにキーの等価操作があります。`?` で全キーマップを表示します。

**aria2t はどこかにデータを送りますか？**
いいえ。アナリティクスもテレメトリもリモート設定もありません — 唯一の通信相手は、あなたが設定した aria2 エンドポイントです。

## 🛠️ 開発

このモジュールは **100% のステートメントカバレッジ** を維持しており、これは強制されます:

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```
## 🙏 謝辞

- **[aria2](https://github.com/aria2/aria2)**（GPL-2.0）— 転送・BitTorrent・Metalink の実際の仕事をすべて担うダウンロードエンジン。aria2t は操作パネルであり、働くのは aria2 です。
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**、**[Bubbles](https://github.com/charmbracelet/bubbles)**、**[Lip Gloss](https://github.com/charmbracelet/lipgloss)**（MIT）— aria2t が乗っている Charm の TUI スタック。
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — 両方のパレットは Tokyo Night / Tokyo Night Day からそのまま採用しています。
