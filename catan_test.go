package main

import (
	"encoding/json"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func catSetup(t *testing.T) *catGame {
	t.Helper()
	g := catNewGame()
	for _, name := range []string{"Ana", "Bruno", "Carla", "Diego"} {
		if _, e := g.join(name, ""); e != nil {
			t.Fatal(e)
		}
	}
	if e := g.start(g.Players[0]); e != nil {
		t.Fatal(e)
	}
	g.First, g.Turn = 0, 0 // Deterministic seat order for individual rule scenarios.
	return g
}
func catPlaceAll(t *testing.T, g *catGame) {
	t.Helper()
	for g.SetupStep < 8 {
		p := g.Players[g.Turn]
		v := -1
		for i := range g.Board.Vertices {
			if g.Board.settlementSpot(i) {
				v = i
				break
			}
		}
		if v < 0 {
			t.Fatal("no setup site")
		}
		if e := g.act(p, catAction{Type: "build", Kind: "settlement", Index: v}); e != nil {
			t.Fatal(e)
		}
		ei := -1
		for i, e := range g.Board.Edges {
			if e.Owner == "" && (e.A == v || e.B == v) {
				ei = i
				break
			}
		}
		if e := g.act(p, catAction{Type: "build", Kind: "road", Index: ei}); e != nil {
			t.Fatal(e)
		}
	}
}
func catConservation(t *testing.T, g *catGame) {
	t.Helper()
	r := g.Bank
	for _, p := range g.Players {
		r.add(p.Hand, 1)
		if !p.Hand.valid() {
			t.Fatal("negative hand")
		}
	}
	if r != (catResources{19, 19, 19, 19, 19}) {
		t.Fatalf("lost resources: %v", r)
	}
}
func TestCatanVariableBoardTopologyAndTokens(t *testing.T) {
	for iteration := 0; iteration < 120; iteration++ {
		b := catNewBoard()
		if len(b.Hexes) != 19 || len(b.Vertices) != 54 || len(b.Edges) != 72 || len(b.Ports) != 9 {
			t.Fatalf("wrong topology %d/%d/%d/%d", len(b.Hexes), len(b.Vertices), len(b.Edges), len(b.Ports))
		}
		counts := map[int]int{}
		tokens := map[int]int{}
		ports := map[int]int{}
		for _, h := range b.Hexes {
			counts[h.Resource]++
			tokens[h.Number]++
		}
		for r, want := range map[int]int{-1: 1, 0: 4, 1: 3, 2: 4, 3: 4, 4: 3} {
			if counts[r] != want {
				t.Fatal("terrain distribution")
			}
		}
		for n, want := range map[int]int{0: 1, 2: 1, 3: 2, 4: 2, 5: 2, 6: 2, 8: 2, 9: 2, 10: 2, 11: 2, 12: 1} {
			if tokens[n] != want {
				t.Fatalf("number distribution %v", tokens)
			}
		}
		for i, h := range b.Hexes {
			if h.Number != 6 && h.Number != 8 {
				continue
			}
			for j, other := range b.Hexes {
				if j <= i || other.Number != 6 && other.Number != 8 {
					continue
				}
				shared := 0
				for _, v := range h.Vertices {
					if slices.Contains(other.Vertices, v) {
						shared++
					}
				}
				if shared == 2 {
					t.Fatal("adjacent sixes/eights")
				}
			}
		}
		for _, p := range b.Ports {
			ports[p.Resource]++
		}
		if ports[-1] != 4 {
			t.Fatal("generic ports")
		}
		for r := 0; r < 5; r++ {
			if ports[r] != 1 {
				t.Fatal("specific ports")
			}
		}
		if b.Hexes[b.Robber].Resource != -1 {
			t.Fatal("robber not in desert")
		}
	}
}
func TestCatanExactlyFourAndSnakeSetup(t *testing.T) {
	g := catNewGame()
	host, _ := g.join("A", "")
	if g.start(host) == nil {
		t.Fatal("started with one")
	}
	g = catSetup(t)
	if _, err := g.join("Fifth", ""); err == nil {
		t.Fatal("fifth joined")
	}
	for _, turn := range []int{0, 1, 2, 3, 3, 2, 1, 0} {
		if g.Turn != turn {
			t.Fatal("wrong snake order")
		}
		p := g.Players[turn]
		v := -1
		for i := range g.Board.Vertices {
			if g.Board.settlementSpot(i) {
				v = i
				break
			}
		}
		if err := g.act(p, catAction{Type: "build", Kind: "settlement", Index: v}); err != nil {
			t.Fatal(err)
		}
		if g.Board.settlementSpot(v) {
			t.Fatal("occupied spot available")
		}
		for _, n := range g.Board.adjacent(v) {
			if g.Board.settlementSpot(n) {
				t.Fatal("distance rule failed")
			}
		}
		edge := -1
		for i, e := range g.Board.Edges {
			if e.Owner == "" && (e.A == v || e.B == v) {
				edge = i
				break
			}
		}
		if err := g.act(p, catAction{Type: "build", Kind: "road", Index: edge}); err != nil {
			t.Fatal(err)
		}
	}
	if g.Phase != "roll" || g.Turn != 0 || g.SetupStep != 8 {
		t.Fatal("setup did not finish")
	}
	for _, p := range g.Players {
		r, s, c := g.pieces(p.ID)
		if r != 2 || s != 2 || c != 0 {
			t.Fatal("initial pieces")
		}
	}
	catConservation(t, g)
}
func TestCatanProductionRobberAndBankShortage(t *testing.T) {
	g := catSetup(t)
	h := &g.Board.Hexes[0]
	h.Resource, h.Number = 0, 5
	g.Board.Robber = 18
	for i := 1; i < len(g.Board.Hexes); i++ {
		g.Board.Hexes[i].Number = 2
	}
	p, q := g.Players[0], g.Players[1]
	g.Board.Vertices[h.Vertices[0]].Owner = p.ID
	g.Board.Vertices[h.Vertices[0]].City = true
	g.Board.Vertices[h.Vertices[2]].Owner = q.ID
	g.produce(5)
	if p.Hand[0] != 2 || q.Hand[0] != 1 {
		t.Fatal("city/settlement production")
	}
	g.Board.Robber = 0
	g.produce(5)
	if p.Hand[0] != 2 {
		t.Fatal("robber did not block")
	}
	g.Board.Robber = 18
	g.Bank[0] = 1
	g.produce(5)
	if p.Hand[0] != 2 || q.Hand[0] != 1 {
		t.Fatal("shortage distributed to multiple players")
	}
	g.Board.Vertices[h.Vertices[2]].Owner = ""
	g.produce(5)
	if p.Hand[0] != 3 || g.Bank[0] != 0 {
		t.Fatal("single recipient shortage")
	}
}
func TestCatanSevenDiscardAndSteal(t *testing.T) {
	g := catSetup(t)
	catPlaceAll(t, g)
	p := g.Players[0]
	p.Hand = catResources{9, 0, 0, 0, 0}
	g.Dice = [2]int{3, 4}
	g.resolveRoll()
	if g.Phase != "discard" || g.Discards[p.ID] != 4 {
		t.Fatal("discard quota")
	}
	if g.act(p, catAction{Type: "discard", Give: catResources{3}}) == nil {
		t.Fatal("wrong discard allowed")
	}
	if e := g.act(p, catAction{Type: "discard", Give: catResources{4}}); e != nil {
		t.Fatal(e)
	}
	if g.Phase != "robber" {
		t.Fatal("did not wait for discards")
	}
	tile := (g.Board.Robber + 1) % 19
	for _, vi := range g.Board.Hexes[tile].Vertices {
		g.Board.Vertices[vi].Owner = ""
	}
	victim := g.Players[1]
	victim.Hand = catResources{0, 1}
	g.Board.Vertices[g.Board.Hexes[tile].Vertices[0]].Owner = victim.ID
	if err := g.act(p, catAction{Type: "robber", Index: tile}); err != nil {
		t.Fatal(err)
	}
	if victim.Hand.total() != 0 || p.Hand[1] != 1 || g.Phase != "main" {
		t.Fatal("steal not resolved")
	}
}
func TestCatanBankPortsAndBuildCosts(t *testing.T) {
	g := catSetup(t)
	catPlaceAll(t, g)
	g.Phase = "main"
	p := g.Players[0]
	p.Hand = catResources{6, 6, 6, 6, 6}
	for i := range g.Board.Vertices {
		g.Board.Vertices[i].Port = -2
	}
	if g.rate(p, 0) != 4 {
		t.Fatal("bank rate")
	}
	for i := range g.Board.Vertices {
		if g.Board.Vertices[i].Owner == p.ID {
			g.Board.Vertices[i].Port = -1
			break
		}
	}
	if g.rate(p, 0) != 3 {
		t.Fatal("generic rate")
	}
	for i := range g.Board.Vertices {
		if g.Board.Vertices[i].Owner == p.ID {
			g.Board.Vertices[i].Port = 0
			break
		}
	}
	if g.rate(p, 0) != 2 {
		t.Fatal("specific rate")
	}
	if err := g.act(p, catAction{Type: "bank", Give: catResources{2}, Want: catResources{0, 0, 0, 1}}); err != nil {
		t.Fatal(err)
	}
	if p.Hand[0] != 4 || p.Hand[3] != 7 {
		t.Fatal("bank exchange")
	}
	if g.act(p, catAction{Type: "bank", Give: catResources{2}, Want: catResources{1}}) == nil {
		t.Fatal("same resource bank trade")
	}
	vi := -1
	for i, v := range g.Board.Vertices {
		if v.Owner == p.ID {
			vi = i
			break
		}
	}
	if err := g.act(p, catAction{Type: "build", Kind: "city", Index: vi}); err != nil {
		t.Fatal(err)
	}
	if !g.Board.Vertices[vi].City || p.Hand[3] != 5 || p.Hand[4] != 3 {
		t.Fatal("city cost")
	}
}
func TestCatanTradingRequiresBothSidesAndRevalidates(t *testing.T) {
	g := catSetup(t)
	catPlaceAll(t, g)
	g.Phase = "main"
	p, q, r := g.Players[0], g.Players[1], g.Players[2]
	p.Hand = catResources{2}
	q.Hand = catResources{0, 2}
	r.Hand = catResources{0, 2}
	if e := g.act(p, catAction{Type: "offer", Give: catResources{1}, Want: catResources{0, 1}}); e != nil {
		t.Fatal(e)
	}
	o := g.Offers[0].ID
	if g.act(p, catAction{Type: "confirm", Offer: o, Target: q.ID}) == nil {
		t.Fatal("unaccepted trade executed")
	}
	if e := g.act(q, catAction{Type: "respond", Offer: o, Response: "accept"}); e != nil {
		t.Fatal(e)
	}
	if p.Hand[0] != 2 {
		t.Fatal("acceptance committed without author")
	}
	q.Hand[1] = 0
	if g.act(p, catAction{Type: "confirm", Offer: o, Target: q.ID}) == nil {
		t.Fatal("stale resources traded")
	}
	q.Hand[1] = 2
	if e := g.act(p, catAction{Type: "confirm", Offer: o, Target: q.ID}); e != nil {
		t.Fatal(e)
	}
	if p.Hand != (catResources{1, 1}) || q.Hand != (catResources{1, 1}) || len(g.Offers) != 0 {
		t.Fatal("trade totals")
	}
	if g.act(p, catAction{Type: "confirm", Offer: o, Target: r.ID}) == nil {
		t.Fatal("trade replayed")
	}
	if e := g.act(q, catAction{Type: "offer", Target: r.ID, Give: catResources{1}, Want: catResources{0, 1}}); e != nil {
		t.Fatal(e)
	}
	if g.Offers[0].To != p.ID {
		t.Fatal("off-turn pair permitted")
	}
	if e := g.act(p, catAction{Type: "end"}); e != nil {
		t.Fatal(e)
	}
	if len(g.Offers) != 0 {
		t.Fatal("offers survived end of turn")
	}
}
func TestCatanDevelopmentMaturityAndVictory(t *testing.T) {
	g := catSetup(t)
	catPlaceAll(t, g)
	g.Phase = "main"
	p := g.Players[0]
	p.Dev = []catDev{{"knight", g.Serial}}
	if g.act(p, catAction{Type: "dev", Kind: "knight"}) == nil {
		t.Fatal("new development played")
	}
	g.Serial++
	p.Knights = 2
	if err := g.act(p, catAction{Type: "dev", Kind: "knight"}); err != nil {
		t.Fatal(err)
	}
	if g.Phase != "robber" || p.Knights != 3 || g.Largest != p.ID {
		t.Fatal("knight/army not awarded")
	}
	g.Phase = "main"
	p.Dev = []catDev{{"monopoly", 0}}
	if g.act(p, catAction{Type: "dev", Kind: "monopoly"}) == nil {
		t.Fatal("two developments played")
	}
	p.Dev = []catDev{}
	for i := 0; i < 10; i++ {
		p.Dev = append(p.Dev, catDev{"vp", g.Serial})
	}
	g.checkWin()
	if g.Phase != "finished" || g.Winner != p.ID {
		t.Fatal("VP cards did not win on purchase turn")
	}
}
func TestCatanPrivacyAndEmbeddedRoutes(t *testing.T) {
	a := catNewApp("")
	a.g = catSetup(t)
	catPlaceAll(t, a.g)
	p, q := a.g.Players[0], a.g.Players[1]
	p.Hand = catResources{9, 8, 7, 6, 5}
	p.Dev = []catDev{{"vp", 0}, {"monopoly", 0}}
	for _, token := range []string{"", q.Token} {
		data := a.snapshot(token)
		var s catState
		if e := json.Unmarshal(data, &s); e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(data), p.Token) || strings.Contains(string(data), "monopoly") {
			t.Fatal("private identity or development leaked")
		}
		if s.Players[0].Points != 2 {
			t.Fatal("hidden VP exposed")
		}
	}
	var s catState
	json.Unmarshal(a.snapshot(p.Token), &s)
	if s.Hand != p.Hand || len(s.Dev) != 2 || s.Players[0].Points != 3 {
		t.Fatal("own private hand missing")
	}
	h := handler(nil)
	for _, route := range []string{"/catan/", "/catan/play", "/catan/app.js", "/catan/style.css", "/catan/api/info", "/catan/qr"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 200 {
			t.Fatal(route, w.Code)
		}
	}
}

func TestCatanLongestRoadBlocksAndKeepsTies(t *testing.T) {
	g := catSetup(t)
	p, q := g.Players[0], g.Players[1]
	g.Board.Vertices = make([]catVertex, 14)
	g.Board.Edges = nil
	for i := 0; i < 6; i++ {
		g.Board.Edges = append(g.Board.Edges, catEdge{A: i, B: i + 1, Owner: p.ID}, catEdge{A: i + 7, B: i + 8, Owner: q.ID})
	}
	g.Longest = p.ID
	g.awardBonuses()
	if g.Board.longest(p.ID) != 6 || g.Longest != p.ID {
		t.Fatal("tie removed incumbent")
	}
	g.Board.Vertices[3].Owner = q.ID
	g.awardBonuses()
	if g.Board.longest(p.ID) != 3 || g.Longest != q.ID {
		t.Fatal("opponent settlement did not break route")
	}
	p.Knights, q.Knights = 3, 3
	g.Largest = p.ID
	g.awardBonuses()
	if g.Largest != p.ID {
		t.Fatal("army tie removed incumbent")
	}
	q.Knights = 4
	g.awardBonuses()
	if g.Largest != q.ID {
		t.Fatal("larger army not transferred")
	}
}

func TestCatanProgressCardsAndFinitePieces(t *testing.T) {
	g := catSetup(t)
	catPlaceAll(t, g)
	g.Phase = "main"
	g.Serial = 2
	p, q := g.Players[0], g.Players[1]
	p.Dev = []catDev{{"plenty", 0}, {"monopoly", 0}, {"roads", 0}}
	before := p.Hand
	if err := g.act(p, catAction{Type: "dev", Kind: "plenty", Want: catResources{1, 1}}); err != nil {
		t.Fatal(err)
	}
	before.add(catResources{1, 1}, 1)
	if p.Hand != before {
		t.Fatal("plenty did not grant resources")
	}
	catConservation(t, g)
	g.PlayedDev = false
	amount := q.Hand[0]
	before = p.Hand
	for i, other := range g.Players {
		if i > 1 {
			amount += other.Hand[0]
		}
	}
	if err := g.act(p, catAction{Type: "dev", Kind: "monopoly", Index: 0}); err != nil {
		t.Fatal(err)
	}
	if p.Hand[0] != before[0]+amount || q.Hand[0] != 0 {
		t.Fatal("monopoly transfer")
	}
	catConservation(t, g)
	g.PlayedDev = false
	before = p.Hand
	r, _, _ := g.pieces(p.ID)
	if err := g.act(p, catAction{Type: "dev", Kind: "roads"}); err != nil {
		t.Fatal(err)
	}
	for g.Phase == "roads" {
		ei := -1
		for i := range g.Board.Edges {
			if g.Board.roadSpot(i, p.ID) {
				ei = i
				break
			}
		}
		if e := g.act(p, catAction{Type: "build", Kind: "road", Index: ei}); e != nil {
			t.Fatal(e)
		}
	}
	count, _, _ := g.pieces(p.ID)
	if count != r+2 || p.Hand != before || g.Phase != "main" {
		t.Fatal("free roads cost resources or failed to resume phase")
	}
	catConservation(t, g)
}

func TestCatanSeveralHundredTurnsConserveResources(t *testing.T) {
	g := catSetup(t)
	catPlaceAll(t, g)
	apply := func(p *catPlayer, a catAction) {
		t.Helper()
		if err := g.act(p, a); err != nil {
			t.Fatalf("%s/%s: %v", g.Phase, a.Type, err)
		}
		g.awardBonuses()
		g.checkWin()
		catConservation(t, g)
	}
	for turn := 0; turn < 400 && g.Phase != "finished"; turn++ {
		p := g.Players[g.Turn]
		apply(p, catAction{Type: "roll"})
		if g.Phase == "discard" {
			for _, other := range g.Players {
				n := g.Discards[other.ID]
				if n == 0 {
					continue
				}
				give := catResources{}
				for r, count := range other.Hand {
					give[r] = min(n, count)
					n -= give[r]
				}
				apply(other, catAction{Type: "discard", Give: give})
			}
		}
		if g.Phase == "robber" {
			apply(p, catAction{Type: "robber", Index: (g.Board.Robber + 1) % 19})
		}
		if g.Phase == "steal" {
			apply(p, catAction{Type: "steal", Target: g.Board.victims(g.Board.Robber, p.ID, g.Players)[0]})
		}
		for action := 0; action < 3 && g.Phase == "main"; action++ {
			built := false
			_, settlements, cities := g.pieces(p.ID)
			if cities < 4 && p.Hand.has(catCosts["city"]) {
				for i, v := range g.Board.Vertices {
					if v.Owner == p.ID && !v.City {
						apply(p, catAction{Type: "build", Kind: "city", Index: i})
						built = true
						break
					}
				}
			}
			if !built && settlements < 5 && p.Hand.has(catCosts["settlement"]) {
				for i := range g.Board.Vertices {
					if g.Board.settlementSpot(i) && g.Board.touchesRoad(i, p.ID) {
						apply(p, catAction{Type: "build", Kind: "settlement", Index: i})
						built = true
						break
					}
				}
			}
			if !built && p.Hand.has(catCosts["buy"]) && len(g.Deck) > 0 {
				apply(p, catAction{Type: "buy"})
				built = true
			}
			r, _, _ := g.pieces(p.ID)
			if !built && r < 15 && p.Hand.has(catCosts["road"]) {
				for i := range g.Board.Edges {
					if g.Board.roadSpot(i, p.ID) {
						apply(p, catAction{Type: "build", Kind: "road", Index: i})
						built = true
						break
					}
				}
			}
			if !built {
				break
			}
		}
		if g.Phase == "main" {
			apply(p, catAction{Type: "end"})
		}
	}
}

func TestCatanTableEndNotifiesAndPreservesSeats(t *testing.T) {
	a := catNewApp("")
	a.g = catSetup(t)
	catPlaceAll(t, a.g)
	p := a.g.Players[0]
	c := &catSubscriber{token: p.Token, updates: make(chan []byte, 1)}
	a.clients[c] = true
	for _, origin := range []string{"http://other.example", ""} {
		req := httptest.NewRequest("POST", "http://localhost:3000/api/table/end", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)
		if w.Code != 403 || a.g.Phase == "lobby" {
			t.Fatal("foreign request ended game")
		}
	}
	req := httptest.NewRequest("POST", "http://localhost:3000/api/table/end", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	if w.Code != 200 || a.g.Phase != "lobby" || a.g.find(p.Token) != p || len(a.g.Players) != 4 || p.Hand.total() != 0 || len(p.Dev) != 0 {
		t.Fatal("table end lost seats or retained private cards")
	}
	select {
	case <-c.updates:
	default:
		t.Fatal("phones were not notified")
	}
}
