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
██████   ██████  ████████
██   ██ ██    ██    ██
██████  ██    ██    ██
██   ██ ██    ██    ██
██████   ██████     ██
*/

type Bot struct {
	Owner      int
	GameMap    hlt.GameMap
	Cells      [][]*Cell
	OwnedCells []*Cell
}

func NewBot(owner int, gameMap hlt.GameMap) *Bot {
	return &Bot{
		Owner:   owner,
		GameMap: gameMap,
	}
}

func (b *Bot) UpdateMap(gameMap hlt.GameMap) {
	b.GameMap = gameMap
	b.Cells = make([][]*Cell, gameMap.Height)
	b.OwnedCells = make([]*Cell, 0)
	for y := 0; y < gameMap.Height; y++ {
		b.Cells[y] = make([]*Cell, b.Width)
		for x := 0; x < gameMap.Width; x++ {
			var cell = NewCell(&gameMap, x, y)
			b.Cells[cell.Y][cell.X] = cell
			if cell.Owner == b.Owner {
				b.OwnedCells = append(b.Cells, cell)
			}
		}
	}
}

func (b *Bot) Moves() hlt.MoveSet {
	var moves = hlt.MoveSet{}
	return moves
}

/*
 ██████ ███████ ██      ██
██      ██      ██      ██
██      █████   ██      ██
██      ██      ██      ██
 ██████ ███████ ███████ ███████
*/

type Cell struct {
	Bot        *Bot
	Y          int
	X          int
	Owner      int
	Strength   int
	Production int
	_CalcDone  bool
	_Border    bool
	_Damage    int
}

func NewCell(bot *Bot, x int, y int) *Cell {
	location := hlt.NewLocation(x%bot.GameMap.Width, y%bot.GameMap.Height)
	site := bot.GameMap.GetSite(location, hlt.STILL)
	return &Cell{
		Bot:        bot,
		Y:          y,
		X:          x,
		Owner:      site.Owner,
		Strength:   site.Strength,
		Production: site.Production,
		_CalcDone:  false,
	}
}

func (c *Cell) Location() hlt.Location {
	return hlt.NewLocation(c.X, c.Y)
}

func (c *Cell) Site() hlt.Site {
	return hlt.Site{
		Owner:      c.Owner,
		Strength:   c.Strength,
		Production: c.Production,
	}
}

func (c *Cell) GetCell(direction hlt.Direction) *Cell {
	l := c.Bot.GameMap.GetLocation(c.Location(), direction)
	return c.Bot.Cells[l.X][l.Y]
}

func (c *Cell) Cells() []*Cell {
	cells := make([]*Cell, 0, 4)
	for i, direction := range hlt.CARDINALS {
		cells[i] = c.GetCell(direction)
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
	for _, direction := range hlt.CARDINALS {
		var other = c.GetCell(direction)
		if other.Owner != c.Bot.Owner {
			if other.Owner != 0 {
				damage += other.Strength
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
	bot.UpdateMap(gameMap)
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
