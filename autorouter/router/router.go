package router

import "autorouter/common"

type Point = common.Point

// Grid defines what the router needs from a grid
type Grid interface {
	IsPassable(p Point, netID int) bool
	Neighbors(p Point, netID int) []Point
}

// Path is an ordered sequence of points from start to end
type Path []Point

// Router finds a path between two points on a grid
type Router interface {
	Route(from, to Point, netID int) (Path, error)
}
