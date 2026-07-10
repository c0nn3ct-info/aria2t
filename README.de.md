[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<h1 align="center">aria2t</h1>

<p align="center"><strong>Terminal-Oberfläche für den Download-Manager aria2</strong></p>
<p align="center"><em>Downloads, Torrents, Magnets und Metalinks — eine TUI für Tastatur und Maus, null Konfiguration.</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
</p>

<p align="center">
  <img alt="aria2t-Demo" src="./docs/media/demo.gif" width="720">
</p>

> [!IMPORTANT]
> aria2t ist eine Steueroberfläche — das Herunterladen erledigt [aria2](https://aria2.github.io/). Standardmäßig startet und verwaltet es für dich einen privaten `aria2c`-Daemon (null Konfiguration); auf ein bereits laufendes aria2 gezeigt, wird es zu einem reinen RPC-Client. Keine Analytik, keine Telemetrie — es spricht nur mit dem aria2-Endpunkt, den du konfigurierst.

aria2t ist eine Terminal-Oberfläche für den Download-Manager aria2 in der Tokyo-Night-Palette. Sie spricht mit aria2 über JSON-RPC — mit einem privaten Daemon, den sie selbst startet und für dich verwaltet, oder mit jedem aria2, das du bereits auf einer Seedbox, einem NAS oder einem entfernten Rechner betreibst — und jede Aktion funktioniert per Tastatur **und** Maus. Ein einziges statisches Go-Binary.

## ✨ Funktionen

- **Fügt alles hinzu, was aria2 annimmt** — URL-Spiegel, `.torrent`, `.metalink`, Magnet-Links, aria2-Eingabedateien — und lässt dich **vor dem Start auswählen, welche Dateien geladen werden**.
- **Null Konfiguration** — findet `aria2c`, startet einen privaten Daemon und verwaltet dessen kompletten Lebenszyklus. Oder zeige mit `--url` auf ein externes aria2.
- **Steuert jeden Download** — einzeln oder alle pausieren/fortsetzen, entfernen, die Warteschlange umsortieren, pro Download oder nach Tageszeit drosseln.
- **Zeigt das Innere eines Downloads** — Piece-Karte, Peers und Spiegel-Geschwindigkeiten, Fortschritt pro Datei, Upload-Ratio und BitTorrent-Seeding-Steuerung.
- **Hält dich ehrlich** — prüft eine fertige Datei gegen eine sha-256 und lädt bei Abweichung neu; erklärt Fehler in Klartext statt in Fehlercodes.
- **Übersteht Neustarts** — fertige und laufende Downloads kommen zurück, und eine Dateiauswahl, die du ohne Antwort geschlossen hast, öffnet sich erneut.
- **Bleibt aus dem Weg** — läutet die Glocke, wenn etwas fertig wird, filtert beim Tippen, wechselt Server mit Latenz-Messung und schaltet zwischen hellem und dunklem Thema um.

## 📦 Unterstützte Quellen

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2-Eingabedatei`

Alles, was aria2 annimmt, fügt aria2t hinzu: einfache URLs — eine pro Zeile bedeutet Spiegel derselben Datei —, Magnet-Links und `.torrent`-Dateien mit DHT und Seeding, `.metalink` sowie aria2s eigenes `--input-file`-Batch-Format mit Optionen pro Download. Mehrdatei-Torrents, Magnets und Metalinks öffnen einen Kästchen-Baum, **bevor** der Download startet — nie wird etwas Ungewolltes geladen.

## 🖥️ Bildschirme

| | |
|---|---|
| ![Download-Liste](./docs/media/list-active.png) Die Liste — Live-Fortschritt, Geschwindigkeiten, eine farbige STATUS-Spalte | ![Dateiauswahl](./docs/media/files-picker.png) Dateien auswählen, bevor etwas lädt |
| ![Detail](./docs/media/detail.png) Detail — Piece-Karte, Peers/Spiegel, Fortschritt pro Datei | ![Hinzufügen](./docs/media/add-overlay.png) Hinzufügen — aus der Zwischenablage vorausgefüllt, grüne/rote Buttons |
| ![Integrität](./docs/media/stopped-integrity.png) sha-256-Prüfung + Klartext-Fehler | ![Zeitplaner](./docs/media/scheduler.png) Bandbreiten-Zeitplaner nach Tageszeit |

## 🧩 Wie es funktioniert

Die TUI lädt selbst nie etwas herunter — das tut ein aria2-Daemon, und zwischen beiden fließt nur JSON-RPC.

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

Standardmäßig findet aria2t `aria2c` in deinem `PATH` und betreibt einen privaten Daemon auf einem zufälligen Port mit zufälligem Secret: Die komplette Sitzung wird beim Beenden gespeichert und beim nächsten Start wiederhergestellt, und der Kindprozess wird beim Beenden sauber gestoppt (ein durch Absturz verwaister Daemon wird beim nächsten Start abgeräumt). Zeigt man stattdessen auf einen externen Server, startet der eingebaute Daemon gar nicht erst — aria2t wird zu einem reinen RPC-Client.

## 📥 Installation

### Bevor du anfängst

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — nur für den Null-Konfigurations-Modus; für die Verbindung zu einem externen Server nicht nötig.
- Ein Terminal mit 256-Farben- oder Truecolor-Unterstützung. Maus ist optional; jede Aktion hat eine Taste.
- Go 1.25+ für den Build aus den Quellen.

### Bauen und starten

```sh
go build -o aria2t ./cmd/aria2t     # or: go install aria2t/cmd/aria2t
./aria2t
```

Der erste Start bringt den privaten Daemon hoch und lässt dich in einer leeren Liste landen, bereit für `a` — oder für `↵`, das direkt einen Link aus der Zwischenablage hinzufügt.

### Mit einem externen aria2 verbinden

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

Die Konfiguration liegt in `~/.config/aria2t/config.json` (`--config` überschreibt); der verwaltete Daemon hält seine Sitzung unter `~/.config/aria2t/daemon/`.

### Aktualisieren

Neu bauen und das Binary ersetzen — Konfiguration, Zeitplaner-Regeln und die Daemon-Sitzung liegen alle unter `~/.config/aria2t/` und überstehen den Tausch.

### Deinstallieren

1. Lösche das Binary.
2. Lösche das Datenverzeichnis: `~/.config/aria2t/` (Konfiguration, Daemon-Sitzung, Logs).

## ⌨️ Bedienung

**Hinzufügen.** `a` öffnet das Formular, aus deiner Zwischenablage vorausgefüllt. Ein URI pro Zeile bedeutet Spiegel derselben Datei; `^t` wechselt durch die Tabs (URL · `.torrent` · `.metalink` · Eingabedatei), `^o` durchsucht die Platte nach einer Datei, `^s` schaltet Start-pausiert um. Auf dem leeren Startbildschirm des ersten Laufs fügt `↵` direkt einen Link aus der Zwischenablage hinzu.

**Dateien auswählen.** Mehrdatei-Torrents, Metalinks und Magnets öffnen einen Kästchen-Baum, **bevor überhaupt etwas lädt** (ein Magnet bleibt pausiert, bis du wählst): `space` schaltet eine Datei oder einen Ordner um, `a`/`n` wählen alles/nichts, `↵` bestätigt. Mit `f` änderst du die Auswahl später. Beendest du, bevor du wählst, öffnet sich die Auswahl beim nächsten Start erneut — nie wird etwas Ungewolltes geladen.

**Die Liste bedienen.** `tab` oder `1`–`4` wechseln zwischen Alle / Active / Waiting / Stopped. `space` pausiert/setzt die gewählte Zeile fort, `P`/`U` betreffen alle, `d` entfernt, `l` limitiert, `/` filtert beim Tippen, `y` kopiert die Quell-URL, `↵` öffnet die Details. Im Waiting-Tab greifen `J`/`K` einen Eintrag und verschieben ihn in der Warteschlange.

**Bandbreite.** `l` drosselt den gewählten Download mit Preset-Chips. Ein in den Einstellungen (`,`) gesetztes globales Limit wird gespeichert und beim Neustart des Daemons erneut angewendet. `S` plant globale Limits nach Tageszeit („5 MiB/s zur Arbeitszeit, nachts unbegrenzt").

**Integrität.** Im Stopped-Tab: `c` speichert eine erwartete sha-256, `v` prüft die lokale Datei, `R` lädt bei Abweichung neu, `X` leert die Liste.

Drücke jederzeit **`?`** für die vollständige Tastenbelegung. Jeder Hinweis in der Tastenleiste ist außerdem klickbar, und Dialoge nutzen grüne (fortfahren) / rote (abbrechen) Buttons.

## ⚙️ Konfiguration

`~/.config/aria2t/config.json` (ausgelassene Felder behalten ihre Defaults):

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

Ein `managed`-Server ist der eingebaute Daemon (Port und Secret werden beim Start gewählt); alles andere ist ein externer Endpunkt. `globalDown`/`globalUp` sind gespeicherte globale Geschwindigkeitslimits und `seedRatio`/`seedTime` die Seeding-Standards — alle werden beim Verbinden erneut auf den Daemon angewendet. Zeitplaner-Regeln (`S`) werden ebenfalls hier gespeichert.

## ❓ FAQ

**Was ist aria2, und warum eine TUI dafür?**
[aria2](https://aria2.github.io/) ist eine schnelle Multi-Protokoll-Download-Engine — HTTP(S), FTP, SFTP, BitTorrent, Metalink —, die headless läuft und JSON-RPC spricht. Eine eigene Oberfläche jenseits von Einmal-Kommandos hat sie nicht. aria2t ist diese Oberfläche: Live-Liste, Dateiauswahl, Umsortieren der Warteschlange, Drosselung und Integritätsprüfungen in einer Vollbild-TUI.

**Was ist der Unterschied zum direkten Aufruf von `aria2c`?**
`aria2c` lädt entweder eine URL und beendet sich, oder es läuft als Daemon, gegen den man Skripte schreibt. aria2t verwaltet diesen Daemon für dich — Start, Sitzung speichern/wiederherstellen, sauberes Herunterfahren, Abräumen eines durch Absturz verwaisten Daemons — und legt alles Interaktive obendrauf.

**Funktioniert es mit einem aria2, das ich schon betreibe?**
Ja. `--url ws://host:6800/jsonrpc --secret …` (oder der eingebaute Server-Umschalter mit Latenz-Messung) verbindet sich mit jedem aria2 über WebSocket oder HTTP(S) — Seedbox, NAS, Docker-Container. Der eingebaute Daemon startet dann nie.

**Überleben Downloads das Beenden?**
Ja. Der verwaltete Daemon speichert seine komplette Sitzung beim Beenden und stellt sie beim nächsten Start wieder her — aktive, wartende, fertige und seedende Downloads kommen alle zurück, und fertige Dateien werden erkannt statt neu geladen. Selbst eine Dateiauswahl, die du ohne Antwort geschlossen hast, öffnet sich erneut.

**Kann ich Dateien in einem Torrent auswählen, bevor er lädt?**
Ja — Torrents, Magnets und Metalinks werden pausiert hinzugefügt, und zuerst öffnet sich ein Kästchen-Baum (ein Magnet wartet auf seine Metadaten und pausiert dann). Nichts wird übertragen, bis du bestätigst, und `f` öffnet die Auswahl später erneut.

**Seedet es?**
Ja. Ein fertiger Torrent lädt weiter hoch, und sein Status wechselt zu einem eigenen *seeding*; globale Ratio-/Zeit-Standards liegen in den Einstellungen und werden bei jedem Daemon-Neustart erneut angewendet.

**Tastatur oder Maus?**
Beides, vollständig: Jeder Hinweis in der Tastenleiste ist klickbar, und jeder Klick hat ein Tasten-Äquivalent. `?` zeigt die ganze Belegung.

**Sendet aria2t irgendetwas irgendwohin?**
Nein. Keine Analytik, keine Telemetrie, keine Fernkonfiguration — der einzige Netzwerk-Gesprächspartner ist der aria2-Endpunkt, den du konfigurierst.

## 🛠️ Entwicklung

Das Modul hält **100 % Statement-Coverage**, durchgesetzt:

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```
## 🙏 Danksagungen

- **[aria2](https://github.com/aria2/aria2)** (GPL-2.0) — die Download-Engine, die die gesamte eigentliche Transfer-, BitTorrent- und Metalink-Arbeit erledigt. aria2t ist eine Steueroberfläche; die Arbeit macht aria2.
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**, **[Bubbles](https://github.com/charmbracelet/bubbles)** und **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** (MIT) — der Charm-TUI-Stack, auf dem aria2t gebaut ist.
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — beide Paletten sind wörtlich aus Tokyo Night / Tokyo Night Day übernommen.
