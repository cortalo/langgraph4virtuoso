package grid

import "errors"
import "autorouter/common"

type Point = common.Point

var ErrOutOfBounds = errors.New("point out of bounds")
var ErrClearOccupiedNotOccupied = errors.New("clear occupied not occupied")
var ErrNetIDMismatch = errors.New("net id mismatch")
var ErrCellNotOccupied = errors.New("cell not occupied by net")

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

func (g *Grid) IsPassable(p Point, netID int) bool {
	return g.isPassable(p, netID, false)
}

func (g *Grid) IsPassableIgnoreOccupied(p Point, netID int) bool {
	return g.isPassable(p, netID, true)
}

func (g *Grid) isPassable(p Point, netID int, ignoreOccupied bool) bool {
	cell, err := g.GetCell(p)
	if err != nil {
		return false
	}
	if ignoreOccupied {
		return cell.State != CellBlocked
	}
	return cell.State == CellEmpty || cell.NetID == netID
}

func (g *Grid) Neighbors(p Point, netID int) []Point {
	return g.neighbors(p, netID, false)
}

func (g *Grid) NeighborsIgnoreOccupied(p Point, netID int) []Point {
	return g.neighbors(p, netID, true)
}

func (g *Grid) neighbors(p Point, netID int, ignoreOccupied bool) []Point {
	dirs := []Point{
		{p.X, p.Y - 1}, // 上
		{p.X, p.Y + 1}, // 下
		{p.X - 1, p.Y}, // 左
		{p.X + 1, p.Y}, // 右
	}

	neighbors := make([]Point, 0, 4)
	for _, candidate := range dirs {
		if g.isPassable(candidate, netID, ignoreOccupied) {
			neighbors = append(neighbors, candidate)
		}
	}
	return neighbors
}
