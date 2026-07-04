# aria2t

[English](README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · **Deutsch** · [日本語](README.ja.md)

aria2t ist eine Terminal-Oberfläche für den Download-Manager [aria2](https://aria2.github.io/) in der Tokyo-Night-Palette. Sie spricht aria2s JSON-RPC-Schnittstelle über WebSocket oder HTTP — mit einem privaten Daemon, den sie selbst startet und verwaltet (der Standard: null Konfiguration), oder mit jedem aria2, das bereits auf einer Seedbox, einem NAS oder einem entfernten Rechner läuft. Ein einziges statisches Go-Binary.

![aria2t-Demo](docs/media/demo.gif)

## Was es kann

- Downloads von Anfang bis Ende verwalten: hinzufügen (URL-Spiegel, `.torrent`, `.metalink`), einzeln oder alle pausieren/fortsetzen, entfernen, neu einreihen.
- Live-Fortschritt, Geschwindigkeiten, ETA und Piece-Karte in den Tabs **Active / Waiting / Stopped**.
- Filtern (`/`), Warteschlange umsortieren (`J`/`K` — greifen und ablegen), Drosselung pro Download oder nach Tageszeit.
- Fertige Downloads gegen eine eingefügte sha-256 prüfen und bei Abweichung neu laden.
- BitTorrent-Seeding steuern: Stop-Ratio, Seed-Zeit, Tracker-Liste; Peers und Dateiauswahl.
- Zwischen Servern wechseln, mit Latenz-Messung; Terminal-Glocke und Download-Name bei Abschluss oder Fehler.
- Volle Mausunterstützung: Klick auf Tabs/Zeilen/Buttons/Chips, Doppelklick öffnet, Mausrad scrollt.

## Bildschirme

| | |
|---|---|
| ![Download-Liste](docs/media/list-active.png) Die Liste — Fortschritt, Geschwindigkeiten, ETA live | ![Detail](docs/media/detail.png) Detail — Piece-Karte, Peers, Dateiauswahl |
| ![Download hinzufügen](docs/media/add-overlay.png) Hinzufügen — aus der Zwischenablage vorausgefüllt, Spiegel, Umbenennen | ![Drosselung](docs/media/throttle.png) Drosselung pro Download mit Preset-Chips |
| ![Filter](docs/media/filter.png) Live-Filter mit `/` | ![Umsortieren](docs/media/reorder.png) Warteschlange — greifen, verschieben, ablegen |
| ![Integrität](docs/media/stopped-integrity.png) sha-256-Prüfung und Fehlerzeilen | ![Statistik](docs/media/stats.png) Globale Statistik — 60-s-Sparkline, Sitzungssummen |
| ![Zeitplaner](docs/media/scheduler.png) Bandbreiten-Zeitplaner nach Tageszeit | ![Server](docs/media/servers.png) Server-Umschalter mit Latenz-Messung |
| ![Einstellungen](docs/media/settings.png) Einstellungen — Live-Editor für `getGlobalOption` | ![Helles Thema](docs/media/list-light.png) Tokyo Night Day (`T` wechselt) |

## Wie es funktioniert

```mermaid
flowchart LR
    subgraph Terminal
      UI["aria2t<br/>Bubble-Tea-TUI"]
    end
    subgraph Machine["Dein Rechner"]
      D["verwalteter aria2c<br/>(bei Bedarf gestartet)"]
    end
    R(("externes aria2<br/>Seedbox · NAS"))

    UI -- "JSON-RPC über ws:// (Push + Polling)" --> D
    UI -. "ws:// oder http(s)://" .-> R
```

Standardmäßig findet aria2t `aria2c` im `PATH`, startet einen privaten Daemon auf einem freien Port mit zufälligem Secret und verwaltet dessen kompletten Lebenszyklus: Die Sitzung wird beim Beenden gespeichert und beim nächsten Start fortgesetzt, der Kindprozess wird sauber gestoppt (saveSession- + shutdown-RPC, Signale nur als Eskalation). Nichts zu konfigurieren, nichts bleibt laufen.

Zeigt man stattdessen auf einen externen Server, startet der eingebaute Daemon gar nicht erst: aria2t ist dann ein reiner RPC-Client, und alles — inklusive der Prüfsummen-Verifikation, die die Datei von der lokalen Platte liest — degradiert sauber, wenn die Dateien woanders liegen.

## Voraussetzungen

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — nur für den Null-Konfigurations-Modus; für externe Server nicht nötig.
- Ein Terminal mit 256 Farben oder Truecolor. Maus ist optional; jede Aktion hat eine Taste.
- Go 1.25+ für den Build aus den Quellen.

## Installation

```sh
go build -o aria2t ./cmd/aria2t     # oder: go install aria2t/cmd/aria2t
./aria2t
```

Das war's — der erste Start bringt den privaten Daemon hoch, und du landest in einer leeren Liste, bereit für `a`.

Verbindung zu einem **externen** aria2:

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret meinsecret
```

oder Server im Umschalter hinzufügen (`s` → `+`). Die Konfiguration liegt in `~/.config/aria2t/config.json` (`--config` überschreibt); der verwaltete Daemon hält Sitzung und Log unter `~/.config/aria2t/daemon/`. `--version` gibt die Build-Version aus.

## Bedienung

**Hinzufügen.** `a` öffnet das Overlay. Liegt in der Zwischenablage eine URL oder ein Magnet-Link, ist er schon eingetragen. Ein URI pro Zeile bedeutet Spiegel derselben Datei; `^t` wechselt in den `.torrent`-/`.metalink`-Dateimodus; `^r` benennt das Ziel um; `^s` startet pausiert.

**Die Liste.** `space` schaltet Pause/Fortsetzen intelligent um, `P`/`U` pausieren und setzen alles fort, `d` entfernt (mit Rückfrage), `D` leert die ganze Stopped-Liste, `y` kopiert die Quell-URL (oder einen aus dem Info-Hash gebauten Magnet-Link) in die Zwischenablage, `/` filtert beim Tippen nach Namen — `enter` behält den Filter (ein `⌕`-Abzeichen zeigt ihn), `esc` löscht ihn.

**Warteschlange.** Im Waiting-Tab greifen `J`/`K` den gewählten Eintrag und verschieben ihn; `gg`/`G` schicken ihn an Anfang/Ende, `↵` legt ab. Während des Ziehens friert die Liste ein, die Endposition wird gegen die live-Warteschlange neu berechnet — der Eintrag landet dort, wo es aussieht, selbst wenn zwischendurch Downloads fertig wurden.

**Bandbreite.** `l` drosselt den gewählten Download mit Preset-Chips (`∞`/1M/5M/10M/eigene). Der Zeitplaner (`S`) wendet globale Limits nach Tageszeit und Wochentag an — Regeln wie „5 MiB/s zur Arbeitszeit, nachts unbegrenzt" laufen über `changeGlobalOption` und überleben Reconnects.

**Integrität.** Im Stopped-Tab speichert `c` die erwartete sha-256, `v` liest die lokale Datei und vergleicht, `R` reiht bei Abweichung anhand der aufgezeichneten URIs neu ein. Fehlgeschlagene Downloads zeigen aria2s Fehlermeldung direkt in der Liste.

**Benachrichtigungen.** Wenn ein Download fertig wird oder scheitert — egal auf welchem Bildschirm — nennt die Statuszeile den Namen und die Terminal-Glocke läutet. Funktioniert auch über reines HTTP-Polling; WebSocket-Push macht es nur sofortig.

## Tastenbelegung

| Kontext | Tasten |
|---|---|
| Liste | `a` hinzufügen · `space` Pause/weiter · `P`/`U` alles pausieren/fortsetzen · `d` entfernen · `y` Quelle kopieren · `/` filtern · `↵` Details · `g` Statistik · `l` Limit · `s` Server · `S` Zeitplaner · `t` Seeding · `,` Einstellungen · `T` Thema · `tab`/`1‑3` Tabs · `?` Hilfe · `q` beenden |
| Maus | Klick wählt/fokussiert · Doppelklick öffnet/verbindet · Rad scrollt · Tabs, Chips, Buttons, Tastenleiste klickbar |
| Waiting-Tab | `J`/`K` greifen + verschieben · `gg`/`G` Anfang/Ende · `↵` ablegen · `esc` abbrechen |
| Stopped-Tab | `c` Prüfsumme einfügen · `v` prüfen · `R` neu laden · `D` Liste leeren · `o` Ordner öffnen |
| Detail | `p` Pause/weiter · `d` entfernen · `f` Dateiauswahl · `t` Tracker · `o` Ordner öffnen |
| Formulare | `tab` nächstes Feld · `space` umschalten · `^s` speichern · `esc` zurück |

## Konfiguration

`~/.config/aria2t/config.json`:

```json
{
  "servers": [
    { "name": "built-in", "host": "localhost", "protocol": "ws", "managed": true },
    { "name": "seedbox", "host": "sb.example.net", "port": 6800, "secret": "…", "protocol": "wss" }
  ],
  "active": 0,
  "theme": "dark",
  "schedulerEnabled": true,
  "rules": [
    { "start": "09:00", "end": "18:00", "days": [false,true,true,true,true,true,false],
      "label": "Working hours", "down": "5M", "up": "256K" }
  ],
  "dir": "~/Downloads",
  "split": 16
}
```

Ein Server mit `managed` ist der eingebaute Daemon (Port und Secret werden beim Start gewählt). Alles andere sind externe Endpunkte; `path` überschreibt den RPC-Pfad für Daemons hinter einem Reverse-Proxy. Ausgelassene Felder behalten ihre Defaults.

## Entwicklung

Das Modul hält **100 % Statement-Coverage** — jede Funktion, jedes Paket — und die Latte wird durchgesetzt:

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # darf nichts ausgeben
```


