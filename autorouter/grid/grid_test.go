package grid_test

import (
	"autorouter/grid"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- New ---

func TestNew_ReturnsSizeCorrectly(t *testing.T) {
	g := grid.New(10, 5)
	assert.Equal(t, 10, g.Width)
	assert.Equal(t, 5, g.Height)
}

func TestNew_AllCellsEmptyByDefault(t *testing.T) {
	g := grid.New(3, 3)
	for x := 0; x < 3; x++ {
		for y := 0; y < 3; y++ {
			cell, err := g.GetCell(grid.Point{X: x, Y: y})
			require.NoError(t, err)
			assert.Equal(t, grid.CellEmpty, cell.State)
		}
	}
}

// --- InBounds ---

func TestInBounds_InsideGrid(t *testing.T) {
	g := grid.New(10, 10)
	cases := []grid.Point{
		{0, 0}, {9, 9}, {5, 5}, {0, 9}, {9, 0},
	}
	for _, p := range cases {
		assert.True(t, g.InBounds(p), "expected %v to be in bounds", p)
	}
}

func TestInBounds_OutsideGrid(t *testing.T) {
	g := grid.New(10, 10)
	cases := []grid.Point{
		{-1, 0}, {0, -1}, {10, 0}, {0, 10}, {-1, -1}, {10, 10},
	}
	for _, p := range cases {
		assert.False(t, g.InBounds(p), "expected %v to be out of bounds", p)
	}
}

// --- SetBlocked / GetCell ---

func TestSetBlocked_CellBecomesBlocked(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 3}

	require.NoError(t, g.SetBlocked(p))

	cell, err := g.GetCell(p)
	require.NoError(t, err)
	assert.Equal(t, grid.CellBlocked, cell.State)
}

func TestSetBlocked_OutOfBounds_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	err := g.SetBlocked(grid.Point{X: 10, Y: 0})
	assert.ErrorIs(t, err, grid.ErrOutOfBounds)
}

func TestGetCell_OutOfBounds_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	_, err := g.GetCell(grid.Point{X: -1, Y: 0})
	assert.ErrorIs(t, err, grid.ErrOutOfBounds)
}

// --- SetOccupied ---

func TestSetOccupied_CellBecomesOccupied(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 1, Y: 1}

	require.NoError(t, g.SetOccupied(p, 42))

	cell, err := g.GetCell(p)
	require.NoError(t, err)
	assert.Equal(t, grid.CellOccupied, cell.State)
	assert.Equal(t, 42, cell.NetID)
}

func TestSetOccupied_OutOfBounds_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	err := g.SetOccupied(grid.Point{X: 99, Y: 0}, 1)
	assert.ErrorIs(t, err, grid.ErrOutOfBounds)
}

// --- IsPassable ---

func TestIsPassable_EmptyCell_IsPassable(t *testing.T) {
	g := grid.New(5, 5)
	assert.True(t, g.IsPassable(grid.Point{X: 2, Y: 2}, 1, 0))
}

func TestIsPassable_BlockedCell_NotPassable(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetBlocked(p))
	assert.False(t, g.IsPassable(p, 1, 0))
}

func TestIsPassable_OccupiedBySameNet_IsPassable(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetOccupied(p, 7))
	assert.True(t, g.IsPassable(p, 7, 0))
}

func TestIsPassable_OccupiedByDifferentNet_NotPassable(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetOccupied(p, 7))
	assert.False(t, g.IsPassable(p, 99, 0))
}

func TestIsPassable_OutOfBounds_NotPassable(t *testing.T) {
	g := grid.New(5, 5)
	assert.False(t, g.IsPassable(grid.Point{X: -1, Y: 0}, 1, 0))
}

// --- Neighbors ---

func TestNeighbors_CenterCell_HasFourNeighbors(t *testing.T) {
	g := grid.New(5, 5)
	neighbors := g.Neighbors(grid.Point{X: 2, Y: 2}, 1, 0)
	assert.Len(t, neighbors, 4)
}

func TestNeighbors_CornerCell_HasTwoNeighbors(t *testing.T) {
	g := grid.New(5, 5)
	neighbors := g.Neighbors(grid.Point{X: 0, Y: 0}, 1, 0)
	assert.Len(t, neighbors, 2)
}

func TestNeighbors_EdgeCell_HasThreeNeighbors(t *testing.T) {
	g := grid.New(5, 5)
	neighbors := g.Neighbors(grid.Point{X: 0, Y: 2}, 1, 0)
	assert.Len(t, neighbors, 3)
}

func TestNeighbors_BlockedNeighbor_Excluded(t *testing.T) {
	g := grid.New(5, 5)
	require.NoError(t, g.SetBlocked(grid.Point{X: 3, Y: 2}))

	neighbors := g.Neighbors(grid.Point{X: 2, Y: 2}, 1, 0)

	assert.Len(t, neighbors, 3)
	assert.NotContains(t, neighbors, grid.Point{X: 3, Y: 2})
}

// --- SetFixed ---

func TestSetFixed_CellBecomesFixed(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}

	require.NoError(t, g.SetFixed(p, 1))

	cell, err := g.GetCell(p)
	require.NoError(t, err)
	assert.Equal(t, grid.CellFixed, cell.State)
	assert.Equal(t, 1, cell.NetID)
}

func TestSetFixed_OutOfBounds_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	err := g.SetFixed(grid.Point{X: 10, Y: 0}, 1)
	assert.ErrorIs(t, err, grid.ErrOutOfBounds)
}

func TestSetFixed_OnBlockedCell_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetBlocked(p))

	err := g.SetFixed(p, 1)
	assert.ErrorIs(t, err, grid.ErrCellBlocked)
}

func TestSetFixed_OnOccupiedCell_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetOccupied(p, 1))

	err := g.SetFixed(p, 1)
	assert.ErrorIs(t, err, grid.ErrCellOccupied)
}

func TestSetFixed_OnAlreadyFixed_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetFixed(p, 1))

	err := g.SetFixed(p, 1)
	assert.ErrorIs(t, err, grid.ErrCellFixed)
}

// --- SetOccupied on fixed cell ---

func TestSetOccupied_OnFixedCellSameNet_ReturnsNil(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetFixed(p, 1))

	err := g.SetOccupied(p, 1)
	assert.NoError(t, err)

	// cell should still be fixed, not occupied
	cell, err := g.GetCell(p)
	require.NoError(t, err)
	assert.Equal(t, grid.CellFixed, cell.State)
}

func TestSetOccupied_OnFixedCellDifferentNet_ReturnsError(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetFixed(p, 1))

	err := g.SetOccupied(p, 2)
	assert.ErrorIs(t, err, grid.ErrNetIDMismatch)
}

// --- ClearOccupied on fixed cell ---

func TestClearOccupied_OnFixedCell_ReturnsNilWithoutClearing(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetFixed(p, 1))

	err := g.ClearOccupied(p, 1)
	assert.NoError(t, err)

	// cell should still be fixed
	cell, err := g.GetCell(p)
	require.NoError(t, err)
	assert.Equal(t, grid.CellFixed, cell.State)
	assert.Equal(t, 1, cell.NetID)
}

// --- IsPassable with fixed cells ---

func TestIsPassable_FixedCellSameNet_IsPassable(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetFixed(p, 1))

	assert.True(t, g.IsPassable(p, 1, 0))
}

func TestIsPassable_FixedCellDifferentNet_NotPassable(t *testing.T) {
	g := grid.New(5, 5)
	p := grid.Point{X: 2, Y: 2}
	require.NoError(t, g.SetFixed(p, 1))

	assert.False(t, g.IsPassable(p, 2, 0))
}

// --- Neighbors with fixed cells ---

func TestNeighbors_FixedCellSameNet_Included(t *testing.T) {
	g := grid.New(5, 5)
	require.NoError(t, g.SetFixed(grid.Point{X: 3, Y: 2}, 1))

	neighbors := g.Neighbors(grid.Point{X: 2, Y: 2}, 1, 0)
	assert.Contains(t, neighbors, grid.Point{X: 3, Y: 2})
}

func TestNeighbors_FixedCellDifferentNet_Excluded(t *testing.T) {
	g := grid.New(5, 5)
	require.NoError(t, g.SetFixed(grid.Point{X: 3, Y: 2}, 99))

	neighbors := g.Neighbors(grid.Point{X: 2, Y: 2}, 1, 0)
	assert.NotContains(t, neighbors, grid.Point{X: 3, Y: 2})
}
