package canvas

import "errors"

var ErrOutOfBounds = errors.New("out bof bounds")

type SegmentStorageImpl struct {
	segments   []*Segment
	LowerLeft  Point
	UpperRight Point
}

func NewSegmentStore(lowerLeft, upperRight Point) *SegmentStorageImpl {
	return &SegmentStorageImpl{
		LowerLeft:  lowerLeft,
		UpperRight: upperRight,
	}
}

func (s *SegmentStorageImpl) IsPassible(seg Segment) bool {
	if !s.inbound(seg) {
		return false
	}
	for _, existing := range s.segments {
		if existing.NetID == seg.NetID {
			continue
		}
		if seg.Overlap(*existing) {
			return false
		}
	}
	return true
}

func (s *SegmentStorageImpl) Occupy(seg Segment) error {
	if !s.inbound(seg) {
		return ErrOutOfBounds
	}
	if !s.IsPassible(seg) {
		return ErrOverlap
	}
	s.segments = append(s.segments, &seg)
	return nil
}

func (s *SegmentStorageImpl) inbound(seg Segment) bool {
	return seg.LowerLeft.X >= s.LowerLeft.X &&
		seg.LowerLeft.Y >= s.LowerLeft.Y &&
		seg.UpperRight.X <= s.UpperRight.X &&
		seg.UpperRight.Y <= s.UpperRight.Y
}
