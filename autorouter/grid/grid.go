package grid

import "errors"
import "autorouter/common"

type Point = common.Point

var ErrOutOfBounds = errors.New("point out of bounds")

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
	cells := make([][]Cell, width)
	for x := range cells {
		cells[x] = make([]Cell, height)
	}
	return &Grid{
		Width:  width,
		Height: height,
		cells:  cells,
	}
}

// InBounds 判断一个点是否在网格范围内
func (g *Grid) InBounds(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < g.Width && p.Y < g.Height
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

// SetOccupied 将某个点标记为被 netID 占用
func (g *Grid) SetOccupied(p Point, netID int) error {
	if !g.InBounds(p) {
		return ErrOutOfBounds
	}
	g.cells[p.X][p.Y].State = CellOccupied
	g.cells[p.X][p.Y].NetID = netID
	return nil
}

// IsPassable 判断某个点是否可以通行（空或已被同一 netID 占用）
func (g *Grid) IsPassable(p Point, netID int) bool {
	cell, err := g.GetCell(p)
	if err != nil {
		return false
	}
	return cell.State == CellEmpty || cell.NetID == netID
}

// Neighbors 返回某个点上下左右四个方向中可通行的邻居
func (g *Grid) Neighbors(p Point, netID int) []Point {
	dirs := []Point{
		{p.X, p.Y - 1}, // 上
		{p.X, p.Y + 1}, // 下
		{p.X - 1, p.Y}, // 左
		{p.X + 1, p.Y}, // 右
	}

	neighbors := make([]Point, 0, 4)
	for _, candidate := range dirs {
		if g.IsPassable(candidate, netID) {
			neighbors = append(neighbors, candidate)
		}
	}
	return neighbors
}
