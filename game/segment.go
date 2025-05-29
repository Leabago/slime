package game

import "math"

type Segment struct {
	a, b         Vector
	closestPoint Vector
	normal       Vector
	savePoint    *SavePoint
	isRed        bool
}

func (s Segment) Normal() Vector {
	dx := s.b.X - s.a.X
	dy := s.b.Y - s.a.Y
	return Vector{-dy, dx}.Normalize()
}

func (s Segment) Offset(d float64) Segment {
	n := s.Normal().Mul(d)
	return Segment{
		a: s.a.Add(n),
		b: s.b.Add(n),
	}
}

func (s Segment) OffsetPoint(p Vector, offset float64) Vector {
	return p.Add(s.Normal().Mul(offset))
}

func (s Segment) MinY() float64 {
	return math.Min(s.a.Y, s.b.Y)
}

func (s Segment) MaxY() float64 {
	return math.Max(s.a.Y, s.b.Y)
}

func (s Segment) AvrX() float64 {
	return (s.b.X + s.a.X) / 2
}
