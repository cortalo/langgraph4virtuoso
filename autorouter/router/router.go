package router

import "autorouter/common"

type Point = common.Point
type Path = common.Path

type Grid interface {
	IsPassable(p Point, netID, halfWidth int) bool
	Neighbors(p Point, netID, halfWidth int) []Point
	NeighborsIgnoreOccupied(p Point, netID, halfWidth int) []Point
}
