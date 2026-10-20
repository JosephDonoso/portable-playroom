package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUnoRoutes(t *testing.T) {
	h := handler([]string{"http://192.168.1.88:3000"})
	for route, kind := range map[string]string{"/uno/": "text/html", "/uno/play": "text/html", "/uno/app.js": "text/javascript", "/uno/style.css": "text/css", "/uno/qr": "image/png"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", route, nil))
		if r.Code != 200 || !strings.HasPrefix(r.Header().Get("Content-Type"), kind) {
			t.Fatalf("%s: %d %s", route, r.Code, r.Header().Get("Content-Type"))
		}
	}
}

func TestUnoEffectsAndPrivateStream(t *testing.T) {
	g := setup(t, 3)
	a := &app{g: g, clients: map[*subscriber]bool{}}
	post := func(p *Player, action Action) int {
		t.Helper()
		action.Token = p.Token
		b, _ := json.Marshal(action)
		r := httptest.NewRecorder()
		a.ServeHTTP(r, httptest.NewRequest("POST", "/api/action", strings.NewReader(string(b))))
		return r.Code
	}
	p := g.Players[0]
	if post(p, Action{Type: "play", Card: "invalid"}) != 400 || g.Revision != 0 {
		t.Fatal("Una jugada rechazada emitió un efecto")
	}
	p.Hand[0] = Card{"reverse-card", "red", "reverse"}
	if post(p, Action{Type: "play", Card: "reverse-card"}) != 200 || g.Revision != 1 || g.Effect.Value != "reverse" || g.Effect.Actor != p.Name {
		t.Fatal("Efecto reversa incorrecto")
	}
	server := httptest.NewServer(a)
	defer server.Close()
	response, err := http.Get(server.URL + "/events?token=" + p.Token)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var s State
	if err = json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &s); err != nil {
		t.Fatal(err)
	}
	if s.Me != p.ID || len(s.Hand) != 2 || s.Revision != 1 || s.Effect.Value != "reverse" {
		t.Fatal("Estado privado o efecto incorrecto")
	}
	if strings.Contains(line, g.Players[1].Token) || strings.Contains(line, `"id":"`+g.Players[1].Hand[0].ID+`"`) {
		t.Fatal("El stream revela información ajena")
	}
}

func TestSpecialEffects(t *testing.T) {
	for _, value := range []string{"skip", "draw2", "draw4", "wild"} {
		t.Run(value, func(t *testing.T) {
			g := setup(t, 3)
			p := g.Players[0]
			color := "red"
			if value == "draw4" || value == "wild" {
				color = "wild"
			}
			p.Hand = []Card{{"special", color, value}, {"other", "blue", "2"}, {"third", "blue", "3"}}
			a := &app{g: g, clients: map[*subscriber]bool{}}
			b, _ := json.Marshal(Action{Type: "play", Card: "special", Color: "green", Token: p.Token})
			r := httptest.NewRecorder()
			a.ServeHTTP(r, httptest.NewRequest("POST", "/api/action", strings.NewReader(string(b))))
			if r.Code != 200 || g.Effect.Value != value || g.Revision != 1 {
				t.Fatal("Efecto especial incorrecto", r.Body.String())
			}
		})
	}
	g := setup(t, 3)
	p := g.Players[0]
	g.Pile = append(g.Pile, Card{"drawn", "red", "1"})
	a := &app{g: g, clients: map[*subscriber]bool{}}
	b, _ := json.Marshal(Action{Type: "draw", Token: p.Token})
	r := httptest.NewRecorder()
	a.ServeHTTP(r, httptest.NewRequest("POST", "/api/action", strings.NewReader(string(b))))
	if r.Code != 200 || g.Effect.Type != "draw" || g.Drawn != "drawn" {
		t.Fatal("Robo incorrecto")
	}
}
