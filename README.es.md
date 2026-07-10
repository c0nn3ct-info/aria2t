[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<h1 align="center">aria2t</h1>

<p align="center"><strong>Interfaz de terminal para el gestor de descargas aria2</strong></p>
<p align="center"><em>Descargas, torrents, magnets y metalinks — una sola TUI de teclado y ratón, cero configuración.</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
</p>

<p align="center">
  <img alt="Demo de aria2t" src="./docs/media/demo.gif" width="720">
</p>

> [!IMPORTANT]
> aria2t es una superficie de control — [aria2](https://aria2.github.io/) es quien descarga. Por defecto lanza y gestiona por ti un demonio `aria2c` privado (cero configuración); apuntado a un aria2 que ya tengas corriendo, se convierte en un cliente RPC puro. Sin analíticas, sin telemetría — solo habla con el endpoint de aria2 que tú configures.

aria2t es una interfaz de terminal para el gestor de descargas aria2, con la paleta Tokyo Night. Habla con aria2 por JSON-RPC — con un demonio privado que él mismo lanza y gestiona por ti, o con cualquier aria2 que ya tengas corriendo en un seedbox, un NAS o una máquina remota — y cada acción funciona con teclado **y** ratón. Un único binario estático de Go.

## ✨ Características

- **Añade todo lo que aria2 acepta** — espejos de URL, `.torrent`, `.metalink`, enlaces magnet, archivos de entrada de aria2 — y te deja **elegir qué archivos descargar antes de que nada empiece**.
- **Cero configuración** — encuentra `aria2c`, lanza un demonio privado y gestiona todo su ciclo de vida. O apúntalo a un aria2 externo con `--url`.
- **Maneja cada descarga** — pausar/reanudar una o todas, eliminar, reordenar la cola, limitar la velocidad por descarga o según un horario del día.
- **Muestra el interior de una descarga** — mapa por piezas, peers y velocidades de los espejos, progreso por archivo, ratio de subida y controles de seeding de BitTorrent.
- **Te mantiene honesto** — verifica un archivo terminado contra un sha-256 y lo vuelve a descargar si no coincide; explica los fallos en lenguaje claro en lugar de códigos de error.
- **Sobrevive a los reinicios** — las descargas terminadas y en curso reaparecen, y un selector de archivos que cerraste sin responder se vuelve a abrir.
- **No te estorba** — hace sonar la campana cuando algo termina, filtra mientras escribes, cambia de servidor con sondas de latencia y alterna entre tema claro y oscuro.

## 📦 Fuentes compatibles

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `archivo de entrada de aria2`

Todo lo que aria2 acepta, aria2t lo añade: URLs planas — una por línea significa espejos del mismo archivo —, enlaces magnet y archivos `.torrent` con DHT y seeding, `.metalink`, y el propio formato por lotes `--input-file` de aria2 con opciones por descarga. Los torrents multiarchivo, los magnets y los metalinks abren un árbol de casillas **antes** de que empiece la descarga — nunca se descarga nada no deseado.

## 🖥️ Pantallas

| | |
|---|---|
| ![Lista de descargas](./docs/media/list-active.png) La lista — progreso y velocidades en vivo, una columna STATUS coloreada | ![Selector de archivos](./docs/media/files-picker.png) Elige archivos antes de que nada se descargue |
| ![Detalle](./docs/media/detail.png) Detalle — mapa de piezas, peers/espejos, progreso por archivo | ![Añadir](./docs/media/add-overlay.png) Añadir — prellenado desde el portapapeles, botones verde/rojo |
| ![Integridad](./docs/media/stopped-integrity.png) Verificación sha-256 + errores en lenguaje claro | ![Planificador](./docs/media/scheduler.png) Planificador de ancho de banda por horario |

## 🧩 Cómo funciona

La TUI nunca descarga nada por sí misma — lo hace un demonio de aria2, y entre ambos solo fluye JSON-RPC.

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

Por defecto aria2t encuentra `aria2c` en tu `PATH` y ejecuta un demonio privado en un puerto aleatorio con un secreto aleatorio: la sesión completa se guarda al salir y se restaura en el siguiente arranque, y el proceso hijo se detiene limpiamente cuando sales (un demonio huérfano por un fallo se recoge en el siguiente arranque). Apúntalo en cambio a un servidor externo y el demonio integrado nunca arranca — aria2t pasa a ser un cliente RPC puro.

## 📥 Instalación

### Antes de empezar

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — solo para el modo cero-configuración; no hace falta para conectar con un servidor externo.
- Un terminal con soporte de 256 colores o truecolor. El ratón es opcional; toda acción tiene su tecla.
- Go 1.25+ para compilar desde el código fuente.

### Compilar y ejecutar

```sh
go build -o aria2t ./cmd/aria2t     # or: go install aria2t/cmd/aria2t
./aria2t
```

El primer arranque lanza el demonio privado y te deja en una lista vacía, lista para `a` — o para `↵`, que añade un enlace directamente desde el portapapeles.

### Conectar con un aria2 externo

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

La configuración persiste en `~/.config/aria2t/config.json` (`--config` la sobreescribe); el demonio gestionado guarda su sesión bajo `~/.config/aria2t/daemon/`.

### Actualizar

Recompila y sustituye el binario — la configuración, las reglas del planificador y la sesión del demonio viven bajo `~/.config/aria2t/` y sobreviven al cambio.

### Desinstalar

1. Borra el binario.
2. Borra el directorio de datos: `~/.config/aria2t/` (configuración, sesión del demonio, registros).

## ⌨️ Uso

**Añadir.** `a` abre el formulario de añadir, prellenado desde tu portapapeles. Un URI por línea significa espejos del mismo archivo; `^t` recorre las pestañas (URL · `.torrent` · `.metalink` · archivo de entrada), `^o` explora el disco en busca de un archivo, `^s` alterna empezar en pausa. En la pantalla vacía del primer arranque, `↵` añade un enlace directamente desde el portapapeles.

**Elegir archivos.** Los torrents multiarchivo, los metalinks y los magnets abren un árbol de casillas **antes de que nada se descargue** (un magnet queda en pausa hasta que elijas): `space` alterna un archivo o una carpeta, `a`/`n` seleccionan todo/nada, `↵` confirma. Pulsa `f` para cambiar la selección más tarde. Sal antes de elegir y el selector se vuelve a abrir en el siguiente arranque — nunca se descarga nada no deseado.

**Maneja la lista.** `tab` o `1`–`4` alternan entre All / Active / Waiting / Stopped. `space` pausa/reanuda la fila seleccionada, `P`/`U` lo hacen con todas, `d` elimina, `l` limita, `/` filtra mientras escribes, `y` copia la URL de origen, `↵` abre los detalles. En la pestaña Waiting, `J`/`K` agarran y mueven un elemento de la cola.

**Ancho de banda.** `l` limita la descarga seleccionada con chips predefinidos. Un límite global fijado en los ajustes (`,`) se guarda y se vuelve a aplicar cuando el demonio se reinicia. `S` planifica los límites globales por hora del día («5 MiB/s en el trabajo, sin límite de noche»).

**Integridad.** En la pestaña Stopped: `c` guarda un sha-256 esperado, `v` verifica el archivo local, `R` vuelve a descargar si no coincide, `X` limpia la lista.

Pulsa **`?`** en cualquier momento para el mapa completo de teclas. Cada indicación de la barra de teclas es también clicable, y los diálogos usan botones verdes (continuar) / rojos (cancelar).

## ⚙️ Configuración

`~/.config/aria2t/config.json` (los campos que omitas conservan sus valores por defecto):

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

Un servidor `managed` es el demonio integrado (el puerto y el secreto se deciden al lanzarlo); cualquier otra cosa es un endpoint externo. `globalDown`/`globalUp` son los límites de velocidad globales guardados y `seedRatio`/`seedTime` los valores de seeding por defecto — todos se vuelven a aplicar al demonio al conectar. Las reglas del planificador (`S`) también se guardan aquí.

## ❓ Preguntas frecuentes

**¿Qué es aria2 y por qué ponerle una TUI?**
[aria2](https://aria2.github.io/) es un motor de descargas rápido y multiprotocolo — HTTP(S), FTP, SFTP, BitTorrent, Metalink — que corre sin interfaz y habla JSON-RPC. No tiene interfaz propia más allá de comandos de un solo uso. aria2t es esa interfaz: lista en vivo, selección de archivos, reordenación de la cola, límites de velocidad y comprobaciones de integridad en una sola TUI a pantalla completa.

**¿En qué se diferencia de ejecutar `aria2c` a secas?**
`aria2c` o bien descarga una URL y termina, o bien corre como un demonio contra el que escribes scripts. aria2t gestiona ese demonio por ti — lanzamiento, guardado/restauración de sesión, apagado limpio, recogida de un demonio huérfano tras un fallo — y pone encima toda la parte interactiva.

**¿Funciona con un aria2 que ya tengo corriendo?**
Sí. `--url ws://host:6800/jsonrpc --secret …` (o el selector de servidores integrado, con sondas de latencia) conecta con cualquier aria2 por WebSocket o HTTP(S) — seedbox, NAS, contenedor Docker. El demonio integrado nunca arranca.

**¿Sobreviven las descargas al salir?**
Sí. El demonio gestionado guarda su sesión completa al salir y la restaura en el siguiente arranque — las descargas activas, en cola, terminadas y en seeding vuelven todas, y los archivos terminados se reconocen en lugar de volver a descargarse. Incluso un selector de archivos que cerraste sin responder se vuelve a abrir.

**¿Puedo elegir archivos dentro de un torrent antes de que se descargue?**
Sí — los torrents, magnets y metalinks se añaden en pausa y primero se abre un árbol de casillas (un magnet espera sus metadatos y luego se pausa). Nada se transfiere hasta que confirmes, y `f` reabre la selección más tarde.

**¿Hace seeding?**
Sí. Un torrent terminado sigue subiendo y su estado cambia a un *seeding* diferenciado; los valores globales de ratio/tiempo viven en los ajustes y se vuelven a aplicar cada vez que el demonio se reinicia.

**¿Teclado o ratón?**
Ambos, del todo: cada indicación de la barra de teclas es clicable y cada clic tiene su tecla equivalente. `?` muestra el mapa completo.

**¿aria2t envía algo a alguna parte?**
No. Sin analíticas, sin telemetría, sin configuración remota — el único interlocutor de red es el endpoint de aria2 que tú configures.

## 🛠️ Desarrollo

El módulo mantiene **cobertura de sentencias del 100 %**, aplicada:

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```


## 🙏 Agradecimientos

- **[aria2](https://github.com/aria2/aria2)** (GPL-2.0) — el motor de descargas que hace todo el trabajo real de transferencia, BitTorrent y Metalink. aria2t es una superficie de control; aria2 hace el trabajo.
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**, **[Bubbles](https://github.com/charmbracelet/bubbles)** y **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** (MIT) — la pila TUI de Charm sobre la que está construido aria2t.
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — ambas paletas están tomadas literalmente de Tokyo Night / Tokyo Night Day.
