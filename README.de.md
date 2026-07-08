# aria2t

[English](README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · [Español](README.es.md) · **Deutsch** · [日本語](README.ja.md)

aria2t ist eine Terminal-Oberfläche für den Download-Manager [aria2](https://aria2.github.io/) in der Tokyo-Night-Palette. Sie spricht aria2s JSON-RPC-Schnittstelle über WebSocket oder HTTP — mit einem privaten Daemon, den sie selbst startet und verwaltet (der Standard: null Konfiguration), oder mit jedem aria2, das bereits auf einer Seedbox, einem NAS oder einem entfernten Rechner läuft. Ein einziges statisches Go-Binary.

![aria2t-Demo](docs/media/demo.gif)

## Was es kann

- Downloads von Anfang bis Ende verwalten: hinzufügen (URL-Spiegel, `.torrent`, `.metalink`), einzeln oder alle pausieren/fortsetzen, entfernen, neu einreihen.
- Eine **Alle**-Ansicht als Standard mit Live-Fortschritt, Geschwindigkeiten und ETA zeigt jeden Download auf einmal: Ein Eintrag bleibt sichtbar und wechselt beim Übergang von aktiv zu fertig nur sein Abzeichen, statt scheinbar zu verschwinden — die Tabs Active / Waiting / Stopped sind einen Tastendruck entfernt.
- Jede Zeile ist eine saubere Spaltentabelle — **NAME · STATUS · PROGRESS · SIZE · SPEED · CONN · ETA** — mit einer eigenen farbigen **STATUS**-Spalte und — wie `aria2c` — Verbindungs- und Seed-Zahlen.
- Wählen, welche Dateien geladen werden, bevor überhaupt etwas lädt — Mehrdatei-Torrents, Metalinks und Magnets öffnen alle zuerst einen einklappbaren Kästchen-Baum (Magnets nach dem Auflösen ihrer Metadaten, was bis zur Auswahl pausiert bleibt); jederzeit mit `f` erreichbar.
- Volle aria2-Dateiunterstützung: `.torrent`, `.metalink`/`.meta4`, Magnet-Links und aria2-Eingabedateien (der Eingabedatei-Tab fügt jede URL mit ihren Optionen pro Download stapelweise hinzu). Das Dateisystem durchsuchen, um mit `^o` eine Datei zu wählen, statt einen Pfad zu tippen.
- Vorübergehende Magnet-Metadaten-Einträge werden beschriftet und automatisch aufgeräumt — nie bleibt ein blanker Hash in der Liste.
- Reichhaltiger Detail-Bildschirm: Piece-Karte, Fortschritt und Flags pro Peer bei Torrents, verbundene HTTP-/FTP-Spiegel und ihre Geschwindigkeiten bei Direkt-Downloads, Fortschritt pro Datei, Upload-Summe und Ratio.
- Filtern (`/`), Warteschlange umsortieren (`J`/`K` — greifen und ablegen), Drosselung pro Download oder nach Tageszeit.
- Fertige Downloads gegen eine eingefügte sha-256 prüfen und bei Abweichung neu laden.
- Erklärt Fehler in Klartext (z. B. „Datei auf dem Server nicht gefunden (404)", „nicht genug Speicherplatz", „Host konnte nicht aufgelöst werden") statt roher Fehlercodes.
- Seedet fertige Torrents (bis zu einer Stop-Ratio, standardmäßig 1.0) und steuert das pro Download oder als gespeicherten globalen Standard — Stop-Ratio, Seed-Zeit, Tracker-Liste.
- Zwischen Servern wechseln, mit Latenz-Messung; Terminal-Glocke und Download-Name bei Abschluss oder Fehler.
- Freundlicher Startbildschirm beim ersten Lauf mit Ein-Tasten-Hinzufügen eines Links aus der Zwischenablage; warnt vor dem Beenden, solange noch Downloads laufen.
- Volle Mausunterstützung ohne Doppelklicks: Ein einfacher Klick auf einen Download öffnet seine Details, und jeder Hinweis in der Tastenleiste ist klickbar (pausieren, entfernen, Dateien wählen, einen Server verbinden…). Jeder Dialog hat einheitliche gefüllte **Cancel**-/Bestätigen-Buttons (ein rotes Bestätigen für alles Destruktive) — alles ist mit der Maus **und** der Tastatur erreichbar. Das Mausrad scrollt.
- **Nichts geht beim Beenden verloren**: Fertige Downloads bleiben über Neustarts hinweg in der Liste, unfertige Übertragungen werden fortgesetzt, und eine Dateiauswahl, die du ohne Antwort geschlossen hast, öffnet sich beim nächsten Start erneut.

## Bildschirme

| | |
|---|---|
| ![Alle-Ansicht](docs/media/list-active.png) Die Liste — die Alle-Ansicht: jeder Download auf einmal, Fortschritt, Geschwindigkeiten, ETA live | ![Dateiauswahl](docs/media/files-picker.png) Dateiauswahl für Mehrdatei-Torrents — einklappbarer Baum, Tri-State-Kästchen |
| ![Detail](docs/media/detail.png) Detail — Piece-Karte, Spiegel/Peers, Fortschritt pro Datei, Ratio | ![Download hinzufügen](docs/media/add-overlay.png) Hinzufügen — aus der Zwischenablage vorausgefüllt, Spiegel, Umbenennen |
| ![Willkommen](docs/media/onboarding.png) Willkommen — Link aus der Zwischenablage per Taste | ![Drosselung](docs/media/throttle.png) Drosselung pro Download mit Preset-Chips |
| ![Filter](docs/media/filter.png) Live-Filter mit `/` | ![Umsortieren](docs/media/reorder.png) Warteschlange — greifen, verschieben, ablegen |
| ![Integrität](docs/media/stopped-integrity.png) sha-256-Prüfung mit Klartext-Fehlern in der Zeile | ![Statistik](docs/media/stats.png) Globale Statistik — 60-s-Sparkline, Sitzungssummen |
| ![Zeitplaner](docs/media/scheduler.png) Bandbreiten-Zeitplaner nach Tageszeit | ![Server](docs/media/servers.png) Server-Umschalter mit Latenz-Messung |
| ![Einstellungen](docs/media/settings.png) Einstellungen — Live-Editor für `getGlobalOption` | ![Dateibrowser](docs/media/file-browser.png) Hinzufügen — im Dateisystem nach einer `.torrent`/`.metalink` suchen (`^o`) |
| ![Helles Thema](docs/media/list-light.png) Tokyo Night Day (`T` wechselt) | |

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

Standardmäßig findet aria2t `aria2c` im `PATH`, startet einen privaten Daemon auf einem freien Port mit zufälligem Secret und verwaltet dessen kompletten Lebenszyklus: Die komplette Sitzung — fertige Downloads wie auch laufende Übertragungen — wird beim Beenden gespeichert und beim nächsten Start wiederhergestellt (fertige erscheinen wieder als abgeschlossen, sie werden nicht neu geladen), der Kindprozess wird sauber gestoppt (saveSession- + shutdown-RPC, Signale nur als Eskalation). Falls ein Absturz einmal einen Daemon am Laufen lässt, räumt der nächste Start ihn ab, bevor er einen frischen startet. Nichts zu konfigurieren, nichts bleibt laufen.

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

**Hinzufügen.** `a` öffnet das Overlay. Liegt in der Zwischenablage eine URL oder ein Magnet-Link, ist er schon eingetragen. Ein URI pro Zeile bedeutet Spiegel derselben Datei; `^t` wechselt durch die Tabs, darunter der Eingabedatei-Tab, der jeden Download aus einer aria2-Eingabedatei (URLs + Optionen pro Download) stapelweise hinzufügt; `^o` durchsucht das Dateisystem nach der Datei, statt den Pfad zu tippen; `^r` benennt das Ziel um; `^s` startet pausiert. Auf dem leeren Startbildschirm des ersten Laufs fügt `↵` direkt einen Link aus der Zwischenablage hinzu.

**Dateien auswählen.** Bei Mehrdatei-Torrents, Metalinks und Magnets öffnet sich der Kästchen-Baum, bevor überhaupt etwas lädt (ein Magnet bleibt nach dem Auflösen seiner Metadaten pausiert, bis du wählst): `space` schaltet eine Datei oder einen ganzen Ordner um, `a`/`n` wählen alles/nichts, `h`/`l` klappen einen Ordner ein/aus, `↵` bestätigt (und startet, falls gewünscht), `esc` bricht ab. Beendest du, bevor du wählst, öffnet sich die Auswahl beim nächsten Start erneut — der Download bleibt bis zur Auswahl pausiert, sodass nie etwas Ungewolltes geladen wird. Später jederzeit mit `f` wieder erreichbar. Ein-Datei-Torrents überspringen den Schritt.

**Die Liste.** Der Standard-Tab **Alle** zeigt jeden Download auf einmal; `tab` oder `1`–`4` wechseln zwischen Alle / Active / Waiting / Stopped. `space` schaltet je nach Zustand des Eintrags Pause/Fortsetzen um, `P`/`U` pausieren und setzen alles fort, `d` entfernt (mit Rückfrage), `D` leert die ganze Stopped-Liste, `y` kopiert die Quell-URL (oder einen aus dem Info-Hash gebauten Magnet-Link) in die Zwischenablage, `/` filtert beim Tippen nach Namen — `enter` behält den Filter (ein `⌕`-Abzeichen zeigt ihn), `esc` löscht ihn. Beim Beenden mit noch laufenden Downloads fragt aria2t erst nach.

**Warteschlange.** Im Waiting-Tab greifen `J`/`K` den gewählten Eintrag und verschieben ihn; `gg`/`G` schicken ihn an Anfang/Ende, `↵` legt ab. Während des Ziehens friert die Liste ein, die Endposition wird gegen die live-Warteschlange neu berechnet — der Eintrag landet dort, wo es aussieht, selbst wenn zwischendurch Downloads fertig wurden.

**Bandbreite.** `l` drosselt den gewählten Download mit Preset-Chips (`∞`/1M/5M/10M/eigene). Ein in den Einstellungen (`,`) gesetztes globales Limit wird gespeichert und beim Neustart des Daemons erneut angewendet. Der Zeitplaner (`S`) wendet globale Limits nach Tageszeit und Wochentag an — Regeln wie „5 MiB/s zur Arbeitszeit, nachts unbegrenzt" laufen über `changeGlobalOption` und überleben Reconnects (und haben, solange er aktiviert ist, Vorrang vor dem gespeicherten manuellen Limit).

**Integrität.** Im Stopped-Tab speichert `c` die erwartete sha-256, `v` liest die lokale Datei und vergleicht, `R` reiht bei Abweichung anhand der aufgezeichneten URIs neu ein. Fehlgeschlagene Downloads nennen den Grund in Klartext direkt in der Liste (etwa „Datei auf dem Server nicht gefunden (404)" oder „nicht genug Speicherplatz") statt eines rohen Fehlercodes.

**Benachrichtigungen.** Wenn ein Download fertig wird oder scheitert — egal auf welchem Bildschirm — nennt die Statuszeile den Namen und die Terminal-Glocke läutet. Funktioniert auch über reines HTTP-Polling; WebSocket-Push macht es nur sofortig.

## Tastenbelegung

| Kontext | Tasten |
|---|---|
| Liste | `a` hinzufügen · `space` Pause/weiter · `P`/`U` alles pausieren/fortsetzen · `d` entfernen · `f` Dateien wählen · `y` Quelle kopieren · `/` filtern · `↵` Details · `g` Statistik · `l` Limit · `s` Server · `S` Zeitplaner · `t` Seeding · `,` Einstellungen · `T` Thema · `tab`/`1‑4` Tabs · `?` Hilfe · `q` beenden |
| Maus | Klick auf eine Zeile → Details · Klick auf jeden Hinweis der Tastenleiste (Pause, Dateien, verbinden…) · Klick auf den Cancel-/Bestätigen-Button eines Dialogs · Klick auf das FILES-Panel → Dateiauswahl · Rad scrollt · kein Doppelklick nötig |
| Waiting-Tab | `J`/`K` greifen + verschieben · `gg`/`G` Anfang/Ende · `↵` ablegen · `esc` abbrechen |
| Stopped-Tab | `c` Prüfsumme einfügen · `v` prüfen · `R` neu laden · `D` Liste leeren · `o` Ordner öffnen |
| Dateiauswahl | `space` umschalten · `a`/`n` alle/keine · `h`/`l` ein-/ausklappen · `↵` bestätigen · `esc` abbrechen · `^o` (in Hinzufügen) Datei suchen |
| Detail | `p` Pause/weiter · `d` entfernen · `f` Dateien wählen · `t` Tracker · `o` Ordner öffnen |
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
  "split": 16,
  "globalDown": "5M",
  "globalUp": "512K",
  "seedRatio": "1.5",
  "seedTime": "0"
}
```

Ein Server mit `managed` ist der eingebaute Daemon (Port und Secret werden beim Start gewählt). Alles andere sind externe Endpunkte; `path` überschreibt den RPC-Pfad für Daemons hinter einem Reverse-Proxy. `globalDown`/`globalUp` sind die gespeicherten globalen Geschwindigkeitslimits (beim Verbinden erneut auf den Daemon angewendet; weglassen oder `""` für unbegrenzt). `seedRatio`/`seedTime` sind die gespeicherten globalen Seeding-Standards (ebenfalls beim Verbinden erneut angewendet, unabhängig vom Zeitplaner). Ausgelassene Felder behalten ihre Defaults.

## Entwicklung

Das Modul hält **100 % Statement-Coverage** — jede Funktion, jedes Paket — und die Latte wird durchgesetzt:

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # darf nichts ausgeben
```


