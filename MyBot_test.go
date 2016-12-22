package main

import (
	"fmt"
	"hlt"
	"testing"
	"time"
)

func MockSite() hlt.Site {
	return hlt.Site{
		Owner:      0,
		Production: 0,
		Strength:   0,
	}
}

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

func TestStack(t *testing.T) {
	stack := NewStack()
	c1 := NewCell(nil, MockSite(), 0, 0)
	c2 := NewCell(nil, MockSite(), 0, 1)
	c3 := NewCell(nil, MockSite(), 0, 2)
	stack.Push(c1)
	stack.Push(c2)
	if cell, _ := stack.Peek(); cell != c2 {
		fmt.Println(stack.Peek())
		t.Fail()
	}
	if !stack.Contains(c1) {
		fmt.Println("Stack should contain c1!")
		t.Fail()
	}
	if !stack.Contains(c2) {
		fmt.Println("Stack should contain c2!")
		t.Fail()
	}
	if stack.Contains(c3) {
		fmt.Println("Stack should NOT contain c3!")
		t.Fail()
	}
}

func TestCellsGetLocation1(t *testing.T) {
	m := MockGameBoard(0, 1, 2, 5, 5)
	cells := NewCells(-1, -1, 3, 3, m)
	cell := cells.Get(4, 4)
	// East neighbor wraps
	if fmt.Sprint(cell.GetNeighbor(hlt.EAST)) != "(x:0, y:4)[o:0, p:1, s:2]" {
		fmt.Println(cell.GetNeighbor(hlt.EAST))
		t.Fail()
	}
	// south neighbor wraps
	if fmt.Sprint(cell.GetNeighbor(hlt.SOUTH)) != "(x:4, y:0)[o:0, p:1, s:2]" {
		fmt.Println(cell.GetNeighbor(hlt.SOUTH))
		t.Fail()
	}
}

func TestCellsGetLocation2(t *testing.T) {
	m := MockGameBoard(0, 1, 2, 5, 5)
	cells := NewCells(4, 4, 3, 3, m)
	cell := cells.Get(4, 4)
	// East neighbor wraps
	if fmt.Sprint(cell.GetNeighbor(hlt.EAST)) != "(x:0, y:4)[o:0, p:1, s:2]" {
		fmt.Println(cell.GetNeighbor(hlt.EAST))
		t.Fail()
	}
	// south neighbor wraps
	if fmt.Sprint(cell.GetNeighbor(hlt.SOUTH)) != "(x:4, y:0)[o:0, p:1, s:2]" {
		fmt.Println(cell.GetNeighbor(hlt.SOUTH))
		t.Fail()
	}
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
	moves := hlt.MoveSet{}
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 2, Y: 3}, Direction: hlt.NORTH})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 2, Y: 1}, Direction: hlt.SOUTH})
	newCells := cells.Simulate(moves)
	// top center
	if fmt.Sprint(newCells.Get(2, 1)) != "(x:2, y:1)[o:0, p:0, s:0]" {
		fmt.Println(newCells.Get(2, 1))
		t.Fail()
	}
	// left center
	if fmt.Sprint(newCells.Get(1, 2)) != "(x:1, y:2)[o:1, p:0, s:5]" {
		fmt.Println(newCells.Get(1, 2))
		t.Fail()
	}
	// middle center
	if fmt.Sprint(newCells.Get(2, 2)) != "(x:2, y:2)[o:1, p:0, s:5]" {
		fmt.Println(newCells.Get(2, 2))
		t.Fail()
	}
	// right center
	if fmt.Sprint(newCells.Get(3, 2)) != "(x:3, y:2)[o:1, p:0, s:5]" {
		fmt.Println(newCells.Get(3, 2))
		t.Fail()
	}
	// bottom center
	if fmt.Sprint(newCells.Get(2, 3)) != "(x:2, y:3)[o:0, p:0, s:0]" {
		fmt.Println(newCells.Get(2, 3))
		t.Fail()
	}

	owned1 := newCells.ByOwner[1]
	if owned1.TotalProduction != 0 || owned1.TotalStrength != 45 || owned1.TotalTerritory != 5 {
		fmt.Printf("Owned 1: p:%d, s:%d, t:%d\n", owned1.TotalProduction, owned1.TotalStrength, owned1.TotalTerritory)
		t.Fail()
	}
	owned2 := newCells.ByOwner[2]
	if owned2.TotalProduction != 0 || owned2.TotalStrength != 0 || owned2.TotalTerritory != 0 {
		fmt.Printf("Owned 2: p:%d, s:%d, t:%d\n", owned2.TotalProduction, owned2.TotalStrength, owned2.TotalTerritory)
		t.Fail()
	}
}

func TestCellsSimulation2(t *testing.T) {
	m := MockGameBoard(0, 0, 0, 5, 5)
	setSite(1, 0, 15, &m.Contents[2][1])
	setSite(1, 0, 15, &m.Contents[1][2])
	setSite(1, 0, 15, &m.Contents[2][3])
	setSite(2, 0, 10, &m.Contents[3][2])

	cells := NewCells(1, 1, 3, 3, m)
	moves := hlt.MoveSet{}
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 2, Y: 3}, Direction: hlt.NORTH})
	newCells := cells.Simulate(moves)
	// top center
	if fmt.Sprint(newCells.Get(2, 1)) != "(x:2, y:1)[o:1, p:0, s:5]" {
		fmt.Println(newCells.Get(2, 1))
		t.Fail()
	}
	// left center
	if fmt.Sprint(newCells.Get(1, 2)) != "(x:1, y:2)[o:1, p:0, s:5]" {
		fmt.Println(newCells.Get(1, 2))
		t.Fail()
	}
	// middle center
	if fmt.Sprint(newCells.Get(2, 2)) != "(x:2, y:2)[o:0, p:0, s:0]" {
		fmt.Println(newCells.Get(2, 2))
		t.Fail()
	}
	// right center
	if fmt.Sprint(newCells.Get(3, 2)) != "(x:3, y:2)[o:1, p:0, s:5]" {
		fmt.Println(newCells.Get(3, 2))
		t.Fail()
	}
	// bottom center
	if fmt.Sprint(newCells.Get(2, 3)) != "(x:2, y:3)[o:2, p:0, s:0]" {
		fmt.Println(newCells.Get(2, 3))
		t.Fail()
	}

	owned1 := newCells.ByOwner[1]
	if owned1.TotalProduction != 0 || owned1.TotalStrength != 15 || owned1.TotalTerritory != 3 {
		fmt.Printf("Owned 1: p:%d, s:%d, t:%d\n", owned1.TotalProduction, owned1.TotalStrength, owned1.TotalTerritory)
		t.Fail()
	}
	owned2 := newCells.ByOwner[2]
	if owned2.TotalProduction != 0 || owned2.TotalStrength != 0 || owned2.TotalTerritory != 1 {
		fmt.Printf("Owned 2: p:%d, s:%d, t:%d\n", owned2.TotalProduction, owned2.TotalStrength, owned2.TotalTerritory)
		t.Fail()
	}
}

func TestCellsSimulation3(t *testing.T) {
	m := MockGameBoard(0, 0, 0, 2, 2)
	setSite(1, 0, 15, &m.Contents[0][0])
	setSite(2, 0, 15, &m.Contents[1][1])

	cells := NewCells(0, 0, 2, 2, m)
	moves := hlt.MoveSet{}
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 0, Y: 0}, Direction: hlt.EAST})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 1, Y: 1}, Direction: hlt.WEST})
	newCells := cells.Simulate(moves)
	// top left
	if fmt.Sprint(newCells.Get(0, 0)) != "(x:0, y:0)[o:0, p:0, s:0]" {
		fmt.Println(newCells.Get(0, 0))
		t.Fail()
	}
	// top right
	if fmt.Sprint(newCells.Get(1, 0)) != "(x:1, y:0)[o:1, p:0, s:15]" {
		fmt.Println(newCells.Get(1, 0))
		t.Fail()
	}
	// bottom left
	if fmt.Sprint(newCells.Get(0, 1)) != "(x:0, y:1)[o:2, p:0, s:15]" {
		fmt.Println(newCells.Get(0, 1))
		t.Fail()
	}
	// bottom right
	if fmt.Sprint(newCells.Get(1, 1)) != "(x:1, y:1)[o:0, p:0, s:0]" {
		fmt.Println(newCells.Get(1, 1))
		t.Fail()
	}

	owned1 := newCells.ByOwner[1]
	if owned1.TotalProduction != 0 || owned1.TotalStrength != 15 || owned1.TotalTerritory != 1 {
		fmt.Printf("Owned 1: p:%d, s:%d, t:%d\n", owned1.TotalProduction, owned1.TotalStrength, owned1.TotalTerritory)
		t.Fail()
	}
	owned2 := newCells.ByOwner[2]
	if owned2.TotalProduction != 0 || owned2.TotalStrength != 15 || owned2.TotalTerritory != 1 {
		fmt.Printf("Owned 2: p:%d, s:%d, t:%d\n", owned2.TotalProduction, owned2.TotalStrength, owned2.TotalTerritory)
		t.Fail()
	}
}

func TestCellsSimulation4(t *testing.T) {
	startTime := time.Now()
	m := MockGameBoard(0, 1, 0, 4, 4)
	setSite(1, 1, 15, &m.Contents[0][0])
	setSite(2, 1, 15, &m.Contents[0][3])
	setSite(3, 1, 15, &m.Contents[3][3])
	setSite(4, 1, 15, &m.Contents[3][0])
	cells := NewCells(0, 0, 4, 4, m)
	moves := hlt.MoveSet{}
	newCells := cells.Simulate(moves)

	owned1 := newCells.ByOwner[1]
	if owned1.TotalProduction != 1 || owned1.TotalStrength != 16 || owned1.TotalTerritory != 1 {
		fmt.Printf("Owned 1: p:%d, s:%d, t:%d\n", owned1.TotalProduction, owned1.TotalStrength, owned1.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned1).SingleScore()) != "0.045" {
		fmt.Printf("Owned 1 Score: %f\n", NewOwnerScore(owned1).SingleScore())
		t.Fail()
	}
	owned2 := newCells.ByOwner[2]
	if owned2.TotalProduction != 1 || owned2.TotalStrength != 16 || owned2.TotalTerritory != 1 {
		fmt.Printf("Owned 2: p:%d, s:%d, t:%d\n", owned2.TotalProduction, owned2.TotalStrength, owned2.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned2).SingleScore()) != "0.045" {
		fmt.Printf("Owned 2 Score: %f\n", NewOwnerScore(owned2).SingleScore())
		t.Fail()
	}
	owned3 := newCells.ByOwner[2]
	if owned3.TotalProduction != 1 || owned3.TotalStrength != 16 || owned3.TotalTerritory != 1 {
		fmt.Printf("Owned 3: p:%d, s:%d, t:%d\n", owned3.TotalProduction, owned3.TotalStrength, owned3.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned3).SingleScore()) != "0.045" {
		fmt.Printf("Owned 3 Score: %f\n", NewOwnerScore(owned3).SingleScore())
		t.Fail()
	}
	owned4 := newCells.ByOwner[2]
	if owned4.TotalProduction != 1 || owned4.TotalStrength != 16 || owned4.TotalTerritory != 1 {
		fmt.Printf("Owned 4: p:%d, s:%d, t:%d\n", owned4.TotalProduction, owned4.TotalStrength, owned4.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned4).SingleScore()) != "0.045" {
		fmt.Printf("Owned 4 Score: %f\n", NewOwnerScore(owned4).SingleScore())
		t.Fail()
	}

	moves = append(moves, hlt.Move{Location: hlt.Location{X: 0, Y: 0}, Direction: hlt.EAST})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 0, Y: 3}, Direction: hlt.NORTH})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 3, Y: 3}, Direction: hlt.WEST})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 3, Y: 0}, Direction: hlt.SOUTH})
	newCells = newCells.Simulate(moves)

	owned1 = newCells.ByOwner[1]
	if owned1.TotalProduction != 2 || owned1.TotalStrength != 16 || owned1.TotalTerritory != 2 {
		fmt.Printf("Owned 1: p:%d, s:%d, t:%d\n", owned1.TotalProduction, owned1.TotalStrength, owned1.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned1).SingleScore()) != "0.077" {
		fmt.Printf("Owned 1 Score: %f\n", NewOwnerScore(owned1).SingleScore())
		t.Fail()
	}
	owned2 = newCells.ByOwner[2]
	if owned2.TotalProduction != 2 || owned2.TotalStrength != 16 || owned2.TotalTerritory != 2 {
		fmt.Printf("Owned 2: p:%d, s:%d, t:%d\n", owned2.TotalProduction, owned2.TotalStrength, owned2.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned2).SingleScore()) != "0.077" {
		fmt.Printf("Owned 2 Score: %f\n", NewOwnerScore(owned2).SingleScore())
		t.Fail()
	}
	owned3 = newCells.ByOwner[2]
	if owned3.TotalProduction != 2 || owned3.TotalStrength != 16 || owned3.TotalTerritory != 2 {
		fmt.Printf("Owned 3: p:%d, s:%d, t:%d\n", owned3.TotalProduction, owned3.TotalStrength, owned3.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned3).SingleScore()) != "0.077" {
		fmt.Printf("Owned 3 Score: %f\n", NewOwnerScore(owned3).SingleScore())
		t.Fail()
	}
	owned4 = newCells.ByOwner[2]
	if owned4.TotalProduction != 2 || owned4.TotalStrength != 16 || owned4.TotalTerritory != 2 {
		fmt.Printf("Owned 4: p:%d, s:%d, t:%d\n", owned4.TotalProduction, owned4.TotalStrength, owned4.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned4).SingleScore()) != "0.077" {
		fmt.Printf("Owned 4 Score: %f\n", NewOwnerScore(owned4).SingleScore())
		t.Fail()
	}

	moves = hlt.MoveSet{}
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 1, Y: 0}, Direction: hlt.SOUTH})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 0, Y: 2}, Direction: hlt.EAST})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 2, Y: 3}, Direction: hlt.NORTH})
	moves = append(moves, hlt.Move{Location: hlt.Location{X: 3, Y: 1}, Direction: hlt.WEST})
	newCells = newCells.Simulate(moves)

	owned1 = newCells.ByOwner[1]
	if owned1.TotalProduction != 2 || owned1.TotalStrength != 1 || owned1.TotalTerritory != 2 {
		fmt.Printf("Owned 1: p:%d, s:%d, t:%d\n", owned1.TotalProduction, owned1.TotalStrength, owned1.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned1).SingleScore()) != "0.065" {
		fmt.Printf("Owned 1 Score: %f\n", NewOwnerScore(owned1).SingleScore())
		t.Fail()
	}
	owned2 = newCells.ByOwner[2]
	if owned2.TotalProduction != 2 || owned2.TotalStrength != 1 || owned2.TotalTerritory != 2 {
		fmt.Printf("Owned 2: p:%d, s:%d, t:%d\n", owned2.TotalProduction, owned2.TotalStrength, owned2.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned2).SingleScore()) != "0.065" {
		fmt.Printf("Owned 2 Score: %f\n", NewOwnerScore(owned2).SingleScore())
		t.Fail()
	}
	owned3 = newCells.ByOwner[2]
	if owned3.TotalProduction != 2 || owned3.TotalStrength != 1 || owned3.TotalTerritory != 2 {
		fmt.Printf("Owned 3: p:%d, s:%d, t:%d\n", owned3.TotalProduction, owned3.TotalStrength, owned3.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned3).SingleScore()) != "0.065" {
		fmt.Printf("Owned 3 Score: %f\n", NewOwnerScore(owned3).SingleScore())
		t.Fail()
	}
	owned4 = newCells.ByOwner[2]
	if owned4.TotalProduction != 2 || owned4.TotalStrength != 1 || owned4.TotalTerritory != 2 {
		fmt.Printf("Owned 4: p:%d, s:%d, t:%d\n", owned4.TotalProduction, owned4.TotalStrength, owned4.TotalTerritory)
		t.Fail()
	}
	if fmt.Sprintf("%.3f", NewOwnerScore(owned4).SingleScore()) != "0.065" {
		fmt.Printf("Owned 4 Score: %f\n", NewOwnerScore(owned4).SingleScore())
		t.Fail()
	}

	fmt.Printf("Time: %v\n", time.Now().Sub(startTime))
}

func TestProjectedMove(t *testing.T) {
	startTime := time.Now()

	m := MockGameBoard(0, 1, 1, 5, 5)
	setSite(1, 1, 10, &m.Contents[2][2])
	setSite(2, 1, 5, &m.Contents[0][2])
	setSite(2, 1, 5, &m.Contents[2][0])
	setSite(2, 1, 5, &m.Contents[2][4])
	setSite(2, 1, 5, &m.Contents[4][2])
	bot := NewBot(1, m)
	location := hlt.NewLocation(2, 2)
	cells := bot.ProjectedCells(location)
	fmt.Println(cells)
	locations := bot.ProjectedMoves(location, cells)
	fmt.Println(locations)

	fmt.Println(cells.InBounds(location))

	fmt.Printf("Time: %v\n", time.Now().Sub(startTime))
}

func BenchmarkSimulation(t *testing.B) {}
