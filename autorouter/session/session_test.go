package session_test

import (
	"autorouter/grid"
	"autorouter/router"
	"autorouter/session"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSession(w, h int) (*grid.Grid, *session.Session) {
	g := grid.New(w, h)
	r := router.NewAStarRouter(g)
	s := session.NewSession(g, r)
	return g, s
}

// --- single net ---

func TestSession_SingleNet_RoutesSuccessfully(t *testing.T) {
	_, s := makeSession(5, 5)
	s.AddNet(session.Net{ID: 1, From: session.Point{X: 0, Y: 0}, To: session.Point{X: 4, Y: 4}})

	results := s.Route()

	require.Len(t, results, 1)
	assert.NoError(t, results[0].Err)
	assert.Equal(t, session.Point{X: 0, Y: 0}, results[0].Path[0])
	assert.Equal(t, session.Point{X: 4, Y: 4}, results[0].Path[len(results[0].Path)-1])
}

// --- multiple nets ---
func TestSession_TwoNets_SecondRoutesAroundFirst(t *testing.T) {
	// Grid 5x3 (width=5, height=3), two nets cross in the middle
	//
	//     Y=0 Y=1 Y=2 Y=3 Y=4
	// X=0  .   .   S   .   .    S = net2 start (0,2)
	// X=1  A   .   +   .   B    A = net1 start (1,0), B = net1 end (1,4), + = conflict (1,2)
	// X=2  .   .   E   .   .    E = net2 end (2,2)
	// X=3  .   .   .   .   .
	//
	// if net1 routes first along X=1, it occupies (1,2) and blocks net2
	// rip-and-reroute should detect the conflict and retry with net2 first
	_, s := makeSession(5, 4)
	s.AddNet(session.Net{ID: 1, From: session.Point{X: 1, Y: 0}, To: session.Point{X: 1, Y: 4}})
	s.AddNet(session.Net{ID: 2, From: session.Point{X: 0, Y: 2}, To: session.Point{X: 2, Y: 2}})

	results := s.Route()

	require.Len(t, results, 2)
	assert.NoError(t, results[0].Err)
	assert.NoError(t, results[1].Err)

	// net2 path must not contain any cell occupied by net1
	net1Cells := make(map[session.Point]bool)
	for _, p := range results[0].Path {
		net1Cells[p] = true
	}
	for _, p := range results[1].Path {
		assert.False(t, net1Cells[p], "net2 path crosses net1 at %v", p)
	}
}

// --- failure cases ---
func TestSession_TwoNets_NoConflict_BothSucceed(t *testing.T) {
	// two nets on opposite sides, no interference
	//
	//     Y=0 Y=1 Y=2 Y=3 Y=4
	// X=0  A   .   .   .   B    net1: (0,0) -> (0,4)
	// X=1  .   .   .   .   .
	// X=2  C   .   .   .   D    net2: (2,0) -> (2,4)
	_, s := makeSession(5, 3)
	s.AddNet(session.Net{ID: 1, From: session.Point{X: 0, Y: 0}, To: session.Point{X: 0, Y: 4}})
	s.AddNet(session.Net{ID: 2, From: session.Point{X: 2, Y: 0}, To: session.Point{X: 2, Y: 4}})

	results := s.Route()

	require.Len(t, results, 2)
	assert.NoError(t, results[0].Err)
	assert.NoError(t, results[1].Err)
}

func TestSession_SecondNet_Blocked_ReturnsError(t *testing.T) {
	// net1 completely walls off net2's only path
	//
	//     Y=0 Y=1 Y=2 Y=3 Y=4
	// X=0  #   #   A   #   #    A = net2 start (0,2), surrounded on all sides
	// X=1  #   S   #   E   #    net1: (1,1)->(1,3), occupies (1,2) sealing off (0,2)
	// X=2  .   .   .   .   .
	// X=3  .   .   .   .   .
	// X=4  .   .   D   .   .    D = net2 end (4,2)
	g, s := makeSession(5, 5)

	// static obstacles seal (0,2) from above and below
	require.NoError(t, g.SetBlocked(grid.Point{X: 0, Y: 0}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 0, Y: 1}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 0, Y: 3}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 0, Y: 4}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 1, Y: 0}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 1, Y: 4}))

	// net1 routes through (1,1)->(1,2)->(1,3), sealing off (0,2)
	s.AddNet(session.Net{ID: 1, From: session.Point{X: 1, Y: 1}, To: session.Point{X: 1, Y: 3}})
	// net2 starts at (0,2) which is now completely surrounded
	s.AddNet(session.Net{ID: 2, From: session.Point{X: 0, Y: 2}, To: session.Point{X: 4, Y: 2}})

	results := s.Route()

	require.Len(t, results, 2)
	assert.NoError(t, results[0].Err)
	assert.Error(t, results[1].Err)
}

func TestSession_OneNetFails_OthersContinue(t *testing.T) {
	// net2 fails, but net1 and net3 should still succeed
	g, s := makeSession(5, 5)

	// wall off net2 completely
	require.NoError(t, g.SetBlocked(grid.Point{X: 1, Y: 2}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 2, Y: 1}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 2, Y: 3}))
	require.NoError(t, g.SetBlocked(grid.Point{X: 3, Y: 2}))

	s.AddNet(session.Net{ID: 1, From: session.Point{X: 0, Y: 0}, To: session.Point{X: 4, Y: 0}})
	s.AddNet(session.Net{ID: 2, From: session.Point{X: 2, Y: 2}, To: session.Point{X: 4, Y: 4}}) // trapped
	s.AddNet(session.Net{ID: 3, From: session.Point{X: 0, Y: 4}, To: session.Point{X: 4, Y: 4}})

	results := s.Route()

	require.Len(t, results, 3)
	assert.NoError(t, results[0].Err)
	assert.Error(t, results[1].Err)
	assert.NoError(t, results[2].Err)
}
