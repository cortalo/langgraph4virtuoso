package router

import (
	"autorouter/common"
)

type Point = common.Point
type Path = common.TwoLayerPath
type Segment = common.Segment
type TrackSegment = common.TrackSegment

const M2Width = 1

type Canvas interface {
	Inbound(p Point) bool
	IsPassibleM2(seg Segment) bool
	IsPassibleM3(seg TrackSegment) bool
	GetLowerLeft() Point
	GetUpperRight() Point
	GetM3TrackWidth() int
}

type TwoLayerRouter struct {
	canvas Canvas
}

func NewTwoLayerRouter(c Canvas) *TwoLayerRouter {
	return &TwoLayerRouter{canvas: c}
}

func (r *TwoLayerRouter) Route(from, to Point, netID int) (int, error) {
	if !r.canvas.Inbound(from) || !r.canvas.Inbound(to) {
		return 0, ErrOutOfBound
	}
	midY := (from.Y + to.Y) / 2
	lowerLeft := r.canvas.GetLowerLeft()
	upperRight := r.canvas.GetUpperRight()
	midTrack := (midY - lowerLeft.Y) / r.canvas.GetM3TrackWidth()
	maxTrack := (upperRight.Y-lowerLeft.Y)/r.canvas.GetM3TrackWidth() - 1
	for delta := 0; (midTrack+delta <= maxTrack) || (midTrack-delta >= 0); delta++ {
		if r.tryTrack(from, to, netID, midTrack+delta) {
			return midTrack + delta, nil
		}
		if r.tryTrack(from, to, netID, midTrack-delta) {
			return midTrack - delta, nil
		}
	}
	return 0, ErrNoPath
}

func (r *TwoLayerRouter) tryTrack(from, to Point, netID, trackID int) bool {
	lowerLeft := r.canvas.GetLowerLeft()
	upperRight := r.canvas.GetUpperRight()
	maxTrack := (upperRight.Y-lowerLeft.Y)/r.canvas.GetM3TrackWidth() - 1
	if trackID < 0 || trackID > maxTrack {
		return false
	}

	trackYLower := lowerLeft.Y + trackID*r.canvas.GetM3TrackWidth()
	trackYUpper := lowerLeft.Y + (trackID+1)*r.canvas.GetM3TrackWidth()
	m2From := Segment{
		LowerLeft:  Point{X: from.X, Y: min(from.Y, trackYLower)},
		UpperRight: Point{X: from.X + M2Width, Y: max(from.Y, trackYUpper)},
		NetID:      netID,
	}
	m2To := Segment{
		LowerLeft:  Point{X: to.X, Y: min(to.Y, trackYLower)},
		UpperRight: Point{X: to.X + M2Width, Y: max(to.Y, trackYUpper)},
		NetID:      netID,
	}
	m3 := TrackSegment{
		TrackID: trackID,
		Start:   min(from.X, to.X),
		End:     max(from.X, to.X) + M2Width,
		NetID:   netID,
	}

	return r.canvas.IsPassibleM2(m2From) &&
		r.canvas.IsPassibleM2(m2To) &&
		r.canvas.IsPassibleM3(m3)
}
