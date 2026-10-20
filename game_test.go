package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func setup(t *testing.T, n int) *Game {
	t.Helper()
	g := newGame()
	for i := 0; i < n; i++ {
		if _, e := g.join(fmt.Sprint("Jugador ", i), ""); e != nil {
			t.Fatal(e)
		}
	}
	if e := g.start(g.Players[0]); e != nil {
		t.Fatal(e)
	}
	g.Top = &Card{"top", "red", "5"}
	g.Discard = []Card{*g.Top}
	g.Color = "red"
	for i, p := range g.Players {
		p.Hand = []Card{{fmt.Sprint("a", i), "red", "8"}, {fmt.Sprint("b", i), "blue", "3"}, {fmt.Sprint("c", i), "green", "6"}}
	}
	return g
}
func mustAct(t *testing.T, g *Game, p *Player, a Action) {
	t.Helper()
	if e := g.act(p, a); e != nil {
		t.Fatal(e)
	}
}
func TestDeck(t *testing.T) {
	d := deck()
	ids := map[string]bool{}
	for _, c := range d {
		ids[c.ID] = true
	}
	if len(d) != 108 || len(ids) != 108 {
		t.Fatal("Baraja incorrecta")
	}
}
func TestPrivateViews(t *testing.T) {
	g := setup(t, 3)
	p := g.Players[0]
	for _, viewer := range []*Player{nil, g.Players[1]} {
		b, _ := json.Marshal(g.state(viewer))
		if bytes.Contains(b, []byte(p.Token)) || bytes.Contains(b, []byte(`"id":"`+p.Hand[0].ID+`"`)) {
			t.Fatal("Información privada expuesta")
		}
	}
	if len(g.state(p).Hand) != 3 {
		t.Fatal("Mano incorrecta")
	}
}
func TestPermissions(t *testing.T) {
	g := setup(t, 3)
	if g.start(g.Players[1]) == nil || g.act(g.Players[1], Action{Type: "play", Card: "a1"}) == nil {
		t.Fatal("Falta validar anfitrión o turno")
	}
	for _, c := range []string{"a1", "b0"} {
		if g.act(g.Players[0], Action{Type: "play", Card: c}) == nil {
			t.Fatal("Carta inválida aceptada")
		}
	}
}
func TestDrawTwo(t *testing.T) {
	g := setup(t, 3)
	g.Players[0].Hand[0] = Card{"x", "red", "draw2"}
	mustAct(t, g, g.Players[0], Action{Type: "play", Card: "x"})
	if len(g.Players[1].Hand) != 5 || g.Turn != 2 {
		t.Fatal("+2 incorrecto")
	}
}
func TestReverseAndSkip(t *testing.T) {
	for _, v := range []string{"reverse", "skip"} {
		g := setup(t, 2)
		g.Players[0].Hand[0] = Card{"x", "red", v}
		mustAct(t, g, g.Players[0], Action{Type: "play", Card: "x"})
		if g.Turn != 0 {
			t.Fatal("Debe repetir turno en dos jugadores")
		}
	}
	g := setup(t, 3)
	g.Players[0].Hand[0] = Card{"x", "red", "reverse"}
	mustAct(t, g, g.Players[0], Action{Type: "play", Card: "x"})
	if g.Direction != -1 || g.Turn != 2 {
		t.Fatal("Reversa incorrecta")
	}
}
func TestWildFour(t *testing.T) {
	g := setup(t, 3)
	p := g.Players[0]
	p.Hand = append(p.Hand, Card{"w", "wild", "draw4"})
	if g.act(p, Action{Type: "play", Card: "w", Color: "green"}) == nil {
		t.Fatal("+4 ilegal aceptado")
	}
	p.Hand = p.Hand[1:]
	if g.act(p, Action{Type: "play", Card: "w", Color: "purple"}) == nil {
		t.Fatal("Color inválido")
	}
	mustAct(t, g, p, Action{Type: "play", Card: "w", Color: "green", Uno: true})
	if g.Color != "green" || len(g.Players[1].Hand) != 7 || g.Turn != 2 {
		t.Fatal("+4 incorrecto")
	}
}
func TestUnoPenalty(t *testing.T) {
	for _, uno := range []bool{false, true} {
		g := setup(t, 3)
		p := g.Players[0]
		p.Hand = p.Hand[:2]
		mustAct(t, g, p, Action{Type: "play", Card: "a0", Uno: uno})
		want := 3
		if uno {
			want = 1
		}
		if len(p.Hand) != want {
			t.Fatal("Penalización UNO incorrecta")
		}
	}
}
func TestDrawAndPass(t *testing.T) {
	g := setup(t, 3)
	p := g.Players[0]
	g.Pile = append(g.Pile, Card{"new", "red", "9"})
	mustAct(t, g, p, Action{Type: "draw"})
	if g.Drawn != "new" {
		t.Fatal("Falta carta robada")
	}
	if g.act(p, Action{Type: "draw"}) == nil || g.act(p, Action{Type: "play", Card: "a0"}) == nil {
		t.Fatal("Acción ilegal tras robar")
	}
	mustAct(t, g, p, Action{Type: "pass"})
	if g.Turn != 1 {
		t.Fatal("Turno incorrecto")
	}
}
func TestRecycle(t *testing.T) {
	g := setup(t, 3)
	g.Pile = nil
	g.Discard = []Card{{"old", "blue", "1"}, *g.Top}
	g.take(g.Players[0], 1)
	if len(g.Discard) != 1 || g.Discard[0].ID != "top" || g.Players[0].Hand[3].ID != "old" {
		t.Fatal("Reciclado incorrecto")
	}
}
func TestWinReconnectNewRound(t *testing.T) {
	g := setup(t, 3)
	p := g.Players[0]
	p.Hand = []Card{{"last", "red", "2"}}
	mustAct(t, g, p, Action{Type: "play", Card: "last"})
	if g.Phase != "finished" || g.Winner != p.ID {
		t.Fatal("Victoria incorrecta")
	}
	rejoined, e := g.join("Otro nombre", p.Token)
	if e != nil || rejoined != p {
		t.Fatal("Reconexión incorrecta")
	}
	if e = g.start(p); e != nil {
		t.Fatal(e)
	}
	if g.Round != 2 || g.Phase != "playing" {
		t.Fatal("Nueva ronda incorrecta")
	}
	for _, p := range g.Players {
		if len(p.Hand) != 7 {
			t.Fatal("Reparto incorrecto")
		}
	}
}
