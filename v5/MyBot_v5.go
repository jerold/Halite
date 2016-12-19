package main

import (
	"errors"
	"fmt"
	"hlt"
	"os"
	"sync"
)

const logFile = "goLog.txt"

/*
███████ ████████  █████   ██████ ██   ██
██         ██    ██   ██ ██      ██  ██
███████    ██    ███████ ██      █████
     ██    ██    ██   ██ ██      ██  ██
███████    ██    ██   ██  ██████ ██   ██
*/

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

/*
███████ ██ ███████ ██      ██████  ███████
██      ██ ██      ██      ██   ██ ██
█████   ██ █████   ██      ██   ██ ███████
██      ██ ██      ██      ██   ██      ██
██      ██ ███████ ███████ ██████  ███████
*/

type Fields struct {
	Border     *FlowField
	Initiative *FlowField
	Support    *FlowField
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
	f.Support = NewSupportField(bot.Owner, bot.OwnedCells, bot)
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
	if cell.Site.Owner > 0 && f.Support.Strength[cell] > 0 {
		switch f.Support.Direction[cell] {
		case hlt.NORTH:
			return "^"
		case hlt.SOUTH:
			return "v"
		case hlt.WEST:
			return "<"
		case hlt.EAST:
			return ">"
		default:
			return "x"
		}
	} else {
		return " "
	}

	// if cell.Site.Owner > 0 {
	// 	if direction, ok := f.Border.Direction[cell]; ok {
	// 		switch direction {
	// 		case hlt.NORTH:
	// 			return "^"
	// 		case hlt.SOUTH:
	// 			return "v"
	// 		case hlt.WEST:
	// 			return "<"
	// 		case hlt.EAST:
	// 			return ">"
	// 		default:
	// 			return "x"
	// 		}
	// 	}
	// 	var topThreat = f.TopThreat(cell)
	// 	if strength, ok := topThreat.Strength[cell]; ok && strength > 0 {
	// 		return ":"
	// 	} else {
	// 		return "."
	// 	}
	// }
	// var topThreat = f.TopThreat(cell)
	// if direction, ok := topThreat.Direction[cell]; ok {
	// 	switch opposite(direction) {
	// 	case hlt.NORTH:
	// 		return "^"
	// 	case hlt.SOUTH:
	// 		return "v"
	// 	case hlt.WEST:
	// 		return "<"
	// 	case hlt.EAST:
	// 		return ">"
	// 	default:
	// 		return "x"
	// 	}
	// }
	// return " "
}

/*
███████ ██       ██████  ██     ██ ███████ ██ ███████ ██      ██████
██      ██      ██    ██ ██     ██ ██      ██ ██      ██      ██   ██
█████   ██      ██    ██ ██  █  ██ █████   ██ █████   ██      ██   ██
██      ██      ██    ██ ██ ███ ██ ██      ██ ██      ██      ██   ██
██      ███████  ██████   ███ ███  ██      ██ ███████ ███████ ██████
*/

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

/*
██████   ██████  ██████  ██████  ███████ ██████      ███████ ██ ███████ ██      ██████
██   ██ ██    ██ ██   ██ ██   ██ ██      ██   ██     ██      ██ ██      ██      ██   ██
██████  ██    ██ ██████  ██   ██ █████   ██████      █████   ██ █████   ██      ██   ██
██   ██ ██    ██ ██   ██ ██   ██ ██      ██   ██     ██      ██ ██      ██      ██   ██
██████   ██████  ██   ██ ██████  ███████ ██   ██     ██      ██ ███████ ███████ ██████
*/

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
				if field.Distance[otherCell] > borderDistance+1 {
					field.Direction[otherCell] = opposite(direction)
					field.Distance[otherCell] = borderDistance + 1
					stack.Push(otherCell)
				}
			}
		}
	}
	return field
}

/*
██ ███    ██ ██ ████████ ██  █████  ████████ ██ ██    ██ ███████     ███████ ██ ███████ ██      ██████
██ ████   ██ ██    ██    ██ ██   ██    ██    ██ ██    ██ ██          ██      ██ ██      ██      ██   ██
██ ██ ██  ██ ██    ██    ██ ███████    ██    ██ ██    ██ █████       █████   ██ █████   ██      ██   ██
██ ██  ██ ██ ██    ██    ██ ██   ██    ██    ██  ██  ██  ██          ██      ██ ██      ██      ██   ██
██ ██   ████ ██    ██    ██ ██   ██    ██    ██   ████   ███████     ██      ██ ███████ ███████ ██████
*/

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
		for _, direction := range hlt.CARDINALS {
			otherCell := bot.GetCell(cell.Location, direction)
			if otherCell.Site.Owner != owner {
				var otherCellStrength = otherCell.Site.Strength
				if otherCell.Site.Owner > 0 {
					otherCellStrength = otherCellStrength + otherCell.Site.Production*(field.Distance[cell]+1)
				}
				var remainingStrength = field.Strength[cell] - otherCellStrength
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

/*
███████ ██    ██ ██████  ██████   ██████  ██████  ████████     ███████ ██ ███████ ██      ██████
██      ██    ██ ██   ██ ██   ██ ██    ██ ██   ██    ██        ██      ██ ██      ██      ██   ██
███████ ██    ██ ██████  ██████  ██    ██ ██████     ██        █████   ██ █████   ██      ██   ██
     ██ ██    ██ ██      ██      ██    ██ ██   ██    ██        ██      ██ ██      ██      ██   ██
███████  ██████  ██      ██       ██████  ██   ██    ██        ██      ██ ███████ ███████ ██████
*/

// Represents a need. Greater needs win
func NewSupportField(owner int, cells []*Cell, bot *Bot) *FlowField {
	var field = NewFlowField(owner)
	var stack = NewStack()
	// push border cells onto the stack
	for _, cell := range cells {
		field.Direction[cell] = hlt.STILL
		field.Distance[cell] = 999999
		field.Strength[cell] = 0
		if cell.isBorder() {
			var target *Cell
			var targetDirection = hlt.STILL
			var targetHeuristic = 0.0
			for _, direction := range hlt.CARDINALS {
				var otherCell = bot.GetCell(cell.Location, direction)
				if otherCell.Site.Owner != owner {
					var otherHeuristic = otherCell.Heuristic(owner)
					if target == nil || otherHeuristic > targetHeuristic {
						target = otherCell
						targetDirection = direction
						targetHeuristic = otherHeuristic
					}
				}
			}
			if targetDirection != hlt.STILL {
				field.Direction[cell] = targetDirection
				field.Distance[cell] = 0
				field.Strength[cell] = target.TotalDamage(owner) - cell.Site.Strength
				if field.Strength[cell] < 0 {
					field.Strength[cell] = 0
				}
				stack.Push(cell)
			}
		}
	}
	// produce strongest threat flow field
	for stack.isNotEmpty() {
		cell, _ := stack.Pop()
		for _, direction := range hlt.CARDINALS {
			otherCell := bot.GetCell(cell.Location, direction)
			if otherCell.Site.Owner == owner {
				if field.Strength[cell] > field.Strength[otherCell] {
					field.Direction[otherCell] = opposite(direction)
					field.Distance[otherCell] = field.Distance[cell] + 1
					field.Strength[otherCell] = field.Strength[cell] - otherCell.Site.Strength
					if field.Strength[otherCell] <= 0 {
						field.Strength[otherCell] = 0
					} else {
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

/*
██████   ██████  ████████
██   ██ ██    ██    ██
██████  ██    ██    ██
██   ██ ██    ██    ██
██████   ██████     ██
*/

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

func (bot *Bot) UpdateMap(gameMap hlt.GameMap) {
	bot.GameMap = &gameMap
	bot.Turn = bot.Turn + 1
	bot.Cells = make([][]*Cell, bot.Height())
	bot.OwnedCells = make([]*Cell, 0)
	bot.EmptyCells = make([]*Cell, 0)
	bot.EnemyCells = make(map[int][]*Cell)

	// Sorts Map Cells into "owned", "empty", and "enemy" lists
	for y := 0; y < bot.Height(); y++ {
		bot.Cells[y] = make([]*Cell, bot.Width())
		for x := 0; x < bot.Width(); x++ {
			var cell = NewCell(&gameMap, x, y)
			bot.SetCell(cell)
		}
	}

	bot.Fields = NewFields(bot)

	// bot.logBot()

	// Threat that overlaps our borders produce Assistence Requesting Agents
	// and Assitence Lending Agents.
	// Call for help radiates out, Agents with required strength answer call
	// and begin moving to assist.  Agents die as call is answered. :D
}

func (bot *Bot) Moves() hlt.MoveSet {
	var moves = hlt.MoveSet{}
	for _, ownedCell := range bot.OwnedCells {
		if ownedCell.isBorder() {
			var targetDirection = hlt.STILL
			var targetCell *Cell
			var targetHeuristic = 0.0
			for _, direction := range hlt.CARDINALS {
				var otherCell = bot.GetCell(ownedCell.Location, direction)
				if otherCell.Site.Owner != bot.Owner {
					var otherHeuristic = otherCell.Heuristic(bot.Owner)
					if targetCell == nil || otherHeuristic > targetHeuristic {
						targetCell = otherCell
						targetDirection = direction
						targetHeuristic = otherHeuristic
					}
				}
			}
			if targetCell != nil && targetCell.Site.Strength < ownedCell.Site.Strength {
				moves = append(moves, hlt.Move{Location: ownedCell.Location, Direction: targetDirection})
			}
			// var direction = bot.DirectionFromProjection(ownedCell)
			// moves = append(moves, hlt.Move{Location: ownedCell.Location, Direction: direction})
		} else if ownedCell.Site.Strength > ownedCell.Site.Production*5 {
			if bot.Fields.Support.Distance[ownedCell] < 4 && bot.Fields.Support.Strength[ownedCell] > 0 {
				moves = append(moves, hlt.Move{Location: ownedCell.Location, Direction: bot.Fields.Support.Direction[ownedCell]})
			} else {
				moves = append(moves, hlt.Move{Location: ownedCell.Location, Direction: bot.Fields.Border.Direction[ownedCell]})
			}
			// var direction = bot.Fields.Border.Direction[ownedCell]
			// moves = append(moves, hlt.Move{Location: ownedCell.Location, Direction: direction})
		} else {
			moves = append(moves, hlt.Move{Location: ownedCell.Location, Direction: hlt.STILL})
		}
		// project game state forward for border pieces, score sample histories
		// and pick best option. Projection should turn assume enemy cells Play
		// the best move as well.
	}
	return moves
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

func (bot *Bot) DirectionFromProjection(cell *Cell) hlt.Direction {
	return InitialProjection(cell, bot).BestDirection()
}

func (bot *Bot) logBot() {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var strArr = make([][]string, bot.GameMap.Height)
	for y := 0; y < bot.GameMap.Height; y++ {
		strArr[y] = make([]string, bot.GameMap.Width)
		for x := 0; x < bot.GameMap.Width; x++ {
			var cell = bot.Cells[y][x]
			strArr[y][x] = bot.Fields.String(cell)
		}
	}

	var mapStr = fmt.Sprintf("Map %d:\n", bot.Turn)
	for y := 0; y < bot.GameMap.Height; y++ {
		var str = ""
		for x := 0; x < bot.GameMap.Width; x++ {
			str = fmt.Sprintf("%v %v  ", str, strArr[y][x])
		}
		mapStr = fmt.Sprintf("%v%v\n", mapStr, str)
	}
	if _, err = f.WriteString(mapStr); err != nil {
		panic(err)
	}
}

/*
██████  ██████   ██████       ██ ███████  ██████ ████████ ██  ██████  ███    ██
██   ██ ██   ██ ██    ██      ██ ██      ██         ██    ██ ██    ██ ████   ██
██████  ██████  ██    ██      ██ █████   ██         ██    ██ ██    ██ ██ ██  ██
██      ██   ██ ██    ██ ██   ██ ██      ██         ██    ██ ██    ██ ██  ██ ██
██      ██   ██  ██████   █████  ███████  ██████    ██    ██  ██████  ██   ████
*/

type Projection struct {
	Owner    int
	Bot      *Bot
	Depth    int
	Cell     *Cell
	Strength int
	Source   *Projection
	Score    int
}

func InitialProjection(cell *Cell, bot *Bot) *Projection {
	return &Projection{
		Owner:    bot.Owner,
		Bot:      bot,
		Depth:    0,
		Cell:     cell,
		Strength: cell.Site.Strength,
		Score:    0,
	}
}

func NewProjection(direction hlt.Direction, source *Projection, bot *Bot) *Projection {
	var cell = bot.GetCell(source.Cell.Location, direction)
	var strength = source.Strength
	var score = 0
	if cell.Site.Owner != source.Owner {
		strength = strength - cell.Site.Strength
		if strength > 0 && !source.InHistory(cell) {
			score = cell.Site.Production
		}
	}
	var p = &Projection{
		Owner:    source.Owner,
		Bot:      bot,
		Depth:    source.Depth + 1,
		Cell:     cell,
		Strength: strength,
		Source:   source,
		Score:    score,
	}
	return p
}

func (p *Projection) BestDirection() hlt.Direction {
	var bestDirection = hlt.STILL
	var highestScore = 0
	for _, direction := range hlt.Directions {
		var score = NewProjection(direction, p, p.Bot).GetScore()
		if score > highestScore {
			bestDirection = direction
			highestScore = score
		}
	}
	return bestDirection
}

func (p *Projection) GetScore() int {
	if p.Depth == 5 {
		return p.Score
	}
	return p.Score + NewProjection(p.BestDirection(), p, p.Bot).GetScore()
}

func (p *Projection) InHistory(cell *Cell) bool {
	if cell.Location.X == p.Cell.Location.X && cell.Location.Y == p.Cell.Location.Y {
		return true
	} else if p.Depth > 0 {
		return p.Source.InHistory(cell)
	}
	return false
}

/*
 ██████ ███████ ██      ██
██      ██      ██      ██
██      █████   ██      ██
██      ██      ██      ██
 ██████ ███████ ███████ ███████
*/

type Cell struct {
	GameMap  *hlt.GameMap
	Y        int
	X        int
	Location hlt.Location
	Site     hlt.Site
}

func NewCell(gameMap *hlt.GameMap, x int, y int) *Cell {
	location := hlt.NewLocation(x%gameMap.Width, y%gameMap.Height)
	site := gameMap.GetSite(location, hlt.STILL)
	return &Cell{
		GameMap:  gameMap,
		Y:        y,
		X:        x,
		Location: location,
		Site:     site,
	}
}

func (c *Cell) Heuristic(owner int) float64 {
	if c.Site.Owner == 0 && c.Site.Strength > 0 {
		return float64(c.Site.Production) / float64(c.Site.Strength)
	} else {
		return float64(c.TotalDamage(owner))
	}
}

func (c *Cell) TotalDamage(owner int) int {
	totalDamage := c.Site.Strength
	for _, direction := range hlt.CARDINALS {
		site := c.GetSite(direction)
		if site.Owner != 0 && site.Owner != owner {
			totalDamage += site.Strength
		}
	}
	return totalDamage
}

func (c *Cell) GetLocation(direction hlt.Direction) hlt.Location {
	return c.GameMap.GetLocation(c.Location, direction)
}

func (c *Cell) GetSite(direction hlt.Direction) hlt.Site {
	return c.GameMap.GetSite(c.Location, direction)
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

/*
███    ███  █████  ██ ███    ██
████  ████ ██   ██ ██ ████   ██
██ ████ ██ ███████ ██ ██ ██  ██
██  ██  ██ ██   ██ ██ ██  ██ ██
██      ██ ██   ██ ██ ██   ████
*/

func main() {
	conn, gameMap := hlt.NewConnection()
	bot := NewBot(conn.PlayerTag)
	bot.UpdateMap(gameMap)
	conn.SendName("v5")
	for {
		var moves hlt.MoveSet
		gameMap = conn.GetFrame()

		// startTime := time.Now()
		bot.UpdateMap(gameMap)
		moves = bot.Moves()
		// log(fmt.Sprintf("Time: %v\n", time.Now().Sub(startTime)))

		conn.SendFrame(moves)
	}
}
