package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
)

func catRandom(n int) int {
	v, e := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if e != nil {
		panic(e)
	}
	return int(v.Int64())
}
func (r catResources) total() int {
	n := 0
	for _, v := range r {
		n += v
	}
	return n
}
func (r catResources) valid() bool {
	for _, n := range r {
		if n < 0 || n > 95 {
			return false
		}
	}
	return true
}
func (r catResources) has(cost catResources) bool {
	for i, n := range cost {
		if n > r[i] {
			return false
		}
	}
	return true
}
func (r *catResources) add(v catResources, m int) {
	for i, n := range v {
		r[i] += n * m
	}
}

var catCosts = map[string]catResources{"road": {1, 1, 0, 0, 0}, "settlement": {1, 1, 1, 1, 0}, "city": {0, 0, 0, 2, 3}, "buy": {0, 0, 1, 1, 1}}

type catDev struct {
	Kind   string `json:"kind"`
	Bought int    `json:"bought"`
}
type catPlayer struct {
	ID, Token, Name, Color string
	Hand                   catResources
	Dev                    []catDev
	Knights                int
}
type catOffer struct {
	ID        string            `json:"id"`
	From      string            `json:"from"`
	To        string            `json:"to"`
	Give      catResources      `json:"give"`
	Want      catResources      `json:"want"`
	Parent    string            `json:"parent"`
	Responses map[string]string `json:"responses"`
}
type catAction struct {
	Type     string       `json:"type"`
	Token    string       `json:"token"`
	Index    int          `json:"index"`
	Target   string       `json:"target"`
	Kind     string       `json:"kind"`
	Give     catResources `json:"give"`
	Want     catResources `json:"want"`
	Offer    string       `json:"offer"`
	Response string       `json:"response"`
}
type catGame struct {
	Players                                                   []*catPlayer
	Board                                                     catBoard
	Phase                                                     string
	Turn, First, Serial, SetupStep, LastSettlement, FreeRoads int
	Dice                                                      [2]int
	Bank                                                      catResources
	Deck                                                      []string
	PlayedDev                                                 bool
	ReturnPhase                                               string
	Discards                                                  map[string]int
	Offers                                                    []catOffer
	Longest, Largest, Winner                                  string
	Events                                                    []string
	Revision                                                  int
}

func catNewGame() *catGame {
	return &catGame{Phase: "lobby", Board: catNewBoard(), Players: []*catPlayer{}, Events: []string{"La sala está abierta."}, Offers: []catOffer{}, Discards: map[string]int{}}
}
func (g *catGame) note(s string) {
	g.Events = append([]string{s}, g.Events...)
	if len(g.Events) > 8 {
		g.Events = g.Events[:8]
	}
}
func (g *catGame) find(token string) *catPlayer {
	for _, p := range g.Players {
		if token != "" && p.Token == token {
			return p
		}
	}
	return nil
}
func (g *catGame) byID(id string) *catPlayer {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}
func (g *catGame) join(name, token string) (*catPlayer, error) {
	if p := g.find(token); p != nil {
		return p, nil
	}
	if g.Phase != "lobby" {
		return nil, fmt.Errorf("La partida ya comenzó.")
	}
	if len(g.Players) >= 4 {
		return nil, fmt.Errorf("La mesa tiene cuatro jugadores.")
	}
	name = catName(name)
	if name == "" {
		return nil, fmt.Errorf("Escribe tu nombre.")
	}
	used := map[string]bool{}
	for _, p := range g.Players {
		used[p.Color] = true
	}
	color := ""
	for _, c := range []string{"#eb705e", "#60a7ed", "#f0bf60", "#e5e4dc"} {
		if !used[c] {
			color = c
			break
		}
	}
	p := &catPlayer{ID: id(), Token: id(), Name: name, Color: color, Dev: []catDev{}}
	g.Players = append(g.Players, p)
	g.note(name + " se unió a Catán.")
	return p, nil
}
func (g *catGame) start(p *catPlayer) error {
	if len(g.Players) != 4 {
		return fmt.Errorf("Se necesitan exactamente cuatro jugadores.")
	}
	if p != g.Players[0] || g.Phase != "lobby" && g.Phase != "finished" {
		return fmt.Errorf("Solo el anfitrión puede iniciar desde la sala.")
	}
	players, revision := g.Players, g.Revision
	*g = *catNewGame()
	g.Players, g.Revision = players, revision
	for _, p := range players {
		p.Hand = catResources{}
		p.Dev = []catDev{}
		p.Knights = 0
	}
	g.Bank = catResources{19, 19, 19, 19, 19}
	g.Deck = []string{}
	for i := 0; i < 14; i++ {
		g.Deck = append(g.Deck, "knight")
	}
	for i := 0; i < 5; i++ {
		g.Deck = append(g.Deck, "vp")
	}
	for _, kind := range []string{"roads", "plenty", "monopoly"} {
		g.Deck = append(g.Deck, kind, kind)
	}
	catShuffle(g.Deck)
	g.Phase = "setup-settlement"
	g.Serial = 1
	pool := []int{0, 1, 2, 3}
	for {
		highest := 0
		tied := []int{}
		for _, i := range pool {
			roll := catRandom(6) + catRandom(6) + 2
			g.note(fmt.Sprintf("%s obtuvo %d en la tirada de orden inicial.", g.Players[i].Name, roll))
			if roll > highest {
				highest = roll
				tied = []int{i}
			} else if roll == highest {
				tied = append(tied, i)
			}
		}
		if len(tied) == 1 {
			g.First = tied[0]
			g.Turn = g.First
			break
		}
		pool = tied
	}
	g.note(g.Players[g.First].Name + " comienza la colocación inicial.")
	g.note("Colocación inicial: cada jugador construye dos poblados y dos caminos.")
	return nil
}
func (g *catGame) pieces(owner string) (roads, settlements, cities int) {
	for _, e := range g.Board.Edges {
		if e.Owner == owner {
			roads++
		}
	}
	for _, v := range g.Board.Vertices {
		if v.Owner == owner {
			if v.City {
				cities++
			} else {
				settlements++
			}
		}
	}
	return
}
func (g *catGame) score(p *catPlayer, hidden bool) int {
	_, s, c := g.pieces(p.ID)
	n := s + c*2
	if g.Longest == p.ID {
		n += 2
	}
	if g.Largest == p.ID {
		n += 2
	}
	if hidden {
		for _, d := range p.Dev {
			if d.Kind == "vp" {
				n++
			}
		}
	}
	return n
}
func (g *catGame) awardBonuses() {
	longest, army := 0, 0
	roads := map[string]int{}
	armies := map[string]int{}
	for _, p := range g.Players {
		roads[p.ID] = g.Board.longest(p.ID)
		longest = max(longest, roads[p.ID])
		armies[p.ID] = p.Knights
		army = max(army, p.Knights)
	}
	choose := func(old string, values map[string]int, highest, threshold int) string {
		if highest < threshold {
			return ""
		}
		if values[old] == highest && old != "" {
			return old
		}
		winner := ""
		for _, p := range g.Players {
			if values[p.ID] == highest {
				if winner != "" {
					return ""
				}
				winner = p.ID
			}
		}
		return winner
	}
	g.Longest = choose(g.Longest, roads, longest, 5)
	g.Largest = choose(g.Largest, armies, army, 3)
}
func (g *catGame) checkWin() {
	if g.Phase == "lobby" || g.Phase == "finished" || len(g.Players) != 4 || g.SetupStep < 8 {
		return
	}
	p := g.Players[g.Turn]
	if g.score(p, true) >= 10 {
		g.Winner = p.ID
		g.Phase = "finished"
		g.Offers = []catOffer{}
		g.note(p.Name + " ganó con diez o más puntos.")
	}
}
func (g *catGame) pay(p *catPlayer, cost catResources) error {
	if !p.Hand.has(cost) {
		return fmt.Errorf("No tienes los recursos necesarios.")
	}
	p.Hand.add(cost, -1)
	g.Bank.add(cost, 1)
	return nil
}
func (g *catGame) grant(p *catPlayer, r catResources) { p.Hand.add(r, 1); g.Bank.add(r, -1) }
func (g *catGame) produce(total int) {
	needs := make([]catResources, len(g.Players))
	for hi, h := range g.Board.Hexes {
		if h.Number != total || hi == g.Board.Robber || h.Resource < 0 {
			continue
		}
		for _, vi := range h.Vertices {
			v := g.Board.Vertices[vi]
			for pi, p := range g.Players {
				if v.Owner == p.ID {
					n := 1
					if v.City {
						n = 2
					}
					needs[pi][h.Resource] += n
				}
			}
		}
	}
	for r := 0; r < 5; r++ {
		n, recipients, only := 0, 0, 0
		for i, need := range needs {
			n += need[r]
			if need[r] > 0 {
				recipients++
				only = i
			}
		}
		if n <= g.Bank[r] {
			for i, p := range g.Players {
				v := catResources{}
				v[r] = needs[i][r]
				g.grant(p, v)
			}
		} else if recipients == 1 {
			v := catResources{}
			v[r] = g.Bank[r]
			g.grant(g.Players[only], v)
		}
	}
}
func (g *catGame) build(p *catPlayer, kind string, index int) error {
	initial := g.Phase == "setup-settlement" || g.Phase == "setup-road"
	free := g.Phase == "roads"
	r, s, c := g.pieces(p.ID)
	if kind == "road" {
		if initial && g.Phase != "setup-road" {
			return fmt.Errorf("Coloca primero un poblado.")
		}
		if r >= 15 || !g.Board.roadSpot(index, p.ID) {
			return fmt.Errorf("Elige un camino libre conectado a tu red.")
		}
		if initial {
			e := g.Board.Edges[index]
			if e.A != g.LastSettlement && e.B != g.LastSettlement {
				return fmt.Errorf("Conecta el camino al poblado que acabas de colocar.")
			}
		}
		if !initial && !free {
			if err := g.pay(p, catCosts[kind]); err != nil {
				return err
			}
		}
		g.Board.Edges[index].Owner = p.ID
	} else if kind == "settlement" {
		if initial && g.Phase != "setup-settlement" || free {
			return fmt.Errorf("Ahora debes colocar un camino.")
		}
		if s >= 5 || !g.Board.settlementSpot(index) || !initial && !g.Board.touchesRoad(index, p.ID) {
			return fmt.Errorf("Elige una intersección libre, separada de otros poblados y conectada a tu red.")
		}
		if !initial {
			if err := g.pay(p, catCosts[kind]); err != nil {
				return err
			}
		}
		g.Board.Vertices[index].Owner = p.ID
		if initial {
			g.LastSettlement = index
			g.Phase = "setup-road"
			if g.SetupStep >= 4 {
				for _, h := range g.Board.Hexes {
					if h.Resource >= 0 && slices.Contains(h.Vertices, index) {
						v := catResources{}
						v[h.Resource] = 1
						g.grant(p, v)
					}
				}
			}
		}
	} else if kind == "city" {
		if initial || free || index < 0 || index >= len(g.Board.Vertices) || c >= 4 {
			return fmt.Errorf("No puedes construir una ciudad aquí.")
		}
		v := &g.Board.Vertices[index]
		if v.Owner != p.ID || v.City {
			return fmt.Errorf("Mejora uno de tus poblados.")
		}
		if err := g.pay(p, catCosts[kind]); err != nil {
			return err
		}
		v.City = true
	} else {
		return fmt.Errorf("Construcción desconocida.")
	}
	g.note(p.Name + " construyó " + map[string]string{"road": "un camino", "settlement": "un poblado", "city": "una ciudad"}[kind] + ".")
	if initial && kind == "road" {
		g.SetupStep++
		if g.SetupStep == 8 {
			g.Turn = g.First
			g.Phase = "roll"
			g.note("Empieza la partida. Lanza los dados.")
		} else {
			g.Turn = (g.First + []int{0, 1, 2, 3, 3, 2, 1, 0}[g.SetupStep]) % 4
			g.Phase = "setup-settlement"
		}
	}
	if free {
		g.FreeRoads--
		g.finishRoads(p)
	}
	g.awardBonuses()
	return nil
}
func (g *catGame) finishRoads(p *catPlayer) {
	r, _, _ := g.pieces(p.ID)
	available := false
	for i := range g.Board.Edges {
		if g.Board.roadSpot(i, p.ID) {
			available = true
			break
		}
	}
	if g.FreeRoads <= 0 || r >= 15 || !available {
		g.FreeRoads = 0
		g.Phase = g.ReturnPhase
	}
}
func (g *catGame) rate(p *catPlayer, resource int) int {
	rate := 4
	for _, v := range g.Board.Vertices {
		if v.Owner == p.ID {
			if v.Port == -1 {
				rate = min(rate, 3)
			} else if v.Port == resource {
				rate = 2
			}
		}
	}
	return rate
}
func (g *catGame) moveRobber(p *catPlayer, tile int) error {
	if tile < 0 || tile >= len(g.Board.Hexes) || tile == g.Board.Robber {
		return fmt.Errorf("Mueve el ladrón a otro hexágono.")
	}
	g.Board.Robber = tile
	victims := g.Board.victims(tile, p.ID, g.Players)
	if len(victims) == 0 {
		g.Phase = g.ReturnPhase
	} else if len(victims) == 1 {
		g.steal(p, victims[0])
	} else {
		g.Phase = "steal"
	}
	g.note(p.Name + " movió el ladrón.")
	return nil
}
func (g *catGame) steal(p *catPlayer, target string) error {
	if !slices.Contains(g.Board.victims(g.Board.Robber, p.ID, g.Players), target) {
		return fmt.Errorf("Elige un rival junto al ladrón que tenga recursos.")
	}
	victim := g.byID(target)
	n := catRandom(victim.Hand.total())
	for r, count := range victim.Hand {
		if n < count {
			victim.Hand[r]--
			p.Hand[r]++
			break
		}
		n -= count
	}
	g.Phase = g.ReturnPhase
	g.note(p.Name + " robó un recurso a " + victim.Name + ".")
	return nil
}
func (g *catGame) develop(p *catPlayer, a catAction) error {
	if g.PlayedDev {
		return fmt.Errorf("Solo puedes jugar una carta de desarrollo por turno.")
	}
	i := slices.IndexFunc(p.Dev, func(d catDev) bool { return d.Kind == a.Kind && d.Bought < g.Serial })
	if i < 0 || a.Kind == "vp" {
		return fmt.Errorf("No tienes esa carta disponible. Las recién compradas esperan al siguiente turno.")
	}
	switch a.Kind {
	case "knight":
		g.ReturnPhase = g.Phase
		g.Phase = "robber"
		p.Knights++
	case "roads":
		r, _, _ := g.pieces(p.ID)
		if r >= 15 {
			return fmt.Errorf("Ya colocaste tus quince caminos.")
		}
		available := false
		for i := range g.Board.Edges {
			if g.Board.roadSpot(i, p.ID) {
				available = true
				break
			}
		}
		if !available {
			return fmt.Errorf("No hay un lugar disponible para construir caminos.")
		}
		g.ReturnPhase = g.Phase
		g.Phase = "roads"
		g.FreeRoads = 2
		g.finishRoads(p)
	case "plenty":
		if !a.Want.valid() || a.Want.total() == 0 || a.Want.total() != min(2, g.Bank.total()) || !g.Bank.has(a.Want) {
			return fmt.Errorf("Selecciona dos recursos disponibles en el banco.")
		}
		g.grant(p, a.Want)
	case "monopoly":
		if a.Index < 0 || a.Index > 4 {
			return fmt.Errorf("Elige un recurso.")
		}
		for _, other := range g.Players {
			if other != p {
				p.Hand[a.Index] += other.Hand[a.Index]
				other.Hand[a.Index] = 0
			}
		}
	default:
		return fmt.Errorf("Carta desconocida.")
	}
	p.Dev = slices.Delete(p.Dev, i, i+1)
	g.PlayedDev = true
	g.awardBonuses()
	g.note(p.Name + " jugó " + map[string]string{"knight": "un caballero", "roads": "construcción de caminos", "plenty": "año de abundancia", "monopoly": "monopolio"}[a.Kind] + ".")
	return nil
}
func (g *catGame) act(p *catPlayer, a catAction) error {
	if p == nil {
		return fmt.Errorf("Entra a la sala para jugar.")
	}
	if a.Type == "start" {
		return g.start(p)
	}
	if a.Type == "lobby" {
		if p != g.Players[0] || g.Phase != "finished" {
			return fmt.Errorf("Solo el anfitrión puede abrir la sala al terminar.")
		}
		g.Phase = "lobby"
		return nil
	}
	if g.Phase == "lobby" || g.Phase == "finished" {
		return fmt.Errorf("No hay una partida en curso.")
	}
	if a.Type == "discard" {
		n := g.Discards[p.ID]
		if g.Phase != "discard" || n == 0 || !a.Give.valid() || a.Give.total() != n || !p.Hand.has(a.Give) {
			return fmt.Errorf("Selecciona exactamente la mitad de tus recursos, redondeando hacia abajo.")
		}
		g.pay(p, a.Give)
		delete(g.Discards, p.ID)
		if len(g.Discards) == 0 {
			g.Phase = "robber"
		}
		g.note(p.Name + " descartó recursos por el siete.")
		return nil
	}
	if g.Phase == "main" && (a.Type == "offer" || a.Type == "respond" || a.Type == "confirm" || a.Type == "cancelOffer") {
		return g.trade(p, a)
	}
	if g.Players[g.Turn] != p {
		return fmt.Errorf("Espera tu turno.")
	}
	if a.Type == "build" && (g.Phase == "main" || g.Phase == "roads" || g.Phase == "setup-settlement" || g.Phase == "setup-road") {
		return g.build(p, a.Kind, a.Index)
	}
	if a.Type == "dev" && (g.Phase == "roll" || g.Phase == "main") {
		return g.develop(p, a)
	}
	if a.Type == "robber" && g.Phase == "robber" {
		return g.moveRobber(p, a.Index)
	}
	if a.Type == "steal" && g.Phase == "steal" {
		return g.steal(p, a.Target)
	}
	if a.Type == "roll" && g.Phase == "roll" {
		g.Dice = [2]int{catRandom(6) + 1, catRandom(6) + 1}
		return g.resolveRoll()
	}
	if g.Phase != "main" {
		return fmt.Errorf("Completa primero la fase actual.")
	}
	switch a.Type {
	case "end":
		g.Turn = (g.Turn + 1) % 4
		g.Serial++
		g.PlayedDev = false
		g.Dice = [2]int{}
		g.Phase = "roll"
		g.Offers = []catOffer{}
		g.note("Turno de " + g.Players[g.Turn].Name + ".")
	case "bank":
		if !a.Give.valid() || !a.Want.valid() || a.Want.total() != 1 || !g.Bank.has(a.Want) {
			return fmt.Errorf("Elige un recurso disponible para recibir.")
		}
		r := -1
		for i, n := range a.Give {
			if n > 0 {
				if r >= 0 {
					return fmt.Errorf("Entrega recursos de un solo tipo.")
				}
				r = i
			}
		}
		if r < 0 || a.Want[r] > 0 || a.Give[r] != g.rate(p, r) {
			return fmt.Errorf("El intercambio no cumple la proporción de tu puerto o del banco.")
		}
		if err := g.pay(p, a.Give); err != nil {
			return err
		}
		g.grant(p, a.Want)
		g.note(p.Name + " comerció con el banco.")
	case "buy":
		if len(g.Deck) == 0 {
			return fmt.Errorf("No quedan cartas de desarrollo.")
		}
		if err := g.pay(p, catCosts["buy"]); err != nil {
			return err
		}
		p.Dev = append(p.Dev, catDev{g.Deck[0], g.Serial})
		g.Deck = g.Deck[1:]
		g.note(p.Name + " compró una carta de desarrollo.")
	default:
		return fmt.Errorf("Acción no disponible.")
	}
	return nil
}
func (g *catGame) resolveRoll() error {
	total := g.Dice[0] + g.Dice[1]
	g.note(fmt.Sprintf("%s sacó %d + %d = %d.", g.Players[g.Turn].Name, g.Dice[0], g.Dice[1], total))
	g.Phase = "main"
	if total != 7 {
		g.produce(total)
		return nil
	}
	g.ReturnPhase = "main"
	g.Discards = map[string]int{}
	for _, p := range g.Players {
		if n := p.Hand.total(); n > 7 {
			g.Discards[p.ID] = n / 2
		}
	}
	if len(g.Discards) > 0 {
		g.Phase = "discard"
	} else {
		g.Phase = "robber"
	}
	return nil
}
