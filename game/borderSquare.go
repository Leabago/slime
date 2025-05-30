package game

const bottomDown = 200

// BorderSquare square with points ABCD clockwise
// clockwise (left - bottom) - (left - top) - (right - top) - (right - bottom)
type BorderSquare struct {
	left   Segment
	top    Segment
	right  Segment
	bottom Segment

	drawLeft  Segment
	drawRight Segment

	// point A - (left - bottom)
	position Vector

	bottomDown float64
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
func newBorderSquare(ground []*Segment, minY, maxY float64) BorderSquare {

	leftX := ground[0].a.X
	rightX := ground[len(ground)-1].b.X

	leftY := ground[0].a.Y
	rightY := ground[len(ground)-1].b.Y

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
			a: Vector{X: rightX, Y: maxY + bottomDown},
			b: Vector{X: leftX, Y: maxY + bottomDown},
		},

		drawLeft: Segment{
			a: Vector{X: leftX, Y: leftY},
			b: Vector{X: leftX, Y: minY},
		},

		drawRight: Segment{
			a: Vector{X: rightX, Y: minY},
			b: Vector{X: rightX, Y: rightY},
		},

		position: Vector{X: leftX, Y: minY},
	}
}
