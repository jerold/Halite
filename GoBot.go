package main

import (
	"errors"
	"fmt"
	"hlt"
	"math/rand"
	"os"
	"sync"
	"time"
)

const logFile = "goLog.txt"

type stack struct {
	lock sync.Mutex // you don't have to do this if you don't want thread safety
	s    []*Cell
}

func NewStack() *stack {
	return &stack{sync.Mutex{}, make([]*Cell, 0)}
}

func (s *stack) Push(v *Cell) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.s = append(s.s, v)
}

func (s *stack) Pop() (*Cell, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := len(s.s)
	if l == 0 {
		return nil, errors.New("Empty Stack")
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res, nil
}

func (s *stack) length() int {
	return len(s.s)
}

func (s *stack) isEmpty() bool {
	return len(s.s) == 0
}

func (s *stack) isNotEmpty() bool {
	return !s.isEmpty()
}

func log(text string) {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	if _, err = f.WriteString(text); err != nil {
		panic(err)
	}
	f.Sync()
	f.Close()
}

func opposite(direction hlt.Direction) hlt.Direction {
	switch direction {
	case hlt.NORTH:
		return hlt.SOUTH
	case hlt.WEST:
		return hlt.EAST
	case hlt.SOUTH:
		return hlt.NORTH
	case hlt.EAST:
		return hlt.WEST
	default:
		return hlt.STILL
	}
}

type Initiative struct {
	Owner int
}

type BotMap struct {
	Owner     int
	GameMap   hlt.GameMap
	Turn      int
	Cells     [][]*Cell
	PrevCells [][]*Cell
	Moves     map[hlt.Location]hlt.Move
	PrevMoves map[hlt.Location]hlt.Move
	OwnedRows []bool
	OwnedCols []bool
}

func NewBotMap(owner int, gameMap hlt.GameMap) BotMap {
	var bm = BotMap{Owner: owner, GameMap: gameMap, Turn: 0}
	return bm
}

func (bm *BotMap) updateMap(gameMap hlt.GameMap) {
	bm.Turn = bm.Turn + 1
	bm.PrevCells = bm.Cells
	bm.Cells = make([][]*Cell, gameMap.Height)
	bm.PrevMoves = bm.Moves
	bm.Moves = make(map[hlt.Location]hlt.Move)

	// init owned rows to true
	bm.OwnedRows = make([]bool, gameMap.Height)
	for r := 0; r < gameMap.Height; r++ {
		bm.OwnedRows[r] = true
	}

	// init owned cols to true
	bm.OwnedCols = make([]bool, gameMap.Width)
	for c := 0; c < gameMap.Width; c++ {
		bm.OwnedCols[c] = true
	}

	ownedBorders := NewStack()
	enemyBorders := NewStack()
	for y := 0; y < gameMap.Height; y++ {
		bm.Cells[y] = make([]*Cell, gameMap.Width)
		for x := 0; x < gameMap.Width; x++ {
			var cell = NewCell(gameMap, x, y)
			if cell.Site.Owner != bm.Owner {
				if cell.Site.Owner != 0 && cell.isBorder() {
					// Track enemby
					cell.ThreatDistance = 0
					cell.ThreatOwner = cell.Site.Owner
					cell.ThreatStrength = cell.Site.Strength
					enemyBorders.Push(cell)
				}
				bm.OwnedRows[y] = false
				bm.OwnedCols[x] = false
			} else if cell.isBorder() {
				cell.BorderDistance = 0
				ownedBorders.Push(cell)
			}
			bm.SetCell(cell)
		}
	}

	// produce nearest border flow field
	for ownedBorders.isNotEmpty() {
		cell, _ := ownedBorders.Pop()
		for _, direction := range hlt.CARDINALS {
			otherCell := bm.GetCell(cell.Location, direction)
			if otherCell.Site.Owner == bm.Owner {
				if otherCell.BorderDistance > cell.BorderDistance+1 {
					otherCell.BorderDirection = opposite(direction)
					otherCell.BorderDistance = cell.BorderDistance + 1
					bm.SetCell(otherCell)
					ownedBorders.Push(otherCell)
				}
			}
		}
	}

	// produce strongest threat flow field
	for enemyBorders.isNotEmpty() {
		var cell, _ = enemyBorders.Pop()
		for _, direction := range hlt.CARDINALS {
			otherCell := bm.GetCell(cell.Location, direction)
			if otherCell.Site.Owner != cell.ThreatOwner {
				var newThreatStrength = cell.ThreatStrength - otherCell.Site.Strength
				if newThreatStrength > 0 && newThreatStrength > otherCell.ThreatStrength {
					otherCell.ThreatDirection = opposite(direction)
					otherCell.ThreatDistance = cell.ThreatDistance + 1
					otherCell.ThreatOwner = cell.ThreatOwner
					otherCell.ThreatStrength = newThreatStrength
					bm.SetCell(otherCell)
					enemyBorders.Push(otherCell)
				}
			}
		}
	}

	bm.logMap()

	// Threat that overlaps our borders produce Assistence Requesting Agents
	// and Assitence Lending Agents.
	// Call for help radiates out, Agents with required strength answer call
	// and begin moving to assist.  Agents die as call is answered. :D
}

func (bm *BotMap) logMap() {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var mapStr = fmt.Sprintf("Map %d:\n", bm.Turn)
	for y := 0; y < bm.GameMap.Height; y++ {
		var str = ""
		for x := 0; x < bm.GameMap.Width; x++ {
			var cell = bm.Cells[y][x]
			if cell.isBorder() && cell.Site.Owner > 0 {
				str = fmt.Sprintf("%v %d ", str, cell.Site.Owner)
			} else if cell.Site.Owner > 0 && cell.Site.Owner != cell.ThreatOwner {
				str = fmt.Sprintf("%v : ", str)
			} else if cell.Site.Owner > 0 {
				str = fmt.Sprintf("%v . ", str)
			} else if cell.ThreatStrength > 0 {
				var dirStr string
				switch opposite(cell.ThreatDirection) {
				case hlt.NORTH:
					dirStr = "^"
				case hlt.SOUTH:
					dirStr = "v"
				case hlt.WEST:
					dirStr = "<"
				case hlt.EAST:
					dirStr = ">"
				default:
					dirStr = "x"
				}
				str = fmt.Sprintf("%v %v ", str, dirStr)
			} else {
				str = fmt.Sprintf("%v   ", str)
			}
		}
		mapStr = fmt.Sprintf("%v%v\n", mapStr, str)
	}
	if _, err = f.WriteString(mapStr); err != nil {
		panic(err)
	}
}

func (bm *BotMap) GetCell(loc hlt.Location, direction hlt.Direction) *Cell {
	loc = bm.GameMap.GetLocation(loc, direction)
	return bm.Cells[loc.Y][loc.X]
}

func (bm *BotMap) SetCell(cell *Cell) {
	bm.Cells[cell.Y][cell.X] = cell
}

type Cell struct {
	GameMap hlt.GameMap
	// Provided Cell data
	Y        int
	X        int
	Location hlt.Location
	Site     hlt.Site
	// For cells belonging to the Owner, flow field to the border
	BorderDirection hlt.Direction
	BorderDistance  int
	// Field of threat radiating from edge of enemy borders
	ThreatDirection hlt.Direction
	ThreatDistance  int
	ThreatOwner     int
	ThreatStrength  int
}

func NewCell(gameMap hlt.GameMap, x int, y int) *Cell {
	location := hlt.NewLocation(x, y)
	site := gameMap.GetSite(location, hlt.STILL)
	return &Cell{
		GameMap:         gameMap,
		Y:               y,
		X:               x,
		Location:        location,
		Site:            site,
		BorderDirection: hlt.STILL,
		BorderDistance:  999999,
		ThreatDirection: hlt.STILL,
		ThreatDistance:  999999,
		ThreatOwner:     site.Owner,
		ThreatStrength:  0,
	}
}

func (c *Cell) GetLocation(direction hlt.Direction) hlt.Location {
	return c.GameMap.GetLocation(hlt.NewLocation(c.X, c.Y), direction)
}

func (c *Cell) GetSite(direction hlt.Direction) hlt.Site {
	return c.GameMap.GetSite(hlt.NewLocation(c.X, c.Y), direction)
}

func (c *Cell) isBorder() bool {
	var site = c.GetSite(hlt.STILL)
	for _, direction := range hlt.CARDINALS {
		var otherSite = c.GetSite(direction)
		if otherSite.Owner != site.Owner {
			return true
		}
	}
	return false
}

func (c *Cell) String() string {
	return fmt.Sprintf("Cell(x: %d, y: %d, dist:%d)", c.X, c.Y, c.BorderDistance)
}

// [ ][ ][ ][ ][ ]
// [ ][ ][N][ ][ ]
// [ ][W][x][E][ ]
// [ ][ ][S][ ][ ]
// [ ][ ][ ][ ][ ]
type Need struct {
	dir hlt.Direction
	str int
}

func NewNeed(dir hlt.Direction, str int) Need {
	return Need{
		dir: dir,
		str: str,
	}
}

func main() {
	conn, gameMap := hlt.NewConnection("BrevBot")
	botMap := NewBotMap(conn.PlayerTag, gameMap)
	for {
		var moves hlt.MoveSet
		gameMap = conn.GetFrame()

		startTime := time.Now()
		botMap.updateMap(gameMap)
		for y := 0; y < gameMap.Height; y++ {
			for x := 0; x < gameMap.Width; x++ {
				loc := hlt.NewLocation(x, y)
				if gameMap.GetSite(loc, hlt.STILL).Owner == conn.PlayerTag {
					var dir = hlt.Direction(rand.Int() % 5)
					moves = append(moves, hlt.Move{
						Location:  loc,
						Direction: dir,
					})
				}
			}
		}
		log(fmt.Sprintf("Time: %v\n", time.Now().Sub(startTime)))

		conn.SendFrame(moves)
	}
}
