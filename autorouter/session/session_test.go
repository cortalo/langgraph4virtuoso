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
	assert.Equal(t, session.Point{X: 0, Y: 0}, results[0].Net.Path[0])
	assert.Equal(t, session.Point{X: 4, Y: 4}, results[0].Net.Path[len(results[0].Net.Path)-1])
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
	for _, p := range results[0].Net.Path {
		net1Cells[p] = true
	}
	for _, p := range results[1].Net.Path {
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

// --- line width ---

func assertValidEndpoints(t *testing.T, g *grid.Grid, net session.Net) {
	t.Helper()
	hw := net.HalfWidth
	assert.True(t, net.From.X >= hw && net.From.X < g.Height-hw &&
		net.From.Y >= hw && net.From.Y < g.Width-hw,
		"From %v is too close to edge for halfWidth=%d", net.From, hw)
	assert.True(t, net.To.X >= hw && net.To.X < g.Height-hw &&
		net.To.Y >= hw && net.To.Y < g.Width-hw,
		"To %v is too close to edge for halfWidth=%d", net.To, hw)
}

func TestSession_SingleNet_WithHalfWidth_RoutesSuccessfully(t *testing.T) {
	// Grid 10x10, net with halfWidth=1 (line width 3)
	// start and end are at least 1 cell from edge
	//
	//      Y=0 Y=1 Y=2 Y=3 Y=4 Y=5 Y=6 Y=7 Y=8 Y=9
	// X=1   .   S   .   .   .   .   .   .   .   .
	// ...
	// X=8   .   .   .   .   .   .   .   E   .   .
	g, s := makeSession(10, 10)
	net := session.Net{ID: 1, From: session.Point{X: 1, Y: 1}, To: session.Point{X: 8, Y: 8}, HalfWidth: 1}
	assertValidEndpoints(t, g, net)
	s.AddNet(net)

	results := s.Route()

	require.Len(t, results, 1)
	assert.NoError(t, results[0].Err)
	assert.Equal(t, session.Point{X: 1, Y: 1}, results[0].Net.Path[0])
	assert.Equal(t, session.Point{X: 8, Y: 8}, results[0].Net.Path[len(results[0].Net.Path)-1])
}

func TestSession_TwoNets_WithHalfWidth_SecondRoutesAroundFirst(t *testing.T) {
	// Grid 12x12, both nets halfWidth=1
	// net1 routes vertically through middle, occupying X=4..6
	// net2 routes horizontally, must detour around net1 occupied area
	//
	//      Y=0 Y=1 Y=2 Y=3 Y=4 Y=5 Y=6 Y=7 Y=8 Y=9 Y=10 Y=11
	// X=0   .   .   .   .   .   .   .   .   .   .   .    .
	// X=1   .   .   .   .   .   .   .   .   .   .   .    .
	// X=2   .   .   .   .   .   .   .   .   .   .   .    .
	// X=3   .   .   .   .   .   .   .   .   .   .   .    .
	// X=4   .   .   .   .   .   S   .   .   .   .   .    .    S = net2 start (2,5)
	// X=5   .   .   .   .   .   .   .   .   .   .   .    .
	// X=6   .   ■   ■   ■   ■   ■   ■   ■   ■   ■   ■    .    net1 expanded (halfWidth=1)
	// X=7   .   A   ■   ■   ■   ■   ■   ■   ■   ■   B    .    net1 center, A=(5,1) B=(5,10)
	// X=8   .   ■   ■   ■   ■   ■   ■   ■   ■   ■   ■    .    net1 expanded (halfWidth=1)
	// X=9   .   .   .   .   .   .   .   .   .   .   .    .
	// X=10  .   .   .   .   .   .   .   .   .   .   .    .
	// X=11  .   .   .   .   .   E   .   .   .   .   .    .    E = net2 end (9,5)
	// X=12  .   .   .   .   .   .   .   .   .   .   .    .
	// X=13  .   .   .   .   .   .   .   .   .   .   .    .
	// X=14  .   .   .   .   .   .   .   .   .   .   .    .
	// X=15  .   .   .   .   .   .   .   .   .   .   .    .
	//
	// net2 center at (2,5)->(9,5) crosses net1 occupied area at X=4..6
	// net2 must detour to X=3 or X=7
	g, s := makeSession(12, 16)
	net1 := session.Net{ID: 1, From: session.Point{X: 7, Y: 1}, To: session.Point{X: 7, Y: 10}, HalfWidth: 1}
	net2 := session.Net{ID: 2, From: session.Point{X: 4, Y: 5}, To: session.Point{X: 11, Y: 5}, HalfWidth: 1}
	assertValidEndpoints(t, g, net1)
	assertValidEndpoints(t, g, net2)
	s.AddNet(net1)
	s.AddNet(net2)

	results := s.Route()

	require.Len(t, results, 2)
	assert.NoError(t, results[0].Err)
	assert.NoError(t, results[1].Err)

	// net2 expanded area must not overlap net1 expanded area
	net1Cells := make(map[session.Point]bool)
	for _, p := range results[0].Net.Path {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				net1Cells[session.Point{X: p.X + dx, Y: p.Y + dy}] = true
			}
		}
	}
	for _, p := range results[1].Net.Path {
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				expanded := session.Point{X: p.X + dx, Y: p.Y + dy}
				assert.False(t, net1Cells[expanded],
					"net2 expanded area overlaps net1 at %v", expanded)
			}
		}
	}
}

func TestSession_TwoNets_WithHalfWidth_RipAndReroute(t *testing.T) {
	// same crossing scenario but order forces rip-and-reroute
	// net1 routes horizontally first, blocking net2 vertical path
	// rip-and-reroute should resolve by ripping net1 and retrying
	g, s := makeSession(12, 16)
	net1 := session.Net{ID: 1, From: session.Point{X: 4, Y: 5}, To: session.Point{X: 11, Y: 5}, HalfWidth: 1}
	net2 := session.Net{ID: 2, From: session.Point{X: 7, Y: 1}, To: session.Point{X: 7, Y: 10}, HalfWidth: 1}
	assertValidEndpoints(t, g, net1)
	assertValidEndpoints(t, g, net2)
	s.AddNet(net1)
	s.AddNet(net2)

	results := s.Route()

	require.Len(t, results, 2)
	assert.NoError(t, results[0].Err)
	assert.NoError(t, results[1].Err)
}

func TestSession_Net_WithHalfWidth_NoRoom_ReturnsError(t *testing.T) {
	// Grid 7x7, obstacle wall at Y=3 leaves gap too narrow for halfWidth=1
	//
	//      Y=0 Y=1 Y=2 Y=3 Y=4 Y=5 Y=6
	// X=1   .   S   .   .   .   .   .
	// X=2   .   .   .   #   .   .   .
	// X=3   .   .   .   #   .   .   .
	// X=4   .   .   .   #   .   .   .
	// X=5   .   .   .   .   .   E   .
	//
	// gap on each side is only 2 cells wide (Y=0..2 and Y=4..6)
	// with halfWidth=1 center needs 1 clear cell on each side
	// Y=1 and Y=5 are only 1 cell from obstacle, not enough room
	g, s := makeSession(7, 7)
	for x := 2; x <= 4; x++ {
		require.NoError(t, g.SetBlocked(grid.Point{X: x, Y: 3}))
	}
	net := session.Net{ID: 1, From: session.Point{X: 1, Y: 1}, To: session.Point{X: 5, Y: 5}, HalfWidth: 1}
	assertValidEndpoints(t, g, net)
	s.AddNet(net)

	results := s.Route()

	require.Len(t, results, 1)
	assert.Error(t, results[0].Err)
}
