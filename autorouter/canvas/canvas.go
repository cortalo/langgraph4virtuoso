package canvas

import (
	"autorouter/common"
	"errors"
)

type Point = common.Point
type Path = common.TwoLayerPath
type Net = common.Net
type Segment = common.Segment

var ErrOutOfBound = errors.New("out of bound")
var ErrInvalidM3TrackID = errors.New("invalid m3 track ID")

type Track interface {
	IsPassible(netID, start, end int) bool
	Occupy(netID, start, end int) error
}

type Canvas struct {
	LeftBottom   Point
	RightTop     Point
	M3TrackWidth int // in nm
	M3Tracks     []Track
}

func (c *Canvas) AddM2(seg Segment, netID int) error {
	return nil
}

func (c *Canvas) AddM3(trackID, startX, endX, netID int) error {
	if !c.isValidM3TrackID(trackID) {
		return ErrInvalidM3TrackID
	}
	return c.M3Tracks[trackID].Occupy(netID, startX, endX)
}

func (c *Canvas) GetM3TrackIndex(y int) (int, error) {
	if y < c.getMinY() || y > c.getMaxY() {
		return 0, ErrOutOfBound
	}
	return (y - c.getMinY()) / c.M3TrackWidth, nil
}

func (c *Canvas) getMinY() int {
	return c.LeftBottom.Y
}

func (c *Canvas) getMaxY() int {
	return c.RightTop.Y
}

func (c *Canvas) isValidM3TrackID(trackID int) bool {
	if trackID < 0 {
		return false
	}
	if c.getMinY()+(trackID+1)*c.M3TrackWidth > c.getMaxY() {
		return false
	}
	return true
}

func (c *Canvas) GetM3YByID(trackID int) (int, int, error) {
	if !c.isValidM3TrackID(trackID) {
		return 0, 0, ErrInvalidM3TrackID
	}
	return c.getMinY() + trackID*c.M3TrackWidth, c.getMinY() + (trackID+1)*c.M3TrackWidth, nil
}

func (c *Canvas) GetM3MaxDelta(index int) (int, error) {
	return 0, nil
}

func (c *Canvas) IsPassibleM3(trackID, netID, startX, endX int) bool {
	return c.isValidM3TrackID(trackID) && c.M3Tracks[trackID].IsPassible(netID, startX, endX)
}

func (c *Canvas) IsPassibleM2(leftX, netID, startY, endY int) bool {
	return false
}
