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

func (r *AStarRouter) Route(from, to Point, netID int) (Path, error) {
	if from == to {
		return Path{from}, nil
	}
	if !r.grid.IsPassable(from, netID) || !r.grid.IsPassable(to, netID) {
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

		for _, neighbor := range r.grid.Neighbors(current.point, netID) {
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
