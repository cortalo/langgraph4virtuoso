package router_test

import (
	"autorouter/canvas"
	"autorouter/common"
	"autorouter/router"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCanvas(width, height, trackWidth int) *canvas.Canvas {
	trackCount := height / trackWidth
	return &canvas.Canvas{
		LowerLeft:  common.Point{X: 0, Y: 0},
		UpperRight: common.Point{X: width, Y: height},
		M2Storage:  canvas.NewSegmentStore(common.Point{X: 0, Y: 0}, common.Point{X: width, Y: height}),
		M3Storage:  canvas.NewTrackSegmentStorage(trackCount, trackWidth),
	}
}

func newRouter(c *canvas.Canvas) *router.TwoLayerRouter {
	return router.NewTwoLayerRouter(c)
}

// --- basic routing ---

func TestRoute_ClearCanvas_FindsMidTrack(t *testing.T) {
	// canvas 1000x1000, trackWidth=100 → 10 tracks (0-9)
	// from=(100,100) to=(900,900), midY=500 → midTrack=5
	c := newCanvas(1000, 1000, 100)
	r := newRouter(c)

	trackID, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)

	require.NoError(t, err)
	assert.Equal(t, 5, trackID)
}

func TestRoute_SameY_FindsMidTrack(t *testing.T) {
	// from and to at same Y=200, midY=200 → midTrack=2
	c := newCanvas(1000, 1000, 100)
	r := newRouter(c)

	trackID, err := r.Route(
		common.Point{X: 100, Y: 200},
		common.Point{X: 900, Y: 200},
		1,
	)

	require.NoError(t, err)
	assert.Equal(t, 2, trackID)
}

func TestRoute_OutOfBounds_ReturnsError(t *testing.T) {
	c := newCanvas(1000, 1000, 100)
	r := newRouter(c)

	_, err := r.Route(common.Point{X: -1, Y: 0}, common.Point{X: 900, Y: 900}, 1)
	assert.ErrorIs(t, err, router.ErrOutOfBound)

	_, err = r.Route(common.Point{X: 100, Y: 100}, common.Point{X: 1001, Y: 900}, 1)
	assert.ErrorIs(t, err, router.ErrOutOfBound)
}

// --- obstacle avoidance ---

func TestRoute_MidTrackM3Blocked_FallsBackToNeighbor(t *testing.T) {
	c := newCanvas(1000, 1000, 100)
	require.NoError(t, c.OccupyM3(common.TrackSegment{TrackID: 5, Start: 0, End: 1000, NetID: 99}))
	r := newRouter(c)

	trackID, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)

	require.NoError(t, err)
	assert.True(t, trackID == 4 || trackID == 6)
}

func TestRoute_M2FromBlocked_SkipsTrack(t *testing.T) {
	c := newCanvas(1000, 1000, 100)
	// block M2 at from.X=100, overlapping track 5's Y range [500,600]
	require.NoError(t, c.OccupyM2(common.Segment{
		LowerLeft:  common.Point{X: 100, Y: 500},
		UpperRight: common.Point{X: 101, Y: 600},
		NetID:      99,
	}))
	r := newRouter(c)

	trackID, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)

	require.NoError(t, err)
	assert.NotEqual(t, 5, trackID)
}

func TestRoute_M2ToBlocked_SkipsTrack(t *testing.T) {
	c := newCanvas(1000, 1000, 100)
	// block M2 at to.X=900, overlapping track 5's Y range [500,600]
	require.NoError(t, c.OccupyM2(common.Segment{
		LowerLeft:  common.Point{X: 900, Y: 500},
		UpperRight: common.Point{X: 901, Y: 600},
		NetID:      99,
	}))
	r := newRouter(c)

	trackID, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)

	require.NoError(t, err)
	assert.NotEqual(t, 5, trackID)
}

func TestRoute_AllTracksBlocked_ReturnsError(t *testing.T) {
	c := newCanvas(1000, 1000, 100)
	for i := 0; i < 10; i++ {
		require.NoError(t, c.OccupyM3(common.TrackSegment{TrackID: i, Start: 0, End: 1000, NetID: 99}))
	}
	r := newRouter(c)

	_, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)

	assert.ErrorIs(t, err, router.ErrNoPath)
}

func TestRoute_SameNetID_IgnoresOwnBlocks(t *testing.T) {
	c := newCanvas(1000, 1000, 100)
	require.NoError(t, c.OccupyM3(common.TrackSegment{TrackID: 5, Start: 0, End: 1000, NetID: 1}))
	r := newRouter(c)

	trackID, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)

	require.NoError(t, err)
	assert.Equal(t, 5, trackID)
}

// --- delta expansion ---

func TestRoute_MidTrackBlocked_ExpandsSymmetrically(t *testing.T) {
	c := newCanvas(1000, 1000, 100)
	// block track 5, should try 4 and 6 next
	require.NoError(t, c.OccupyM3(common.TrackSegment{TrackID: 5, Start: 0, End: 1000, NetID: 99}))
	// also block track 6, should fall to 4
	require.NoError(t, c.OccupyM3(common.TrackSegment{TrackID: 6, Start: 0, End: 1000, NetID: 99}))
	r := newRouter(c)

	trackID, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)

	require.NoError(t, err)
	assert.Equal(t, 4, trackID)
}

func TestRoute_MultipleNets_DoNotConflict(t *testing.T) {
	// two nets routed sequentially, second should not overlap first
	c := newCanvas(1000, 1000, 100)
	r := newRouter(c)

	// route net1
	trackID1, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		1,
	)
	require.NoError(t, err)

	// mark net1 as occupied
	require.NoError(t, c.OccupyM3(common.TrackSegment{
		TrackID: trackID1,
		Start:   100,
		End:     900,
		NetID:   1,
	}))

	// route net2 with same endpoints, should find different track
	trackID2, err := r.Route(
		common.Point{X: 100, Y: 100},
		common.Point{X: 900, Y: 900},
		2,
	)
	require.NoError(t, err)
	assert.NotEqual(t, trackID1, trackID2)
}
