package common

type Point struct {
	X, Y int
}

type Path []Point

type Net struct {
	ID        int
	From      Point
	To        Point
	HalfWidth int
	Path      Path
}
