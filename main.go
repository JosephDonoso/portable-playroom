package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const version = "0.4.3"

//go:embed web/* THIRD_PARTY_NOTICES.txt
var assets embed.FS

func handler(addresses []string) http.Handler {
	mux := http.NewServeMux()
	base := ""
	if len(addresses) > 0 {
		base = addresses[0]
	}
	uno := &app{g: newGame(), clients: map[*subscriber]bool{}, url: base, addresses: addresses}
	mux.Handle("/uno/", http.StripPrefix("/uno", uno))
	mux.HandleFunc("/uno", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/uno/", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			Version   string   `json:"version"`
			Addresses []string `json:"addresses"`
			Games     []string `json:"games"`
		}{version, addresses, []string{"uno"}})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}
		files := map[string]string{"/": "web/index.html", "/style.css": "web/style.css", "/app.js": "web/app.js", "/licenses": "THIRD_PARTY_NOTICES.txt"}
		name, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		content, err := assets.ReadFile(name)
		if err != nil {
			http.Error(w, "Recurso no disponible", 500)
			return
		}
		kind := "text/html"
		if strings.HasSuffix(name, ".css") {
			kind = "text/css"
		}
		if strings.HasSuffix(name, ".js") {
			kind = "text/javascript"
		}
		if strings.HasSuffix(name, ".txt") {
			kind = "text/plain"
		}
		w.Header().Set("Content-Type", kind+"; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != "HEAD" {
			_, _ = w.Write(content)
		}
	})
	return mux
}

func networkURLs(port int) []string {
	result := []string{}
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.Contains(name, "vethernet") || strings.Contains(name, "virtual") || strings.Contains(name, "bluetooth") {
			continue
		}
		addresses, _ := iface.Addrs()
		for _, addr := range addresses {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip.To4() == nil || ip.IsLinkLocalUnicast() || ip.IsLoopback() {
				continue
			}
			result = append(result, fmt.Sprintf("http://%s:%d", ip, port))
		}
	}
	// Determine the preferred outgoing adapter without transmitting application data.
	if conn, err := net.Dial("udp4", "192.0.2.1:9"); err == nil {
		preferred := fmt.Sprintf("http://%s:%d", conn.LocalAddr().(*net.UDPAddr).IP, port)
		_ = conn.Close()
		for i, url := range result {
			if url == preferred {
				result[0], result[i] = result[i], result[0]
				break
			}
		}
	}
	return result
}

func openBrowser(url string) {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err == nil {
		go func() { _ = command.Wait() }()
	}
}

func listen(port int) (net.Listener, error) {
	// Default port falls back to nearby ports when another local app is running.
	for attempt := 0; attempt < 20; attempt++ {
		p := port
		if p != 0 {
			p += attempt
		}
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			return listener, nil
		}
		if port == 0 {
			return nil, err
		}
	}
	return nil, fmt.Errorf("no hay un puerto disponible entre %d y %d", port, port+19)
}

func main() {
	port := flag.Int("port", 3000, "Puerto inicial (se prueba con los siguientes si está ocupado)")
	noBrowser := flag.Bool("no-browser", false, "No abrir el navegador automáticamente")
	flag.Parse()
	if *port < 0 || *port > 65516 {
		fmt.Println("Puerto inválido: elige entre 0 y 65516.")
		return
	}
	listener, err := listen(*port)
	if err != nil {
		fmt.Println("No se pudo abrir Portable Playroom:", err)
		fmt.Println("Presiona Enter para salir.")
		var line string
		_, _ = fmt.Scanln(&line)
		return
	}
	actual := listener.Addr().(*net.TCPAddr).Port
	local := fmt.Sprintf("http://localhost:%d", actual)
	addresses := networkURLs(actual)
	fmt.Printf("PORTABLE PLAYROOM · v%s\n\nMenú: %s\n", version, local)
	if len(addresses) > 0 {
		fmt.Printf("Desde otro dispositivo en la misma red: %s\n", addresses[0])
	}
	fmt.Println("\nUNO disponible. Entra al juego desde el menú.\nMantén esta ventana abierta. Para cerrar, presiona Ctrl+C o cierra esta ventana.")
	server := &http.Server{Handler: handler(addresses), ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	if !*noBrowser {
		openBrowser(local)
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Println("El servidor se detuvo:", err)
	}
}
