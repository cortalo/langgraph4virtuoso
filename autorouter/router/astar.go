package router

import "autorouter/common"

type node struct {
	point  Point
	g      int
	f      int
	parent *node
}

func manhattan(a, b Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

type AStarRouter struct {
	grid Grid
}

func NewAStarRouter(g Grid) *AStarRouter {
	return &AStarRouter{grid: g}
}

func (r *AStarRouter) Route(net Net) (Path, error) {
	return r.route(net, false)
}

func (r *AStarRouter) RouteIgnoreOccupied(net Net) (Path, error) {
	return r.route(net, true)
}

func reconstructPath(n *node) Path {
	var path Path
	for n != nil {
		path = append(path, n.point)
		n = n.parent
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func (r *AStarRouter) route(net Net, ignoreOccupied bool) (Path, error) {
	from := net.From
	to := net.To
	netID := net.ID
	halfWidth := net.HalfWidth
	if from == to {
		return Path{from}, nil
	}
	if !r.grid.IsPassable(from, netID, halfWidth) || !r.grid.IsPassable(to, netID, halfWidth) {
		return nil, ErrNoPath
	}

	closed := make(map[Point]bool)
	open := common.NewMinHeap[*node](func(a, b *node) bool {
		return a.f < b.f
	})
	open.PushItem(&node{
		point: from,
		g:     0,
		f:     manhattan(from, to),
	})

	for open.Len() > 0 {
		current := open.PopItem()
		if current.point == to {
			return reconstructPath(current), nil
		}
		if closed[current.point] {
			continue
		}
		closed[current.point] = true

		var neighbors []Point
		if ignoreOccupied {
			neighbors = r.grid.NeighborsIgnoreOccupied(current.point, netID, halfWidth)
		} else {
			neighbors = r.grid.Neighbors(current.point, netID, halfWidth)
		}
		for _, neighbor := range neighbors {
			if closed[neighbor] {
				continue
			}
			g := current.g + 1
			open.PushItem(&node{
				point:  neighbor,
				g:      g,
				f:      g + manhattan(neighbor, to),
				parent: current,
			})
		}
	}
	return nil, ErrNoPath
}
