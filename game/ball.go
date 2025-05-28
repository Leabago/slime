package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Ball struct {
	pos         Vector
	oldPos      Vector
	vel         Vector
	radius      float64
	onGround    bool
	facingRight bool

	jumpVel      Vector
	currPhyState *BallPhysic
	savePoint    *SavePoint
}

func NewBall(spawnPos Vector) *Ball {

	ball := &Ball{
		pos:          Vector{spawnPos.X, spawnPos.Y},
		vel:          Vector{0, 0},
		radius:       ballPhysicA.radius,
		currPhyState: &ballPhysicA,
	}

	return ball
}

func (b *Ball) Update(score *int, collisionSeg *[]Segment, fraction *[]Vector, ground []*Segment, lastX *float64, minLeftX float64, maxLeftX float64) error {

	// change state
	b.currPhyState = &ballPhysicA
	if ebiten.IsKeyPressed(ebiten.KeyP) {
		if b.currPhyState.state == phyStateA {
			b.currPhyState = &ballPhysicB
		}
	}

	// Move left/right
	if ebiten.IsKeyPressed(ebiten.KeyRight) {

		b.vel.X += b.currPhyState.speedRun
		b.facingRight = false

		if b.vel.X < 0 {
			b.vel.X = 0
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {

		b.vel.X -= b.currPhyState.speedRun
		b.facingRight = true

		if b.vel.X > 0 {
			b.vel.X = 0
		}
	}

	// Jump if on ground
	if ebiten.IsKeyPressed(ebiten.KeySpace) && b.onGround && *score > 0 {
		*score--
		b.onGround = false
		b.jumpVel = b.jumpVel.Mul(b.currPhyState.jumpForce)
		b.vel = b.vel.Add(b.jumpVel)

		for _, seg := range *collisionSeg {
			*fraction = append(*fraction, seg.closestPoint)
		}
	}

	// chane radius
	b.radius = b.currPhyState.radius

	// Gravity
	b.vel.Y += b.currPhyState.gravity

	// limit velocity
	if b.vel.Y > 20 {
		b.vel.Y = 20
	}
	if b.vel.X > 10 {
		b.vel.X = 10
	}
	if b.vel.X < -10 {
		b.vel.X = -10
	}

	// previos position

	b.oldPos = b.pos

	// limit edge X
	if b.pos.X < 0+b.radius {
		b.pos.X = b.radius
		b.vel.X = 1
	}
	// if b.pos.X < minLeftX+b.radius {
	// 	b.pos.X = minLeftX + b.radius
	// 	b.vel.X = 1
	// }

	// if b.pos.X > maxLeftX-b.radius {
	// 	b.pos.X = maxLeftX - b.radius
	// 	b.vel.X = -1
	// }

	// Apply velocity
	b.pos = b.pos.Add(b.vel)

	// Dampen
	b.vel.X *= 0.995
	b.vel.Y *= 0.995

	b.CheckCollisions(score, collisionSeg, ground, lastX)

	return nil
}

func (b *Ball) CheckCollisions(score *int, gameCollSeg *[]Segment, ground []*Segment, lastX *float64) {

	var penetrationSum float64
	// average normal
	avgNormal := Vector{0, 0}

	collisionSeg := []Segment{}

	wallThickness := 3.0 // or any thickness you want

	// var savePoint *SavePoint

	for _, seg := range ground {
		// current position
		closest := closestPointOnSegment(seg.a, seg.b, b.pos)
		distVec := b.pos.Sub(closest)
		dist := distVec.Len()

		// true - collision with segment
		if dist < b.radius+wallThickness {

			*lastX = seg.a.X

			// Push the wheel out of the ground
			normal := distVec.Normalize()

			// params
			seg.closestPoint = closest
			seg.normal = normal
			collisionSeg = append(collisionSeg, *seg)
			penetration := b.radius + wallThickness - dist
			penetrationSum += penetration

			// if segment is red then minus score
			if seg.isRed && *score > 0 {
				*score--
			}
		}

		// check collision with save point
		if seg.savePoint != nil {
			if circleToCircle(b.pos, b.radius, seg.savePoint.Position, seg.savePoint.Radius) {
				b.savePoint = seg.savePoint
				b.onGround = true
				seg.savePoint = nil
				*score += savePointScore
			}
		}
	}

	if len(collisionSeg) > 0 {
		for _, n := range collisionSeg {
			avgNormal = avgNormal.Add(n.normal)
		}
		avgNormal = avgNormal.Normalize()

		// Apply averaged correction
		avgPenetration := penetrationSum / float64(len(collisionSeg))
		b.pos = b.pos.Add(avgNormal.Mul(avgPenetration))

		// Handle velocity response
		velDot := b.vel.Dot(avgNormal)
		if velDot < 0 {
			b.vel = b.vel.Sub(avgNormal.Mul(velDot)).Mul(b.currPhyState.friction)
		}

		if velDot < 0 {
			// Reflect velocity along the collision normal
			reflected := b.vel.Sub(avgNormal.Mul(velDot))
			b.vel = reflected.Mul(b.currPhyState.bounceFactor)
		}

		if avgPenetration > 10 {
			b.vel = b.vel.Add(Vector{1, 0})
		}

		b.onGround = true
	}

	// get average angle
	angle := SlopeAngleFromNormal(avgNormal)

	// if state "A" then the ball cannot climb a high slope
	if b.currPhyState.state == phyStateA {
		if angle > 75 {
			b.jumpVel = avgNormal.Add(b.currPhyState.scrambleWall)
		} else {
			b.jumpVel = b.currPhyState.jump
		}
	}
	// if state "B" then the ball can slide a slope
	if b.currPhyState.state == phyStateB {
		b.jumpVel = Vector{0, 0}.Add(Vector{0, -0.4})
	}

	// if savePoint

	// if savePoint != nil {
	// b.pos = savePoint.position
	// b.onGround = true

	// b.vel = b.vel.Mul(4)
	// b.pos = seg.savePoint.position
	// // b.savePoint = seg.savePoint
	// b.onGround = true
	// seg.savePoint = nil
	// b.vel = b.vel.Sub(Vector{0, -20})

	// }

	*gameCollSeg = collisionSeg
}

func circleToCircle(posA Vector, rA float64, posB Vector, rB float64) bool {
	distX := posB.X - posA.X
	distY := posB.Y - posA.Y
	distance := math.Sqrt((distX * distX) + (distY * distY))

	if distance <= rA+rB {
		return true
	}
	return false
}
