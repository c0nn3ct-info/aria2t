[English](./README.md) · [Русский](./README.ru.md) · [简体中文](./README.zh-CN.md) · [Español](./README.es.md) · [Deutsch](./README.de.md) · [日本語](./README.ja.md)

<h1 align="center">aria2t</h1>

<p align="center"><strong>Cliente de terminal para el gestor de descargas aria2</strong></p>
<p align="center"><em>Gestión de descargas, torrents, enlaces magnet y Metalink desde el terminal.</em></p>

<p align="center">
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+"></a>
  <a href="https://aria2.github.io/"><img src="https://img.shields.io/badge/engine-aria2-5c7cfa" alt="Engine: aria2"></a>
  <a href="https://github.com/charmbracelet/bubbletea"><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-ff69b4" alt="TUI: Bubble Tea"></a>
  <img src="https://img.shields.io/badge/coverage-100%25-brightgreen" alt="Coverage: 100%">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License: Apache-2.0"></a>
</p>

<p align="center">
  <img alt="Demo de aria2t" src="./tui/docs/media/demo.gif" width="720">
</p>

> [!IMPORTANT]
> Las descargas las realiza [aria2](https://aria2.github.io/); aria2t es su panel de control. Por defecto aria2t lanza y gestiona su propio demonio `aria2c`, así que no hace falta configurar nada. Si se le indica un aria2 que ya está en marcha, se conecta a él como un cliente RPC normal. aria2t no recopila analíticas ni telemetría y solo se comunica por red con el servidor aria2 configurado.

aria2t es un cliente de terminal para el gestor de descargas aria2, con la paleta Tokyo Night. Se comunica con aria2 por JSON-RPC: bien con un demonio privado que él mismo lanza y gestiona, bien con cualquier aria2 que ya esté corriendo en un seedbox, un NAS o una máquina remota. Todas las acciones están disponibles tanto con el teclado como con el ratón, y la aplicación se compila en un único binario estático de Go.

## ✨ Características

- **Compatibilidad con todas las fuentes de aria2** — espejos de URL, `.torrent`, `.metalink`, enlaces magnet y archivos de entrada de aria2.
- **Arranque sin configuración** — aria2t encuentra `aria2c`, lanza un demonio privado y gestiona todo su ciclo de vida; un aria2 externo se conecta con la opción `--url`.
- **Gestión de descargas** — pausa y reanudación de una o de todas, eliminación, reordenación de la cola, límites de velocidad por descarga o según un horario.
- **Detalles de cada descarga** — mapa de piezas, peers y velocidad de los espejos, progreso por archivo, ratio y control del seeding de BitTorrent.
- **Comprobación de integridad** — el archivo terminado se coteja con una suma sha-256 y se vuelve a descargar si no coincide; los errores se describen en lenguaje claro, no con códigos.
- **Conservación del estado entre sesiones** — las descargas terminadas y en curso se restauran, igual que una ventana de selección de archivos cerrada sin responder.

## 📦 Fuentes compatibles

`HTTP(S)` · `FTP` · `SFTP` · `BitTorrent` · `magnet:` · `.torrent` · `Metalink` · `archivo de entrada de aria2`

En aria2t se puede añadir todo lo que aria2 acepta: URLs normales (varias líneas se tratan como espejos del mismo archivo), enlaces magnet y archivos `.torrent` con soporte de DHT y seeding, `.metalink`, y el formato por lotes `--input-file` de aria2 con opciones separadas para cada entrada.

## 🧩 Cómo funciona

El demonio aria2 descarga los archivos; la interfaz lo controla por JSON-RPC.

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

Por defecto aria2t encuentra `aria2c` en el `PATH` y arranca un demonio privado en un puerto aleatorio con un secreto aleatorio. La sesión se guarda al salir y se restaura en el siguiente arranque, y el proceso hijo se detiene limpiamente junto con el programa; un demonio que quedó huérfano tras un fallo se apaga en el siguiente arranque. Si se ha configurado un servidor externo, el demonio integrado no se arranca y aria2t funciona como un cliente RPC normal.

## 📥 Instalación

### Antes de empezar

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — solo para el demonio integrado; no hace falta para conectar con un servidor externo.
- Un terminal con soporte de 256 colores o truecolor. El ratón es opcional: todas las acciones están disponibles desde el teclado.
- Go 1.25 o más reciente para compilar desde el código fuente.

### Compilar y ejecutar

```sh
git clone https://github.com/c0nn3ct-info/aria2t.git
cd aria2t/tui && go build -o aria2t ./cmd/aria2t
./aria2t
```

En el primer arranque aria2t lanza el demonio privado y abre una lista de descargas vacía. La tecla `a` abre el formulario de añadir y `↵` añade un enlace desde el portapapeles.

### Conectar con un aria2 externo

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

La configuración se guarda en `~/.config/aria2t/config.json` (la ruta puede cambiarse con `--config`); el demonio gestionado mantiene su sesión bajo `~/.config/aria2t/daemon/`.

### Actualizar

Recompile el binario y sustituya el antiguo. La configuración, las reglas del planificador y la sesión del demonio se guardan bajo `~/.config/aria2t/` y sobreviven a la sustitución.

### Desinstalar

1. Borre el binario.
2. Borre el directorio de datos `~/.config/aria2t/` (configuración, sesión del demonio, registros).

## ⌨️ Uso

**Añadir.** `a` abre el formulario de añadir, prellenado desde el portapapeles. Varias URIs, una por línea, se tratan como espejos del mismo archivo. `^t` cambia de pestaña (URL · `.torrent` · `.metalink` · archivo de entrada), `^o` abre un explorador de archivos y `^s` añade la descarga en pausa. En la pantalla vacía del primer arranque, `↵` añade un enlace desde el portapapeles.

**Elegir archivos.** Los torrents multiarchivo, Metalink y los enlaces magnet abren un árbol de archivos con casillas; un enlace magnet permanece en pausa hasta que se confirma la selección. `space` marca un archivo o una carpeta, `a` y `n` seleccionan todo o nada, `↵` confirma la selección. La selección puede cambiarse más tarde con `f`. Si se sale antes de confirmar, la ventana se abre de nuevo en el siguiente arranque.

**La lista.** `tab` o las teclas `1`–`4` cambian entre las pestañas All / Active / Waiting / Stopped. `space` pausa y reanuda la descarga seleccionada, `P` y `U` hacen lo mismo con todas, `d` elimina, `l` limita la velocidad, `/` filtra por nombre, `y` copia la URL de origen, `↵` abre los detalles. En la pestaña Waiting, `J` y `K` mueven el elemento seleccionado dentro de la cola.

**Velocidad.** `l` limita la velocidad de la descarga seleccionada; los valores se eligen entre preajustes. El límite global se fija en los ajustes (`,`), se guarda y se vuelve a aplicar tras el reinicio del demonio. `S` abre el planificador, donde los límites globales se definen por hora del día, por ejemplo 5 MiB/s en horario laboral y sin límite por la noche.

**Integridad.** En la pestaña Stopped, `c` guarda la suma sha-256 esperada, `v` coteja el archivo local con ella, `R` vuelve a descargar el archivo si no coincide y `X` limpia la lista.

La lista completa de atajos de teclado se abre con **`?`**. Todas las indicaciones de la barra inferior también responden al clic; en los diálogos el botón verde confirma la acción y el rojo la cancela.

## ⚙️ Configuración

`~/.config/aria2t/config.json` (los campos omitidos conservan sus valores por defecto):

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

Un servidor con el campo `managed` es el demonio integrado; su puerto y su secreto se eligen al arrancarlo. El resto de entradas describe servidores externos. `globalDown` y `globalUp` definen los límites de velocidad globales guardados, y `seedRatio` y `seedTime` los parámetros de seeding por defecto; al conectar, todos estos valores se vuelven a aplicar al demonio. Las reglas del planificador (`S`) también se guardan en este archivo.

## ❓ Preguntas frecuentes

**¿Qué es aria2 y por qué necesita una interfaz aparte?**
[aria2](https://aria2.github.io/) es un motor de descargas rápido compatible con HTTP(S), FTP, SFTP, BitTorrent y Metalink. Funciona en segundo plano, se controla por JSON-RPC y no tiene interfaz propia más allá de comandos puntuales. aria2t le añade una lista de descargas en tiempo real, selección de archivos, gestión de la cola, límites de velocidad y comprobación de integridad.

**¿En qué se diferencia de ejecutar `aria2c` directamente?**
`aria2c` o bien descarga un archivo y termina, o bien funciona como un demonio controlado por scripts. aria2t asume la gestión de ese demonio: lo arranca, guarda y restaura la sesión, lo detiene limpiamente al salir y apaga un demonio que quedó huérfano tras un fallo. Sobre esa base, aria2t ofrece una interfaz interactiva.

**¿Funciona con un aria2 que ya tengo en marcha?**
Sí. La opción `--url ws://host:6800/jsonrpc --secret …` o el selector de servidores integrado con sondas de latencia conectan aria2t con cualquier aria2 por WebSocket o HTTP(S), por ejemplo en un seedbox, un NAS o un contenedor Docker. En ese caso el demonio integrado no se arranca.

**¿Se conservan las descargas al salir del programa?**
Sí. El demonio gestionado guarda la sesión al salir y la restaura en el siguiente arranque: las descargas activas, en espera, terminadas y en seeding se restauran, y los archivos ya descargados se reconocen en lugar de descargarse de nuevo. Una ventana de selección de archivos cerrada sin responder también se abre otra vez.

**¿Se puede elegir qué archivos de un torrent descargar?**
Sí. Los torrents, los enlaces magnet y Metalink se añaden en pausa y primero se abre un árbol de archivos; un enlace magnet se pausa tras recibir sus metadatos. La transferencia de datos empieza solo después de confirmar la selección, y esta puede cambiarse más tarde con `f`.

**¿Hace seeding de torrents?**
Sí. Un torrent terminado sigue compartiéndose y su estado cambia a seeding. Los parámetros globales de seeding — ratio y tiempo — se definen en los ajustes y se vuelven a aplicar en cada reinicio del demonio.

**¿Teclado o ratón?**
Ambos están plenamente soportados. Todas las indicaciones de la barra de teclas responden al clic, cada acción del ratón tiene su equivalente de teclado y la lista completa de atajos se abre con `?`.

**¿aria2t envía datos a alguna parte?**
No. El programa no recopila analíticas ni telemetría, no obtiene configuración remota y solo se comunica por red con el servidor aria2 configurado.

## 🛠️ Desarrollo

La cobertura de sentencias del módulo es del **100 %**, y se comprueba automáticamente:

```sh
cd tui
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```

## 🙏 Agradecimientos

- **[aria2](https://github.com/aria2/aria2)** (GPL-2.0) — el motor de descargas que realiza todo el trabajo de transferencia, BitTorrent y Metalink; aria2t es su panel de control.
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**, **[Bubbles](https://github.com/charmbracelet/bubbles)** y **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** (MIT) — la pila TUI de Charm sobre la que está construido aria2t.
- **[Tokyo Night](https://github.com/folke/tokyonight.nvim)** — ambas paletas están tomadas sin cambios de los temas Tokyo Night y Tokyo Night Day.
