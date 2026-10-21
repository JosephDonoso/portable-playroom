package main

import (
	"fmt"
	"time"
)

type Options struct {
	TurnSeconds       int  `json:"turnSeconds"`
	DrawUntilPlayable bool `json:"drawUntilPlayable"`
	ForcePlay         bool `json:"forcePlay"`
	MixDrawCards      bool `json:"mixDrawCards"`
}
type PenaltyPayment struct {
	Player   string `json:"player"`
	Amount   int    `json:"amount"`
	Received int    `json:"received"`
}

func (g *Game) configure(p *Player, o *Options) error {
	if len(g.Players) == 0 || p != g.Players[0] {
		return fmt.Errorf("Solo el anfitrión puede configurar la partida.")
	}
	if g.Phase == "playing" {
		return fmt.Errorf("Configura las reglas antes de iniciar una ronda.")
	}
	if o == nil || o.TurnSeconds < 0 || o.TurnSeconds > 600 || o.TurnSeconds > 0 && o.TurnSeconds < 5 {
		return fmt.Errorf("El límite debe ser 0 (sin límite) o entre 5 y 600 segundos.")
	}
	g.Options = *o
	g.note("El anfitrión actualizó las opciones de la próxima ronda.")
	return nil
}
func (g *Game) resetDeadline() {
	g.Deadline = time.Time{}
	if g.Phase == "playing" && g.Options.TurnSeconds > 0 {
		g.Deadline = time.Now().Add(time.Duration(g.Options.TurnSeconds) * time.Second)
	}
}
func (g *Game) canStack(c Card) bool {
	if g.Pending <= 0 {
		return false
	}
	if c.Value == g.PendingKind {
		return true
	}
	if !g.Options.MixDrawCards {
		return false
	}
	return c.Value == "draw4" && g.PendingKind == "draw2" || c.Value == "draw2" && g.PendingKind == "draw4" && c.Color == g.Color
}
func (g *Game) hasPlayable(p *Player) bool {
	for _, c := range p.Hand {
		if g.playable(c, p) {
			return true
		}
	}
	return false
}
func (g *Game) win(p *Player) {
	g.Phase = "finished"
	g.Winner = p.ID
	g.Drawn = ""
	g.Deadline = time.Time{}
	g.note("¡" + p.Name + " ganó la ronda!")
}
func (g *Game) finishCandidates() bool {
	for _, id := range g.FinishCandidates {
		for _, p := range g.Players {
			if p.ID == id && len(p.Hand) == 0 {
				g.win(p)
				g.FinishCandidates = nil
				return true
			}
		}
	}
	g.FinishCandidates = nil
	return false
}
func (g *Game) payPending(p *Player) {
	n := g.Pending
	before := len(p.Hand)
	g.take(p, n)
	g.LastPayment = &PenaltyPayment{p.Name, n, len(p.Hand) - before}
	g.note(fmt.Sprintf("%s recibió %d cartas de una deuda de +%d y pierde el turno.", p.Name, len(p.Hand)-before, n))
	g.Pending = 0
	g.PendingKind = ""
	if !g.finishCandidates() {
		g.next(1)
	}
}
func (g *Game) settlePending() {
	if g.Pending > 0 && !g.hasPlayable(g.Players[g.Turn]) {
		g.payPending(g.Players[g.Turn])
	}
}
func (g *Game) drawForTurn(p *Player) {
	before := len(p.Hand)
	for {
		count := len(p.Hand)
		g.take(p, 1)
		if count == len(p.Hand) {
			break
		}
		c := p.Hand[len(p.Hand)-1]
		if g.playable(c, p) {
			g.Drawn = c.ID
			break
		}
		if !g.Options.DrawUntilPlayable {
			break
		}
	}
	g.note(fmt.Sprintf("%s robó %d carta(s).", p.Name, len(p.Hand)-before))
}
func (g *Game) expireTurn(now time.Time) bool {
	if g.Phase != "playing" || g.Deadline.IsZero() || now.Before(g.Deadline) {
		return false
	}
	p := g.Players[g.Turn]
	g.LastPayment = nil
	g.note("Se agotó el tiempo de " + p.Name + ".")
	if g.Pending > 0 {
		g.payPending(p)
	} else {
		if g.Drawn == "" {
			g.drawForTurn(p)
		}
		g.next(1)
	}
	return true
}

// Called under app.mu. A single server timer keeps running without any browsers.
func (a *app) scheduleClock() {
	if a.timer != nil {
		a.timer.Stop()
	}
	if a.g.Phase != "playing" || a.g.Deadline.IsZero() {
		return
	}
	a.timer = time.AfterFunc(time.Until(a.g.Deadline), func() { a.mu.Lock(); defer a.mu.Unlock(); a.expireIfNeeded(time.Now()) })
}
func (a *app) expireIfNeeded(now time.Time) bool {
	if a.g.Phase != "playing" {
		return false
	}
	actor := a.g.Players[a.g.Turn].Name
	if !a.g.expireTurn(now) {
		return false
	}
	a.g.Revision++
	a.g.Effect = &Effect{Type: "timeout", Actor: actor}
	a.broadcast()
	a.scheduleClock()
	return true
}
