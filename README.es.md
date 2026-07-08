# aria2t

[English](README.md) · [Русский](README.ru.md) · [简体中文](README.zh-CN.md) · **Español** · [Deutsch](README.de.md) · [日本語](README.ja.md)

aria2t es una interfaz de terminal para el gestor de descargas [aria2](https://aria2.github.io/), con la paleta Tokyo Night. Habla con aria2 por JSON-RPC — con un demonio privado que él mismo lanza y gestiona por ti (cero configuración), o con cualquier aria2 que ya tengas corriendo en un seedbox, un NAS o una máquina remota. Un único binario estático de Go, manejado con teclado **y** ratón.

![Demo de aria2t](docs/media/demo.gif)

## Qué hace

- **Añade todo lo que aria2 acepta** — espejos de URL, `.torrent`, `.metalink`, enlaces magnet, archivos de entrada de aria2 — y te deja **elegir qué archivos descargar antes de que nada empiece**.
- **Cero configuración**: encuentra `aria2c`, lanza un demonio privado y gestiona todo su ciclo de vida. O apúntalo a un aria2 externo con `--url`.
- **Maneja cada descarga**: pausar/reanudar una o todas, eliminar, reordenar la cola, limitar la velocidad por descarga o según un horario del día.
- **Muestra el interior de una descarga**: mapa por piezas, peers y velocidades de los espejos, progreso por archivo, ratio de subida y controles de seeding de BitTorrent.
- **Te mantiene honesto**: verifica un archivo terminado contra un sha-256 y lo vuelve a descargar si no coincide; explica los fallos en lenguaje claro en lugar de códigos de error.
- **Sobrevive a los reinicios**: las descargas terminadas y en curso reaparecen, y un selector de archivos que cerraste sin responder se vuelve a abrir.
- **No te estorba**: hace sonar la campana cuando algo termina, filtra mientras escribes, cambia de servidor con sondas de latencia y alterna entre tema claro y oscuro.

## Pantallas

| | |
|---|---|
| ![Lista de descargas](docs/media/list-active.png) La lista — progreso y velocidades en vivo, una columna STATUS coloreada | ![Selector de archivos](docs/media/files-picker.png) Elige archivos antes de que nada se descargue |
| ![Detalle](docs/media/detail.png) Detalle — mapa de piezas, peers/espejos, progreso por archivo | ![Añadir](docs/media/add-overlay.png) Añadir — prellenado desde el portapapeles, botones verde/rojo |
| ![Integridad](docs/media/stopped-integrity.png) Verificación sha-256 + errores en lenguaje claro | ![Planificador](docs/media/scheduler.png) Planificador de ancho de banda por horario |

## Cómo funciona

```mermaid
flowchart LR
    subgraph Terminal
      UI["aria2t<br/>Bubble Tea TUI"]
    end
    subgraph Machine["Your machine"]
      D["managed aria2c<br/>(spawned on demand)"]
    end
    R(("external aria2<br/>seedbox · NAS"))

    UI -- "JSON-RPC over ws:// (push + poll)" --> D
    UI -. "ws:// or http(s)://" .-> R
```

Por defecto aria2t encuentra `aria2c` en tu `PATH` y ejecuta un demonio privado en un puerto aleatorio con un secreto aleatorio: la sesión completa se guarda al salir y se restaura en el siguiente arranque, y el proceso hijo se detiene limpiamente cuando sales (un demonio huérfano por un fallo se recoge en el siguiente arranque). Apúntalo en cambio a un servidor externo y el demonio integrado nunca arranca — aria2t pasa a ser un cliente RPC puro.

## Requisitos

- [aria2](https://aria2.github.io/) (`brew install aria2` / `apt install aria2`) — solo para el modo cero-configuración; no hace falta para conectar con un servidor externo.
- Un terminal con soporte de 256 colores o truecolor. El ratón es opcional; toda acción tiene su tecla.
- Go 1.25+ para compilar desde el código fuente.

## Instalación

```sh
go build -o aria2t ./cmd/aria2t     # or: go install aria2t/cmd/aria2t
./aria2t
```

El primer arranque lanza el demonio privado y te deja en una lista vacía, lista para `a`. Para usar un aria2 que ya tengas corriendo:

```sh
./aria2t --url ws://seedbox:6800/jsonrpc --secret mysecret
```

La configuración persiste en `~/.config/aria2t/config.json` (`--config` la sobreescribe); el demonio gestionado guarda su sesión bajo `~/.config/aria2t/daemon/`.

## Uso

**Añadir.** `a` abre el formulario de añadir, prellenado desde tu portapapeles. Un URI por línea significa espejos del mismo archivo; `^t` recorre las pestañas (URL · `.torrent` · `.metalink` · archivo de entrada), `^o` explora el disco en busca de un archivo, `^s` alterna empezar en pausa. En la pantalla vacía del primer arranque, `↵` añade un enlace directamente desde el portapapeles.

**Elegir archivos.** Los torrents multiarchivo, los metalinks y los magnets abren un árbol de casillas **antes de que nada se descargue** (un magnet queda en pausa hasta que elijas): `space` alterna un archivo o una carpeta, `a`/`n` seleccionan todo/nada, `↵` confirma. Pulsa `f` para cambiar la selección más tarde. Sal antes de elegir y el selector se vuelve a abrir en el siguiente arranque — nunca se descarga nada no deseado.

**Maneja la lista.** `tab` o `1`–`4` alternan entre All / Active / Waiting / Stopped. `space` pausa/reanuda la fila seleccionada, `P`/`U` lo hacen con todas, `d` elimina, `l` limita, `/` filtra mientras escribes, `y` copia la URL de origen, `↵` abre los detalles. En la pestaña Waiting, `J`/`K` agarran y mueven un elemento de la cola.

**Ancho de banda.** `l` limita la descarga seleccionada con chips predefinidos. Un límite global fijado en los ajustes (`,`) se guarda y se vuelve a aplicar cuando el demonio se reinicia. `S` planifica los límites globales por hora del día («5 MiB/s en el trabajo, sin límite de noche»).

**Integridad.** En la pestaña Stopped: `c` guarda un sha-256 esperado, `v` verifica el archivo local, `R` vuelve a descargar si no coincide, `X` limpia la lista.

Pulsa **`?`** en cualquier momento para el mapa completo de teclas. Cada indicación de la barra de teclas es también clicable, y los diálogos usan botones verdes (continuar) / rojos (cancelar).

## Configuración

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

## Desarrollo

El módulo mantiene **cobertura de sentencias del 100 %**, aplicada:

```sh
go vet ./... && gofmt -l .
go test ./... -count=1 -coverprofile=cover.out -coverpkg=./...
go tool cover -func=cover.out | awk '$3!="100.0%"'      # must print nothing
```

