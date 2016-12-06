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

type Stack struct {
	lock sync.Mutex // you don't have to do this if you don't want thread safety
	s    []*Cell
}

func NewStack() *Stack {
	return &Stack{sync.Mutex{}, make([]*Cell, 0)}
}

func (s *Stack) Push(v *Cell) {
	// s.lock.Lock()
	// defer s.lock.Unlock()

	s.s = append(s.s, v)
}

func (s *Stack) Pop() (*Cell, error) {
	// s.lock.Lock()
	// defer s.lock.Unlock()

	l := len(s.s)
	if l == 0 {
		return nil, errors.New("Empty Stack")
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res, nil
}

func (s *Stack) length() int {
	return len(s.s)
}

func (s *Stack) isEmpty() bool {
	return len(s.s) == 0
}

func (s *Stack) isNotEmpty() bool {
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

type Fields struct {
	Border     *FlowField
	Initiative *FlowField
	Threat     map[int]*FlowField
}

// Assumes Bot Cell lists have been populated
func NewFields(bot *Bot) *Fields {
	var f = &Fields{
		Threat: make(map[int]*FlowField),
	}
	f.Border = NewBorderField(bot.Owner, bot.OwnedCells, bot)
	f.Initiative = NewInitiativeField(bot.Owner, bot.OwnedCells, bot)
	for enemy, cells := range bot.EnemyCells {
		f.Threat[enemy] = NewInitiativeField(enemy, cells, bot)
	}
	return f
}

func (f *Fields) TopThreat(cell *Cell) *FlowField {
	var topField *FlowField
	for _, field := range f.Threat {
		if topField == nil || field.Strength[cell] > topField.Strength[cell] {
			topField = field
		}
	}
	return topField
}

func (f *Fields) String(cell *Cell) string {
	if direction, ok := f.Border.Direction[cell]; ok {
		if direction == hlt.STILL {
			return fmt.Sprint(cell.Site.Owner)
		} else {

		}
	}
	return " "
}

type FlowField struct {
	Direction map[*Cell]hlt.Direction
	Distance  map[*Cell]int
	Strength  map[*Cell]int
}

func NewFlowField(owner int) *FlowField {
	return &FlowField{
		Direction: make(map[*Cell]hlt.Direction),
		Distance:  make(map[*Cell]int),
		Strength:  make(map[*Cell]int),
	}
}

// Direction to nearest owned border cell
// Distance to nearest owned border cell
// Strength of nearest owned border cell
func NewBorderField(owner int, cells []*Cell, bot *Bot) *FlowField {
	var field = NewFlowField(owner)
	var stack = NewStack()
	// push border cells onto the stack
	for _, cell := range cells {
		field.Direction[cell] = hlt.STILL
		if cell.isBorder() {
			field.Distance[cell] = 0
			stack.Push(cell)
		} else {
			field.Distance[cell] = 999999
		}
	}
	// produce nearest border flow field by working inward from border
	for stack.isNotEmpty() {
		cell, _ := stack.Pop()
		var borderDistance = field.Distance[cell]
		for _, direction := range hlt.CARDINALS {
			otherCell := bot.GetCell(cell.Location, direction)
			if otherCell.Site.Owner == owner {
				if otherDist, ok := field.Distance[cell]; (ok && otherDist > borderDistance+1) || !ok {
					field.Direction[otherCell] = opposite(direction)
					field.Distance[otherCell] = borderDistance + 1
					stack.Push(otherCell)
				}
			}
		}
	}
	return field
}

// Direction of strongest threat
// Distance to threat source
// Strength of threat if it were to travel to this cell without reinforcements
func NewInitiativeField(owner int, cells []*Cell, bot *Bot) *FlowField {
	var field = NewFlowField(owner)
	var stack = NewStack()
	// push border cells onto the stack
	for _, cell := range cells {
		field.Direction[cell] = hlt.STILL
		if cell.isBorder() {
			field.Distance[cell] = 0
			field.Strength[cell] = cell.Site.Strength
			stack.Push(cell)
		}
	}
	// produce strongest threat flow field
	for stack.isNotEmpty() {
		cell, _ := stack.Pop()
		// var borderDistance = field.Distance[cell]
		for _, direction := range hlt.CARDINALS {
			otherCell := bot.GetCell(cell.Location, direction)
			if otherCell.Site.Owner != owner {
				var remainingStrength = field.Strength[cell] - otherCell.Site.Strength
				if remainingStrength > 0 {
					previousStrength, ok := field.Strength[otherCell]
					if !ok || remainingStrength > previousStrength {
						field.Direction[otherCell] = opposite(direction)
						field.Distance[otherCell] = field.Distance[cell] + 1
						field.Strength[otherCell] = remainingStrength
						stack.Push(otherCell)
					}
				}
			}
		}
	}
	return field
}

type Agent struct {
}

type Bot struct {
	Owner      int
	Agents     map[int]*Agent
	GameMap    *hlt.GameMap
	Turn       int
	Fields     *Fields
	Cells      [][]*Cell
	OwnedCells []*Cell
	EmptyCells []*Cell
	EnemyCells map[int][]*Cell
}

func NewBot(owner int) *Bot {
	return &Bot{
		Owner:  owner,
		Agents: make(map[int]*Agent),
		Turn:   0,
	}
}

func (bot *Bot) UpdateMap(gameMap *hlt.GameMap) {
	bot.GameMap = gameMap
	bot.Turn = bot.Turn + 1
	bot.Cells = make([][]*Cell, gameMap.Height)
	bot.OwnedCells = make([]*Cell, 1)
	bot.EmptyCells = make([]*Cell, 1)
	bot.EnemyCells = make(map[int][]*Cell)

	// Sorts Map Cells into "owned", "empty", and "enemy" lists
	for y := 0; y < gameMap.Height; y++ {
		bot.Cells[y] = make([]*Cell, gameMap.Width)
		for x := 0; x < gameMap.Width; x++ {
			var cell = NewCell(*gameMap, x, y)
			bot.SetCell(cell)
		}
	}

	bot.Fields = NewFields(bot)

	bot.logBot()

	// Threat that overlaps our borders produce Assistence Requesting Agents
	// and Assitence Lending Agents.
	// Call for help radiates out, Agents with required strength answer call
	// and begin moving to assist.  Agents die as call is answered. :D
}

func (bot *Bot) Height() int {
	return bot.GameMap.Height
}

func (bot *Bot) Width() int {
	return bot.GameMap.Width
}

func (bot *Bot) GetCell(loc hlt.Location, direction hlt.Direction) *Cell {
	loc = bot.GameMap.GetLocation(loc, direction)
	return bot.Cells[loc.Y][loc.X]
}

func (bot *Bot) SetCell(cell *Cell) {
	bot.Cells[cell.Y][cell.X] = cell
	var cellOwner = cell.Site.Owner
	if cellOwner > 0 {
		if cellOwner == bot.Owner {
			bot.OwnedCells = append(bot.OwnedCells, cell)
		} else {
			bot.EnemyCells[cellOwner] = append(bot.EnemyCells[cellOwner], cell)
		}
	} else {
		bot.EmptyCells = append(bot.EmptyCells, cell)
	}
}

func (bot *Bot) logBot() {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var mapStr = fmt.Sprintf("Map %d:\n", bot.Turn)
	for y := 0; y < bot.GameMap.Height; y++ {
		var str = ""
		for x := 0; x < bot.GameMap.Width; x++ {
			var cell = bot.Cells[y][x]
			if cell.isBorder() && cell.Site.Owner > 0 {
				str = fmt.Sprintf("%v %d ", str, cell.Site.Owner)
				// } else if cell.Site.Owner > 0 && cell.Site.Owner != cell.ThreatOwner {
				// 	str = fmt.Sprintf("%v : ", str)
			} else if cell.Site.Owner > 0 {
				str = fmt.Sprintf("%v . ", str)
				// } else if cell.ThreatStrength > 0 {
				// 	var dirStr string
				// 	switch opposite(cell.ThreatDirection) {
				// 	case hlt.NORTH:
				// 		dirStr = "^"
				// 	case hlt.SOUTH:
				// 		dirStr = "v"
				// 	case hlt.WEST:
				// 		dirStr = "<"
				// 	case hlt.EAST:
				// 		dirStr = ">"
				// 	default:
				// 		dirStr = "x"
				// 	}
				// 	str = fmt.Sprintf("%v %v ", str, dirStr)
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

type Cell struct {
	GameMap hlt.GameMap
	// Provided Cell data
	Y        int
	X        int
	Location hlt.Location
	Site     hlt.Site
}

func NewCell(gameMap hlt.GameMap, x int, y int) *Cell {
	location := hlt.NewLocation(x, y)
	site := gameMap.GetSite(location, hlt.STILL)
	return &Cell{
		GameMap:  gameMap,
		Y:        y,
		X:        x,
		Location: location,
		Site:     site,
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
	return fmt.Sprintf("Cell(x: %d, y: %d, owner:%d)", c.X, c.Y, c.Site.Owner)
}

func main() {
	conn, gameMap := hlt.NewConnection("BrevBot")
	bot := NewBot(conn.PlayerTag)
	bot.UpdateMap(&gameMap)
	for {
		var moves hlt.MoveSet
		gameMap = conn.GetFrame()

		startTime := time.Now()
		bot.UpdateMap(&gameMap)
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
