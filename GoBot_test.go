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
	for y := range m.Contents {
		for x := range m.Contents[y] {
			setSite(owner, production, strength, &m.Contents[y][x])
		}
	}
	return m
}

func TestCellsSimulation1(t *testing.T) {
	m := MockGameBoard(0, 0, 0, 5, 5)
	setSite(1, 0, 15, &m.Contents[1][1])
	setSite(1, 0, 15, &m.Contents[1][3])
	setSite(1, 0, 15, &m.Contents[2][1])
	setSite(1, 0, 15, &m.Contents[1][2])
	setSite(1, 0, 15, &m.Contents[2][3])
	setSite(2, 0, 10, &m.Contents[3][2])

	cells := NewCells(1, 1, 3, 3, m)
	fmt.Println(cells)
	moves := hlt.MoveSet{}
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 2, Y: 3}, Direction: hlt.NORTH})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 2, Y: 1}, Direction: hlt.SOUTH})
	newCells := cells.Simulate(moves)
	fmt.Println("\"", newCells, "\"")
}

func TestCellsSimulation2(t *testing.T) {
	m := MockGameBoard(0, 0, 0, 5, 5)
	setSite(1, 0, 15, &m.Contents[2][1])
	setSite(1, 0, 15, &m.Contents[1][2])
	setSite(1, 0, 15, &m.Contents[2][3])
	setSite(2, 0, 10, &m.Contents[3][2])

	cells := NewCells(1, 1, 3, 3, m)
	fmt.Println(cells)
	moves := hlt.MoveSet{}
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 2, Y: 3}, Direction: hlt.NORTH})
	newCells := cells.Simulate(moves)
	fmt.Println(newCells)
}

func TestCellsSimulation3(t *testing.T) {
	m := MockGameBoard(0, 0, 0, 2, 2)
	setSite(1, 0, 15, &m.Contents[0][0])
	setSite(2, 0, 15, &m.Contents[1][1])

	cells := NewCells(0, 0, 2, 2, m)
	fmt.Println(cells)
	moves := hlt.MoveSet{}
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 0, Y: 0}, Direction: hlt.EAST})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 1, Y: 1}, Direction: hlt.WEST})
	newCells := cells.Simulate(moves)
	fmt.Println(newCells)
}

func BenchmarkSimulation(t *testing.B) {

}
