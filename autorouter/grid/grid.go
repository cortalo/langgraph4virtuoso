package grid

import "errors"
import "autorouter/common"

type Point = common.Point

var ErrOutOfBounds = errors.New("point out of bounds")
var ErrClearOccupiedNotOccupied = errors.New("clear occupied not occupied")
var ErrNetIDMismatch = errors.New("net id mismatch")
var ErrCellNotOccupied = errors.New("cell not occupied by net")
var ErrCellNotEmpty = errors.New("cell is not empty")

// CellState 表示一个格子的状态
type CellState int

const (
	CellEmpty    CellState = iota
	CellBlocked            // 障碍物
	CellOccupied           // 被某条线占用
)

// Cell 表示网格上的一个格子
type Cell struct {
	State CellState
	NetID int // 当 State == CellOccupied 时，记录是哪条线占用的
}

// Grid 是一个二维网格
type Grid struct {
	Width  int
	Height int
	cells  [][]Cell
}

// New 创建一个指定宽高的空网格
func New(width, height int) *Grid {
	cells := make([][]Cell, height)
	for x := range cells {
		cells[x] = make([]Cell, width)
	}
	return &Grid{
		Width:  width,
		Height: height,
		cells:  cells,
	}
}

// InBounds 判断一个点是否在网格范围内
func (g *Grid) InBounds(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < g.Height && p.Y < g.Width
}

// SetBlocked 将某个点标记为障碍物
func (g *Grid) SetBlocked(p Point) error {
	if !g.InBounds(p) {
		return ErrOutOfBounds
	}
	g.cells[p.X][p.Y].State = CellBlocked
	return nil
}

// GetCell 返回某个点的 Cell，越界时返回 error
func (g *Grid) GetCell(p Point) (Cell, error) {
	if !g.InBounds(p) {
		return Cell{}, ErrOutOfBounds
	}
	return g.cells[p.X][p.Y], nil
}

func (g *Grid) GetNetID(p Point) (int, error) {
	if !g.InBounds(p) {
		return 0, ErrOutOfBounds
	}
	if g.cells[p.X][p.Y].State != CellOccupied {
		return 0, ErrCellNotOccupied
	}
	return g.cells[p.X][p.Y].NetID, nil
}

// SetOccupied 将某个点标记为被 netID 占用
func (g *Grid) SetOccupied(p Point, netID int) error {
	if !g.InBounds(p) {
		return ErrOutOfBounds
	}
	if g.cells[p.X][p.Y].State != CellEmpty {
		return ErrCellNotEmpty
	}
	g.cells[p.X][p.Y].State = CellOccupied
	g.cells[p.X][p.Y].NetID = netID
	return nil
}

func (g *Grid) ClearOccupied(p Point, netID int) error {
	if !g.InBounds(p) {
		return ErrOutOfBounds
	}
	if g.cells[p.X][p.Y].State != CellOccupied {
		return ErrClearOccupiedNotOccupied
	}
	if g.cells[p.X][p.Y].NetID != netID {
		return ErrNetIDMismatch
	}
	g.cells[p.X][p.Y].State = CellEmpty
	g.cells[p.X][p.Y].NetID = 0
	return nil
}

func (g *Grid) IsPassable(p Point, netID, halfWidth int) bool {
	return g.isPassable(p, netID, halfWidth, false)
}

func (g *Grid) IsPassableIgnoreOccupied(p Point, netID, halfWidth int) bool {
	return g.isPassable(p, netID, halfWidth, true)
}

func (g *Grid) isPassable(p Point, netID, halfWidth int, ignoreOccupied bool) bool {
	for dx := -halfWidth; dx <= halfWidth; dx++ {
		for dy := -halfWidth; dy <= halfWidth; dy++ {
			neighbor := Point{X: p.X + dx, Y: p.Y + dy}
			cell, err := g.GetCell(neighbor)
			if err != nil {
				// out of bounds
				return false
			}
			if cell.State == CellBlocked {
				return false
			}
			if !ignoreOccupied {
				if cell.State != CellEmpty && cell.NetID != netID {
					return false
				}
			}
		}
	}
	return true
}

func (g *Grid) Neighbors(p Point, netID, halfWidth int) []Point {
	return g.neighbors(p, netID, halfWidth, false)
}

func (g *Grid) NeighborsIgnoreOccupied(p Point, netID, halfWidth int) []Point {
	return g.neighbors(p, netID, halfWidth, true)
}

func (g *Grid) neighbors(p Point, netID, halfWidth int, ignoreOccupied bool) []Point {
	dirs := []Point{
		{X: -1, Y: 0}, // up
		{X: 1, Y: 0},  // down
		{X: 0, Y: -1}, // left
		{X: 0, Y: 1},  // right
	}

	neighbors := make([]Point, 0, 4)
	for _, dir := range dirs {
		candidate := Point{X: p.X + dir.X, Y: p.Y + dir.Y}
		if g.isNewStripPassable(candidate, netID, halfWidth, dir, ignoreOccupied) {
			neighbors = append(neighbors, candidate)
		}
	}
	return neighbors
}

func (g *Grid) isNewStripPassable(p Point, netID, halfWidth int, dir Point, ignoreOccupied bool) bool {
	if halfWidth == 0 {
		return g.isPassable(p, netID, halfWidth, ignoreOccupied)
	}
	if dir.X != 0 {
		// moving vertically
		stripX := p.X + dir.X*halfWidth
		for dy := -halfWidth; dy <= halfWidth; dy++ {
			if !g.isPassable(Point{X: stripX, Y: p.Y + dy}, netID, 0, ignoreOccupied) {
				return false
			}
		}
	} else {
		// moving horizontally
		stripY := p.Y + dir.Y*halfWidth
		for dx := -halfWidth; dx <= halfWidth; dx++ {
			if !g.isPassable(Point{X: p.X + dx, Y: stripY}, netID, 0, ignoreOccupied) {
				return false
			}
		}
	}
	return true
}
