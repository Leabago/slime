package game

import "math"

type Vector struct {
	X float64
	Y float64
}

func (v Vector) Sub(o Vector) Vector  { return Vector{v.X - o.X, v.Y - o.Y} }
func (v Vector) Add(o Vector) Vector  { return Vector{v.X + o.X, v.Y + o.Y} }
func (v Vector) Mul(f float64) Vector { return Vector{v.X * f, v.Y * f} }
func (v Vector) Dot(o Vector) float64 { return v.X*o.X + v.Y*o.Y }
func (v Vector) Len() float64         { return math.Hypot(v.X, v.Y) }
func (v Vector) Normalize() Vector {
	l := v.Len()
	if l == 0 {
		return Vector{0, 0}
	}
	return Vector{v.X / l, v.Y / l}
}

func SlopeAngleFromNormal(normal Vector) float64 {
	// Ensure normal points "up" (away from slope)
	if normal.Y > 0 {
		normal.X, normal.Y = -normal.X, -normal.Y
	}

	// Calculate angle between slope and horizontal
	angleRad := math.Atan2(math.Abs(normal.X), math.Abs(normal.Y))
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
