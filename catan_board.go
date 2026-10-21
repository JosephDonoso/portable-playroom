package main

import (
	"fmt"
	"math"
	"slices"
)

// Resource order: lumber, brick, wool, grain, ore.
type catResources [5]int
type catVertex struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Owner string  `json:"owner"`
	City  bool    `json:"city"`
	Port  int     `json:"port"` // -2: none; -1: generic 3:1; 0..4: specific 2:1
}
type catEdge struct {
	A     int    `json:"a"`
	B     int    `json:"b"`
	Owner string `json:"owner"`
}
type catHex struct {
	Q        int   `json:"q"`
	R        int   `json:"r"`
	Resource int   `json:"resource"`
	Number   int   `json:"number"`
	Vertices []int `json:"vertices"`
}
type catPort struct {
	Edge     int `json:"edge"`
	Resource int `json:"resource"`
}
type catBoard struct {
	Hexes    []catHex    `json:"hexes"`
	Vertices []catVertex `json:"vertices"`
	Edges    []catEdge   `json:"edges"`
	Ports    []catPort   `json:"ports"`
	Robber   int         `json:"robber"`
}

func catShuffle[T any](v []T) {
	for i := len(v) - 1; i > 0; i-- {
		j := catRandom(i + 1)
		v[i], v[j] = v[j], v[i]
	}
}
func catNewBoard() catBoard {
	b := catBoard{}
	terrain := []int{0, 0, 0, 0, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, -1}
	catShuffle(terrain)
	vertexIDs := map[string]int{}
	edgeIDs := map[[2]int]int{}
	edgeCounts := map[int]int{}
	for r := -2; r <= 2; r++ {
		for q := -2; q <= 2; q++ {
			if abs(q+r) > 2 {
				continue
			}
			h := catHex{Q: q, R: r, Resource: terrain[len(b.Hexes)]}
			for k := 0; k < 6; k++ {
				a := (30 + 60*float64(k)) * math.Pi / 180
				x, y := math.Sqrt(3)*(float64(q)+float64(r)/2)+math.Cos(a), 1.5*float64(r)+math.Sin(a)
				key := fmt.Sprintf("%.0f,%.0f", x*10000, y*10000)
				// Normalize negative zero before deduplication.
				key = fmt.Sprintf("%d,%d", int(math.Round(x*10000)), int(math.Round(y*10000)))
				vi, ok := vertexIDs[key]
				if !ok {
					vi = len(b.Vertices)
					vertexIDs[key] = vi
					b.Vertices = append(b.Vertices, catVertex{X: x, Y: y, Port: -2})
				}
				h.Vertices = append(h.Vertices, vi)
			}
			for k, a := range h.Vertices {
				c := h.Vertices[(k+1)%6]
				key := [2]int{min(a, c), max(a, c)}
				ei, ok := edgeIDs[key]
				if !ok {
					ei = len(b.Edges)
					edgeIDs[key] = ei
					b.Edges = append(b.Edges, catEdge{A: a, B: c})
				}
				edgeCounts[ei]++
			}
			if h.Resource < 0 {
				b.Robber = len(b.Hexes)
			}
			b.Hexes = append(b.Hexes, h)
		}
	}
	// Number tokens A–R, clockwise from an outer corner, skipping the desert.
	numbers := []int{5, 2, 6, 3, 8, 10, 9, 12, 11, 4, 8, 10, 9, 4, 5, 6, 3, 11}
	positions := [][2]int{}
	dirs := [][2]int{{1, 0}, {1, -1}, {0, -1}, {-1, 0}, {-1, 1}, {0, 1}}
	for radius := 2; radius >= 1; radius-- {
		q, r := -radius, radius
		for _, d := range dirs {
			for k := 0; k < radius; k++ {
				positions = append(positions, [2]int{q, r})
				q += d[0]
				r += d[1]
			}
		}
	}
	positions = append(positions, [2]int{0, 0})
	ni := 0
	for _, pos := range positions {
		for i := range b.Hexes {
			h := &b.Hexes[i]
			if h.Q == pos[0] && h.R == pos[1] && h.Resource >= 0 {
				h.Number = numbers[ni]
				ni++
			}
		}
	}
	coast := []int{}
	for i := range b.Edges {
		if edgeCounts[i] == 1 {
			coast = append(coast, i)
		}
	}
	// Walk the coastline so a port always uses the two ends of one sea edge.
	ordered := []int{coast[0]}
	at := b.Edges[coast[0]].B
	used := map[int]bool{coast[0]: true}
	for len(ordered) < len(coast) {
		for _, ei := range coast {
			e := b.Edges[ei]
			if !used[ei] && (e.A == at || e.B == at) {
				ordered = append(ordered, ei)
				used[ei] = true
				if e.A == at {
					at = e.B
				} else {
					at = e.A
				}
				break
			}
		}
	}
	ports := []int{-1, -1, -1, -1, 0, 1, 2, 3, 4}
	catShuffle(ports)
	for i, offset := range []int{0, 3, 7, 10, 13, 17, 20, 23, 27} {
		ei := ordered[offset]
		b.Ports = append(b.Ports, catPort{ei, ports[i]})
		e := b.Edges[ei]
		b.Vertices[e.A].Port = ports[i]
		b.Vertices[e.B].Port = ports[i]
	}
	return b
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func (b *catBoard) adjacent(v int) []int {
	result := []int{}
	for _, e := range b.Edges {
		if e.A == v {
			result = append(result, e.B)
		} else if e.B == v {
			result = append(result, e.A)
		}
	}
	return result
}
func (b *catBoard) settlementSpot(v int) bool {
	if v < 0 || v >= len(b.Vertices) || b.Vertices[v].Owner != "" {
		return false
	}
	for _, n := range b.adjacent(v) {
		if b.Vertices[n].Owner != "" {
			return false
		}
	}
	return true
}
func (b *catBoard) touchesRoad(v int, owner string) bool {
	for _, e := range b.Edges {
		if e.Owner == owner && (e.A == v || e.B == v) {
			return true
		}
	}
	return false
}
func (b *catBoard) roadSpot(ei int, owner string) bool {
	if ei < 0 || ei >= len(b.Edges) || b.Edges[ei].Owner != "" {
		return false
	}
	e := b.Edges[ei]
	for _, v := range []int{e.A, e.B} {
		o := b.Vertices[v].Owner
		if o == owner || o == "" && b.touchesRoad(v, owner) {
			return true
		}
	}
	return false
}
func (b *catBoard) longest(owner string) int {
	best := 0
	var walk func(int, map[int]bool) int
	walk = func(v int, used map[int]bool) int {
		if len(used) > 0 && b.Vertices[v].Owner != "" && b.Vertices[v].Owner != owner {
			return 0
		}
		n := 0
		for i, e := range b.Edges {
			if e.Owner != owner || used[i] || e.A != v && e.B != v {
				continue
			}
			next := e.A
			if next == v {
				next = e.B
			}
			used[i] = true
			n = max(n, 1+walk(next, used))
			delete(used, i)
		}
		return n
	}
	for v := range b.Vertices {
		best = max(best, walk(v, map[int]bool{}))
	}
	return best
}
func (b *catBoard) victims(tile int, me string, players []*catPlayer) []string {
	result := []string{}
	for _, vi := range b.Hexes[tile].Vertices {
		owner := b.Vertices[vi].Owner
		if owner == "" || owner == me || slices.Contains(result, owner) {
			continue
		}
		for _, p := range players {
			if p.ID == owner && p.Hand.total() > 0 {
				result = append(result, owner)
			}
		}
	}
	return result
}
