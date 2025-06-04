package game

import "math"

type Segment struct {
	A            Vector `json:"a"`
	B            Vector `json:"b"`
	closestPoint Vector
	normal       Vector
	savePoint    *SavePoint
	isRed        bool
	isMovingWall bool
}

func (s Segment) Normal() Vector {
	dx := s.B.X - s.A.X
	dy := s.B.Y - s.A.Y
	return Vector{-dy, dx}.Normalize()
}

func (s Segment) Offset(d float64) Segment {
	n := s.Normal().Mul(d)
	return Segment{
		A: s.A.Add(n),
		B: s.B.Add(n),
	}
}

func (s Segment) OffsetPoint(p Vector, offset float64) Vector {
	return p.Add(s.Normal().Mul(offset))
}

func (s Segment) MinY() float64 {
	return math.Min(s.A.Y, s.B.Y)
}

func (s Segment) MaxY() float64 {
	return math.Max(s.A.Y, s.B.Y)
}

func (s Segment) AvrX() float64 {
	return (s.B.X + s.A.X) / 2
}

func (s Segment) GetPosWithMinY() Vector {
	minY := math.Min(s.A.Y, s.B.Y)

	if minY == s.A.Y {
		return s.A
	} else {
		return s.B
	}
}
