package router_test

import (
	"autorouter/grid"
	"autorouter/router"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeGrid(w, h int) *grid.Grid {
	return grid.New(w, h)
}

// --- basic routing ---

func TestRoute_SamePoint_ReturnsSinglePointPath(t *testing.T) {
	g := makeGrid(5, 5)
	r := router.NewAStarRouter(g)
	p := grid.Point{X: 2, Y: 2}

	path, err := r.Route(p, p, 1, 0)

	require.NoError(t, err)
	assert.Equal(t, router.Path{p}, path)
}

func TestRoute_AdjacentPoints_ReturnsPathOfTwo(t *testing.T) {
	g := makeGrid(5, 5)
	r := router.NewAStarRouter(g)
	from := grid.Point{X: 0, Y: 0}
	to := grid.Point{X: 1, Y: 0}

	path, err := r.Route(from, to, 1, 0)

	require.NoError(t, err)
	assert.Equal(t, router.Path{from, to}, path)
}

func TestRoute_ClearGrid_ConnectsEndpoints(t *testing.T) {
	g := makeGrid(10, 10)
	r := router.NewAStarRouter(g)
	from := grid.Point{X: 0, Y: 0}
	to := grid.Point{X: 9, Y: 9}

	path, err := r.Route(from, to, 1, 0)

	require.NoError(t, err)
	assert.Equal(t, from, path[0])
	assert.Equal(t, to, path[len(path)-1])
}

// --- obstacles ---

func TestRoute_ObstacleOnDirectLine_FindsDetour(t *testing.T) {
	g := makeGrid(5, 5)
	// block the direct horizontal path
	g.SetBlocked(grid.Point{X: 1, Y: 0})
	g.SetBlocked(grid.Point{X: 2, Y: 0})
	g.SetBlocked(grid.Point{X: 3, Y: 0})
	r := router.NewAStarRouter(g)

	path, err := r.Route(grid.Point{X: 0, Y: 0}, grid.Point{X: 4, Y: 0}, 1, 0)

	require.NoError(t, err)
	assert.Equal(t, grid.Point{X: 0, Y: 0}, path[0])
	assert.Equal(t, grid.Point{X: 4, Y: 0}, path[len(path)-1])
}

func TestRoute_StartIsBlocked_ReturnsError(t *testing.T) {
	g := makeGrid(5, 5)
	from := grid.Point{X: 0, Y: 0}
	g.SetBlocked(from)
	r := router.NewAStarRouter(g)

	_, err := r.Route(from, grid.Point{X: 4, Y: 4}, 1, 0)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

func TestRoute_EndIsBlocked_ReturnsError(t *testing.T) {
	g := makeGrid(5, 5)
	to := grid.Point{X: 4, Y: 4}
	g.SetBlocked(to)
	r := router.NewAStarRouter(g)

	_, err := r.Route(grid.Point{X: 0, Y: 0}, to, 1, 0)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

func TestRoute_NoPathExists_ReturnsError(t *testing.T) {
	g := makeGrid(5, 5)
	// wall off the start point completely
	g.SetBlocked(grid.Point{X: 1, Y: 0})
	g.SetBlocked(grid.Point{X: 0, Y: 1})
	r := router.NewAStarRouter(g)

	_, err := r.Route(grid.Point{X: 0, Y: 0}, grid.Point{X: 4, Y: 4}, 1, 0)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

// --- path validity ---

func TestRoute_PathIsContiguous(t *testing.T) {
	g := makeGrid(10, 10)
	g.SetBlocked(grid.Point{X: 5, Y: 0})
	g.SetBlocked(grid.Point{X: 5, Y: 1})
	g.SetBlocked(grid.Point{X: 5, Y: 2})
	r := router.NewAStarRouter(g)

	path, err := r.Route(grid.Point{X: 0, Y: 0}, grid.Point{X: 9, Y: 0}, 1, 0)

	require.NoError(t, err)
	for i := 1; i < len(path); i++ {
		dx := abs(path[i].X - path[i-1].X)
		dy := abs(path[i].Y - path[i-1].Y)
		assert.Equal(t, 1, dx+dy, "path step %d->%d is not adjacent", i-1, i)
	}
}

func TestRoute_MazeWithObstacles(t *testing.T) {
	// Grid 7x5, routing from (0,2) to (6,2)
	// A vertical wall at x=3, blocking y=0..3, open only at y=4
	//
	//  0 1 2 3 4 5 6
	//  . . . # . . .   y=0
	//  . . . # . . .   y=1
	//  S . . # . . E   y=2
	//  . . . # . . .   y=3
	//  . . . . . . .   y=4  ← only gap
	//
	// only valid path must go through y=4
	// minimum length = 11
	// (0,2)→(0,3)→(0,4)→(1,4)→(2,4)→(3,4)→(4,4)→(5,4)→(6,4)→(6,3)→(6,2) = wrong, let me recalc
	// actually: down to y=4, across, back up = 2+6+2 = 10 moves, length 11

	g := grid.New(5, 7)
	from := grid.Point{X: 0, Y: 2}
	to := grid.Point{X: 6, Y: 2}

	wall := []grid.Point{
		{X: 3, Y: 0},
		{X: 3, Y: 1},
		{X: 3, Y: 2},
		{X: 3, Y: 3},
	}
	for _, p := range wall {
		require.NoError(t, g.SetBlocked(p))
	}

	r := router.NewAStarRouter(g)
	path, err := r.Route(from, to, 1, 0)

	require.NoError(t, err)

	// starts and ends correctly
	assert.Equal(t, from, path[0])
	assert.Equal(t, to, path[len(path)-1])

	// minimum possible length given the wall
	assert.Equal(t, 11, len(path))

	// path is contiguous
	for i := 1; i < len(path); i++ {
		dx := abs(path[i].X - path[i-1].X)
		dy := abs(path[i].Y - path[i-1].Y)
		assert.Equal(t, 1, dx+dy, "step %d->%d is not adjacent", i-1, i)
	}

	// no point in path is blocked
	blockedSet := make(map[grid.Point]bool)
	for _, p := range wall {
		blockedSet[p] = true
	}
	for _, p := range path {
		assert.False(t, blockedSet[p], "path goes through blocked cell %v", p)
	}
}

// --- line width routing ---

func TestRoute_WithLineWidth_ClearPath(t *testing.T) {
	// Grid 10x10, line width 3 cells (half=1)
	// routing from (1,1) to (1,8), enough space on all sides
	g := grid.New(10, 10)
	r := router.NewAStarRouter(g)

	path, err := r.Route(
		grid.Point{X: 1, Y: 1},
		grid.Point{X: 1, Y: 8},
		1, 1, // netID=1, halfWidth=1
	)

	require.NoError(t, err)
	assert.Equal(t, grid.Point{X: 1, Y: 1}, path[0])
	assert.Equal(t, grid.Point{X: 1, Y: 8}, path[len(path)-1])
}

func TestRoute_WithLineWidth_ObstacleTooClose_FindsDetour(t *testing.T) {
	// Grid 10x10, line width 3 cells (half=1)
	// obstacle at X=5, Y=3..7
	// S=(5,1) and E=(5,8) are 1 cell away from grid edge to allow halfWidth=1
	//
	//     Y=0 Y=1 Y=2 Y=3 Y=4 Y=5 Y=6 Y=7 Y=8 Y=9
	// X=0  .   .   .   .   .   .   .   .   .   .
	// X=1  .   .   .   .   .   .   .   .   .   .
	// X=2  .   .   .   .   .   .   .   .   .   .
	// X=3  .   .   .   .   .   .   .   .   .   .
	// X=4  .   .   .   .   .   .   .   .   .   .
	// X=5  .   S   .   #   #   #   #   .   E   .
	// X=6  .   .   .   .   .   .   .   .   .   .
	// X=7  .   .   .   .   .   .   .   .   .   .
	// X=8  .   .   .   .   .   .   .   .   .   .
	// X=9  .   .   .   .   .   .   .   .   .   .
	//
	// with halfWidth=1, center line must stay 1 cell away from obstacle
	// X=4 and X=6 are also effectively blocked for center line
	// must detour to X=3 or X=7
	g := grid.New(10, 10)
	for y := 3; y <= 6; y++ {
		require.NoError(t, g.SetBlocked(grid.Point{X: 5, Y: y}))
	}
	r := router.NewAStarRouter(g)

	from := grid.Point{X: 5, Y: 1}
	to := grid.Point{X: 5, Y: 8}
	path, err := r.Route(from, to, 1, 1)

	require.NoError(t, err)
	assert.Equal(t, from, path[0])
	assert.Equal(t, to, path[len(path)-1])

	// center line must not come within halfWidth of any obstacle
	for _, p := range path {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				neighbor := grid.Point{X: p.X + dx, Y: p.Y + dy}
				if !g.InBounds(neighbor) {
					continue
				}
				cell, err := g.GetCell(neighbor)
				require.NoError(t, err)
				assert.NotEqual(t, grid.CellBlocked, cell.State,
					"path at %v comes within halfWidth of blocked cell %v", p, neighbor)
			}
		}
	}
}

func TestRoute_WithLineWidth_NoRoomToPass_ReturnsError(t *testing.T) {
	// Grid 5x5, line width 3 cells (half=1)
	// obstacle wall at X=2 with only 1 cell gap, not enough for line width 3
	//
	//     Y=0 Y=1 Y=2 Y=3 Y=4
	// X=0  S   .   .   .   .
	// X=1  .   .   #   .   .
	// X=2  .   .   #   .   .
	// X=3  .   .   #   .   .
	// X=4  .   .   .   .   E
	//
	// gap is only 2 cells wide on each side, center line needs 1 clear cell on each side
	// X=1 col is only 1 cell from obstacle, not passable with halfWidth=1
	g := grid.New(5, 5)
	for x := 1; x <= 3; x++ {
		require.NoError(t, g.SetBlocked(grid.Point{X: x, Y: 2}))
	}
	r := router.NewAStarRouter(g)

	_, err := r.Route(
		grid.Point{X: 0, Y: 0},
		grid.Point{X: 4, Y: 4},
		1, 1,
	)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
