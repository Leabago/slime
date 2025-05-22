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
