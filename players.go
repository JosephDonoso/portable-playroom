package main

import (
	"fmt"
	"slices"
	"time"
)

func (g *Game) openLobby() {
	g.Phase, g.Turn, g.Direction = "lobby", 0, 1
	g.Top, g.Pile, g.Discard = nil, nil, nil
	g.Pending, g.PendingKind, g.LastPayment = 0, "", nil
	g.Winner, g.Drawn, g.Color = "", "", ""
	g.FinishCandidates = nil
	g.Deadline = time.Time{}
	for _, p := range g.Players {
		p.Hand = []Card{}
	}
}

func (g *Game) kick(host *Player, targetID string) error {
	if host == nil || len(g.Players) == 0 || host != g.Players[0] {
		return fmt.Errorf("Solo el anfitrión puede expulsar jugadores.")
	}
	i := slices.IndexFunc(g.Players, func(p *Player) bool { return p.ID == targetID })
	if i < 0 {
		return fmt.Errorf("Ese jugador ya no está en la sala.")
	}
	if i == 0 {
		return fmt.Errorf("El anfitrión no puede expulsarse a sí mismo.")
	}
	target := g.Players[i]
	current := g.Players[g.Turn]
	wasTurn := current == target
	if wasTurn {
		current = g.Players[(g.Turn+g.Direction+len(g.Players))%len(g.Players)]
	}
	g.Players = slices.Delete(g.Players, i, i+1)
	g.FinishCandidates = slices.DeleteFunc(g.FinishCandidates, func(id string) bool { return id == targetID })
	g.note(host.Name + " expulsó a " + target.Name + ".")
	if len(g.Players) < 2 || g.Phase == "finished" && g.Winner == targetID {
		g.openLobby()
		g.note("La sala está abierta.")
		if len(g.Players) < 2 {
			g.note("Se necesitan al menos dos jugadores para jugar.")
		}
	} else {
		g.Turn = slices.Index(g.Players, current)
		if g.Phase == "playing" {
			g.Pile = append(g.Pile, target.Hand...)
			shuffle(g.Pile)
			if wasTurn {
				g.Drawn = ""
				g.resetDeadline()
				g.settlePending()
			}
		}
	}
	target.Hand = []Card{}
	return nil
}
