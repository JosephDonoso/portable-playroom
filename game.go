package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
)

type Card struct {
	ID    string `json:"id"`
	Color string `json:"color"`
	Value string `json:"value"`
}
type Player struct {
	ID, Token, Name string
	Hand            []Card
}
type Action struct {
	Type  string `json:"type"`
	Card  string `json:"card"`
	Color string `json:"color"`
	Uno   bool   `json:"uno"`
	Token string `json:"token"`
}
type Effect struct {
	Type  string `json:"type"`
	Actor string `json:"actor"`
	Value string `json:"value"`
}
type Game struct {
	Revision               int
	Effect                 *Effect
	Players                []*Player
	Phase                  string
	Round, Turn, Direction int
	Events                 []string
	Top                    *Card
	Color, Drawn, Winner   string
	Pile, Discard          []Card
}

var colors = []string{"red", "yellow", "green", "blue"}

func id() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func shuffle(cards []Card) {
	for i := len(cards) - 1; i > 0; i-- {
		j, e := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if e != nil {
			panic(e)
		}
		cards[i], cards[int(j.Int64())] = cards[int(j.Int64())], cards[i]
	}
}
func deck() []Card {
	cards := []Card{}
	add := func(c, v string) { cards = append(cards, Card{id(), c, v}) }
	for _, c := range colors {
		add(c, "0")
		for _, v := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "skip", "reverse", "draw2"} {
			add(c, v)
			add(c, v)
		}
	}
	for i := 0; i < 4; i++ {
		add("wild", "wild")
		add("wild", "draw4")
	}
	shuffle(cards)
	return cards
}
func newGame() *Game {
	return &Game{Players: []*Player{}, Phase: "lobby", Direction: 1, Events: []string{"La mesa está abierta."}}
}
func (g *Game) note(s string) {
	g.Events = append([]string{s}, g.Events...)
	if len(g.Events) > 8 {
		g.Events = g.Events[:8]
	}
}
func (g *Game) find(token string) *Player {
	if token == "" {
		return nil
	}
	for _, p := range g.Players {
		if p.Token == token {
			return p
		}
	}
	return nil
}
func (g *Game) join(name, token string) (*Player, error) {
	if p := g.find(token); p != nil {
		return p, nil
	}
	if g.Phase != "lobby" {
		return nil, errors.New("La partida ya comenzó. Podrás entrar en la siguiente.")
	}
	if len(g.Players) >= 10 {
		return nil, errors.New("La mesa está llena (10 jugadores).")
	}
	r := []rune(strings.TrimSpace(name))
	if len(r) > 24 {
		r = r[:24]
	}
	if len(r) == 0 {
		return nil, errors.New("Escribe tu nombre.")
	}
	p := &Player{ID: id(), Token: id(), Name: string(r), Hand: []Card{}}
	g.Players = append(g.Players, p)
	g.note(p.Name + " se unió a la mesa.")
	return p, nil
}
func (g *Game) start(p *Player) error {
	if len(g.Players) == 0 || p != g.Players[0] {
		return errors.New("Solo el anfitrión puede iniciar.")
	}
	if g.Phase == "playing" {
		return errors.New("Ya hay una partida en curso.")
	}
	if len(g.Players) < 2 {
		return errors.New("Se necesitan al menos dos jugadores.")
	}
	g.Pile = deck()
	g.Discard = nil
	g.Direction = 1
	g.Turn = 0
	g.Drawn = ""
	g.Winner = ""
	g.Round++
	for _, p := range g.Players {
		p.Hand = append([]Card{}, g.Pile[:7]...)
		g.Pile = g.Pile[7:]
	}
	for i, c := range g.Pile {
		if len(c.Value) == 1 && c.Value[0] >= '0' && c.Value[0] <= '9' {
			g.Top = &c
			g.Pile = append(g.Pile[:i], g.Pile[i+1:]...)
			g.Discard = append(g.Discard, c)
			g.Color = c.Color
			break
		}
	}
	g.Phase = "playing"
	g.note(fmt.Sprintf("¡Comienza la ronda %d! Turno de %s.", g.Round, g.Players[0].Name))
	return nil
}
func (g *Game) take(p *Player, n int) {
	for i := 0; i < n; i++ {
		if len(g.Pile) == 0 && len(g.Discard) > 1 {
			top := g.Discard[len(g.Discard)-1]
			g.Pile = append([]Card{}, g.Discard[:len(g.Discard)-1]...)
			shuffle(g.Pile)
			g.Discard = []Card{top}
		}
		if len(g.Pile) == 0 {
			break
		}
		p.Hand = append(p.Hand, g.Pile[len(g.Pile)-1])
		g.Pile = g.Pile[:len(g.Pile)-1]
	}
}
func (g *Game) next(steps int) {
	g.Turn = (g.Turn + g.Direction*steps + len(g.Players)*steps) % len(g.Players)
	g.Drawn = ""
}
func (g *Game) playable(c Card, p *Player) bool {
	if c.Color == "wild" {
		if c.Value != "draw4" {
			return true
		}
		for _, x := range p.Hand {
			if x.Color == g.Color {
				return false
			}
		}
		return true
	}
	return c.Color == g.Color || g.Top != nil && c.Value == g.Top.Value
}
func label(c Card) string {
	if s, ok := map[string]string{"skip": "salto", "reverse": "reversa", "draw2": "+2", "wild": "comodín", "draw4": "+4"}[c.Value]; ok {
		return s
	}
	return c.Value
}
func (g *Game) act(p *Player, a Action) error {
	if p == nil {
		return errors.New("Vuelve a entrar a la sala.")
	}
	if a.Type == "start" {
		return g.start(p)
	}
	if a.Type == "lobby" {
		if p != g.Players[0] || g.Phase == "playing" {
			return errors.New("No se puede abrir la sala ahora.")
		}
		g.Phase = "lobby"
		for _, x := range g.Players {
			x.Hand = []Card{}
		}
		g.Top = nil
		return nil
	}
	if g.Phase != "playing" {
		return errors.New("No hay una ronda en curso.")
	}
	if g.Players[g.Turn] != p {
		return errors.New("Espera tu turno.")
	}
	if a.Type == "draw" {
		if g.Drawn != "" {
			return errors.New("Ya robaste una carta. Puedes jugarla o pasar.")
		}
		before := len(p.Hand)
		g.take(p, 1)
		g.note(p.Name + " robó una carta.")
		if len(p.Hand) == before || !g.playable(p.Hand[len(p.Hand)-1], p) {
			g.next(1)
		} else {
			g.Drawn = p.Hand[len(p.Hand)-1].ID
		}
		return nil
	}
	if a.Type == "pass" {
		if g.Drawn == "" {
			return errors.New("Debes robar antes de pasar.")
		}
		g.note(p.Name + " pasó.")
		g.next(1)
		return nil
	}
	if a.Type != "play" {
		return errors.New("Acción desconocida.")
	}
	index := -1
	for i, c := range p.Hand {
		if c.ID == a.Card {
			index = i
			break
		}
	}
	if index < 0 {
		return errors.New("Esa carta no está en tu mano.")
	}
	c := p.Hand[index]
	if g.Drawn != "" && c.ID != g.Drawn {
		return errors.New("Solo puedes jugar la carta que acabas de robar.")
	}
	if !g.playable(c, p) {
		return errors.New("La carta debe coincidir en color o símbolo. +4 solo si no tienes el color actual.")
	}
	if c.Color == "wild" && !slices.Contains(colors, a.Color) {
		return errors.New("Elige un color para el comodín.")
	}
	p.Hand = append(p.Hand[:index], p.Hand[index+1:]...)
	g.Top = &c
	g.Discard = append(g.Discard, c)
	g.Color = c.Color
	if c.Color == "wild" {
		g.Color = a.Color
	}
	g.note(p.Name + " jugó " + label(c) + ".")
	if len(p.Hand) == 1 {
		if a.Uno {
			g.note("¡" + p.Name + " dijo UNO!")
		} else {
			g.take(p, 2)
			g.note(p.Name + " olvidó decir UNO y robó 2.")
		}
	}
	steps := 1
	if c.Value == "reverse" {
		g.Direction *= -1
		if len(g.Players) == 2 {
			steps = 2
		}
	}
	if c.Value == "skip" {
		steps = 2
	}
	if c.Value == "draw2" || c.Value == "draw4" {
		victim := g.Players[(g.Turn+g.Direction+len(g.Players))%len(g.Players)]
		n := 2
		if c.Value == "draw4" {
			n = 4
		}
		g.take(victim, n)
		g.note(fmt.Sprintf("%s robó %d y pierde su turno.", victim.Name, n))
		steps = 2
	}
	if len(p.Hand) == 0 {
		g.Phase = "finished"
		g.Winner = p.ID
		g.note("¡" + p.Name + " ganó la ronda!")
		g.Drawn = ""
	} else {
		g.next(steps)
	}
	return nil
}

type PublicPlayer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Host  bool   `json:"host"`
}
type State struct {
	Revision  int            `json:"revision"`
	Effect    *Effect        `json:"effect"`
	Phase     string         `json:"phase"`
	Round     int            `json:"round"`
	Players   []PublicPlayer `json:"players"`
	Top       *Card          `json:"top"`
	Color     any            `json:"color"`
	Turn      any            `json:"turn"`
	Direction int            `json:"direction"`
	Winner    any            `json:"winner"`
	Events    []string       `json:"events"`
	Me        any            `json:"me"`
	Hand      []Card         `json:"hand"`
	Drawn     any            `json:"drawn"`
	Playable  []string       `json:"playable"`
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func (g *Game) state(p *Player) State {
	s := State{Revision: g.Revision, Effect: g.Effect, Phase: g.Phase, Round: g.Round, Players: []PublicPlayer{}, Top: g.Top, Color: nullable(g.Color), Direction: g.Direction, Winner: nullable(g.Winner), Events: append([]string{}, g.Events...), Hand: []Card{}, Playable: []string{}}
	for i, x := range g.Players {
		s.Players = append(s.Players, PublicPlayer{x.ID, x.Name, len(x.Hand), i == 0})
	}
	if len(g.Players) > 0 {
		s.Turn = g.Players[g.Turn].ID
	}
	if p != nil {
		s.Me = p.ID
		s.Hand = append([]Card{}, p.Hand...)
		if len(g.Players) > 0 && g.Players[g.Turn] == p {
			s.Drawn = nullable(g.Drawn)
			if g.Phase == "playing" {
				for _, c := range p.Hand {
					if (g.Drawn == "" || g.Drawn == c.ID) && g.playable(c, p) {
						s.Playable = append(s.Playable, c.ID)
					}
				}
			}
		}
	}
	return s
}
