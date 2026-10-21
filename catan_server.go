package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

func catName(name string) string {
	r := []rune(strings.TrimSpace(name))
	if len(r) > 24 {
		r = r[:24]
	}
	return string(r)
}

type catPublicPlayer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Host        bool   `json:"host"`
	Count       int    `json:"count"`
	Development int    `json:"development"`
	Knights     int    `json:"knights"`
	Points      int    `json:"points"`
	RoadLength  int    `json:"roadLength"`
	Connected   bool   `json:"connected"`
}
type catState struct {
	Phase         string            `json:"phase"`
	Board         catBoard          `json:"board"`
	Players       []catPublicPlayer `json:"players"`
	Me            string            `json:"me"`
	Turn          string            `json:"turn"`
	Winner        string            `json:"winner"`
	Hand          catResources      `json:"hand"`
	Dev           []catDev          `json:"dev"`
	Bank          catResources      `json:"bank"`
	Rates         catResources      `json:"rates"`
	Dice          [2]int            `json:"dice"`
	Offers        []catOffer        `json:"offers"`
	Events        []string          `json:"events"`
	Discards      map[string]int    `json:"discards"`
	NeedDiscard   int               `json:"needDiscard"`
	Serial        int               `json:"serial"`
	Revision      int               `json:"revision"`
	PlayedDev     bool              `json:"playedDev"`
	Longest       string            `json:"longest"`
	Largest       string            `json:"largest"`
	DeckCount     int               `json:"deckCount"`
	LegalVertices []int             `json:"legalVertices"`
	LegalCities   []int             `json:"legalCities"`
	LegalEdges    []int             `json:"legalEdges"`
	Victims       []string          `json:"victims"`
}
type catSubscriber struct {
	token   string
	updates chan []byte
}
type catApp struct {
	mu      sync.Mutex
	g       *catGame
	clients map[*catSubscriber]bool
	seen    map[string]time.Time
	url     string
}

func catNewApp(base string) *catApp {
	return &catApp{g: catNewGame(), clients: map[*catSubscriber]bool{}, seen: map[string]time.Time{}, url: base}
}
func (a *catApp) snapshot(token string) []byte {
	g := a.g
	p := g.find(token)
	s := catState{Phase: g.Phase, Board: g.Board, Players: []catPublicPlayer{}, Bank: g.Bank, Dice: g.Dice, Offers: g.Offers, Events: g.Events, Discards: g.Discards, Winner: g.Winner, Serial: g.Serial, Revision: g.Revision, PlayedDev: g.PlayedDev, Longest: g.Longest, Largest: g.Largest, DeckCount: len(g.Deck), Dev: []catDev{}, LegalVertices: []int{}, LegalCities: []int{}, LegalEdges: []int{}, Victims: []string{}}
	if len(g.Players) > 0 {
		s.Turn = g.Players[g.Turn].ID
	}
	for i, other := range g.Players {
		connected := false
		for client := range a.clients {
			if client.token == other.Token && time.Since(a.seen[other.ID]) < heartbeatLease {
				connected = true
				break
			}
		}
		s.Players = append(s.Players, catPublicPlayer{other.ID, other.Name, other.Color, i == 0, other.Hand.total(), len(other.Dev), other.Knights, g.score(other, g.Phase == "finished"), g.Board.longest(other.ID), connected})
	}
	if p != nil {
		s.Me = p.ID
		s.Hand = p.Hand
		s.Dev = p.Dev
		s.NeedDiscard = g.Discards[p.ID]
		for r := 0; r < 5; r++ {
			s.Rates[r] = g.rate(p, r)
		}
		for i := range s.Players {
			if s.Players[i].ID == p.ID {
				s.Players[i].Points = g.score(p, true)
			}
		}
		if s.Turn == p.ID {
			r, settlements, cities := g.pieces(p.ID)
			if g.Phase == "setup-settlement" || g.Phase == "main" && settlements < 5 && p.Hand.has(catCosts["settlement"]) {
				for i := range g.Board.Vertices {
					if g.Board.settlementSpot(i) && (g.Phase == "setup-settlement" || g.Board.touchesRoad(i, p.ID)) {
						s.LegalVertices = append(s.LegalVertices, i)
					}
				}
			}
			if g.Phase == "main" && cities < 4 && p.Hand.has(catCosts["city"]) {
				for i, v := range g.Board.Vertices {
					if v.Owner == p.ID && !v.City {
						s.LegalCities = append(s.LegalCities, i)
					}
				}
			}
			if r < 15 && (g.Phase == "setup-road" || g.Phase == "roads" || g.Phase == "main" && p.Hand.has(catCosts["road"])) {
				for i, e := range g.Board.Edges {
					if g.Board.roadSpot(i, p.ID) && (g.Phase != "setup-road" || e.A == g.LastSettlement || e.B == g.LastSettlement) {
						s.LegalEdges = append(s.LegalEdges, i)
					}
				}
			}
			if g.Phase == "steal" {
				s.Victims = g.Board.victims(g.Board.Robber, p.ID, g.Players)
			}
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return b
}
func (a *catApp) broadcast() {
	for c := range a.clients {
		data := a.snapshot(c.token)
		select {
		case <-c.updates:
		default:
		}
		c.updates <- data
	}
}
func (a *catApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == "GET" {
		switch r.URL.Path {
		case "/api/info":
			reply(w, 200, map[string]any{"url": a.url, "tableControl": localTable(r)})
			return
		case "/events":
			a.events(w, r)
			return
		case "/qr":
			b, e := qrcode.Encode(a.url+"/catan/play", qrcode.Medium, 230)
			if e != nil {
				http.Error(w, "QR no disponible", 500)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Write(b)
			return
		}
		files := map[string]string{"/": "index.html", "/play": "index.html", "/app.js": "app.js", "/style.css": "style.css"}
		if f, ok := files[r.URL.Path]; ok {
			data, err := assets.ReadFile("web/catan/" + f)
			if err != nil {
				http.NotFound(w, r)
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
			w.Write(data)
			return
		}
	}
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	fail := func(err error) { reply(w, 400, map[string]string{"error": err.Error()}) }
	if r.URL.Path == "/api/table/end" {
		origin, e := url.Parse(r.Header.Get("Origin"))
		if !localTable(r) || e != nil || origin.Host != r.Host || origin.Scheme != "http" {
			reply(w, 403, map[string]string{"error": "Usa la pantalla del PC anfitrión."})
			return
		}
		players, revision := a.g.Players, a.g.Revision
		a.g = catNewGame()
		a.g.Players = players
		a.g.Revision = revision + 1
		for _, p := range players {
			p.Hand = catResources{}
			p.Dev = []catDev{}
			p.Knights = 0
		}
		a.broadcast()
		reply(w, 200, map[string]bool{"ok": true})
		return
	}
	if r.URL.Path == "/api/join" {
		var b struct{ Name, Token string }
		if decode(w, r, &b) != nil {
			fail(fmt.Errorf("Solicitud inválida."))
			return
		}
		p, e := a.g.join(b.Name, b.Token)
		if e != nil {
			fail(e)
			return
		}
		a.seen[p.ID] = time.Now()
		a.g.Revision++
		a.broadcast()
		reply(w, 200, map[string]string{"token": p.Token})
		return
	}
	if r.URL.Path != "/api/action" && r.URL.Path != "/api/heartbeat" {
		http.NotFound(w, r)
		return
	}
	var action catAction
	if decode(w, r, &action) != nil {
		fail(fmt.Errorf("Solicitud inválida."))
		return
	}
	p := a.g.find(action.Token)
	if p == nil {
		fail(fmt.Errorf("Vuelve a entrar a la sala."))
		return
	}
	a.seen[p.ID] = time.Now()
	if r.URL.Path == "/api/heartbeat" {
		reply(w, 200, map[string]bool{"ok": true})
		return
	}
	if err := a.g.act(p, action); err != nil {
		fail(err)
		return
	}
	a.g.awardBonuses()
	a.g.checkWin()
	a.g.Revision++
	a.broadcast()
	reply(w, 200, map[string]bool{"ok": true})
}
func (a *catApp) events(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming no disponible", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	a.mu.Lock()
	c := &catSubscriber{r.URL.Query().Get("token"), make(chan []byte, 1)}
	a.clients[c] = true
	if p := a.g.find(c.token); p != nil {
		a.seen[p.ID] = time.Now()
	}
	a.broadcast()
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.clients, c); a.broadcast(); a.mu.Unlock() }()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-c.updates:
			http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			f.Flush()
		case <-ticker.C:
			a.mu.Lock()
			data := a.snapshot(c.token)
			a.mu.Unlock()
			http.NewResponseController(w).SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			f.Flush()
		}
	}
}
