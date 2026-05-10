package session

import (
	"autorouter/common"
	"errors"
	"fmt"
)
import "github.com/samber/lo"

var ErrRouteFailed = errors.New("route failed")

type Point = common.Point
type Path = common.Path
type Net = common.Net

// NetResult holds the outcome of routing a single Net
type NetResult struct {
	Net *Net
	Err error
}

// Grid defines what the session needs from a grid
type Grid interface {
	GetNetID(p Point) (int, error)
	SetOccupied(p Point, netID int) error
	ClearOccupied(p Point, netID int) error
}

// Router defines what the session needs from a router
type Router interface {
	Route(net Net) (Path, error)
	RouteIgnoreOccupied(net Net) (Path, error)
}

// Session holds a grid and a set of nets to route
type Session struct {
	grid   Grid
	router Router
	nets   []*Net
}

// NewSession creates a new Session for the given grid and router
func NewSession(g Grid, r Router) *Session {
	return &Session{grid: g, router: r}
}

// AddNet adds a net to be routed in this session
func (s *Session) AddNet(net Net) {
	s.nets = append(s.nets, &net)
}

const maxIterations = 10

func (s *Session) Route() []NetResult {
	results := make(map[int]NetResult)
	pending := make(map[int]*Net)
	lo.ForEach(s.nets, func(net *Net, _ int) {
		pending[net.ID] = net
	})

	for iteration := 0; iteration < maxIterations && len(pending) > 0; iteration++ {
		for len(pending) > 0 {
			net, ok := s.nextPending(pending)
			if !ok {
				panic("there should be nets remaining in the pending")
			}
			path, err := s.router.Route(*net)
			if err == nil {
				// success: mark grid and record result
				net.Path = path
				s.markOccupied(*net)
				results[net.ID] = NetResult{Net: net, Err: nil}
				delete(pending, net.ID)
				continue
			}

			// failed: find blocking nets
			blockingIDs := s.findBlockingNets(*net)
			if len(blockingIDs) == 0 {
				// no blocking nets, permanently unreachable
				results[net.ID] = NetResult{Net: net, Err: err}
				delete(pending, net.ID)
				continue
			}

			// rip out blocking nets
			for _, blockingID := range blockingIDs {
				blockingNetResult, ok := results[blockingID]
				if !ok {
					panic(fmt.Sprintf("blocking net %d not found in results", blockingID))
				}
				blockingNet := blockingNetResult.Net
				s.clearOccupied(*blockingNet)
				blockingNet.Path = nil
				pending[blockingID] = results[blockingID].Net
				delete(results, blockingID)
			}
			path, err = s.router.Route(*net)
			if err != nil {
				panic("route should success after rip out blocking nets")
			}
			net.Path = path
			s.markOccupied(*net)
			results[net.ID] = NetResult{Net: net, Err: nil}
			delete(pending, net.ID)
			break
		}
	}
	for _, net := range pending {
		_, ok := results[net.ID]
		if ok {
			panic("pending net should not exist in results")
		}
		results[net.ID] = NetResult{Net: net, Err: ErrRouteFailed}
	}
	return s.orderedResults(results)
}

func (s *Session) findBlockingNets(net Net) []int {
	path, err := s.router.RouteIgnoreOccupied(net)
	if err != nil || len(path) == 0 {
		return nil
	}

	seen := make(map[int]bool)
	var blockingIDs []int
	collect := func(p Point) {
		id, err := s.grid.GetNetID(p)
		if err != nil {
			return
		}
		if id != net.ID && !seen[id] {
			seen[id] = true
			blockingIDs = append(blockingIDs, id)
		}
	}
	// first point: check full area
	for dx := -net.HalfWidth; dx <= net.HalfWidth; dx++ {
		for dy := -net.HalfWidth; dy <= net.HalfWidth; dy++ {
			collect(Point{X: path[0].X + dx, Y: path[0].Y + dy})
		}
	}

	// subsequent points: only check new strip
	for i := 1; i < len(path); i++ {
		prev := path[i-1]
		curr := path[i]
		dir := Point{X: curr.X - prev.X, Y: curr.Y - prev.Y}

		if dir.X != 0 {
			stripX := curr.X + dir.X*net.HalfWidth
			for dy := -net.HalfWidth; dy <= net.HalfWidth; dy++ {
				collect(Point{X: stripX, Y: curr.Y + dy})
			}
		} else {
			stripY := curr.Y + dir.Y*net.HalfWidth
			for dx := -net.HalfWidth; dx <= net.HalfWidth; dx++ {
				collect(Point{X: curr.X + dx, Y: stripY})
			}
		}
	}

	return blockingIDs
}

func (s *Session) orderedResults(results map[int]NetResult) []NetResult {
	ordered := make([]NetResult, len(results))
	for i, net := range s.nets {
		ordered[i] = results[net.ID]
		if ordered[i].Err == nil && ordered[i].Net.Path == nil {
			panic("Both Err and Path are nil")
		}
	}
	return ordered
}

func (s *Session) nextPending(pending map[int]*Net) (*Net, bool) {
	for _, net := range s.nets {
		if _, ok := pending[net.ID]; ok {
			return net, true
		}
	}
	return nil, false
}

func (s *Session) markOccupied(net Net) {
	path := net.Path
	if len(path) == 0 {
		return
	}
	s.markArea(path[0], net.ID, net.HalfWidth)
	for i := 1; i < len(path); i++ {
		prev := path[i-1]
		curr := path[i]
		dir := Point{X: curr.X - prev.X, Y: curr.Y - prev.Y}
		s.markStrip(curr, net.ID, net.HalfWidth, dir)
	}
}

func (s *Session) clearOccupied(net Net) {
	path := net.Path
	halfWidth := net.HalfWidth
	if len(path) == 0 {
		return
	}
	s.clearArea(path[0], halfWidth, net.ID)
	for i := 1; i < len(path); i++ {
		prev := path[i-1]
		curr := path[i]
		dir := Point{X: curr.X - prev.X, Y: curr.Y - prev.Y}
		s.clearStrip(curr, halfWidth, net.ID, dir)
	}
}

func (s *Session) markArea(p Point, netID int, halfWidth int) {
	for dx := -halfWidth; dx <= halfWidth; dx++ {
		for dy := -halfWidth; dy <= halfWidth; dy++ {
			expanded := Point{X: p.X + dx, Y: p.Y + dy}
			lo.Must0(s.grid.SetOccupied(expanded, netID))
		}
	}
}

func (s *Session) markStrip(p Point, netID, halfWidth int, dir Point) {
	if dir.X != 0 {
		stripX := p.X + dir.X*halfWidth
		for dy := -halfWidth; dy <= halfWidth; dy++ {
			expanded := Point{X: stripX, Y: p.Y + dy}
			lo.Must0(s.grid.SetOccupied(expanded, netID))
		}
	} else {
		stripY := p.Y + dir.Y*halfWidth
		for dx := -halfWidth; dx <= halfWidth; dx++ {
			expanded := Point{X: p.X + dx, Y: stripY}
			lo.Must0(s.grid.SetOccupied(expanded, netID))
		}
	}
}

func (s *Session) clearArea(p Point, halfWidth, netID int) {
	for dx := -halfWidth; dx <= halfWidth; dx++ {
		for dy := -halfWidth; dy <= halfWidth; dy++ {
			expanded := Point{X: p.X + dx, Y: p.Y + dy}
			lo.Must0(s.grid.ClearOccupied(expanded, netID))
		}
	}
}

func (s *Session) clearStrip(p Point, halfWidth, netID int, dir Point) {
	if dir.X != 0 {
		stripX := p.X + dir.X*halfWidth
		for dy := -halfWidth; dy <= halfWidth; dy++ {
			expanded := Point{X: stripX, Y: p.Y + dy}
			lo.Must0(s.grid.ClearOccupied(expanded, netID))
		}
	} else {
		stripY := p.Y + dir.Y*halfWidth
		for dx := -halfWidth; dx <= halfWidth; dx++ {
			expanded := Point{X: p.X + dx, Y: stripY}
			lo.Must0(s.grid.ClearOccupied(expanded, netID))
		}
	}
}
