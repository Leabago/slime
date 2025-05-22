package game

import "math"

type Vector struct {
	x, y float64
}

func (v Vector) Sub(o Vector) Vector  { return Vector{v.x - o.x, v.y - o.y} }
func (v Vector) Add(o Vector) Vector  { return Vector{v.x + o.x, v.y + o.y} }
func (v Vector) Mul(f float64) Vector { return Vector{v.x * f, v.y * f} }
func (v Vector) Dot(o Vector) float64 { return v.x*o.x + v.y*o.y }
func (v Vector) Len() float64         { return math.Hypot(v.x, v.y) }
func (v Vector) Normalize() Vector {
	l := v.Len()
	if l == 0 {
		return Vector{0, 0}
	}
	return Vector{v.x / l, v.y / l}
}

func SlopeAngleFromNormal(normal Vector) float64 {
	// Ensure normal points "up" (away from slope)
	if normal.y > 0 {
		normal.x, normal.y = -normal.x, -normal.y
	}

	// Calculate angle between slope and horizontal
	angleRad := math.Atan2(math.Abs(normal.x), math.Abs(normal.y))
	angleDeg := angleRad * (180 / math.Pi)

	return angleDeg
}

func closestPointOnSegment(a, b, p Vector) Vector {
	ap := p.Sub(a)
	ab := b.Sub(a)
	t := ap.Dot(ab) / ab.Dot(ab)
	t = math.Max(0, math.Min(1, t))
	return a.Add(ab.Mul(t))
}
