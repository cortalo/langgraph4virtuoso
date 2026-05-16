package common

type Point struct {
	X, Y int
}

type Path []Point

type Segment struct {
	// lower left corner point
	Point     Point
	LineWidth int
	Length    int
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
