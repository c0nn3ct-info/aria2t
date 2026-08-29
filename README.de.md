[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./tui/docs/media/logo-dark.svg">
    <img alt="" src="./tui/docs/media/logo-light.svg" width="128">
  </picture>
</p>

<h1 align="center">Aria2t</h1>

<p align="center"><strong>Download-Manager für aria2</strong></p>
<p align="center"><em>Verwalte aria2 im Terminal oder Browser.</em></p>

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
    <img alt="Aria2t-Demo" src="./tui/docs/media/demo-light.gif" width="720">
  </picture>
</p>

> [!IMPORTANT]
> Die Downloads erledigt [aria2](https://aria2.github.io/); Aria2t ist sein Bedienpanel. Standardmäßig startet und verwaltet Aria2t einen eigenen `aria2c`-Daemon, sodass keine Einrichtung nötig ist. Verweist man es auf ein bereits laufendes aria2, verbindet es sich damit als gewöhnlicher RPC-Client. Aria2t sammelt weder Analytik noch Telemetrie und kommuniziert über das Netzwerk ausschließlich mit dem konfigurierten aria2-Server.

Aria2t ist ein Download-Manager für aria2 mit Oberflächen für Terminal und Browser. Es startet einen privaten Daemon oder verbindet sich per JSON-RPC mit aria2 auf einer Seedbox, einem NAS oder einem entfernten Rechner.

## ✨ Funktionen

- **Unterstützung aller aria2-Quellen** — URL-Spiegel, `.torrent`, `.metalink`, Magnet-Links und aria2-Eingabedateien.
- **Browser-Erweiterung** — eine Chrome-Erweiterung schickt Browser-Downloads und Magnet-Links an denselben Daemon, gefiltert nach Größe, Domain oder Dateityp. Installation unter <https://aria2t.c0nn3ct.info/extension>.
- **Start ohne Einrichtung** — Aria2t findet `aria2c`, startet einen privaten Daemon und verwaltet dessen gesamten Lebenszyklus; ein externes aria2 wird über die Option `--url` angebunden.
- **Download-Verwaltung** — Pausieren und Fortsetzen einzeln oder für alle, Entfernen, Umsortieren der Warteschlange, Geschwindigkeitslimits pro Download oder nach Zeitplan.
- **Details zu jedem Download** — Piece-Karte, Peers und Spiegel-Geschwindigkeiten, Fortschritt pro Datei, Ratio und Steuerung des BitTorrent-Seedings.
- **Integritätsprüfung** — eine fertige Datei wird gegen eine sha-256-Prüfsumme abgeglichen und bei Abweichung neu geladen; Fehler werden in Klartext statt mit Codes beschrieben.
- **Erhalt des Zustands über Neustarts** — fertige und laufende Downloads werden wiederhergestellt, ebenso ein ohne Antwort geschlossenes Dateiauswahl-Fenster.

## 📦 Unterstützte Quellen

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `aria2-Eingabedatei`

In Aria2t lässt sich alles hinzufügen, was aria2 annimmt: gewöhnliche URLs (mehrere Zeilen gelten als Spiegel derselben Datei), Magnet-Links und `.torrent`-Dateien mit DHT- und Seeding-Unterstützung, `.metalink` sowie das Batch-Format `--input-file` von aria2 mit eigenen Optionen je Eintrag.

## 🧩 Wie es funktioniert

Die Dateien lädt der aria2-Daemon; eine Oberfläche steuert ihn über JSON-RPC. Der Terminal-Client und die Browser-Erweiterung sind zwei solche Oberflächen und steuern denselben Daemon.

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

Standardmäßig findet Aria2t `aria2c` im `PATH` und startet einen privaten Daemon auf einem zufälligen Port mit einem zufälligen Secret. Die Sitzung wird beim Beenden gespeichert und beim nächsten Start wiederhergestellt, und der Kindprozess wird zusammen mit dem Programm sauber gestoppt; ein nach einem Absturz zurückgebliebener Daemon wird beim nächsten Start beendet. Ist ein externer Server konfiguriert, wird der eingebaute Daemon nicht gestartet, und Aria2t arbeitet als gewöhnlicher RPC-Client.

## 📥 Installation

### Bevor Sie anfangen

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — nur für den eingebauten Daemon; für die Verbindung zu einem externen Server nicht nötig.
- Ein Terminal mit 256-Farben- oder Truecolor-Unterstützung. Eine Maus ist optional: Alle Aktionen sind über die Tastatur erreichbar.
- Go 1.25 oder neuer für den Build aus den Quellen.

### Bauen und starten

```sh
git clone https://github.com/c0nn3ct-info/aria2t.git
cd aria2t/tui && go build -o aria2t ./cmd/aria2t
./aria2t
```

Beim ersten Start bringt Aria2t den privaten Daemon hoch und öffnet eine leere Download-Liste. Die Taste `a` öffnet das Hinzufügen-Formular, und `↵` fügt einen Link aus der Zwischenablage hinzu.

### Verbindung zu einem externen aria2

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

Die Konfiguration liegt in `~/.config/aria2t/config.json` (der Pfad lässt sich mit `--config` ändern); der verwaltete Daemon hält seine Sitzung unter `~/.config/aria2t/daemon/`.

### Aktualisieren

Bauen Sie das Binary neu und ersetzen Sie das alte. Konfiguration, Zeitplaner-Regeln und die Daemon-Sitzung liegen unter `~/.config/aria2t/` und überstehen den Austausch.

### Deinstallieren

1. Löschen Sie das Binary.
2. Löschen Sie das Datenverzeichnis `~/.config/aria2t/` (Konfiguration, Daemon-Sitzung, Logs).

## ⌨️ Bedienung

**Hinzufügen.** `a` öffnet das Hinzufügen-Formular, aus der Zwischenablage vorausgefüllt. Mehrere URIs, eine pro Zeile, gelten als Spiegel derselben Datei. `^t` wechselt die Tabs (URL · `.torrent` · `.metalink` · Eingabedatei), `^o` öffnet einen Dateibrowser, und `^s` fügt den Download pausiert hinzu. Auf dem leeren Bildschirm des ersten Starts fügt `↵` einen Link aus der Zwischenablage hinzu.

**Dateien auswählen.** Mehrdatei-Torrents, Metalink und Magnet-Links öffnen einen Dateibaum mit Kontrollkästchen; ein Magnet-Link bleibt pausiert, bis die Auswahl bestätigt ist. `space` markiert eine Datei oder einen Ordner, `a` und `n` wählen alles oder nichts, `↵` bestätigt die Auswahl. Später lässt sich die Auswahl mit `f` ändern. Beendet man das Programm vor der Bestätigung, öffnet sich das Fenster beim nächsten Start erneut.

**Die Liste.** `tab` oder die Tasten `1`–`4` wechseln zwischen den Tabs All / Active / Waiting / Stopped. `space` pausiert und setzt den gewählten Download fort, `P` und `U` tun dasselbe für alle, `d` entfernt, `l` begrenzt die Geschwindigkeit, `/` filtert nach Namen, `y` kopiert die Quell-URL, `↵` öffnet die Details. Im Waiting-Tab verschieben `J` und `K` den gewählten Eintrag innerhalb der Warteschlange.

**Geschwindigkeit.** `l` begrenzt die Geschwindigkeit des gewählten Downloads; die Werte werden aus Voreinstellungen gewählt. Ein globales Limit wird in den Einstellungen (`,`) gesetzt, gespeichert und nach einem Neustart des Daemons erneut angewendet. `S` öffnet den Zeitplaner, in dem globale Limits nach Tageszeit festgelegt werden, etwa 5 MiB/s während der Arbeitszeit und unbegrenzt nachts.

**Integrität.** Im Stopped-Tab speichert `c` die erwartete sha-256-Prüfsumme, `v` gleicht die lokale Datei damit ab, `R` lädt die Datei bei Abweichung neu, und `X` leert die Liste.

Die vollständige Liste der Tastenkombinationen öffnet sich mit **`?`**. Jeder Hinweis in der unteren Leiste reagiert auch auf Mausklick; in Dialogen bestätigt der grüne Knopf die Aktion, der rote bricht ab.

## ⚙️ Konfiguration

`~/.config/aria2t/config.json` (ausgelassene Felder behalten ihre Standardwerte):

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

Ein Server mit dem Feld `managed` ist der eingebaute Daemon; Port und Secret werden bei seinem Start gewählt. Die übrigen Einträge beschreiben externe Server. `globalDown` und `globalUp` legen die gespeicherten globalen Geschwindigkeitslimits fest, `seedRatio` und `seedTime` die Standard-Seeding-Parameter; beim Verbinden werden alle diese Werte erneut auf den Daemon angewendet. Zeitplaner-Regeln (`S`) werden ebenfalls in dieser Datei gespeichert.

## ❓ FAQ

**Was ist aria2, und warum braucht es eine eigene Oberfläche?**
[aria2](https://aria2.github.io/) ist eine schnelle Download-Engine mit Unterstützung für HTTP(S), FTP, SFTP, BitTorrent und Metalink. Sie läuft im Hintergrund, wird über JSON-RPC gesteuert und hat außer Einmal-Kommandos keine eigene Oberfläche. Aria2t ergänzt sie um eine Download-Liste in Echtzeit, Dateiauswahl, Verwaltung der Warteschlange, Geschwindigkeitslimits und Integritätsprüfung.

**Was ist der Unterschied zum direkten Aufruf von `aria2c`?**
`aria2c` lädt entweder eine Datei und beendet sich, oder es läuft als Daemon, der über Skripte gesteuert wird. Aria2t übernimmt die Verwaltung dieses Daemons: Es startet ihn, speichert und restauriert die Sitzung, stoppt ihn beim Beenden sauber und beendet einen nach einem Absturz zurückgebliebenen Daemon. Darauf aufbauend stellt Aria2t eine interaktive Oberfläche bereit.

**Funktioniert es mit einem aria2, das bereits läuft?**
Ja. Die Option `--url ws://host:6800/jsonrpc --secret …` oder der eingebaute Server-Umschalter mit Latenzmessung verbindet Aria2t mit jedem aria2 über WebSocket oder HTTP(S), etwa auf einer Seedbox, einem NAS oder in einem Docker-Container. Der eingebaute Daemon wird in diesem Fall nicht gestartet.

**Bleiben Downloads nach dem Beenden erhalten?**
Ja. Der verwaltete Daemon speichert die Sitzung beim Beenden und stellt sie beim nächsten Start wieder her: Aktive, wartende, fertige und seedende Downloads werden wiederhergestellt, und bereits geladene Dateien werden erkannt statt erneut geladen. Auch ein ohne Antwort geschlossenes Dateiauswahl-Fenster öffnet sich wieder.

**Kann man auswählen, welche Dateien eines Torrents geladen werden?**
Ja. Torrents, Magnet-Links und Metalink werden pausiert hinzugefügt, und zuerst öffnet sich ein Dateibaum; ein Magnet-Link wird nach dem Eintreffen seiner Metadaten pausiert. Die Datenübertragung beginnt erst nach Bestätigung der Auswahl, und mit `f` lässt sie sich später ändern.

**Seedet es Torrents?**
Ja. Ein fertiger Torrent wird weiter verteilt, und sein Status wechselt zu seeding. Die globalen Seeding-Parameter — Ratio und Zeit — werden in den Einstellungen festgelegt und bei jedem Neustart des Daemons erneut angewendet.

**Tastatur oder Maus?**
Beides wird vollständig unterstützt. Jeder Hinweis in der Tastenleiste reagiert auf Klick, jede Mausaktion hat ein Tastatur-Äquivalent, und die vollständige Liste der Kombinationen öffnet sich mit `?`.

**Sendet Aria2t irgendwelche Daten nach außen?**
Nein. Das Programm sammelt weder Analytik noch Telemetrie, bezieht keine Fernkonfiguration und kommuniziert über das Netzwerk ausschließlich mit dem konfigurierten aria2-Server.

## 🛠️ Entwicklung

Die Statement-Coverage des Moduls liegt bei **100 %**, und das wird automatisch geprüft:

```sh
cd tui
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```

## 🙏 Danksagungen

- **[aria2](https://github.com/aria2/aria2)** (GPL-2.0) — die Download-Engine, die die gesamte Transfer-, BitTorrent- und Metalink-Arbeit leistet; Aria2t ist ihr Bedienpanel.
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**, **[Bubbles](https://github.com/charmbracelet/bubbles)** und **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** (MIT) — der Charm-TUI-Stack, auf dem Aria2t gebaut ist.
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — beide Paletten sind unverändert aus den Themes Tokyo Night und Tokyo Night Day übernommen.
