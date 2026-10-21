package main

import (
	"fmt"
	"time"
)

const heartbeatLease = 9 * time.Second
const reconnectGrace = 8 * time.Second

type presenceEntry struct {
	lastSeen     time.Time
	offlineSince time.Time
	status       string
}

// All presence methods run under app.mu. Connection counts include all tabs.
func (a *app) entry(p *Player) *presenceEntry {
	if a.presence == nil {
		a.presence = map[string]*presenceEntry{}
	}
	if a.presence[p.ID] == nil {
		a.presence[p.ID] = &presenceEntry{}
	}
	return a.presence[p.ID]
}
func (a *app) touch(p *Player, now time.Time) { a.entry(p).lastSeen = now }
func (a *app) updatePresence(now time.Time) bool {
	changed := false
	for _, p := range a.g.Players {
		entry := a.entry(p)
		hasStream := false
		for s := range a.clients {
			if s.p == p {
				hasStream = true
				break
			}
		}
		connected := hasStream && !entry.lastSeen.IsZero() && now.Sub(entry.lastSeen) <= heartbeatLease
		status := "connected"
		if connected {
			entry.offlineSince = time.Time{}
		} else {
			if entry.offlineSince.IsZero() {
				entry.offlineSince = now
			}
			status = "reconnecting"
			if now.Sub(entry.offlineSince) >= reconnectGrace {
				status = "disconnected"
			}
		}
		if entry.status != status {
			changed = true
			entry.status = status
		}
	}
	return changed
}
func (a *app) skipDisconnected(host *Player, action Action) error {
	if host == nil || len(a.g.Players) == 0 || host != a.g.Players[0] {
		return fmt.Errorf("Solo el anfitrión puede saltar un turno desconectado.")
	}
	if a.g.Phase != "playing" {
		return fmt.Errorf("No hay una ronda en curso.")
	}
	target := a.g.Players[a.g.Turn]
	if action.Target != target.ID || action.ExpectedRevision != a.g.Revision {
		return fmt.Errorf("El turno cambió. Revisa la partida antes de volver a intentar.")
	}
	a.updatePresence(time.Now())
	if target == host || a.presence[target.ID].status != "disconnected" {
		return fmt.Errorf("El jugador está conectado o aún tiene margen para reconectarse.")
	}
	a.g.note(fmt.Sprintf("%s saltó el turno de %s (desconectado).", host.Name, target.Name))
	if a.g.Pending > 0 {
		a.g.payPending(target)
	} else {
		a.g.next(1)
	}
	return nil
}
