# Portable Playroom

Una sala de juegos de mesa portable y autohosteada: el PC se conecta a la tele como pantalla compartida y los jugadores usan sus teléfonos desde el navegador, en la misma red local.

**v0.1.1: menú simplificado con tema oscuro, todavía sin juegos.** Esta versión permite abrir y explorar el menú desde el PC o desde otro dispositivo en la misma red. Aún no incluye salas de partida, QR de acceso, jugadores, cartas ni reglas. Esas funciones llegarán con los juegos.

El menú usa texto grande, fondo gris carbón y acentos azul violeta. La ayuda de conexión se despliega solo cuando la necesitas.

## Descargar y abrir en Windows

1. Entra a [Releases](https://github.com/JosephDonoso/portable-playroom/releases/latest).
2. Descarga `Portable-Playroom-Windows-amd64.exe` (Windows de 64 bits, Intel o AMD).
3. Guárdalo en una carpeta y ábrelo con doble clic.
4. Se abrirá el menú en tu navegador y una consola con la dirección de la sala.

**No necesitas instalar Go, Node.js ni dependencias.** El ejecutable contiene el servidor y todos los archivos de la interfaz. No requiere internet para funcionar ni descarga recursos externos. El navegador predeterminado debe estar disponible en el PC.

Mantén la consola abierta mientras uses la aplicación. Para detenerla, presiona `Ctrl+C` en esa ventana o ciérrala. Cerrar solamente la pestaña del navegador no detiene el servidor.

Si Windows solicita permiso de firewall, permite el acceso en redes privadas para abrir el menú desde otros dispositivos. El ejecutable todavía no está firmado digitalmente, por lo que Windows puede mostrar un aviso al descargarlo o ejecutarlo.

## Probar desde el teléfono

Conecta el PC y el teléfono al mismo router; el PC puede usar cable y el teléfono Wi-Fi. Abre en el teléfono la dirección que muestra la consola o la sección de red local del menú, por ejemplo `http://192.168.1.88:3000`.

`localhost` se refiere al dispositivo en el que lo escribes: en el teléfono debes usar la IP del PC. La IP puede cambiar entre redes o reconexiones. Las redes de invitados y algunas VPN pueden impedir el acceso entre dispositivos. Esta primera versión sirve HTTP y está pensada para una red local de confianza; no publica un túnel ni abre puertos del router.

El puerto inicial es `3000`. Si está ocupado, el programa busca el siguiente disponible, hasta `3019`. Usa siempre el enlace mostrado por el programa.

## La idea

- Un único programa portable aloja los juegos en el PC.
- La tele muestra el tablero y los cambios públicos.
- Los jugadores, incluido el anfitrión, controlan la partida desde sus teléfonos.
- La información privada se envía únicamente al jugador correspondiente.
- La experiencia funciona en red local sin depender de servicios externos.

Estas son las metas del proyecto; esta publicación contiene únicamente el menú.

## Desarrollo

Requiere Go 1.24 o posterior **solo para desarrollar o compilar**. El servidor usa la biblioteca estándar y la interfaz utiliza HTML, CSS y JavaScript sin dependencias externas.

```sh
go run .
go test ./...
go vet ./...
```

Opciones:

```sh
go run . -port 4000
go run . -no-browser
```

La interfaz está en `web/` y se incorpora al ejecutable con `go:embed`. Los cambios en estos archivos requieren volver a compilar.

### Compilar para Windows

En PowerShell:

```powershell
./scripts/build-windows.ps1
```

Genera el ejecutable, un ZIP y un archivo SHA-256 en `dist/`. También se puede compilar directamente en Windows con:

```sh
go build -trimpath -ldflags="-s -w" -o dist/Portable-Playroom-Windows-amd64.exe .
```

GitHub Actions comprueba el código y genera el ejecutable en cada actualización de `main`. Los ejecutables publicados se encuentran en Releases; los binarios no se guardan en el historial de código.

## Estructura

```text
main.go                  Servidor local y apertura del navegador
main_test.go             Pruebas de rutas, recursos y puertos
web/                     Menú incrustado en el ejecutable
scripts/                 Compilación y empaquetado
.github/workflows/       Validación y compilación en GitHub
THIRD_PARTY_NOTICES.txt   Licencia del runtime de Go, incluida en el ejecutable
```
