package main

import (
	"errors"
	"fmt"
	"hlt"
	"os"
	"sync"
	"time"
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
	Owner  int
	Cells  *Cells
	Agents map[hlt.Location]*Agent
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
		Owner:  owner,
		Cells:  NewCells(0, 0, gameMap.Width, gameMap.Height, gameMap),
		Agents: make(map[hlt.Location]*Agent),
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
	Adds     map[int][]*Cell
	Subs     map[int][]*Cell
}

func NewCells(x int, y int, width int, height int, gameMap hlt.GameMap) *Cells {
	cells := &Cells{
		Height: height,
		Width:  width,
		X:      x,
		Y:      y,
		Adds:   make(map[int][]*Cell),
		Subs:   make(map[int][]*Cell),
	}
	contents := make(map[int]map[int]*Cell)
	for dy := 0; dy < height; dy++ {
		yf := y + dy
		contents[yf] = make(map[int]*Cell)
		for dx := 0; dx < width; dx++ {
			xf := x + dx
			contents[yf][xf] = NewCell(cells, gameMap.Contents[yf][xf], xf, yf)
		}
	}
	cells.Contents = contents
	return cells
}

// Produce a copy of the Cells containing new copies of all contained cells
func (c *Cells) Clone() *Cells {
	clone := &Cells{
		Height: c.Height,
		Width:  c.Width,
		X:      c.X,
		Y:      c.Y,
		Adds:   make(map[int][]*Cell),
		Subs:   make(map[int][]*Cell),
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
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			// TODO: update Cells.Adds/Subs if owners changed to optimize adding Agents
			cell := c.Get(x, y)
			oldOwner := cell.Owner
			cell.Update(gameMap)
			newOwner := cell.Owner
			if newOwner != oldOwner {
				c.Adds[newOwner] = append(c.Adds[newOwner], cell)
				c.Adds[oldOwner] = append(c.Subs[oldOwner], cell)
			}
		}
	}
}

// applies moves in the same way halite.io would... I think.
func (c *Cells) Simulate(hlt.MoveSet) {

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
			loc.Y -= 1
		}
	case hlt.EAST:
		if loc.X == c.Width-1 {
			loc.X = 0
		} else {
			loc.X += 1
		}
	case hlt.SOUTH:
		if loc.Y == c.Height-1 {
			loc.Y = 0
		} else {
			loc.Y += 1
		}
	case hlt.WEST:
		if loc.X == 0 {
			loc.X = c.Width - 1
		} else {
			loc.X -= 1
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

func (c *Cells) Get(x int, y int) *Cell {
	return c.Contents[y][x]
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

func (c *Cell) Update(gameMap hlt.GameMap) {
	site := gameMap.GetSite(c.Location(), hlt.STILL)
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
