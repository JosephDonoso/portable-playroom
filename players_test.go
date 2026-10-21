package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKickRequiresHostAndCannotRemoveHost(t *testing.T) {
	g := setup(t, 3)
	for _, actor := range []*Player{nil, g.Players[1]} {
		if g.kick(actor, g.Players[2].ID) == nil {
			t.Fatal("non-host expelled player")
		}
	}
	if g.kick(g.Players[0], g.Players[0].ID) == nil {
		t.Fatal("host expelled self")
	}
	if g.kick(g.Players[0], "missing") == nil || len(g.Players) != 3 {
		t.Fatal("invalid target changed room")
	}
}

func TestKickKeepsTurnAndDeadlineWhenAnotherPlayerLeaves(t *testing.T) {
	g := setup(t, 4)
	g.Turn = 3
	current, target := g.Players[3], g.Players[1]
	deadline := time.Now().Add(time.Minute)
	g.Deadline, g.Drawn = deadline, current.Hand[0].ID
	count := len(g.Pile) + len(target.Hand)
	if err := g.kick(g.Players[0], target.ID); err != nil {
		t.Fatal(err)
	}
	if g.Players[g.Turn] != current || g.Deadline != deadline || g.Drawn != current.Hand[0].ID {
		t.Fatal("unrelated turn changed")
	}
	if len(g.Pile) != count || len(target.Hand) != 0 || g.find(target.Token) != nil {
		t.Fatal("player or cards not removed correctly")
	}
}

func TestKickCurrentTurnFollowsDirectionAndPassesDebt(t *testing.T) {
	for _, direction := range []int{1, -1} {
		g := setup(t, 4)
		g.Turn, g.Direction = 2, direction
		g.Options.TurnSeconds = 30
		target, next := g.Players[2], g.Players[2+direction]
		g.Pending, g.PendingKind, g.Drawn = 6, "draw2", "old-drawn"
		next.Hand = []Card{{"counter", "blue", "draw2"}}
		if err := g.kick(g.Players[0], target.ID); err != nil {
			t.Fatal(err)
		}
		if g.Players[g.Turn] != next || g.Pending != 6 || g.Drawn != "" || g.Deadline.IsZero() {
			t.Fatal("wrong successor or debt after kick")
		}
	}
}

func TestKickLastOpponentOpensLobby(t *testing.T) {
	g := setup(t, 2)
	g.Pending, g.PendingKind = 4, "draw4"
	g.Deadline = time.Now().Add(time.Minute)
	if err := g.kick(g.Players[0], g.Players[1].ID); err != nil {
		t.Fatal(err)
	}
	if g.Phase != "lobby" || g.Pending != 0 || !g.Deadline.IsZero() || g.Top != nil || len(g.Players[0].Hand) != 0 {
		t.Fatal("one-player game continued")
	}
	if _, err := g.join("Replacement", ""); err != nil {
		t.Fatal(err)
	}
	if err := g.start(g.Players[0]); err != nil {
		t.Fatal(err)
	}
}

func TestKickInvalidatesTokenAndExistingPrivateStream(t *testing.T) {
	g := setup(t, 3)
	host, target := g.Players[0], g.Players[1]
	a := &app{g: g, clients: map[*subscriber]bool{}}
	s := &subscriber{p: target, updates: make(chan []byte, 1)}
	a.clients[s] = true
	r := httptest.NewRequest("POST", "/api/action", strings.NewReader(`{"type":"kick","token":"`+host.Token+`","target":"`+target.ID+`"}`))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var state State
	if err := json.Unmarshal(<-s.updates, &state); err != nil {
		t.Fatal(err)
	}
	if !state.Expelled || state.Me != nil || len(state.Hand) != 0 || len(state.Playable) != 0 {
		t.Fatal("removed stream retained private identity")
	}
	r = httptest.NewRequest("POST", "/api/action", strings.NewReader(`{"type":"draw","token":"`+target.Token+`"}`))
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	if w.Code == 200 {
		t.Fatal("removed token could act")
	}
}

func TestKickInLobbyAndAfterWinner(t *testing.T) {
	for _, phase := range []string{"lobby", "finished"} {
		g := setup(t, 3)
		g.Phase, g.Winner = phase, g.Players[1].ID
		if err := g.kick(g.Players[0], g.Players[1].ID); err != nil {
			t.Fatal(err)
		}
		if g.Phase != "lobby" {
			t.Fatal("removed winner left broken result screen")
		}
	}
}
