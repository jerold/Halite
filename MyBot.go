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
const unowned = 0
const maxCost = 999999
const maxStrength = 255

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

type stackable struct {
	Content  *Cell
	Previous *stackable
}

// Stack is a FILO collection
type Stack struct {
	_top    *stackable
	_length int
}

// NewStack is a constructor
func NewStack() *Stack {
	return &Stack{_length: 0}
}

// Peek at top item
func (s *Stack) Peek() (*Cell, error) {
	if s.isEmpty() {
		return nil, errors.New("Empty Stack")
	}
	return s._top.Content, nil
}

// Push new item into the Stack
func (s *Stack) Push(c *Cell) {
	newTop := &stackable{Content: c, Previous: s._top}
	s._top = newTop
	s._length++
}

// Pop the top item off of the stack
func (s *Stack) Pop() (*Cell, error) {
	if s.isEmpty() {
		return nil, errors.New("Empty Stack")
	}
	pop := s._top
	cell := pop.Content
	s._top = pop.Previous
	pop.Previous = nil
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

// CellTest is a function used to filter or include cells that meet described criteria
type CellTest func(cell *Cell) bool

// CellCost is a function used to calculate the cost of moving to a cell from a another cell that itself has a cost to get to.
type CellCost func(via *Cell, cell *Cell, field FlowField) float32

// FlowField is a location and a map of strength costs to that location
type FlowField map[hlt.Location]float32

// NewFlowField is a constructor
func NewFlowField(dests []*Cell, cf CellCost) FlowField {
	field := FlowField{}
	stack := NewStack()
	for _, dest := range dests {
		field[dest.Location] = cf(nil, dest, field)
		stack.Push(dest)
	}
	oppCount := 0
	for stack.isNotEmpty() {
		if cell, err := stack.Pop(); err == nil {
			oppCount++
			for _, neighbor := range cell.Neighbors() {
				newCost := cf(cell, neighbor, field)
				if value, ok := field[neighbor.Location]; newCost < maxCost && (!ok || value > newCost) {
					field[neighbor.Location] = newCost
					stack.Push(neighbor)
				}
			}
		}
	}
	return field
}

// NewBorderFlow is a constructor
func NewBorderFlow(owner int, borders []*Cell) FlowField {
	return NewFlowField(borders, func(via *Cell, cell *Cell, field FlowField) float32 {
		if cell.Owner != owner {
			return maxCost
		}
		if via != nil {
			if via.Strength+cell.Strength > maxStrength {
				return maxCost
			}
			return field[via.Location] + float32(cell.Production)
		}
		return float32(cell.Production)
	})
}

/*
██████   ██████  ████████
██   ██ ██    ██    ██
██████  ██    ██    ██
██   ██ ██    ██    ██
██████   ██████     ██
*/

// Bot is in control of Agents for all owned cells.
type Bot struct {
	Owner             int
	Cells             *Cells
	ToBorder          FlowField
	StartingLocations map[int]hlt.Location
}

// NewBot is a constructor
func NewBot(owner int, gameMap hlt.GameMap) *Bot {
	bot := &Bot{
		Owner:             owner,
		Cells:             NewCells(0, 0, gameMap.Width, gameMap.Height, gameMap),
		StartingLocations: make(map[int]hlt.Location),
	}
	bot.ToBorder = NewBorderFlow(bot.Owner, bot.BorderCells())
	for team, ownedCells := range bot.Cells.ByOwner {
		bot.StartingLocations[team] = ownedCells.OwnedCells()[0].Location
	}
	return bot
}

// Update takes in new map data and updates agents following a turn
func (b *Bot) Update(gameMap hlt.GameMap) {
	b.Cells.Update(gameMap)
	b.ToBorder = NewBorderFlow(b.Owner, b.BorderCells())
}

// OwnedCells returns the Cells owned by this Bot
func (b *Bot) OwnedCells() []*Cell {
	return b.Cells.ByOwner[b.Owner].OwnedCells()
}

// BorderCells returns the Cells owned by this Bot that have a non-owner-owned neighbor
func (b *Bot) BorderCells() []*Cell {
	return b.Cells.ByOwner[b.Owner].BorderCells()
}

// BodyCells returns the Cells owned by this Bot that have only owner-owned neighbors
func (b *Bot) BodyCells() []*Cell {
	return b.Cells.ByOwner[b.Owner].BodyCells()
}

// Moves puts together a list of Moves for each Agent owned
func (b *Bot) Moves() hlt.MoveSet {
	var moves = hlt.MoveSet{}
	return moves
}

// OwnedCells collection of Cells belonging to the same owner with some stats
type OwnedCells struct {
	TotalProduction int
	TotalStrength   int
	TotalTerritory  int
	_cells          []*Cell
	_borderCells    []*Cell
	_bodyCells      []*Cell
	_totalX         int
	_totalY         int
	_calcDone       bool
}

// NewOwnedCells is a constructor
func NewOwnedCells() *OwnedCells {
	return &OwnedCells{
		_cells:       make([]*Cell, 0, 1),
		_borderCells: make([]*Cell, 0, 1),
		_bodyCells:   make([]*Cell, 0, 1),
		_calcDone:    false,
	}
}

// Add a cell to the owned cells
func (o *OwnedCells) Add(cell *Cell) {
	o.TotalProduction += cell.Production
	o.TotalProduction += cell.Production
	o.TotalProduction++
	o._cells = append(o._cells, cell)
	o._calcDone = false
}

// Reset removes all owned cells and resets stats
func (o *OwnedCells) Reset() {
	o._cells = make([]*Cell, 0, len(o._cells))
	o.TotalProduction = 0
	o.TotalStrength = 0
	o.TotalTerritory = 0
	o._borderCells = make([]*Cell, 0, len(o._borderCells))
	o._bodyCells = make([]*Cell, 0, len(o._bodyCells))
	o._totalX = 0
	o._totalY = 0
	o._calcDone = false
}

// CenterOfMass is the centroid for the owned shape
func (o *OwnedCells) CenterOfMass() hlt.Location {
	return hlt.NewLocation(o._totalX/len(o._cells), o._totalY/len(o._cells))
}

// OwnedCells is the list of all owned cells
func (o *OwnedCells) OwnedCells() []*Cell {
	return o._cells
}

// BorderCells is the list of only border owned cells
func (o *OwnedCells) BorderCells() []*Cell {
	if !o._calcDone {
		o.Calc()
	}
	return o._borderCells
}

// BodyCells is the list of only body owned cells
func (o *OwnedCells) BodyCells() []*Cell {
	if !o._calcDone {
		o.Calc()
	}
	return o._bodyCells
}

// Calc splits the cells into Border and Body lists. Expensive.
func (o *OwnedCells) Calc() {
	for _, cell := range o._cells {
		if cell.Border() {
			o._borderCells = append(o._borderCells, cell)
		} else {
			o._bodyCells = append(o._bodyCells, cell)
		}
	}
	o._calcDone = true
}

/*
 ██████ ███████ ██      ██      ███████
██      ██      ██      ██      ██
██      █████   ██      ██      ███████
██      ██      ██      ██           ██
 ██████ ███████ ███████ ███████ ███████
*/

// Cells represents a subview of the gameMap. Simulated forward with
// a set of moves, or updated from turn to turn by a bot.
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
	ByOwner map[int]*OwnedCells
	// Production stats
	AvgProduction int
	MaxProduction int
	MinProduction int
}

// NewCells is a constructor
func NewCells(x int, y int, width int, height int, gameMap hlt.GameMap) *Cells {
	cells := &Cells{
		Height:        height,
		Width:         width,
		X:             x,
		Y:             y,
		_sourceHeight: gameMap.Height,
		_sourceWidth:  gameMap.Width,
		ByOwner:       make(map[int]*OwnedCells),
		AvgProduction: 0,
		MaxProduction: 0,
		MinProduction: 255,
	}
	contents := make(map[int]map[int]*Cell)
	sum := 0
	for i := y; i < y+height; i++ {
		yf := i % cells._sourceHeight
		contents[yf] = make(map[int]*Cell)
		for j := x; j < x+width; j++ {
			xf := j % cells._sourceWidth
			site := gameMap.Contents[yf][xf]
			cell := NewCell(cells, gameMap.Contents[yf][xf], xf, yf)
			contents[yf][xf] = cell
			// Add to Owner's OwnedCells
			if _, ok := cells.ByOwner[site.Owner]; !ok {
				cells.ByOwner[site.Owner] = NewOwnedCells()
			}
			cells.ByOwner[site.Owner].Add(cell)
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

// Clone produces a copy of the Cells containing new copies of all contained cells
func (c *Cells) Clone() *Cells {
	clone := &Cells{
		Height:        c.Height,
		Width:         c.Width,
		X:             c.X,
		Y:             c.Y,
		_sourceHeight: c._sourceHeight,
		_sourceWidth:  c._sourceWidth,
		ByOwner:       make(map[int]*OwnedCells),
		AvgProduction: c.AvgProduction,
		MaxProduction: c.MaxProduction,
		MinProduction: c.MinProduction,
	}
	contents := make(map[int]map[int]*Cell)
	for y := c.Y; y < c.Y+c.Height; y++ {
		yf := y % c._sourceHeight
		contents[yf] = make(map[int]*Cell)
		for x := c.X; x < c.X+c.Width; x++ {
			xf := x % c._sourceWidth
			cell := c.Contents[yf][xf].Clone(clone)
			contents[yf][xf] = cell
			// Add to Owner's OwnedCells
			if _, ok := clone.ByOwner[cell.Owner]; !ok {
				clone.ByOwner[cell.Owner] = NewOwnedCells()
			}
			clone.ByOwner[cell.Owner].Add(cell)
		}
	}
	clone.Contents = contents
	return clone
}

// Update Cells with Site data from provided GameMap
func (c *Cells) Update(gameMap hlt.GameMap) {
	for _, ownedCells := range c.ByOwner {
		ownedCells.Reset()
	}
	// Cells my not be the full size of gameMap, only iterate Cells contents
	for y := c.Y; y < c.Y+c.Height; y++ {
		yf := y % c._sourceHeight
		for x := c.X; x < c.X+c.Width; x++ {
			xf := x % c._sourceWidth
			site := gameMap.Contents[yf][xf]
			cell := c.Get(xf, yf)
			cell.Update(site)
			// Add to Owner's OwnedCells
			if _, ok := c.ByOwner[site.Owner]; !ok {
				c.ByOwner[site.Owner] = NewOwnedCells()
			}
			c.ByOwner[site.Owner].Add(cell)
		}
	}
}

// Simulate applies moves in the same way halite.io would... I think.
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
			conflictLocations[fromCell.Location] = true
			conflictLocations[toCell.Location] = true
			if _, ok := activeForces[toCell.Location]; !ok {
				activeForces[toCell.Location] = make(map[int]int)
			}
			force := activeForces[toCell.Location][fromCell.Owner] + fromCell.Strength
			// combine strength from one owner coming from multiple cells to a max of 255
			activeForces[toCell.Location][fromCell.Owner] = min(255, force)
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
				neighborCell := clone.GetCell(toCell.Location, direction)
				conflictLocations[neighborCell.Location] = true
				if _, ok := activeForces[neighborCell.Location][neighborCell.Owner]; ok {
					force := activeForces[neighborCell.Location][neighborCell.Owner] + neighborCell.Strength
					activeForces[neighborCell.Location][neighborCell.Owner] = min(255, force)
					neighborCell.Strength = 0
				} else {
					if _, ok := passiveForces[neighborCell.Location]; !ok {
						passiveForces[neighborCell.Location] = make(map[int]int)
					}
					force := passiveForces[neighborCell.Location][neighborCell.Owner] + neighborCell.Strength
					// combine strength from one owner coming from multiple cells to a max of 255
					passiveForces[neighborCell.Location][neighborCell.Owner] = min(255, force)
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
				otherCell := clone.GetCell(cell.Location, direction)
				for otherOwner := range activeForces[otherCell.Location] {
					if otherOwner != owner {
						if _, ok := effect[otherCell.Location]; !ok {
							effect[otherCell.Location] = make(map[int]int)
						}
						effect[otherCell.Location][otherOwner] += activeCellForce
						// Cell was in conflict, it is possible there will not be an owner here following combat
						otherCell.Owner = unowned
					}
				}
				for otherOwner, otherPassiveCellForce := range passiveForces[otherCell.Location] {
					if otherOwner != owner {
						if _, ok := effect[otherCell.Location]; !ok {
							effect[otherCell.Location] = make(map[int]int)
						}
						effect[otherCell.Location][otherOwner] += activeCellForce
						if otherOwner != unowned {
							effect[cell.Location][owner] += otherPassiveCellForce
						}
						// Cell was in conflict, it is possible there will not be an owner here following combat
						otherCell.Owner = unowned
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
		_, ok := conflictLocations[cell.Location]
		return !ok && cell.Owner != unowned
	}) {
		strength := cell.Strength + cell.Production
		cell.Strength = min(255, strength)
	}
	// This sucks but we now have to go through and reassign all OwnedCells
	clone.ByOwner = make(map[int]*OwnedCells)
	for i := clone.Y; i < clone.Y+clone.Height; i++ {
		yf := i % clone._sourceHeight
		for j := clone.X; j < clone.X+clone.Width; j++ {
			xf := j % clone._sourceWidth
			cell := clone.Contents[yf][xf]
			// Add to Owner's OwnedCells
			if _, ok := clone.ByOwner[cell.Owner]; !ok {
				clone.ByOwner[cell.Owner] = NewOwnedCells()
			}
			clone.ByOwner[cell.Owner].Add(cell)
		}
	}
	return clone
}

// InBounds allows the user to check if a location is within the Cells bounds
func (c *Cells) InBounds(location hlt.Location) bool {
	_, ok := c.Contents[location.X][location.X]
	return ok
}

// GetLocation returns a Location for the requested Location
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

// GetCell returns a Cell for the given Locations
func (c *Cells) GetCell(location hlt.Location, direction hlt.Direction) *Cell {
	loc := c.GetLocation(location, direction)
	return c.Contents[loc.Y][loc.X]
}

// GetCells returns all the cells that pass the provided test
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

// Get returns the cell for a given x, y coordinate
func (c *Cells) Get(x int, y int) *Cell {
	return c.Contents[y][x]
}

// String convert the Cells into a string
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

// Cell is a combination of Location and Site
type Cell struct {
	Cells      *Cells
	Location   hlt.Location
	Y          int
	X          int
	Owner      int
	Strength   int
	Production int
	_CalcDone  bool
	_Border    bool
	_Damage    int
}

// NewCell is a constructor
func NewCell(cells *Cells, site hlt.Site, x int, y int) *Cell {
	return &Cell{
		Cells:      cells,
		Location:   hlt.NewLocation(x, y),
		Y:          y,
		X:          x,
		Owner:      site.Owner,
		Strength:   site.Strength,
		Production: site.Production,
		_CalcDone:  false,
	}
}

// Clone produces a copy of the given Cell
func (c *Cell) Clone(cells *Cells) *Cell {
	return &Cell{
		Cells:      cells,
		Location:   c.Location,
		Y:          c.Y,
		X:          c.X,
		Owner:      c.Owner,
		Strength:   c.Strength,
		Production: c.Production,
		_CalcDone:  false,
	}
}

// Update to reflect the new Site info
func (c *Cell) Update(site hlt.Site) {
	c.Owner = site.Owner
	c.Production = site.Production
	c.Strength = site.Strength
	c._CalcDone = false
}

// Site returns a hlt.Site for the Cell
func (c *Cell) Site() hlt.Site {
	return hlt.Site{
		Owner:      c.Owner,
		Strength:   c.Strength,
		Production: c.Production,
	}
}

// GetNeighbor returns a list of surrounding cells
func (c *Cell) GetNeighbor(direction hlt.Direction) *Cell {
	return c.Cells.GetCell(c.Location, direction)
}

// Neighbors returns a cell in the given direction from this one
func (c *Cell) Neighbors() []*Cell {
	cells := make([]*Cell, 0, 4)
	for _, direction := range hlt.CARDINALS {
		cells = append(cells, c.GetNeighbor(direction))
		// cells[i] = c.GetNeighbor(direction)
	}
	return cells
}

// Heuristic returns the value of the Cell
func (c *Cell) Heuristic(owner int) float64 {
	if c.Owner == 0 && c.Strength > 0 {
		return float64(c.Production) / float64(c.Strength)
	}
	return float64(c.Damage())
}

// Border is true if the Cell has at least one neighbor not owned by the Cell's owner
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

// Calc evaluates sites orthogonal to the given cell
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
	conn, gameMap := hlt.VeryNewConnection()
	// conn, gameMap := hlt.OldConnection("BrevBot")
	bot := NewBot(conn.PlayerTag, gameMap)
	conn.SendName("BrevBot")
	log("Name Sent!")
	turn := 0
	for {
		turn++
		log("Turn:", turn)
		startTime := time.Now()
		gameMap = conn.GetFrame()
		bot.Update(gameMap)
		moves := bot.Moves()
		conn.SendFrame(moves)
		stopTime := time.Now()
		log(fmt.Sprintf("Time: %v", stopTime.Sub(startTime)))
	}
}
