package common

type Point struct {
	X, Y int
}

type Segment struct {
	LowerLeft  Point
	UpperRight Point
	NetID      int
}

func (s Segment) Overlap(other Segment) bool {
	return s.LowerLeft.X < other.UpperRight.X && s.UpperRight.X > other.LowerLeft.X &&
		s.LowerLeft.Y < other.UpperRight.Y && s.UpperRight.Y > other.LowerLeft.Y
}

type TrackSegment struct {
	TrackID int
	Start   int
	End     int
	NetID   int
}

type TwoLayerPath struct {
	M2Start Segment
	M2End   Segment
	M3      Segment
}

type Net struct {
	ID   int
	From Point
	To   Point
}
