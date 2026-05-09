package session

import (
	"autorouter/common"
	"errors"
)
import "github.com/samber/lo"

var ErrRouteFailed = errors.New("route failed")

type Point = common.Point
type Path = common.Path

// Net represents a pair of endpoints that need to be connected
type Net struct {
	ID   int
	From Point
	To   Point
}

// NetResult holds the outcome of routing a single Net
type NetResult struct {
	Net  Net
	Path Path
	Err  error
}

// Grid defines what the session needs from a grid
type Grid interface {
	GetNetID(p Point) (int, error)
	SetOccupied(p Point, netID int) error
	ClearOccupied(p Point, netID int) error
}

// Router defines what the session needs from a router
type Router interface {
	Route(from, to Point, netID, halfWidth int) (Path, error)
	RouteIgnoreOccupied(from, to Point, netID, halfWidth int) (Path, error)
}

// Session holds a grid and a set of nets to route
type Session struct {
	grid   Grid
	router Router
	nets   []Net
}

// NewSession creates a new Session for the given grid and router
func NewSession(g Grid, r Router) *Session {
	return &Session{grid: g, router: r}
}

// AddNet adds a net to be routed in this session
func (s *Session) AddNet(net Net) {
	s.nets = append(s.nets, net)
}

const maxIterations = 10

func (s *Session) Route() []NetResult {
	results := make(map[int]NetResult)
	routed := make(map[int]Path)
	pending := make(map[int]Net)
	lo.ForEach(s.nets, func(net Net, _ int) {
		pending[net.ID] = net
	})

	for iteration := 0; iteration < maxIterations && len(pending) > 0; iteration++ {
		for len(pending) > 0 {
			net, _ := lo.First(lo.Values(pending))
			path, err := s.router.Route(net.From, net.To, net.ID, 0)
			if err == nil {
				// success: mark grid and record result
				lo.ForEach(path, func(p Point, _ int) {
					lo.Must0(s.grid.SetOccupied(p, net.ID))
				})
				results[net.ID] = NetResult{Net: net, Path: path, Err: nil}
				routed[net.ID] = path
				delete(pending, net.ID)
				continue
			}

			// failed: find blocking nets
			blockingIDs := s.findBlockingNets(net)
			if len(blockingIDs) == 0 {
				// no blocking nets, permanently unreachable
				results[net.ID] = NetResult{Net: net, Path: nil, Err: err}
				delete(pending, net.ID)
				continue
			}

			// rip out blocking nets
			for _, blockingID := range blockingIDs {
				if path, ok := routed[blockingID]; ok {
					lo.ForEach(path, func(p Point, _ int) {
						lo.Must0(s.grid.ClearOccupied(p, blockingID))
					})
					pending[blockingID] = results[blockingID].Net
					delete(results, blockingID)
					delete(routed, blockingID)
				}
			}
			path, err = s.router.Route(net.From, net.To, net.ID, 0)
			if err != nil {
				results[net.ID] = NetResult{Net: net, Path: nil, Err: err}
			} else {
				lo.ForEach(path, func(p Point, _ int) {
					lo.Must0(s.grid.SetOccupied(p, net.ID))
				})
				results[net.ID] = NetResult{Net: net, Path: path, Err: nil}
				routed[net.ID] = path
			}
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
	path, err := s.router.RouteIgnoreOccupied(net.From, net.To, net.ID, 0)
	if err != nil {
		return nil
	}
	netIDs := lo.FilterMap(path, func(p Point, _ int) (int, bool) {
		netID, err := s.grid.GetNetID(p)
		if err != nil || netID == net.ID {
			return 0, false
		}
		return netID, true
	})
	return lo.Uniq(netIDs)
}

func (s *Session) orderedResults(results map[int]NetResult) []NetResult {
	ordered := make([]NetResult, len(results))
	for i, net := range s.nets {
		ordered[i] = results[net.ID]
	}
	return ordered
}
