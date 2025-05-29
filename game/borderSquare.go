package game

// BorderSquare square with points ABCD clockwise
// clockwise (left - bottom) - (left - top) - (right - top) - (right - bottom)
type BorderSquare struct {
	left   Segment
	top    Segment
	right  Segment
	bottom Segment

	// point A - (left - bottom)
	position Vector
}

func (b *BorderSquare) leftX() float64 {
	return b.top.a.X
}

func (b *BorderSquare) rightX() float64 {
	return b.top.b.X
}

func (b *BorderSquare) topY() float64 {
	return b.left.b.Y
}

func (b *BorderSquare) bottomY() float64 {
	return b.left.a.Y
}

// minY - higest point
// maxY - lowest point
func newBorderSquare(leftX, rightX, minY, maxY float64) BorderSquare {
	return BorderSquare{
		left: Segment{
			a: Vector{X: leftX, Y: maxY},
			b: Vector{X: leftX, Y: minY},
		},
		top: Segment{
			a: Vector{X: leftX, Y: minY},
			b: Vector{X: rightX, Y: minY},
		},
		right: Segment{
			a: Vector{X: rightX, Y: minY},
			b: Vector{X: rightX, Y: maxY},
		},
		bottom: Segment{
			a: Vector{X: rightX, Y: maxY},
			b: Vector{X: leftX, Y: maxY},
		},

		position: Vector{X: leftX, Y: minY},
	}
}
