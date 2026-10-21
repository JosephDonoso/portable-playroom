package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestTableEndResetsRoundAndNotifiesPlayers(t *testing.T) {
	g := setup(t, 2)
	g.Options.TurnSeconds = 30
	g.Deadline = time.Now().Add(time.Minute)
	g.Pending = 6
	a := &app{g: g, clients: map[*subscriber]bool{}}
	p := g.Players[0]
	s := &subscriber{p: p, updates: make(chan []byte, 1)}
	a.clients[s] = true
	a.scheduleClock()
	req := httptest.NewRequest("POST", "http://localhost:3000/api/table/end", nil)
	req.RemoteAddr = "127.0.0.1:4321"
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if a.g.Phase != "lobby" || !a.g.Deadline.IsZero() || a.g.Pending != 0 || a.g.Top != nil || a.g.Winner != "" {
		t.Fatal("round was not cleared")
	}
	if a.g.find(p.Token) != p || len(p.Hand) != 0 || a.g.Options.TurnSeconds != 30 {
		t.Fatal("player identity or options lost")
	}
	select {
	case <-s.updates:
	default:
		t.Fatal("phones were not notified")
	}
	if a.g.expireTurn(time.Now().Add(time.Hour)) {
		t.Fatal("ended round timed out")
	}
	if err := a.g.start(p); err != nil {
		t.Fatalf("cannot start again: %v", err)
	}
}

func TestTableEndRequiresLocalSameOriginRequest(t *testing.T) {
	for _, tc := range []struct{ remote, origin string }{
		{"192.168.1.42:4321", "http://localhost:3000"},
		{"127.0.0.1:4321", "http://other.example"},
		{"127.0.0.1:4321", ""},
	} {
		a := &app{g: setup(t, 2), clients: map[*subscriber]bool{}}
		req := httptest.NewRequest("POST", "http://localhost:3000/api/table/end", nil)
		req.RemoteAddr = tc.remote
		req.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)
		if w.Code != 403 || a.g.Phase != "playing" {
			t.Fatalf("unauthorized end: %s %s status %d", tc.remote, tc.origin, w.Code)
		}
	}
}
