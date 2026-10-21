package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func connectPlayer(t *testing.T, base, token string) (*http.Response, State) {
	t.Helper()
	response, err := http.Get(base + "/events?token=" + token)
	if err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	var state State
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &state); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	return response, state
}

func waitForConnections(t *testing.T, a *app, count int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		n := len(a.clients)
		a.mu.Unlock()
		if n == count {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("No se cerró la conexión del dispositivo")
}

func TestDisconnectKeepsHandAndWaitsForPlayer(t *testing.T) {
	g := setup(t, 3)
	p := g.Players[1]
	a := &app{g: g, clients: map[*subscriber]bool{}}
	server := httptest.NewServer(a)
	defer server.Close()
	response, before := connectPlayer(t, server.URL, p.Token)
	response.Body.Close()
	waitForConnections(t, a, 0)
	mustAct(t, g, g.Players[0], Action{Type: "play", Card: "a0"})
	if g.Turn != 1 || len(g.Players) != 3 {
		t.Fatal("La desconexión alteró el asiento o el turno")
	}
	if g.act(g.Players[2], Action{Type: "play", Card: "a2"}) == nil {
		t.Fatal("Otro jugador pudo saltar al ausente")
	}
	response, after := connectPlayer(t, server.URL, p.Token)
	defer response.Body.Close()
	if after.Me != before.Me || after.Turn != p.ID || !reflect.DeepEqual(after.Hand, before.Hand) || after.Top.ID == before.Top.ID {
		t.Fatal("La reconexión no recuperó mano, turno o tablero actual")
	}
	restored, err := g.join("Un nombre distinto", p.Token)
	if err != nil || restored != p || len(g.Players) != 3 {
		t.Fatal("Se duplicó el jugador al regresar")
	}
	mustAct(t, g, p, Action{Type: "play", Card: "a1"})
	if g.Turn != 2 {
		t.Fatal("No pudo continuar después de reconectar")
	}
}

func TestHostDisconnectDoesNotTransferControl(t *testing.T) {
	g := setup(t, 3)
	host := g.Players[0]
	a := &app{g: g, clients: map[*subscriber]bool{}}
	server := httptest.NewServer(a)
	defer server.Close()
	response, before := connectPlayer(t, server.URL, host.Token)
	response.Body.Close()
	waitForConnections(t, a, 0)
	response, after := connectPlayer(t, server.URL, host.Token)
	defer response.Body.Close()
	if after.Me != host.ID || after.Turn != host.ID || !reflect.DeepEqual(before.Hand, after.Hand) || g.Players[0] != host {
		t.Fatal("Se perdió el anfitrión o su turno")
	}
	// A browser with no saved identity cannot take an existing seat just by name.
	if _, err := g.join(host.Name, ""); err == nil {
		t.Fatal("Un navegador sin credencial tomó el lugar del anfitrión")
	}
}

func TestTwoTabsShareSeatAndReconnectToLatestState(t *testing.T) {
	g := setup(t, 3)
	p := g.Players[0]
	a := &app{g: g, clients: map[*subscriber]bool{}}
	server := httptest.NewServer(a)
	defer server.Close()
	first, one := connectPlayer(t, server.URL, p.Token)
	second, two := connectPlayer(t, server.URL, p.Token)
	if one.Me != two.Me || len(g.Players) != 3 {
		t.Fatal("Las pestañas crearon jugadores distintos")
	}
	first.Body.Close()
	waitForConnections(t, a, 1)
	second.Body.Close()
	waitForConnections(t, a, 0)
	mustAct(t, g, p, Action{Type: "play", Card: "a0"})
	response, current := connectPlayer(t, server.URL, p.Token)
	defer response.Body.Close()
	if len(current.Hand) != len(one.Hand)-1 || current.Top.ID != "a0" || current.Turn != g.Players[1].ID {
		t.Fatal("La reconexión recibió un estado viejo")
	}
}
