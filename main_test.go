package main

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http/httptest"
	"testing"
)

func TestEmbeddedMenu(t *testing.T) {
	h := handler([]string{"http://192.168.1.88:3000"})
	for route, file := range map[string]string{"/": "web/index.html", "/style.css": "web/style.css", "/app.js": "web/app.js", "/licenses": "THIRD_PARTY_NOTICES.txt"} {
		t.Run(route, func(t *testing.T) {
			r := httptest.NewRecorder()
			h.ServeHTTP(r, httptest.NewRequest("GET", route, nil))
			want, e := assets.ReadFile(file)
			if e != nil {
				t.Fatal(e)
			}
			if r.Code != 200 || !bytes.Equal(r.Body.Bytes(), want) {
				t.Fatal("Recurso incrustado incorrecto")
			}
		})
	}
}
func TestMenuHasNoGames(t *testing.T) {
	r := httptest.NewRecorder()
	handler([]string{}).ServeHTTP(r, httptest.NewRequest("GET", "/api/info", nil))
	var v struct {
		Version   string
		Addresses []string
		Games     []string
	}
	if e := json.Unmarshal(r.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if r.Code != 200 || v.Version != version || v.Games == nil || len(v.Games) != 0 || v.Addresses == nil {
		t.Fatal("Estado inicial incorrecto")
	}
	for _, route := range []string{"/play", "/api/join", "/api/action", "/events", "/game.go", "/../main.go"} {
		r := httptest.NewRecorder()
		handler(nil).ServeHTTP(r, httptest.NewRequest("GET", route, nil))
		if r.Code == 200 {
			t.Fatal("Ruta inesperada expuesta:", route)
		}
	}
}
func TestMethodAndHead(t *testing.T) {
	h := handler(nil)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("POST", "/", nil))
	if r.Code != 405 {
		t.Fatal("POST debe rechazarse")
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("HEAD", "/", nil))
	if r.Code != 200 || r.Body.Len() != 0 {
		t.Fatal("HEAD incorrecto")
	}
}
func TestPortFallback(t *testing.T) {
	first, e := net.Listen("tcp", ":0")
	if e != nil {
		t.Fatal(e)
	}
	defer first.Close()
	p := first.Addr().(*net.TCPAddr).Port
	second, e := listen(p)
	if e != nil {
		t.Fatal(e)
	}
	defer second.Close()
	if second.Addr().(*net.TCPAddr).Port == p {
		t.Fatal("No cambió el puerto ocupado")
	}
}
