package main

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func offlineTurn(t *testing.T) (*app, *Player, *Player) {
	t.Helper()
	g := setup(t, 3)
	g.Turn = 1
	a := &app{g: g, clients: map[*subscriber]bool{}}
	a.updatePresence(time.Now())
	a.presence[g.Players[1].ID].offlineSince = time.Now().Add(-reconnectGrace - time.Second)
	a.updatePresence(time.Now())
	return a, g.Players[0], g.Players[1]
}
func TestPresenceGraceHeartbeatAndReconnect(t *testing.T) {
	a, _, p := offlineTurn(t)
	now := time.Now()
	s := &subscriber{p: p, updates: make(chan []byte, 1)}
	a.clients[s] = true
	a.touch(p, now)
	a.updatePresence(now)
	if a.presence[p.ID].status != "connected" {
		t.Fatal("No marcó conectado")
	}
	a.updatePresence(now.Add(heartbeatLease + time.Second))
	if a.presence[p.ID].status != "reconnecting" {
		t.Fatal("No detectó pérdida silenciosa de red")
	}
	a.updatePresence(now.Add(heartbeatLease + reconnectGrace + 2*time.Second))
	if a.presence[p.ID].status != "disconnected" {
		t.Fatal("No terminó el margen de reconexión")
	}
	a.touch(p, time.Now())
	a.updatePresence(time.Now())
	if a.presence[p.ID].status != "connected" || !a.presence[p.ID].offlineSince.IsZero() {
		t.Fatal("No recuperó conectado")
	}
}
func TestPresenceCountsAllTabs(t *testing.T) {
	a, _, p := offlineTurn(t)
	one := &subscriber{p: p}
	two := &subscriber{p: p}
	a.clients[one] = true
	a.clients[two] = true
	a.touch(p, time.Now())
	a.updatePresence(time.Now())
	delete(a.clients, one)
	a.updatePresence(time.Now())
	if a.presence[p.ID].status != "connected" {
		t.Fatal("Cerrar una pestaña desconectó a otra")
	}
	delete(a.clients, two)
	a.updatePresence(time.Now())
	if a.presence[p.ID].status != "reconnecting" {
		t.Fatal("Cerrar la última pestaña no inició reconexión")
	}
}
func TestHostCanSkipOfflineTurnWithoutChangingHand(t *testing.T) {
	a, host, target := offlineTurn(t)
	hand := append([]Card{}, target.Hand...)
	a.g.Drawn = target.Hand[0].ID
	action := Action{Type: "skipDisconnected", Token: host.Token, Target: target.ID, ExpectedRevision: a.g.Revision}
	b, _ := json.Marshal(action)
	r := httptest.NewRecorder()
	a.ServeHTTP(r, httptest.NewRequest("POST", "/api/action", strings.NewReader(string(b))))
	if r.Code != 200 || a.g.Turn != 2 || a.g.Drawn != "" || a.g.Revision != 1 || !reflect.DeepEqual(hand, target.Hand) {
		t.Fatal("Salto incorrecto", r.Body.String())
	}
	if a.g.Effect.Type != "skipDisconnected" {
		t.Fatal("No anunció la acción")
	}
	if a.skipDisconnected(host, action) == nil {
		t.Fatal("Repitió una acción vieja")
	}
}
func TestSkipChecksHostGraceConnectedAndStaleTurn(t *testing.T) {
	for _, scenario := range []string{"nonhost", "grace", "reconnected", "target", "revision", "self", "lobby"} {
		t.Run(scenario, func(t *testing.T) {
			a, host, target := offlineTurn(t)
			actor := host
			action := Action{Type: "skipDisconnected", Target: target.ID, ExpectedRevision: a.g.Revision}
			switch scenario {
			case "nonhost":
				actor = a.g.Players[2]
			case "grace":
				a.presence[target.ID].offlineSince = time.Now()
			case "reconnected":
				a.clients[&subscriber{p: target}] = true
				a.touch(target, time.Now())
			case "target":
				action.Target = host.ID
			case "revision":
				action.ExpectedRevision++
			case "self":
				a.g.Turn = 0
				action.Target = host.ID
			case "lobby":
				a.g.Phase = "lobby"
			}
			turn := a.g.Turn
			if a.skipDisconnected(actor, action) == nil || a.g.Turn != turn {
				t.Fatal("Salto inválido aceptado")
			}
		})
	}
}
func TestSkipRespectsReverseDirectionAndPrivatePermission(t *testing.T) {
	a, host, target := offlineTurn(t)
	a.g.Direction = -1
	var hostState, publicState State
	if e := json.Unmarshal(a.snapshot(host), &hostState); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(a.snapshot(nil), &publicState); e != nil {
		t.Fatal(e)
	}
	if !hostState.CanSkipTurn || publicState.CanSkipTurn || hostState.Players[1].Connection != "disconnected" {
		t.Fatal("Permisos de salto incorrectos")
	}
	if e := a.skipDisconnected(host, Action{Target: target.ID, ExpectedRevision: a.g.Revision}); e != nil {
		t.Fatal(e)
	}
	if a.g.Turn != 0 {
		t.Fatal("No respetó el sentido inverso")
	}
}
