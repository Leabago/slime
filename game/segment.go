package game

type Segment struct {
	a, b         Vector
	closestPoint Vector
	normal       Vector
}

func (s Segment) Normal() Vector {
	dx := s.b.x - s.a.x
	dy := s.b.y - s.a.y
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
