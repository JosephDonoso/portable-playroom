package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStackingAndAutomaticPayment(t *testing.T) {
	for _, kind := range []string{"draw2", "draw4"} {
		t.Run(kind, func(t *testing.T) {
			g := setup(t, 3)
			color := "red"
			n := 2
			if kind == "draw4" {
				color = "wild"
				n = 4
			}
			g.Players[0].Hand[0] = Card{"first", color, kind}
			g.Players[1].Hand[0] = Card{"second", color, kind}
			mustAct(t, g, g.Players[0], Action{Type: "play", Card: "first", Color: "green"})
			if g.Pending != n || g.Turn != 1 {
				t.Fatal("No ofreció acumular")
			}
			mustAct(t, g, g.Players[1], Action{Type: "play", Card: "second", Color: "red"})
			if g.Pending != 0 || g.LastPayment.Amount != 2*n || len(g.Players[2].Hand) != 3+2*n || g.Turn != 0 {
				t.Fatal("No cobró la deuda automáticamente")
			}
		})
	}
}
func TestMixedStackColorMatrix(t *testing.T) {
	cases := []struct {
		mix                bool
		last, color, value string
		want               bool
	}{
		{false, "draw2", "wild", "draw4", false}, {false, "draw4", "red", "draw2", false},
		{true, "draw2", "wild", "draw4", true}, {true, "draw4", "red", "draw2", true}, {true, "draw4", "blue", "draw2", false},
		{false, "draw2", "blue", "draw2", true}, {false, "draw4", "wild", "draw4", true},
		{true, "draw4", "red", "skip", false}, {true, "draw2", "wild", "wild", false},
	}
	for _, c := range cases {
		g := setup(t, 3)
		g.Pending = 4
		g.PendingKind = c.last
		g.Options.MixDrawCards = c.mix
		got := g.playable(Card{"test", c.color, c.value}, g.Players[0])
		if got != c.want {
			t.Fatalf("%+v: %v", c, got)
		}
	}
}
func TestMixedDebtTotalsAndRejectWrongColor(t *testing.T) {
	g := setup(t, 3)
	g.Options.MixDrawCards = true
	g.Players[0].Hand[0] = Card{"two", "red", "draw2"}
	g.Players[1].Hand[0] = Card{"four", "wild", "draw4"}
	g.Players[2].Hand = []Card{{"wrong", "blue", "draw2"}, {"right", "green", "draw2"}, {"extra", "yellow", "3"}}
	mustAct(t, g, g.Players[0], Action{Type: "play", Card: "two"})
	mustAct(t, g, g.Players[1], Action{Type: "play", Card: "four", Color: "green"})
	if g.Pending != 6 || g.Turn != 2 {
		t.Fatal("No sumó +6")
	}
	if g.act(g.Players[2], Action{Type: "play", Card: "wrong"}) == nil || g.Pending != 6 {
		t.Fatal("Aceptó +2 de otro color")
	}
	mustAct(t, g, g.Players[2], Action{Type: "play", Card: "right"})
	if g.LastPayment.Amount != 8 || g.Pending != 0 {
		t.Fatal("No sumó +8")
	}
}
func TestDrawOptionsAndForcedPlay(t *testing.T) {
	for _, until := range []bool{false, true} {
		g := setup(t, 3)
		g.Options.DrawUntilPlayable = until
		p := g.Players[0]
		p.Hand = []Card{{"old", "blue", "0"}}
		g.Pile = []Card{{"good", "red", "2"}, {"bad2", "yellow", "1"}, {"bad1", "green", "1"}}
		mustAct(t, g, p, Action{Type: "draw"})
		want := 2
		if until {
			want = 4
		}
		if len(p.Hand) != want {
			t.Fatal("Cantidad incorrecta")
		}
		if until && (g.Drawn != "good" || g.Turn != 0) {
			t.Fatal("No detuvo el robo al encontrar carta")
		}
	}
	g := setup(t, 3)
	p := g.Players[0]
	g.Options.ForcePlay = true
	if g.act(p, Action{Type: "draw"}) == nil || g.act(p, Action{Type: "pass"}) == nil {
		t.Fatal("Permitió evitar carta disponible")
	}
	g.Options.ForcePlay = false
	mustAct(t, g, p, Action{Type: "pass"})
	if g.Turn != 1 {
		t.Fatal("No permitió pasar voluntariamente")
	}
}
func TestForcedDrawnCardAndEmptyDeck(t *testing.T) {
	g := setup(t, 3)
	g.Options = Options{ForcePlay: true, DrawUntilPlayable: true}
	p := g.Players[0]
	p.Hand = []Card{{"old", "blue", "0"}}
	g.Pile = []Card{{"good", "red", "2"}}
	mustAct(t, g, p, Action{Type: "draw"})
	if g.act(p, Action{Type: "pass"}) == nil {
		t.Fatal("Permitió pasar con carta recién robada válida")
	}
	mustAct(t, g, p, Action{Type: "play", Card: "good", Uno: true})
	g = setup(t, 3)
	g.Options.DrawUntilPlayable = true
	g.Pile = nil
	g.Discard = []Card{*g.Top}
	mustAct(t, g, g.Players[0], Action{Type: "draw"})
	if g.Turn != 1 {
		t.Fatal("Mazo agotado bloqueó el turno")
	}
}
func TestSettingsHostOnlyAndBetweenRounds(t *testing.T) {
	g := setup(t, 3)
	g.Phase = "lobby"
	o := &Options{TurnSeconds: 30, DrawUntilPlayable: true, ForcePlay: true, MixDrawCards: true}
	if g.configure(g.Players[1], o) == nil {
		t.Fatal("No validó anfitrión")
	}
	if e := g.configure(g.Players[0], o); e != nil {
		t.Fatal(e)
	}
	if g.configure(g.Players[0], &Options{TurnSeconds: 2}) == nil {
		t.Fatal("Aceptó límite inválido")
	}
	if e := g.start(g.Players[0]); e != nil {
		t.Fatal(e)
	}
	if g.Deadline.IsZero() || g.configure(g.Players[0], o) == nil {
		t.Fatal("Configuración activa incorrecta")
	}
}
func TestTimerDoesNotResetOnDrawAndExpiresOnce(t *testing.T) {
	g := setup(t, 3)
	g.Options.TurnSeconds = 15
	g.resetDeadline()
	deadline := g.Deadline
	g.Pile = append(g.Pile, Card{"new", "red", "2"})
	p := g.Players[0]
	mustAct(t, g, p, Action{Type: "draw"})
	if !g.Deadline.Equal(deadline) {
		t.Fatal("Robar reinició el reloj")
	}
	count := len(p.Hand)
	if !g.expireTurn(deadline.Add(time.Millisecond)) || g.Turn != 1 || len(p.Hand) != count {
		t.Fatal("Expiración tras robo incorrecta")
	}
	if g.expireTurn(time.Now()) {
		t.Fatal("Expiró de nuevo el turno siguiente")
	}
}
func TestTimeoutPaysDebtAndPendingWin(t *testing.T) {
	g := setup(t, 2)
	g.Options.TurnSeconds = 5
	g.Players[0].Hand = []Card{{"last", "red", "draw2"}}
	g.Players[1].Hand = []Card{{"counter", "blue", "draw2"}, {"extra", "green", "1"}}
	mustAct(t, g, g.Players[0], Action{Type: "play", Card: "last"})
	if g.Phase != "playing" || g.Pending != 2 {
		t.Fatal("Terminó antes de resolver acumulación")
	}
	if !g.expireTurn(g.Deadline.Add(time.Second)) || g.Phase != "finished" || g.Winner != g.Players[0].ID || len(g.Players[1].Hand) != 4 {
		t.Fatal("No cobró deuda al vencer tiempo")
	}
}
func TestLastDrawCardCanBeCounteredBackToEmptyPlayer(t *testing.T) {
	g := setup(t, 2)
	g.Players[0].Hand = []Card{{"first", "red", "draw2"}}
	g.Players[1].Hand = []Card{{"last", "blue", "draw2"}}
	mustAct(t, g, g.Players[0], Action{Type: "play", Card: "first"})
	mustAct(t, g, g.Players[1], Action{Type: "play", Card: "last"})
	if g.Winner != g.Players[1].ID || len(g.Players[0].Hand) != 4 {
		t.Fatal("Victoria no respetó devolución de deuda")
	}
}
func TestServerTimerRunsWithoutAnyBrowser(t *testing.T) {
	g := setup(t, 3)
	g.Options.TurnSeconds = 60
	g.Deadline = time.Now().Add(30 * time.Millisecond)
	a := &app{g: g, clients: map[*subscriber]bool{}}
	a.scheduleClock()
	defer func() {
		a.mu.Lock()
		if a.timer != nil {
			a.timer.Stop()
		}
		a.mu.Unlock()
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		done := g.Revision == 1 && g.Turn == 1 && g.Effect.Type == "timeout"
		a.mu.Unlock()
		if done {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("El reloj dependía de navegadores abiertos")
}
func TestDebtAndOptionsArePublicButHandsRemainPrivate(t *testing.T) {
	g := setup(t, 3)
	g.Pending = 6
	g.PendingKind = "draw4"
	g.Options.MixDrawCards = true
	g.Options.TurnSeconds = 20
	g.resetDeadline()
	b, _ := json.Marshal(g.state(nil))
	var s State
	json.Unmarshal(b, &s)
	if s.Pending != 6 || !s.Options.MixDrawCards || s.DeadlineMS == 0 || len(s.Hand) != 0 {
		t.Fatal("Vista pública incorrecta")
	}
	a := &app{g: g, clients: map[*subscriber]bool{}}
	r := httptest.NewRecorder()
	a.ServeHTTP(r, httptest.NewRequest("POST", "/api/action", strings.NewReader(`{"type":"configure","token":"invalid","options":{"turnSeconds":30}}`)))
	if r.Code != 400 {
		t.Fatal("Configuración sin autorización")
	}
}
