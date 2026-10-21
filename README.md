# Portable Playroom

Una sala de juegos de mesa portable y autohosteada: el PC se conecta a la tele como pantalla compartida y los jugadores usan sus teléfonos desde el navegador, en la misma red local.

**v0.4.3: expulsión de jugadores por el anfitrión y botón «Abrir sala» más corto.** Incluye reloj de turno, opciones del anfitrión y acumulaciones de +2/+4. Incluye reparto y recorridos entre mazo, mano y descarte; descarte con cartas superpuestas e inclinadas; inspección táctil de la mano. Se mantienen los indicadores de conexión y el salto de turnos desconectados por el anfitrión.

El menú usa texto grande, fondo gris carbón y acentos azul violeta. La ayuda de conexión se despliega solo cuando la necesitas.

## Descargar y abrir en Windows

1. Entra a [Releases](https://github.com/JosephDonoso/portable-playroom/releases/latest).
2. Descarga `Portable-Playroom-Windows-amd64.exe` (Windows de 64 bits, Intel o AMD).
3. Guárdalo en una carpeta y ábrelo con doble clic.
4. Se abrirá el menú en tu navegador y una consola con la dirección de la sala.
5. Elige **UNO → Abrir en la tele** y escanea el QR desde cada teléfono, conectado al mismo router.
6. El primer jugador será el anfitrión. Podrá iniciar cuando haya al menos dos jugadores.

**No necesitas instalar Go, Node.js ni dependencias.** El ejecutable contiene el servidor y todos los archivos de la interfaz. No requiere internet para funcionar ni descarga recursos externos. El navegador predeterminado debe estar disponible en el PC.

Mantén la consola abierta mientras uses la aplicación. Para detenerla, presiona `Ctrl+C` en esa ventana o ciérrala. Cerrar solamente la pestaña del navegador no detiene el servidor.

**Cerrar el servidor borra la partida.** Recargar el mismo navegador recupera tu lugar mientras el servidor siga abierto. Cada jugador debe usar su propio dispositivo o un perfil de navegador independiente: las pestañas del mismo navegador comparten la identidad.

### Administrar jugadores

Desde su teléfono, el anfitrión puede desplegar **Administrar jugadores** y pulsar **Expulsar** junto al nombre de otro jugador. Está disponible en la sala, durante la ronda y entre rondas. El anfitrión no puede expulsarse.

El expulsado pierde su lugar y sus cartas, recibe un aviso y su identidad anterior deja de permitir acciones. Si tenía el turno, continúa el siguiente jugador según el sentido de juego y se conserva la deuda pendiente. Si quedan menos de dos jugadores, se abre la sala. Las cartas retiradas vuelven al mazo durante una ronda. La expulsión no impide volver a entrar como un jugador nuevo cuando la sala esté abierta.

### Jugadores desconectados

Cada jugador muestra **Conectado**, **Reconectando…** o **Desconectado**, tanto en la tele como en los teléfonos. Al detectar una desconexión se conceden 8 segundos para regresar. Si es el turno de ese jugador y sigue ausente, el anfitrión puede pulsar **Saltar su turno** desde su teléfono. El salto no roba ni elimina cartas, conserva su asiento y respeta el sentido de juego. No es automático: si llega otro turno del mismo jugador, el anfitrión decide de nuevo.

Cerrar una pestaña se detecta al cerrarse su conexión. Un corte silencioso de Wi-Fi puede tardar unos 9 segundos en detectarse antes del margen de 8 segundos, más un pequeño intervalo de actualización. Se comprueba actividad cada 3 segundos. Si hay varias pestañas del mismo jugador, sigue conectado mientras una mantenga la conexión y la actividad. Un teléfono que suspenda el navegador en segundo plano también puede aparecer como desconectado; al volver a abrirlo recupera su estado.

El servidor comprueba que quien solicita el salto sea el anfitrión y que el turno siga siendo el mismo. Si el jugador se reconecta antes de procesar el salto, este se rechaza. El anfitrión no se reemplaza automáticamente si se desconecta.

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

## UNO: controles y reglas

Un borde violeta en el teléfono indica cuándo puedes jugar. El centro de la mesa se atenúa mientras esperas y recupera su iluminación cuando te toca. El turno, el reloj y la acumulación ocupan espacios fijos para que la mano y los controles no salten al cambiar los avisos. Toca una carta iluminada o roba del mazo. Al tocar un comodín, aparece una ventana con cuatro botones grandes de color; puedes cancelar antes de jugar. Hay animaciones al robar, recibir y jugar cartas, y efectos para bloqueo, reversa, +2, +4 y cambio de color. Se respeta la preferencia de movimiento reducido del dispositivo.

Las cartas salen visualmente del mazo y viajan a la mano; al jugarlas, viajan al descarte. Los movimientos de otros jugadores muestran únicamente dorsos. Las últimas cinco cartas públicas forman una pila con ángulos y desplazamientos estables, también al reconectar. El reparto se anima carta por carta; reconectarse no repite movimientos antiguos.

La mano se ordena con los comodines multicolor primero (comodín y +4), seguidos por rojo, amarillo, verde y azul. Dentro de cada color aparecen los números de menor a mayor y luego bloqueo, reversa y +2. El orden es visual y no cambia las reglas ni la identidad de las cartas.

Mantener el dedo en un borde desplaza la mano continuamente en esa dirección, más rápido cuanto más cerca esté del extremo. Se detiene al soltar el dedo, cancelar el gesto o volver al centro. No es necesario seguir moviendo el dedo para recorrer una mano larga.

La mano usa cartas superpuestas, con margen lateral para ampliar las cartas de los extremos sin recortarlas. Solo la carta seleccionada se eleva; los eventos de ratón emulados después de un toque no cambian la selección. En el teléfono, toca o desliza el dedo sobre ellas para ampliar la carta seleccionada; luego pulsa **Jugar carta**. Deslizar no juega una carta accidentalmente. El borde de la mano permite desplazarse cuando tienes muchas cartas. En PC puedes inspeccionar con el ratón o el teclado y jugar haciendo clic en una carta válida. Las cartas que no sirven también pueden inspeccionarse. El selector de color aparece después de confirmar un comodín.

De 2 a 10 jugadores, 7 cartas por persona. Juega por color o símbolo. +4 se permite aunque tengas cartas del color actual. Bloqueo y reversa siguen activos; con dos jugadores, reversa permite jugar de nuevo. Marca **¡UNO!** antes de jugar tu penúltima carta: olvidarlo añade dos cartas.

### Controles desde la tele

En el PC anfitrión, abre la mesa mediante `http://localhost:3000/uno/` (o el puerto indicado). **Terminar partida** cierra la ronda y devuelve a los teléfonos a la sala, conservando los jugadores y las opciones. **Menú de juegos** termina la ronda en curso y vuelve al selector de juegos. Estos controles se habilitan en el propio PC.

La vista del celular reúne una mesa más pequeña, la mano y botones de al menos 44 px; el historial está plegado en **Últimas jugadas**. Los jugadores aparecen en una fila horizontal y conservan sus indicadores de conexión.

### Opciones del anfitrión

Se configuran antes de iniciar cada ronda, desde su teléfono. Se guardan con **Guardar opciones** o automáticamente al iniciar:

- **Tiempo por turno:** 0 para desactivarlo; entre 5 y 600 segundos para activarlo. Cuenta regresiva visible en tele y teléfonos, con aviso visual en los últimos cinco segundos.
- **Robo:** una carta o robar hasta encontrar una que se pueda jugar. La carta válida recién robada es la única que se puede jugar después del robo. Si se agota el mazo disponible, el turno avanza sin bloquearse.
- **Jugada obligatoria:** si está activada, no puedes robar ni pasar si tienes una carta válida; tampoco pasar después de robar una válida. Si está desactivada, puedes robar voluntariamente o pasar si tienes una carta disponible.
- **Mezclar +2 y +4:** permite combinar ambos tipos en una misma deuda, con la restricción de color explicada abajo.

El reloj lo controla el servidor, incluso sin navegadores abiertos. Robar no reinicia el contador. Al agotarse, el jugador recibe la deuda pendiente y pierde el turno; si no hay deuda, roba según el modo configurado y pasa. Si ya había robado una carta válida, no roba otra vez al vencer el tiempo. El reloj sigue corriendo si se pierde la conexión.

### Acumulaciones: reglas fijas

- **+2 sobre +2** siempre se acumula, aunque cambie el color.
- **+4 sobre +4** siempre se acumula.
- Si se activa la mezcla: **+4 sobre +2** es válido; **+2 sobre +4** solo si ese +2 tiene el color elegido por el +4.
- No se puede responder con bloqueo, reversa, un número ni un comodín normal a una deuda.
- Si el siguiente jugador no tiene una carta válida para acumular, recibe automáticamente el total y pierde el turno. Si tiene una y la jugada no es obligatoria, puede elegir **Recibir +N** en lugar de acumular.
- La deuda pendiente aparece en ambas interfaces. Cuando se cobra automáticamente, también se muestra quién recibió cuántas cartas. Si el mazo no alcanza, se entregan las disponibles y se informa la cantidad real.
- Saltar por desconexión o vencer el tiempo no evita pagar la deuda.

Una última carta +2 o +4 todavía puede recibir respuesta: la victoria se decide al resolver la deuda, entre quienes sigan sin cartas, priorizando a quien se quedó sin ellas primero. Así, una acumulación que vuelve al jugador puede obligarlo a robar y evitar su victoria.

Sin desafíos de +4, puntuación acumulada, expulsión de jugadores ni guardado en disco. Una sala UNO por ejecutable. Proyecto independiente, sin afiliación con los titulares de UNO.

## Desarrollo

Requiere Go 1.24 o posterior **solo para desarrollar o compilar**. La interfaz utiliza HTML, CSS y JavaScript sin dependencias externas. La única dependencia de Go genera el QR y queda incluida en el ejecutable.

```sh
go run .
go test ./...
go vet ./...
```

Las pruebas de gestos de la interfaz requieren Node.js solo en desarrollo: `node --test tests/motion.test.cjs`. No se incluye Node.js ni se requiere en los equipos que usan el ejecutable.

Opciones:

```sh
go run . -port 4000
go run . -no-browser
```

La interfaz está en `web/` y se incorpora al ejecutable con `go:embed`. Los cambios en estos archivos requieren volver a compilar.

El menú está en `/`, el tablero en `/uno/` y los controles privados en `/uno/play`. Las jugadas se validan en el PC: las manos privadas no se envían a la pantalla pública ni a otros jugadores.

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
uno.go                   API de UNO y sincronización con los dispositivos
presence.go              Estado de conexión y salto autorizado por el anfitrión
game.go                  Reglas y vistas privadas de la partida
rules.go                 Opciones, acumulaciones y reloj del servidor
rules_test.go            Pruebas de opciones, cadenas, vencimientos y fin de ronda
web/uno/motion.js         Recorridos de cartas, pila e inspección de la mano
tests/motion.test.cjs     Pruebas de gestos y descarte estable
main_test.go             Pruebas de rutas, recursos y puertos
game_test.go             Pruebas de reglas, turnos y privacidad
presence_test.go         Pruebas de conexión, reconexión y permisos de salto
uno_connection_test.go   Pruebas de pérdida de conexión y recuperación de identidad
web/                     Menú e interfaz de UNO incrustados en el ejecutable
scripts/                 Compilación y empaquetado
.github/workflows/       Validación y compilación en GitHub
THIRD_PARTY_NOTICES.txt   Licencias incorporadas en el ejecutable
```
