package main

import (
	"encoding/json"
	"fmt"
	qrcode "github.com/skip2/go-qrcode"
	"net/http"
	"strings"
	"sync"
	"time"
)

type subscriber struct {
	p       *Player
	updates chan []byte
}
type app struct {
	mu        sync.Mutex
	g         *Game
	clients   map[*subscriber]bool
	url       string
	addresses []string
}

func (a *app) snapshot(p *Player) []byte {
	b, e := json.Marshal(a.g.state(p))
	if e != nil {
		panic(e)
	}
	return b
}

// Called under mu. Slow devices receive the latest full snapshot, never hold up a turn.
func (a *app) broadcast() {
	for s := range a.clients {
		data := a.snapshot(s.p)
		select {
		case <-s.updates:
		default:
		}
		s.updates <- data
	}
}
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(v)
}
func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == "GET" {
		switch r.URL.Path {
		case "/api/info":
			reply(w, 200, map[string]any{"url": a.url, "addresses": a.addresses})
			return
		case "/qr":
			b, e := qrcode.Encode(a.url+"/uno/play", qrcode.Medium, 230)
			if e != nil {
				http.Error(w, "No se pudo generar el QR", 500)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(b)
			return
		case "/events":
			a.events(w, r)
			return
		case "/licenses":
			b, _ := assets.ReadFile("THIRD_PARTY_NOTICES.txt")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write(b)
			return
		default:
			files := map[string]string{"/": "index.html", "/play": "index.html", "/app.js": "app.js", "/style.css": "style.css"}
			if f, ok := files[r.URL.Path]; ok {
				b, e := assets.ReadFile("web/uno/" + f)
				if e != nil {
					http.Error(w, "Archivo no disponible", 500)
					return
				}
				kind := "text/html"
				if strings.HasSuffix(f, ".js") {
					kind = "text/javascript"
				}
				if strings.HasSuffix(f, ".css") {
					kind = "text/css"
				}
				w.Header().Set("Content-Type", kind+"; charset=utf-8")
				_, _ = w.Write(b)
				return
			}
		}
	}
	if r.Method == "POST" && (r.URL.Path == "/api/join" || r.URL.Path == "/api/action") {
		a.mu.Lock()
		defer a.mu.Unlock()
		fail := func(e error) { reply(w, 400, map[string]string{"error": e.Error()}) }
		if r.URL.Path == "/api/join" {
			var b struct {
				Name  string `json:"name"`
				Token string `json:"token"`
			}
			if e := decode(w, r, &b); e != nil {
				fail(fmt.Errorf("Solicitud inválida."))
				return
			}
			p, e := a.g.join(b.Name, b.Token)
			if e != nil {
				fail(e)
				return
			}
			reply(w, 200, map[string]string{"token": p.Token, "id": p.ID})
			a.broadcast()
			return
		}
		var b Action
		if e := decode(w, r, &b); e != nil {
			fail(fmt.Errorf("Solicitud inválida."))
			return
		}
		player := a.g.find(b.Token)
		if e := a.g.act(player, b); e != nil {
			fail(e)
			return
		}
		a.g.Revision++
		a.g.Effect = &Effect{Type: b.Type, Actor: player.Name}
		if b.Type == "play" {
			a.g.Effect.Value = a.g.Top.Value
		}
		reply(w, 200, map[string]bool{"ok": true})
		a.broadcast()
		return
	}
	http.NotFound(w, r)
}
func (a *app) events(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming no disponible", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	a.mu.Lock()
	s := &subscriber{a.g.find(r.URL.Query().Get("token")), make(chan []byte, 1)}
	a.clients[s] = true
	s.updates <- a.snapshot(s.p)
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.clients, s); a.mu.Unlock() }()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	send := func(b []byte) error {
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, e := w.Write(b)
		if e == nil {
			f.Flush()
		}
		return e
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-s.updates:
			if send(append(append([]byte("data: "), data...), []byte("\n\n")...)) != nil {
				return
			}
		case <-ticker.C:
			if send([]byte(": ping\n\n")) != nil {
				return
			}
		}
	}
}
