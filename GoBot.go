package main

import (
	"bytes"
	"errors"
	"fmt"
	"hlt"
	"os"
	"sync"
	"time"
)

const logFile = "goLog.txt"
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
	for dy := 0; dy < cells.Height; dy++ {
		yf := cells.Y + dy
		field[yf] = make(map[int]int)
		for dx := 0; dx < cells.Width; dx++ {
			xf := cells.X + dx
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
	Dead                           // Agent died in the last turn
	Expanding                      // on Boundry, tasked with capturing new territory
	Initializing                   // new Agent hasn't decided what to do yet
	Transporting                   // within Body, moving strength to edge
)

type Agent struct {
	Boss     Boss       // Get Owner and other Agents
	State    AgentState // What has the Agent done with its life
	LastMove hlt.Move
	Location hlt.Location
}

func NewAgent(boss Boss, location hlt.Location) *Agent {
	return &Agent{
		Boss:     boss,
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
	return a.Boss.GetAgents()
}

func (a *Agent) GetOwner() int {
	return a.Boss.GetOwner()
}

func (a *Agent) GetCell() *Cell {
	return a.GetCells().GetCell(a.Location, hlt.STILL)
}

func (a *Agent) GetCells() *Cells {
	return a.Boss.GetCells()
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
██████   ██████  ███████ ███████
██   ██ ██    ██ ██      ██
██████  ██    ██ ███████ ███████
██   ██ ██    ██      ██      ██
██████   ██████  ███████ ███████
*/

type Boss interface {
	GetAgents() map[hlt.Location]*Agent
	GetCells() *Cells
	GetOwner() int
}

type Bot struct {
	Agents           map[hlt.Location]*Agent
	Owner            int
	Cells            *Cells
	PointsOfInterest []*PointOfInterest
}

/*
██████   ██████  ████████
██   ██ ██    ██    ██
██████  ██    ██    ██
██   ██ ██    ██    ██
██████   ██████     ██
*/

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
	// Update Agents

	newAgents := make(map[hlt.Location]*Agent)
	for _, agent := range b.Agents {
		updatedAgentLocation := agent.Update()
		if agent.State != Dead {
			newAgents[updatedAgentLocation] = agent
		}
	}
	b.Agents = newAgents
}

func (b *Bot) GetAgents() map[hlt.Location]*Agent {
	return b.Agents
}

func (b *Bot) GetCells() *Cells {
	return b.Cells
}

func (b *Bot) GetOwner() int {
	return b.Owner
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
	// Ownership changes between turns
	Adds map[int][]*Cell
	Subs map[int][]*Cell
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
		Adds:            make(map[int][]*Cell),
		Subs:            make(map[int][]*Cell),
		AvgProduction:   0,
		MaxProduction:   255,
		MinProduction:   0,
		TotalProduction: make(map[int]int),
		TotalStrength:   make(map[int]int),
		TotalTerritory:  make(map[int]int),
	}
	contents := make(map[int]map[int]*Cell)
	sum := 0
	for dy := 0; dy < height; dy++ {
		yf := y + dy
		contents[yf] = make(map[int]*Cell)
		for dx := 0; dx < width; dx++ {
			xf := x + dx
			site := gameMap.Contents[yf][xf]
			contents[yf][xf] = NewCell(cells, gameMap.Contents[yf][xf], xf, yf)
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
		Adds:            make(map[int][]*Cell),
		Subs:            make(map[int][]*Cell),
		AvgProduction:   c.AvgProduction,
		MaxProduction:   c.MaxProduction,
		MinProduction:   c.MinProduction,
		TotalProduction: c.TotalProduction,
		TotalStrength:   c.TotalStrength,
		TotalTerritory:  c.TotalTerritory,
	}
	contents := make(map[int]map[int]*Cell)
	for dy := 0; dy < c.Height; dy++ {
		yf := c.Y + dy
		contents[yf] = make(map[int]*Cell)
		for dx := 0; dx < c.Width; dx++ {
			xf := c.X + dx
			contents[yf][xf] = c.Contents[yf][xf].Clone(clone)
		}
	}
	clone.Contents = contents
	for owner, cells := range c.Adds {
		adds := make([]*Cell, 0, len(c.Adds[owner]))
		for _, cell := range cells {
			loc := cell.Location()
			adds = append(adds, contents[loc.Y][loc.X])
		}
		clone.Adds[owner] = adds
	}
	for owner, cells := range c.Subs {
		subs := make([]*Cell, 0, len(c.Subs[owner]))
		for _, cell := range cells {
			loc := cell.Location()
			subs = append(subs, contents[loc.Y][loc.X])
		}
		clone.Subs[owner] = subs
	}
	return clone
}

// update Cells with Site data from provided GameMap
func (c *Cells) Update(gameMap hlt.GameMap) {
	c.Adds = make(map[int][]*Cell)
	c.Subs = make(map[int][]*Cell)
	c.TotalProduction = make(map[int]int)
	c.TotalStrength = make(map[int]int)
	c.TotalTerritory = make(map[int]int)
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			site := gameMap.Contents[y][x]
			c.TotalProduction[site.Owner] += site.Strength
			c.TotalStrength[site.Owner] += site.Strength
			c.TotalTerritory[site.Owner]++
			cell := c.Get(x, y)
			if cell.Owner != site.Owner {
				c.Adds[site.Owner] = append(c.Adds[site.Owner], cell)
				c.Subs[cell.Owner] = append(c.Subs[cell.Owner], cell)
			}
			cell.Update(site)
		}
	}
}

// applies moves in the same way halite.io would... I think.
func (c *Cells) Simulate(moves hlt.MoveSet) *Cells {
	clone := c.Clone()
	// used to prevent production on cells that had movement this round
	fightingLocations := make(map[hlt.Location]bool)
	mapForces := make(map[hlt.Location]map[int]int)
	effect := make(map[hlt.Location]map[int]int)
	// forces moving into new cells
	for _, move := range moves {
		if move.Direction != hlt.STILL {
			fromCell := clone.Get(move.Location.X, move.Location.Y)
			toCell := clone.GetCell(move.Location, move.Direction)
			fightingLocations[fromCell.Location()] = true
			fightingLocations[toCell.Location()] = true
			if _, ok := mapForces[toCell.Location()]; !ok {
				mapForces[toCell.Location()] = make(map[int]int)
			}
			force := mapForces[toCell.Location()][fromCell.Owner] + fromCell.Strength
			// combine strength from one owner coming from multiple cells to a max of 255
			mapForces[toCell.Location()][fromCell.Owner] = min(255, force)
			fromCell.Strength = 0
		}
	}
	// apply damage for each force moved to each opposing strength
	for location, cellForces := range mapForces {
		cell := clone.Get(location.X, location.Y)
		if _, ok := effect[location]; !ok {
			effect[location] = make(map[int]int)
		}
		for owner, force := range cellForces {
			// take on previous cell owner's strength
			if cell.Owner != owner {
				effect[location][cell.Owner] += force
			}
			// damage other forces moving into this cell
			for otherOwner := range cellForces {
				// cell Owner effect already accounted for
				if otherOwner != owner && otherOwner != cell.Owner {
					effect[location][otherOwner] += force
				}
			}
			// Overkill from cardinals only for non neutral strengths
			for _, direction := range hlt.CARDINALS {
				neighborCell := c.GetCell(location, direction)
				if owner != neighborCell.Owner && neighborCell.Owner != NEUTRAL {
					fightingLocations[neighborCell.Location()] = true
					// take on neighbor strength
					if _, ok := effect[neighborCell.Location()]; !ok {
						effect[neighborCell.Location()] = make(map[int]int)
					}
					effect[neighborCell.Location()][neighborCell.Owner] += force
					// neighbor takes its turn out on force
					effect[location][owner] += neighborCell.Strength
				}
			}
		}
	}
	// apply effects
	for location, cellEffects := range effect {
		cell := clone.Get(location.X, location.Y)
		cellOwnerSet := false
		if cellForces, ok := mapForces[location]; ok {
			// apply effect to previous owner of the cell
			if ownerEffect, ok := cellEffects[cell.Owner]; ok {
				if cell.Strength-ownerEffect > 0 {
					cell.Strength -= ownerEffect
				} else {
					cell.Owner = NEUTRAL
					cell.Strength = 0
				}
			}
			// apply effects to forces moving into the cell
			for owner, force := range cellForces {
				if ownerEffect, ok := cellEffects[owner]; ok {
					if force-ownerEffect > 0 {
						cell.Strength = force - ownerEffect
						cell.Owner = owner
						if cellOwnerSet {
							log("Cell Owner set more than once!")
						} else {
							cellOwnerSet = true
						}
					}
				}
			}
		}
	}
	for _, cell := range clone.GetCells(func(cell *Cell) bool {
		_, ok := fightingLocations[cell.Location()]
		return !ok
	}) {
		strength := cell.Strength + cell.Production
		cell.Strength = min(255, strength)
	}
	return clone
}

func (c *Cells) GetLocation(location hlt.Location, direction hlt.Direction) hlt.Location {
	loc := hlt.Location{
		X: location.X - c.X,
		Y: location.Y - c.Y,
	}
	switch direction {
	case hlt.NORTH:
		if loc.Y == 0 {
			loc.Y = c.Height - 1
		} else {
			loc.Y--
		}
	case hlt.EAST:
		if loc.X == c.Width-1 {
			loc.X = 0
		} else {
			loc.X++
		}
	case hlt.SOUTH:
		if loc.Y == c.Height-1 {
			loc.Y = 0
		} else {
			loc.Y++
		}
	case hlt.WEST:
		if loc.X == 0 {
			loc.X = c.Width - 1
		} else {
			loc.X--
		}
	}
	return hlt.Location{
		X: loc.X + c.X,
		Y: loc.Y + c.Y,
	}
}

func (c *Cells) GetCell(location hlt.Location, direction hlt.Direction) *Cell {
	loc := c.GetLocation(location, direction)
	return c.Contents[loc.Y][loc.X]
}

func (c *Cells) GetCells(cellTest CellTest) []*Cell {
	results := make([]*Cell, 0)
	for dy := 0; dy < c.Height; dy++ {
		yf := c.Y + dy
		for dx := 0; dx < c.Width; dx++ {
			xf := c.X + dx
			if cellTest(c.Contents[yf][xf]) {
				results = append(results, c.Contents[yf][xf])
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
	for dy := 0; dy < c.Height; dy++ {
		yf := c.Y + dy
		for dx := 0; dx < c.Width; dx++ {
			xf := c.X + dx
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
	return fmt.Sprintf("[%d, %3d, %3d]", c.Owner, c.Production, c.Strength)
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
