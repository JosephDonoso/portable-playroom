package main

import (
	"fmt"
	"slices"
)

func (g *catGame) trade(p *catPlayer, a catAction) error {
	active := g.Players[g.Turn].ID
	if a.Type == "offer" {
		if len(g.Offers) >= 12 {
			return fmt.Errorf("Cierra alguna oferta antes de publicar otra.")
		}
		to := a.Target
		if p.ID != active {
			to = active
		}
		if to == p.ID || to != "" && g.byID(to) == nil {
			return fmt.Errorf("Selecciona a otro jugador.")
		}
		if !a.Give.valid() || !a.Want.valid() || a.Give.total() == 0 || a.Want.total() == 0 || !p.Hand.has(a.Give) {
			return fmt.Errorf("Ofrece recursos que tengas y elige qué pides a cambio.")
		}
		for i, n := range a.Give {
			if n > 0 && a.Want[i] > 0 {
				return fmt.Errorf("No puedes ofrecer y pedir el mismo recurso.")
			}
		}
		if a.Offer != "" {
			parent := slices.IndexFunc(g.Offers, func(o catOffer) bool { return o.ID == a.Offer })
			if parent < 0 {
				return fmt.Errorf("La oferta original ya no existe.")
			}
			o := g.Offers[parent]
			if o.From == p.ID || o.To != "" && o.To != p.ID || to != o.From {
				return fmt.Errorf("Esta contraoferta debe responder al autor de la oferta.")
			}
		}
		g.Offers = append(g.Offers, catOffer{ID: id(), From: p.ID, To: to, Give: a.Give, Want: a.Want, Parent: a.Offer, Responses: map[string]string{}})
		g.note(p.Name + " publicó una oferta de intercambio.")
		return nil
	}
	i := slices.IndexFunc(g.Offers, func(o catOffer) bool { return o.ID == a.Offer })
	if i < 0 {
		return fmt.Errorf("La oferta ya no está disponible.")
	}
	o := &g.Offers[i]
	if a.Type == "cancelOffer" {
		if o.From != p.ID {
			return fmt.Errorf("Solo el autor puede retirar la oferta.")
		}
		g.Offers = slices.Delete(g.Offers, i, i+1)
		return nil
	}
	if a.Type == "respond" {
		if o.From == p.ID || o.To != "" && o.To != p.ID {
			return fmt.Errorf("Esta oferta no está dirigida a ti.")
		}
		if a.Response != "accept" && a.Response != "reject" {
			return fmt.Errorf("Respuesta inválida.")
		}
		if a.Response == "accept" && !p.Hand.has(o.Want) {
			return fmt.Errorf("No tienes los recursos para aceptar.")
		}
		o.Responses[p.ID] = a.Response
		return nil
	}
	if a.Type == "confirm" {
		other := g.byID(a.Target)
		if o.From != p.ID || other == nil || o.Responses[other.ID] != "accept" || o.To != "" && o.To != other.ID || p.ID != active && other.ID != active {
			return fmt.Errorf("Confirma con un jugador que haya aceptado la oferta y participe en el turno.")
		}
		if !p.Hand.has(o.Give) || !other.Hand.has(o.Want) {
			return fmt.Errorf("Los recursos cambiaron; revisa la oferta antes de intercambiar.")
		}
		p.Hand.add(o.Give, -1)
		other.Hand.add(o.Give, 1)
		other.Hand.add(o.Want, -1)
		p.Hand.add(o.Want, 1)
		g.note(p.Name + " intercambió recursos con " + other.Name + ".")
		g.Offers = slices.Delete(g.Offers, i, i+1)
		return nil
	}
	return fmt.Errorf("Negociación no disponible.")
}
