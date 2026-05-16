package canvas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStore() *SegmentStorageImpl {
	return NewSegmentStore(Point{0, 0}, Point{100, 100})
}

func seg(x1, y1, x2, y2, netID int) Segment {
	return Segment{LowerLeft: Point{x1, y1}, UpperRight: Point{x2, y2}, NetID: netID}
}

// --- IsPassible ---

func TestSegmentStore_IsPassible_EmptyStore_AlwaysPassable(t *testing.T) {
	s := newStore()
	assert.True(t, s.IsPassible(seg(10, 10, 20, 20, 1)))
	assert.True(t, s.IsPassible(seg(50, 50, 60, 60, 2)))
}

func TestSegmentStore_IsPassible_OutOfBounds_NotPassable(t *testing.T) {
	s := newStore()
	assert.False(t, s.IsPassible(seg(-1, 0, 10, 10, 1)))
	assert.False(t, s.IsPassible(seg(0, 0, 101, 10, 1)))
	assert.False(t, s.IsPassible(seg(0, -1, 10, 10, 1)))
	assert.False(t, s.IsPassible(seg(0, 0, 10, 101, 1)))
}

func TestSegmentStore_IsPassible_SameNetID_AlwaysPassable(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(10, 10, 20, 20, 1)))
	assert.True(t, s.IsPassible(seg(10, 10, 20, 20, 1)))
	assert.True(t, s.IsPassible(seg(5, 5, 25, 25, 1)))
	assert.True(t, s.IsPassible(seg(15, 15, 30, 30, 1)))
}

func TestSegmentStore_IsPassible_DifferentNetID_Overlap_NotPassable(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(10, 10, 20, 20, 1)))
	assert.False(t, s.IsPassible(seg(10, 10, 20, 20, 2))) // exact same
	assert.False(t, s.IsPassible(seg(5, 5, 15, 15, 2)))   // overlaps lower-left
	assert.False(t, s.IsPassible(seg(15, 15, 25, 25, 2))) // overlaps upper-right
	assert.False(t, s.IsPassible(seg(5, 5, 25, 25, 2)))   // contains
	assert.False(t, s.IsPassible(seg(12, 12, 18, 18, 2))) // inside
}

func TestSegmentStore_IsPassible_DifferentNetID_NotOverlapping_Passable(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(10, 10, 20, 20, 1)))
	assert.True(t, s.IsPassible(seg(20, 10, 30, 20, 2))) // adjacent right
	assert.True(t, s.IsPassible(seg(0, 10, 10, 20, 2)))  // adjacent left
	assert.True(t, s.IsPassible(seg(10, 20, 20, 30, 2))) // adjacent top
	assert.True(t, s.IsPassible(seg(10, 0, 20, 10, 2)))  // adjacent bottom
	assert.True(t, s.IsPassible(seg(30, 30, 40, 40, 2))) // far away
}

func TestSegmentStore_IsPassible_MultipleSegments(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(0, 0, 10, 10, 1)))
	require.NoError(t, s.Occupy(seg(20, 20, 30, 30, 2)))
	require.NoError(t, s.Occupy(seg(40, 40, 50, 50, 3)))

	assert.True(t, s.IsPassible(seg(10, 0, 20, 10, 4)))   // gap between seg1 and seg2
	assert.False(t, s.IsPassible(seg(5, 5, 25, 25, 4)))   // spans seg1 and seg2
	assert.False(t, s.IsPassible(seg(45, 45, 55, 55, 4))) // overlaps seg3 and out of bounds
}

// --- Occupy ---

func TestSegmentStore_Occupy_Basic_Succeeds(t *testing.T) {
	s := newStore()
	err := s.Occupy(seg(10, 10, 20, 20, 1))
	assert.NoError(t, err)
	assert.False(t, s.IsPassible(seg(10, 10, 20, 20, 2)))
}

func TestSegmentStore_Occupy_OutOfBounds_ReturnsError(t *testing.T) {
	s := newStore()
	assert.ErrorIs(t, s.Occupy(seg(-1, 0, 10, 10, 1)), ErrOutOfBounds)
	assert.ErrorIs(t, s.Occupy(seg(0, 0, 101, 10, 1)), ErrOutOfBounds)
}

func TestSegmentStore_Occupy_Overlap_DifferentNet_ReturnsError(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(10, 10, 20, 20, 1)))
	assert.ErrorIs(t, s.Occupy(seg(15, 15, 25, 25, 2)), ErrOverlap)
	assert.ErrorIs(t, s.Occupy(seg(5, 5, 15, 15, 2)), ErrOverlap)
	assert.ErrorIs(t, s.Occupy(seg(5, 5, 25, 25, 2)), ErrOverlap)
}

func TestSegmentStore_Occupy_SameNet_Overlap_Succeeds(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(10, 10, 20, 20, 1)))
	assert.NoError(t, s.Occupy(seg(15, 15, 25, 25, 1)))
}

func TestSegmentStore_Occupy_Adjacent_DifferentNet_Succeeds(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(10, 10, 20, 20, 1)))
	assert.NoError(t, s.Occupy(seg(20, 10, 30, 20, 2))) // touching edge, not overlapping
}

func TestSegmentStore_Occupy_MultipleDifferentNets_NonOverlapping_AllSucceed(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Occupy(seg(0, 0, 10, 10, 1)))
	require.NoError(t, s.Occupy(seg(10, 0, 20, 10, 2)))
	require.NoError(t, s.Occupy(seg(20, 0, 30, 10, 3)))
	assert.False(t, s.IsPassible(seg(0, 0, 30, 10, 4)))
}
