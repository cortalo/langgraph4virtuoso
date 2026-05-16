package session

import (
	"autorouter/common"
)
import "github.com/samber/lo"

type Point = common.Point
type Path = common.TwoLayerPath
type Segment = common.Segment
type Net = common.Net

type Canvas interface {
	AddM2(seg Segment) error
	AddM3(seg Segment) error
}

type Router interface {
	Route(net Net) (Path, error)
}

type Session struct {
	canvas Canvas
	router Router
	nets   []*Net
}

type NetResult struct {
	Net *Net
	Err error
}

func (s *Session) Route() []NetResult {
	results := make([]NetResult, len(s.nets))
	for i, net := range s.nets {
		path, err := s.router.Route(*net)
		results[i] = NetResult{Net: net, Err: err}
		if err == nil {
			lo.Must0(s.canvas.AddM2(path.M2Start))
			lo.Must0(s.canvas.AddM2(path.M2End))
			lo.Must0(s.canvas.AddM3(path.M3))
		}
	}
	return results
}
