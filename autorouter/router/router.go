package router

import "autorouter/common"

type Point = common.Point
type Path = common.Path

type Grid interface {
	IsPassable(p Point, netID int) bool
	Neighbors(p Point, netID int) []Point
	NeighborsIgnoreOccupied(p Point, netID int) []Point
}
