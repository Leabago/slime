package game

import "github.com/hajimehoshi/ebiten/v2"

type Ball struct {
	pos         Vector
	vel         Vector
	radius      float64
	onGround    bool
	facingRight bool

	jumpVel      Vector
	currPhyState *BallPhysic
}

func NewBall(seg Segment) *Ball {
	ball := &Ball{
		pos:          Vector{seg.a.x, seg.a.y - ballPhysicA.radius},
		vel:          Vector{0, 0},
		radius:       ballPhysicA.radius,
		currPhyState: &ballPhysicA,
	}

	return ball
}

func (b *Ball) Update(score *int, collisionSeg []Segment, fraction *[]Vector) error {

	// change state
	b.currPhyState = &ballPhysicA
	if ebiten.IsKeyPressed(ebiten.KeyP) {
		if b.currPhyState.state == phyStateA {
			b.currPhyState = &ballPhysicB
		}
	}

	// Move left/right
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		b.vel.x += b.currPhyState.speedRun
		b.facingRight = false
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		b.vel.x -= b.currPhyState.speedRun
		b.facingRight = true
	}

	// Jump if on ground
	if ebiten.IsKeyPressed(ebiten.KeySpace) && b.onGround && *score > 0 {
		*score -= 1
		b.onGround = false
		b.jumpVel = b.jumpVel.Mul(b.currPhyState.jumpForce)
		b.vel = b.vel.Add(b.jumpVel)

		for _, seg := range collisionSeg {
			*fraction = append(*fraction, seg.closestPoint)
		}
	}

	// chane radius
	b.radius = b.currPhyState.radius

	// Gravity
	b.vel.y += b.currPhyState.gravity

	// limit velocity
	if b.vel.y > 20 {
		b.vel.y = 20
	}
	if b.vel.x > 10 {
		b.vel.x = 10
	}
	if b.vel.x < -10 {
		b.vel.x = -10
	}

	// limit left x
	if b.pos.x < 0+b.radius {
		b.pos.x = b.radius
		b.vel.x = 1
	}

	// Apply velocity
	b.pos.x += b.vel.x
	b.pos.y += b.vel.y

	// Dampen
	b.vel.x *= 0.995
	b.vel.y *= 0.995

	return nil
}

func (b *Ball) CheckCollisions(gameCollSeg *[]Segment, ground []Segment) {

	var penetrationSum float64
	// average normal
	avgNormal := Vector{0, 0}

	collisionSeg := []Segment{}

	for _, seg := range ground {
		// current position
		closest := closestPointOnSegment(seg.a, seg.b, b.pos)
		distVec := b.pos.Sub(closest)
		dist := distVec.Len()

		if dist < b.radius {
			// Push the wheel out of the ground
			normal := distVec.Normalize()

			// params
			seg.closestPoint = closest
			seg.normal = normal
			collisionSeg = append(collisionSeg, seg)
			penetration := b.radius - dist
			penetrationSum += penetration
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

	*gameCollSeg = collisionSeg
}
