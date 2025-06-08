package game

const bottomDown = 400

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
	return b.top.A.X
}

func (b *BorderSquare) rightX() float64 {
	return b.top.B.X
}

func (b *BorderSquare) topY() float64 {
	return b.left.B.Y
}

func (b *BorderSquare) bottomY() float64 {
	return b.left.A.Y
}

// minY - higest point
// maxY - lowest point
func newBorderSquare(ground []*Segment) BorderSquare {

	minY, maxY := findMinMaxY(ground)
	minY -= wallHeight

	leftX := ground[0].A.X
	rightX := ground[len(ground)-1].B.X

	leftY := ground[0].A.Y
	rightY := ground[len(ground)-1].B.Y

	return BorderSquare{
		left: Segment{
			A:        Vector{X: leftX, Y: maxY + bottomDown},
			B:        Vector{X: leftX, Y: minY},
			isRed:    true,
			isBorder: true,
		},
		top: Segment{
			A:        Vector{X: leftX, Y: minY},
			B:        Vector{X: rightX, Y: minY},
			isRed:    true,
			isBorder: true,
		},
		right: Segment{
			A:        Vector{X: rightX, Y: minY},
			B:        Vector{X: rightX, Y: maxY + bottomDown},
			isRed:    true,
			isBorder: true,
		},
		bottom: Segment{
			A:        Vector{X: rightX, Y: maxY + bottomDown},
			B:        Vector{X: leftX, Y: maxY + bottomDown},
			isRed:    true,
			isBorder: true,
		},

		drawLeft: Segment{
			A:        Vector{X: leftX, Y: leftY},
			B:        Vector{X: leftX, Y: minY},
			isBorder: true,
		},

		drawRight: Segment{
			A:        Vector{X: rightX, Y: minY},
			B:        Vector{X: rightX, Y: rightY},
			isBorder: true,
		},

		position: Vector{X: leftX, Y: minY},
	}
}
