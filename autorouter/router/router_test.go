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

	path, err := r.Route(p, p, 1)

	require.NoError(t, err)
	assert.Equal(t, router.Path{p}, path)
}

func TestRoute_AdjacentPoints_ReturnsPathOfTwo(t *testing.T) {
	g := makeGrid(5, 5)
	r := router.NewAStarRouter(g)
	from := grid.Point{X: 0, Y: 0}
	to := grid.Point{X: 1, Y: 0}

	path, err := r.Route(from, to, 1)

	require.NoError(t, err)
	assert.Equal(t, router.Path{from, to}, path)
}

func TestRoute_ClearGrid_ConnectsEndpoints(t *testing.T) {
	g := makeGrid(10, 10)
	r := router.NewAStarRouter(g)
	from := grid.Point{X: 0, Y: 0}
	to := grid.Point{X: 9, Y: 9}

	path, err := r.Route(from, to, 1)

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

	path, err := r.Route(grid.Point{X: 0, Y: 0}, grid.Point{X: 4, Y: 0}, 1)

	require.NoError(t, err)
	assert.Equal(t, grid.Point{X: 0, Y: 0}, path[0])
	assert.Equal(t, grid.Point{X: 4, Y: 0}, path[len(path)-1])
}

func TestRoute_StartIsBlocked_ReturnsError(t *testing.T) {
	g := makeGrid(5, 5)
	from := grid.Point{X: 0, Y: 0}
	g.SetBlocked(from)
	r := router.NewAStarRouter(g)

	_, err := r.Route(from, grid.Point{X: 4, Y: 4}, 1)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

func TestRoute_EndIsBlocked_ReturnsError(t *testing.T) {
	g := makeGrid(5, 5)
	to := grid.Point{X: 4, Y: 4}
	g.SetBlocked(to)
	r := router.NewAStarRouter(g)

	_, err := r.Route(grid.Point{X: 0, Y: 0}, to, 1)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

func TestRoute_NoPathExists_ReturnsError(t *testing.T) {
	g := makeGrid(5, 5)
	// wall off the start point completely
	g.SetBlocked(grid.Point{X: 1, Y: 0})
	g.SetBlocked(grid.Point{X: 0, Y: 1})
	r := router.NewAStarRouter(g)

	_, err := r.Route(grid.Point{X: 0, Y: 0}, grid.Point{X: 4, Y: 4}, 1)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

// --- path validity ---

func TestRoute_PathIsContiguous(t *testing.T) {
	g := makeGrid(10, 10)
	g.SetBlocked(grid.Point{X: 5, Y: 0})
	g.SetBlocked(grid.Point{X: 5, Y: 1})
	g.SetBlocked(grid.Point{X: 5, Y: 2})
	r := router.NewAStarRouter(g)

	path, err := r.Route(grid.Point{X: 0, Y: 0}, grid.Point{X: 9, Y: 0}, 1)

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
	path, err := r.Route(from, to, 1)

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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
