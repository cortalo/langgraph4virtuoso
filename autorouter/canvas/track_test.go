package canvas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- IsPassable ---

func TestTrack_IsPassable_EmptyTrack_AlwaysPassable(t *testing.T) {
	track := NewTrackImpl()
	assert.True(t, track.IsPassible(1, 0, 10))
	assert.True(t, track.IsPassible(1, 100, 200))
}

func TestTrack_IsPassable_SameNetID_IsPassable(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 20))
	assert.True(t, track.IsPassible(1, 10, 20))
	assert.True(t, track.IsPassible(1, 5, 15))
	assert.True(t, track.IsPassible(1, 15, 25))
}

func TestTrack_IsPassable_DifferentNetID_NotPassable(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 20))
	assert.False(t, track.IsPassible(2, 10, 20))
	assert.False(t, track.IsPassible(2, 5, 15))  // overlaps left
	assert.False(t, track.IsPassible(2, 15, 25)) // overlaps right
	assert.False(t, track.IsPassible(2, 5, 25))  // contains
	assert.False(t, track.IsPassible(2, 12, 18)) // inside
}

func TestTrack_IsPassable_DifferentNetID_NotOverlapping_IsPassable(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 20))
	assert.True(t, track.IsPassible(2, 0, 10))  // adjacent left, not overlapping
	assert.True(t, track.IsPassible(2, 20, 30)) // adjacent right, not overlapping
	assert.True(t, track.IsPassible(2, 0, 5))   // far left
	assert.True(t, track.IsPassible(2, 25, 30)) // far right
}

func TestTrack_IsPassable_IntervalJustBeforeStart_ExtendsIntoRange(t *testing.T) {
	// this tests the DescendLessOrEqual path
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 5, 15))
	// interval [5,15] starts before 10 but extends into [10,20]
	assert.False(t, track.IsPassible(2, 10, 20))
}

func TestTrack_IsPassable_MultipleIntervals(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 0, 10))
	require.NoError(t, track.Occupy(2, 20, 30))
	require.NoError(t, track.Occupy(3, 40, 50))

	assert.True(t, track.IsPassible(4, 10, 20))  // gap between net1 and net2
	assert.False(t, track.IsPassible(4, 5, 25))  // spans net1 and net2
	assert.False(t, track.IsPassible(1, 5, 25))  // same as net1, but hits net2
	assert.False(t, track.IsPassible(4, 45, 55)) // overlaps net3
}

// --- Occupy ---

func TestTrack_Occupy_BasicOccupy_Succeeds(t *testing.T) {
	track := NewTrackImpl()
	err := track.Occupy(1, 10, 20)
	assert.NoError(t, err)
	assert.False(t, track.IsPassible(2, 10, 20))
}

func TestTrack_Occupy_DifferentNetID_Overlap_ReturnsError(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 20))
	assert.ErrorIs(t, track.Occupy(2, 15, 25), ErrOverlap)
	assert.ErrorIs(t, track.Occupy(2, 5, 15), ErrOverlap)
	assert.ErrorIs(t, track.Occupy(2, 5, 25), ErrOverlap)
	assert.ErrorIs(t, track.Occupy(2, 12, 18), ErrOverlap)
}

func TestTrack_Occupy_SameNetID_Adjacent_MergesIntoOne(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 20))
	require.NoError(t, track.Occupy(1, 20, 30))

	// should be merged into [10, 30]
	assert.False(t, track.IsPassible(2, 10, 30))
	assert.True(t, track.IsPassible(2, 0, 10))
	assert.True(t, track.IsPassible(2, 30, 40))

	// internal structure: only one interval
	count := 0
	track.occupied.Ascend(func(_ interval) bool {
		count++
		return true
	})
	assert.Equal(t, 1, count)
}

func TestTrack_Occupy_SameNetID_Overlapping_MergesIntoOne(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 25))
	require.NoError(t, track.Occupy(1, 20, 30))

	// merged into [10, 30]
	assert.False(t, track.IsPassible(2, 10, 30))
	assert.True(t, track.IsPassible(2, 0, 10))
	assert.True(t, track.IsPassible(2, 30, 40))

	count := 0
	track.occupied.Ascend(func(_ interval) bool {
		count++
		return true
	})
	assert.Equal(t, 1, count)
}

func TestTrack_Occupy_SameNetID_MergesMultiple(t *testing.T) {
	// three separate same-netID intervals, then one big one that merges all
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 0, 10))
	require.NoError(t, track.Occupy(1, 20, 30))
	require.NoError(t, track.Occupy(1, 40, 50))
	require.NoError(t, track.Occupy(1, 5, 45)) // merges all three

	count := 0
	track.occupied.Ascend(func(_ interval) bool {
		count++
		return true
	})
	assert.Equal(t, 1, count)
	assert.False(t, track.IsPassible(2, 0, 50))
	assert.True(t, track.IsPassible(2, 50, 60))
}

func TestTrack_Occupy_SameNetID_ExtendLeft(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 20, 30))
	require.NoError(t, track.Occupy(1, 10, 25)) // extends left

	// merged into [10, 30]
	count := 0
	track.occupied.Ascend(func(iv interval) bool {
		count++
		assert.Equal(t, 10, iv.start)
		assert.Equal(t, 30, iv.end)
		return true
	})
	assert.Equal(t, 1, count)
}

func TestTrack_Occupy_SameNetID_ExtendRight(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 20))
	require.NoError(t, track.Occupy(1, 15, 30)) // extends right

	count := 0
	track.occupied.Ascend(func(iv interval) bool {
		count++
		assert.Equal(t, 10, iv.start)
		assert.Equal(t, 30, iv.end)
		return true
	})
	assert.Equal(t, 1, count)
}

func TestTrack_Occupy_DifferentNets_NonOverlapping_BothSucceed(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 0, 10))
	require.NoError(t, track.Occupy(2, 10, 20))
	require.NoError(t, track.Occupy(3, 20, 30))

	assert.False(t, track.IsPassible(4, 0, 10))
	assert.False(t, track.IsPassible(4, 10, 20))
	assert.False(t, track.IsPassible(4, 20, 30))
	assert.True(t, track.IsPassible(4, 30, 40))
}

func TestTrack_Occupy_SamePosition_SameNetID_NoError(t *testing.T) {
	track := NewTrackImpl()
	require.NoError(t, track.Occupy(1, 10, 20))
	require.NoError(t, track.Occupy(1, 10, 20)) // exact same, should merge cleanly

	count := 0
	track.occupied.Ascend(func(_ interval) bool {
		count++
		return true
	})
	assert.Equal(t, 1, count)
}
