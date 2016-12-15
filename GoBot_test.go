package main

import (
	"fmt"
	"hlt"
	"testing"
)

func setSite(owner, production, strength int, site *hlt.Site) {
	site.Owner = owner
	site.Production = production
	site.Strength = strength
}

// Assumes 2D array containing arrays of 3 elements [Owner,Production,Strength]
func MockGameBoard(owner, production, strength, width, height int) hlt.GameMap {
	m := hlt.NewGameMap(width, height)
	for y, _ := range m.Contents {
		for x, _ := range m.Contents[y] {
			setSite(owner, production, strength, &m.Contents[y][x])
		}
	}
	return m
}

func TestCells(t *testing.T) {
	m := MockGameBoard(0, 10, 15, 5, 5)
	setSite(1, 1, 1, &m.Contents[1][1])
	setSite(1, 1, 1, &m.Contents[2][1])
	setSite(1, 1, 1, &m.Contents[3][1])
	setSite(2, 1, 1, &m.Contents[2][3])

	cells := NewCells(1, 1, 3, 3, m)
	fmt.Println(cells)
	moves := hlt.MoveSet{}
	moves = append(moves, hlt.Move{hlt.Location{2, 3}, hlt.WEST})
	newCells := cells.Simulate(moves)
	fmt.Println(newCells)
}

func BenchmarkSimulation(t *testing.B) {

}
