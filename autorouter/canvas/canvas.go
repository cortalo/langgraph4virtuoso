package canvas

import (
	"autorouter/common"
	"errors"
)

type Point = common.Point
type Segment = common.Segment
type TrackSegment = common.TrackSegment

var ErrInvalidTrackID = errors.New("invalid m3 track ID")

type Track interface {
	IsPassible(netID, start, end int) bool
	Occupy(netID, start, end int) error
}

type TrackSegmentStorage interface {
	IsPassible(seg TrackSegment) bool
	Occupy(seg TrackSegment) error
	GetM3TrackWidth() int
}

type SegmentStorage interface {
	IsPassible(seg Segment) bool
	Occupy(seg Segment) error
}

type Canvas struct {
	LowerLeft  Point
	UpperRight Point
	M3Storage  TrackSegmentStorage
	M2Storage  SegmentStorage
}

func (c *Canvas) Inbound(p Point) bool {
	return p.X >= c.LowerLeft.X && p.X <= c.UpperRight.X &&
		p.Y >= c.LowerLeft.Y && p.Y <= c.UpperRight.Y
}

func (c *Canvas) IsPassibleM2(seg Segment) bool {
	return c.M2Storage.IsPassible(seg)
}

func (c *Canvas) IsPassibleM3(seg TrackSegment) bool {
	return c.M3Storage.IsPassible(seg)
}

func (c *Canvas) OccupyM2(seg Segment) error {
	return c.M2Storage.Occupy(seg)
}

func (c *Canvas) OccupyM3(seg TrackSegment) error {
	return c.M3Storage.Occupy(seg)
}

func (c *Canvas) GetLowerLeft() Point {
	return c.LowerLeft
}

func (c *Canvas) GetUpperRight() Point {
	return c.UpperRight
}

func (c *Canvas) GetM3TrackWidth() int {
	return c.M3Storage.GetM3TrackWidth()
}
