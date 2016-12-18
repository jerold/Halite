package main

import (
	"bytes"
	"errors"
	"fmt"
	"hlt"
	"os"
	"time"
)

const logFile = "log.txt"
const NEUTRAL = 0

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

/*
███████ ████████  █████   ██████ ██   ██
██         ██    ██   ██ ██      ██  ██
███████    ██    ███████ ██      █████
     ██    ██    ██   ██ ██      ██  ██
███████    ██    ██   ██  ██████ ██   ██
*/

type Stackable struct {
	Content  *Cell
	Previous *Stackable
}

type Stack struct {
	_top    *Stackable
	_length int
}

func NewStack() *Stack {
	return &Stack{_length: 0}
}

func (s *Stack) Peek() (*Cell, error) {
	if s.isEmpty() {
		return nil, errors.New("Empty Stack")
	}
	return s._top.Content, nil
}

func (s *Stack) Push(c *Cell) {
	newTop := &Stackable{Content: c, Previous: s._top}
	s._top = newTop
	s._length++
}

func (s *Stack) Pop() (*Cell, error) {
	if s.isEmpty() {
		return nil, errors.New("Empty Stack")
	}
	popStackable := s._top
	cell := popStackable.Content
	s._top = popStackable.Previous
	popStackable.Previous = nil
	s._length--
	return cell, nil
}

func (s *Stack) isEmpty() bool {
	return s._length == 0
}

func (s *Stack) isNotEmpty() bool {
	return !s.isEmpty()
}

func log(a ...interface{}) {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	str := fmt.Sprintln(a)
	if _, err = f.WriteString(str); err != nil {
		panic(err)
	}
	f.Sync()
	f.Close()
}

// Test function used to filter or include cells that meet described criteria
type CellTest func(*Cell) bool

type PointOfInterest struct {
	X, Y  int
	Field map[int]map[int]int
}

// map of strength between destination and each cell in cells
func NewPOI(destination *Cell, cells *Cells) *PointOfInterest {
	field := make(map[int]map[int]int)
	stack := NewStack()
	for y := cells.Y; y < cells.Y+cells.Height; y++ {
		yf := y % cells._sourceHeight
		field[yf] = make(map[int]int)
		for x := cells.X; x < cells.X+cells.Width; x++ {
			xf := x % cells._sourceWidth
			field[yf][xf] = 999999
		}
	}
	field[destination.Y][destination.X] = destination.Strength
	stack.Push(destination)
	for stack.isNotEmpty() {
		if cell, err := stack.Pop(); err == nil {
			cost := field[cell.Y][cell.X]
			for _, neighbor := range cell.Neighbors() {
				// cost is strength required to take the cell + 1 for turn cost.
				if field[neighbor.Y][neighbor.X] > cost+neighbor.Strength+1 {
					field[neighbor.Y][neighbor.X] = cost + neighbor.Strength + 1
					stack.Push(neighbor)
				}
			}
		}
	}
	return &PointOfInterest{
		X:     destination.X,
		Y:     destination.Y,
		Field: field,
	}
}

/*
 █████   ██████  ███████ ███    ██ ████████
██   ██ ██       ██      ████   ██    ██
███████ ██   ███ █████   ██ ██  ██    ██
██   ██ ██    ██ ██      ██  ██ ██    ██
██   ██  ██████  ███████ ██   ████    ██
*/

type AgentState int

const (
	Assisting    AgentState = iota // recruited to assist another Agent
	Attacking                      // on Boundry, advancing into enemy territory
	Expanding                      // on Boundry, tasked with capturing new territory
	Farming                        // waiting to harvest strength from this site
	Initializing                   // new Agent hasn't decided what to do yet
	Transporting                   // within Body, moving strength to edge
)

type DestinationType int

const (
	Enemy DestinationType = iota
	Friendly
	Plunder
)

type Agent struct {
	Bot      Bot        // Get Owner and other Agents
	State    AgentState // What has the Agent done with its life
	LastMove hlt.Move
	Location hlt.Location
}

func NewAgent(bot Bot, location hlt.Location) *Agent {
	return &Agent{
		Bot:      bot,
		State:    Initializing,
		LastMove: hlt.Move{Location: location, Direction: hlt.STILL},
		Location: location,
	}
}

// called at beginning of a new turn. Update current location based on last turn.
func (a *Agent) Update() hlt.Location {
	a.Location = a.GetCells().GetLocation(a.LastMove.Location, a.LastMove.Direction)
	return a.Location
}

func (a *Agent) GetAgents() map[hlt.Location]*Agent {
	return a.Bot.Agents
}

func (a *Agent) GetOwner() int {
	return a.Bot.Owner
}

func (a *Agent) GetCell() *Cell {
	return a.GetCells().GetCell(a.Location, hlt.STILL)
}

func (a *Agent) GetCells() *Cells {
	return a.Bot.Cells
}

func (a *Agent) GetMove() hlt.Move {
	nextMove := hlt.Move{
		Location:  a.Location,
		Direction: hlt.STILL,
	}
	a.LastMove = nextMove
	return nextMove
}

/*
██████   ██████  ████████
██   ██ ██    ██    ██
██████  ██    ██    ██
██   ██ ██    ██    ██
██████   ██████     ██
*/

type Bot struct {
	Agents           map[hlt.Location]*Agent
	Owner            int
	Cells            *Cells
	PointsOfInterest []*PointOfInterest
}

func NewBot(owner int, gameMap hlt.GameMap) *Bot {
	bot := &Bot{
		Agents:           make(map[hlt.Location]*Agent),
		Owner:            owner,
		Cells:            NewCells(0, 0, gameMap.Width, gameMap.Height, gameMap),
		PointsOfInterest: make([]*PointOfInterest, 0, 1),
	}
	for _, cell := range bot.Cells.GetCells(func(c *Cell) bool {
		return c.Production == bot.Cells.MaxProduction
	}) {
		bot.PointsOfInterest = append(bot.PointsOfInterest, NewPOI(cell, bot.Cells))
	}
	return bot
}

func (b *Bot) Update(gameMap hlt.GameMap) {
	b.Cells.Update(gameMap)
	// Update Current Agents
	newAgents := make(map[hlt.Location]*Agent)
	for _, agent := range b.Agents {
		updatedAgentLocation := agent.Update()
		newAgents[updatedAgentLocation] = agent
	}
	b.Agents = newAgents
}

func (b *Bot) Moves() hlt.MoveSet {
	var moves = hlt.MoveSet{}
	for _, agent := range b.Agents {
		moves = append(moves, agent.GetMove())
	}
	return moves
}

/*
 ██████ ███████ ██      ██      ███████
██      ██      ██      ██      ██
██      █████   ██      ██      ███████
██      ██      ██      ██           ██
 ██████ ███████ ███████ ███████ ███████
*/

type Cells struct {
	Contents map[int]map[int]*Cell
	Height   int
	Width    int
	X        int
	Y        int
	// Original map dimensions
	_sourceHeight int
	_sourceWidth  int
	// Ownership changes between turns
	ByOwner map[int][]*Cell
	// Production stats
	AvgProduction   int
	MaxProduction   int
	MinProduction   int
	TotalProduction map[int]int
	TotalStrength   map[int]int
	TotalTerritory  map[int]int
}

func NewCells(x int, y int, width int, height int, gameMap hlt.GameMap) *Cells {
	cells := &Cells{
		Height:          height,
		Width:           width,
		X:               x,
		Y:               y,
		_sourceHeight:   gameMap.Height,
		_sourceWidth:    gameMap.Width,
		ByOwner:         make(map[int][]*Cell),
		AvgProduction:   0,
		MaxProduction:   255,
		MinProduction:   0,
		TotalProduction: make(map[int]int),
		TotalStrength:   make(map[int]int),
		TotalTerritory:  make(map[int]int),
	}
	contents := make(map[int]map[int]*Cell)
	sum := 0
	for i := y; i < y+height; i++ {
		yf := i % cells._sourceHeight
		contents[yf] = make(map[int]*Cell)
		for j := x; j < x+width; j++ {
			xf := j % cells._sourceWidth
			site := gameMap.Contents[yf][xf]
			contents[yf][xf] = NewCell(cells, gameMap.Contents[yf][xf], xf, yf)
			cells.ByOwner[site.Owner] = append(cells.ByOwner[site.Owner], contents[yf][xf])
			cells.TotalProduction[site.Owner] += site.Strength
			cells.TotalStrength[site.Owner] += site.Strength
			cells.TotalTerritory[site.Owner]++
			// update Max
			if site.Production > cells.MaxProduction {
				cells.MaxProduction = site.Production
			}
			// update Min
			if site.Production < cells.MinProduction {
				cells.MinProduction = site.Production
			}
			// update Sum for Avg
			sum += site.Production
		}
	}
	cells.AvgProduction = sum / (width * height)
	cells.Contents = contents
	return cells
}

// Produce a copy of the Cells containing new copies of all contained cells
func (c *Cells) Clone() *Cells {
	clone := &Cells{
		Height:          c.Height,
		Width:           c.Width,
		X:               c.X,
		Y:               c.Y,
		_sourceHeight:   c._sourceHeight,
		_sourceWidth:    c._sourceWidth,
		ByOwner:         make(map[int][]*Cell),
		AvgProduction:   c.AvgProduction,
		MaxProduction:   c.MaxProduction,
		MinProduction:   c.MinProduction,
		TotalProduction: c.TotalProduction,
		TotalStrength:   c.TotalStrength,
		TotalTerritory:  c.TotalTerritory,
	}
	contents := make(map[int]map[int]*Cell)
	for y := c.Y; y < c.Y+c.Height; y++ {
		yf := y % c._sourceHeight
		contents[yf] = make(map[int]*Cell)
		for x := c.X; x < c.X+c.Width; x++ {
			xf := x % c._sourceWidth
			cell := c.Contents[yf][xf].Clone(clone)
			contents[yf][xf] = cell
			clone.ByOwner[cell.Owner] = append(clone.ByOwner[cell.Owner], cell)
		}
	}
	clone.Contents = contents
	return clone
}

// update Cells with Site data from provided GameMap
func (c *Cells) Update(gameMap hlt.GameMap) {
	c.TotalProduction = make(map[int]int)
	c.TotalStrength = make(map[int]int)
	c.TotalTerritory = make(map[int]int)
	c.ByOwner = make(map[int][]*Cell)
	// Cells my not be the full size of gameMap, only iterate Cells contents
	for y := c.Y; y < c.Y+c.Height; y++ {
		yf := y % c._sourceHeight
		for x := c.X; x < c.X+c.Width; x++ {
			xf := x % c._sourceWidth
			site := gameMap.Contents[yf][xf]
			c.TotalProduction[site.Owner] += site.Strength
			c.TotalStrength[site.Owner] += site.Strength
			c.TotalTerritory[site.Owner]++
			cell := c.Get(xf, yf)
			cell.Update(site)
			c.ByOwner[cell.Owner] = append(c.ByOwner[cell.Owner], cell)
		}
	}
}

// applies moves in the same way halite.io would... I think.
func (c *Cells) Simulate(moves hlt.MoveSet) *Cells {
	clone := c.Clone()
	// used to prevent production on cells that had movement this round
	conflictLocations := make(map[hlt.Location]bool)
	// forces which have moved or been recruited by moving forces,
	// and will attack destination + Cardinal opposing forces
	activeForces := make(map[hlt.Location]map[int]int)
	// forces moving into new cells
	for _, move := range moves {
		if move.Direction != hlt.STILL {
			fromCell := clone.Get(move.Location.X, move.Location.Y)
			toCell := clone.GetCell(move.Location, move.Direction)
			conflictLocations[fromCell.Location()] = true
			conflictLocations[toCell.Location()] = true
			if _, ok := activeForces[toCell.Location()]; !ok {
				activeForces[toCell.Location()] = make(map[int]int)
			}
			force := activeForces[toCell.Location()][fromCell.Owner] + fromCell.Strength
			// combine strength from one owner coming from multiple cells to a max of 255
			activeForces[toCell.Location()][fromCell.Owner] = min(255, force)
			// fromCell.Owner = NEUTRAL
			fromCell.Strength = 0
		}
	}
	// forces which are brought into conflict by orthogonally adjacent active forces
	passiveForces := make(map[hlt.Location]map[int]int)
	// forces moving into new cells
	for _, move := range moves {
		if move.Direction != hlt.STILL {
			toCell := clone.GetCell(move.Location, move.Direction)
			for _, direction := range hlt.Directions {
				neighborCell := clone.GetCell(toCell.Location(), direction)
				conflictLocations[neighborCell.Location()] = true
				if _, ok := activeForces[neighborCell.Location()][neighborCell.Owner]; ok {
					force := activeForces[neighborCell.Location()][neighborCell.Owner] + neighborCell.Strength
					activeForces[neighborCell.Location()][neighborCell.Owner] = min(255, force)
					neighborCell.Strength = 0
				} else {
					if _, ok := passiveForces[neighborCell.Location()]; !ok {
						passiveForces[neighborCell.Location()] = make(map[int]int)
					}
					force := passiveForces[neighborCell.Location()][neighborCell.Owner] + neighborCell.Strength
					// combine strength from one owner coming from multiple cells to a max of 255
					passiveForces[neighborCell.Location()][neighborCell.Owner] = min(255, force)
					neighborCell.Strength = 0
				}
				// neighborCell.Owner = NEUTRAL
			}
		}
	}
	// generate effects caused by all active and passive forces
	effect := make(map[hlt.Location]map[int]int)
	// combine passive and active forces
	for location, activeCellForces := range activeForces {
		cell := clone.Get(location.X, location.Y)
		if _, ok := effect[location]; !ok {
			effect[location] = make(map[int]int)
		}
		for owner, activeCellForce := range activeCellForces {
			for _, direction := range hlt.Directions {
				otherCell := clone.GetCell(cell.Location(), direction)
				for otherOwner := range activeForces[otherCell.Location()] {
					if otherOwner != owner {
						if _, ok := effect[otherCell.Location()]; !ok {
							effect[otherCell.Location()] = make(map[int]int)
						}
						effect[otherCell.Location()][otherOwner] += activeCellForce
						// Cell was in conflict, it is possible there will not be an owner here following combat
						otherCell.Owner = NEUTRAL
					}
				}
				for otherOwner, otherPassiveCellForce := range passiveForces[otherCell.Location()] {
					if otherOwner != owner {
						if _, ok := effect[otherCell.Location()]; !ok {
							effect[otherCell.Location()] = make(map[int]int)
						}
						effect[otherCell.Location()][otherOwner] += activeCellForce
						if otherOwner != NEUTRAL {
							effect[cell.Location()][owner] += otherPassiveCellForce
						}
						// Cell was in conflict, it is possible there will not be an owner here following combat
						otherCell.Owner = NEUTRAL
					}
				}
			}
		}
	}
	// apply effects
	for location, cellEffects := range effect {
		for owner, effect := range cellEffects {
			if force, ok := activeForces[location][owner]; ok {
				activeForces[location][owner] = max(0, force-effect)
			}
			if force, ok := passiveForces[location][owner]; ok {
				passiveForces[location][owner] = max(0, force-effect)
			}
		}
	}
	for location, remainingActiveForces := range activeForces {
		cell := clone.Get(location.X, location.Y)
		for owner, remainingForce := range remainingActiveForces {
			if remainingForce > 0 {
				cell.Owner = owner
				cell.Strength = remainingForce
			}
		}
	}
	for location, remainingPassiveForces := range passiveForces {
		cell := clone.Get(location.X, location.Y)
		for owner, remainingForce := range remainingPassiveForces {
			if remainingForce > 0 {
				cell.Owner = owner
				cell.Strength = remainingForce
			}
		}
	}
	// Production for cells that didn't move or fight
	for _, cell := range clone.GetCells(func(cell *Cell) bool {
		_, ok := conflictLocations[cell.Location()]
		return !ok && cell.Owner != NEUTRAL
	}) {
		strength := cell.Strength + cell.Production
		cell.Strength = min(255, strength)
	}
	return clone
}

func (c *Cells) InBounds(location hlt.Location) bool {
	_, ok := c.Contents[location.X][location.X]
	return ok
}

func (c *Cells) GetLocation(location hlt.Location, direction hlt.Direction) hlt.Location {
	switch direction {
	case hlt.NORTH:
		if location.Y == 0 {
			location.Y = c._sourceHeight - 1
		} else {
			location.Y--
		}
	case hlt.EAST:
		if location.X == c._sourceWidth-1 {
			location.X = 0
		} else {
			location.X++
		}
	case hlt.SOUTH:
		if location.Y == c._sourceHeight-1 {
			location.Y = 0
		} else {
			location.Y++
		}
	case hlt.WEST:
		if location.X == 0 {
			location.X = c._sourceWidth - 1
		} else {
			location.X--
		}
	}
	return location
}

func (c *Cells) GetCell(location hlt.Location, direction hlt.Direction) *Cell {
	loc := c.GetLocation(location, direction)
	return c.Contents[loc.Y][loc.X]
}

func (c *Cells) GetCells(cellTest CellTest) []*Cell {
	results := make([]*Cell, 0)
	for y := c.Y; y < c.Y+c.Height; y++ {
		yf := y % c._sourceHeight
		for x := c.X; x < c.X+c.Width; x++ {
			xf := x % c._sourceWidth
			cell := c.Get(yf, xf)
			if cellTest(cell) {
				results = append(results, cell)
			}
		}
	}
	return results
}

func (c *Cells) Get(x int, y int) *Cell {
	return c.Contents[y][x]
}

func (c *Cells) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Cells(x:%d, y:%d, w:%d, h:%d)\n", c.X, c.Y, c.Width, c.Height))
	for y := c.Y; y < c.Y+c.Height; y++ {
		yf := y % c._sourceHeight
		for x := c.X; x < c.X+c.Width; x++ {
			xf := x % c._sourceWidth
			buffer.WriteString(fmt.Sprintf("%v, ", c.Contents[yf][xf].String()))
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
}

/*
 ██████ ███████ ██      ██
██      ██      ██      ██
██      █████   ██      ██
██      ██      ██      ██
 ██████ ███████ ███████ ███████
*/

type Cell struct {
	Cells      *Cells
	Y          int
	X          int
	Owner      int
	Strength   int
	Production int
	_CalcDone  bool
	_Border    bool
	_Damage    int
}

func NewCell(cells *Cells, site hlt.Site, x int, y int) *Cell {
	return &Cell{
		Cells:      cells,
		Y:          y,
		X:          x,
		Owner:      site.Owner,
		Strength:   site.Strength,
		Production: site.Production,
		_CalcDone:  false,
	}
}

func (c *Cell) Clone(cells *Cells) *Cell {
	return &Cell{
		Cells:      cells,
		Y:          c.Y,
		X:          c.X,
		Owner:      c.Owner,
		Strength:   c.Strength,
		Production: c.Production,
		_CalcDone:  false,
	}
}

func (c *Cell) Update(site hlt.Site) {
	c.Owner = site.Owner
	c.Production = site.Production
	c.Strength = site.Strength
	c._CalcDone = false
}

func (c *Cell) Location() hlt.Location {
	return hlt.Location{
		X: c.X,
		Y: c.Y,
	}
}

func (c *Cell) Site() hlt.Site {
	return hlt.Site{
		Owner:      c.Owner,
		Strength:   c.Strength,
		Production: c.Production,
	}
}

func (c *Cell) GetNeighbor(direction hlt.Direction) *Cell {
	return c.Cells.GetCell(c.Location(), direction)
}

func (c *Cell) Neighbors() []*Cell {
	cells := make([]*Cell, 0, 4)
	for i, direction := range hlt.CARDINALS {
		cells[i] = c.GetNeighbor(direction)
	}
	return cells
}

func (c *Cell) Heuristic(owner int) float64 {
	if c.Owner == 0 && c.Strength > 0 {
		return float64(c.Production) / float64(c.Strength)
	} else {
		return float64(c.Damage())
	}
}

func (c *Cell) Border() bool {
	if !c._CalcDone {
		c.Calc()
	}
	return c._Border
}

// Damage to be suffered by the Cell's Bot's Owner upon taking this cell.
func (c *Cell) Damage() int {
	if !c._CalcDone {
		c.Calc()
	}
	return c._Damage
}

// evaluate sites orthogonal to the given cell
func (c *Cell) Calc() {
	border := false
	damage := c.Strength
	for _, neighbor := range c.Neighbors() {
		if neighbor.Owner != c.Owner {
			if neighbor.Owner != 0 {
				damage += neighbor.Strength
			}
			border = true
		}
	}
	c._Border = border
	c._Damage = damage
}

func (c *Cell) String() string {
	return fmt.Sprintf("(x:%d, y:%d)[o:%d, p:%d, s:%d]", c.X, c.Y, c.Owner, c.Production, c.Strength)
}

/*
███    ███  █████  ██ ███    ██
████  ████ ██   ██ ██ ████   ██
██ ████ ██ ███████ ██ ██ ██  ██
██  ██  ██ ██   ██ ██ ██  ██ ██
██      ██ ██   ██ ██ ██   ████
*/

func main() {
	conn, gameMap := hlt.NewConnection("BrevBot")
	bot := NewBot(conn.PlayerTag, gameMap)
	turn := 0
	for {
		turn++
		var moves hlt.MoveSet
		gameMap = conn.GetFrame()

		startTime := time.Now()
		bot.Update(gameMap)
		moves = bot.Moves()
		log(fmt.Sprintf("Time: %v\n", time.Now().Sub(startTime)))

		conn.SendFrame(moves)
	}
}
