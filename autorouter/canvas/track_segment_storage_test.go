package canvas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackStorage_IsPassable_EmptyStorage_AlwaysPassable(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 0, Start: 0, End: 100, NetID: 1}))
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 4, Start: 0, End: 100, NetID: 1}))
}

func TestTrackStorage_IsPassable_InvalidTrackID_NotPassable(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	assert.False(t, s.IsPassible(TrackSegment{TrackID: -1, Start: 0, End: 100, NetID: 1}))
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 5, Start: 0, End: 100, NetID: 1}))
}

func TestTrackStorage_IsPassable_SameNetID_AlwaysPassable(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 1}))
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 1}))
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 0, End: 60, NetID: 1}))
}

func TestTrackStorage_IsPassable_DifferentNetID_Overlap_NotPassable(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 1}))
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 2}))
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 5, End: 20, NetID: 2}))
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 40, End: 60, NetID: 2}))
}

func TestTrackStorage_IsPassable_DifferentTrack_NotAffected(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 1}))
	// different track, same range should be passable
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 1, Start: 10, End: 50, NetID: 2}))
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 3, Start: 10, End: 50, NetID: 2}))
}

func TestTrackStorage_IsPassable_DifferentNetID_NotOverlapping_Passable(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 1}))
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 50, End: 100, NetID: 2}))
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 0, End: 10, NetID: 2}))
}

// --- Occupy ---

func TestTrackStorage_Occupy_Basic_Succeeds(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	err := s.Occupy(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 1})
	assert.NoError(t, err)
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 2}))
}

func TestTrackStorage_Occupy_InvalidTrackID_ReturnsError(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	assert.ErrorIs(t, s.Occupy(TrackSegment{TrackID: -1, Start: 0, End: 10, NetID: 1}), ErrInvalidTrackID)
	assert.ErrorIs(t, s.Occupy(TrackSegment{TrackID: 5, Start: 0, End: 10, NetID: 1}), ErrInvalidTrackID)
}

func TestTrackStorage_Occupy_Overlap_DifferentNet_ReturnsError(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 2, Start: 10, End: 50, NetID: 1}))
	assert.ErrorIs(t, s.Occupy(TrackSegment{TrackID: 2, Start: 20, End: 60, NetID: 2}), ErrOverlap)
}

func TestTrackStorage_Occupy_MultipleTracks_Independent(t *testing.T) {
	s := NewTrackSegmentStorage(5, 10)
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 0, Start: 0, End: 100, NetID: 1}))
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 1, Start: 0, End: 100, NetID: 2}))
	require.NoError(t, s.Occupy(TrackSegment{TrackID: 2, Start: 0, End: 100, NetID: 3}))
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 0, Start: 0, End: 100, NetID: 4}))
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 1, Start: 0, End: 100, NetID: 4}))
	assert.False(t, s.IsPassible(TrackSegment{TrackID: 2, Start: 0, End: 100, NetID: 4}))
	assert.True(t, s.IsPassible(TrackSegment{TrackID: 3, Start: 0, End: 100, NetID: 4}))
}

func TestTrackStorage_GetM3TrackWidth_ReturnsCorrectWidth(t *testing.T) {
	s := NewTrackSegmentStorage(5, 46)
	assert.Equal(t, 46, s.GetM3TrackWidth())
}
