package router

import (
	"autorouter/common"
)

type Point = common.Point
type Path = common.TwoLayerPath
type Net = common.Net
type Segment = common.Segment

const M2Width = 1

type Canvas interface {
	AddM2(seg Segment, netID int) error
	AddM3(trackID, startX, endX, netID int) error
	GetM3TrackIndex(y int) (int, error)
	GetM3YByID(trackID int) (int, int, error)
	GetM3MaxDelta(index int) (int, error)
	IsPassibleM3(trackID, netID, startX, endX int) bool
	IsPassibleM2(leftX, netID, startY, endY int) bool
}

type TwoLayerRouter struct {
	canvas Canvas
}

func (r *TwoLayerRouter) Route(net Net) (Path, error) {
	midY := (net.From.Y + net.To.Y) / 2
	midTrack, err := r.canvas.GetM3TrackIndex(midY)
	if err != nil {
		return Path{}, err
	}
	maxDelta, err := r.canvas.GetM3MaxDelta(midTrack)
	if err != nil {
		return Path{}, err
	}
	for delta := 0; delta <= maxDelta; delta++ {
		track := midTrack + delta
		if r.tryPath(net, track) {
			// TODO: return path
			return Path{}, nil
		}
		track = midTrack - delta
		if r.tryPath(net, track) {
			// TODO: return path
			return Path{}, nil
		}
	}
	return Path{}, ErrNoPath
}

func (r *TwoLayerRouter) tryPath(net Net, track int) bool {
	leftPoint, rightPoint := net.From, net.To
	if net.From.X > net.To.X {
		leftPoint, rightPoint = net.To, net.From
	}
	if r.canvas.IsPassibleM3(track, net.ID, leftPoint.X, rightPoint.X+M2Width) {
		m3Y1, m3Y2, err := r.canvas.GetM3YByID(track)
		if err != nil {
			panic(err)
		}
		if r.canvas.IsPassibleM2(net.From.X, net.ID, min(net.From.Y, m3Y1), max(net.From.Y, m3Y2)) &&
			r.canvas.IsPassibleM2(net.To.X, net.ID, min(net.To.Y, m3Y1), max(net.To.Y, m3Y2)) {
			//m2FromSegment := Segment{
			//	Point:     Point{X: net.From.X, Y: min(net.From.Y, m3Y1)},
			//	LineWidth: M2Width,
			//	Length:    max(net.From.Y, m3Y2) - min(net.From.Y, m3Y1),
			//}
			//m2ToSegment := Segment{
			//	Point:     Point{X: net.To.X, Y: min(net.To.Y, m3Y1)},
			//	LineWidth: M2Width,
			//	Length:    max(net.To.Y, m3Y2) - min(net.To.Y, m3Y1),
			//}
			return true
		}
	}
	return false
}
