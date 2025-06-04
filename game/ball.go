package game

import (
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

	// check if a moving wall collision has occurred
	isDied bool
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

func (b *Ball) Update(collisionSeg *[]Segment, ground []*Segment, lastX *float64, game *Game) error {

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
	if ebiten.IsKeyPressed(ebiten.KeySpace) && b.onGround && game.getCurrentLevel().Score > 0 {
		game.getCurrentLevel().Score--
		b.onGround = false
		b.jumpVel = b.jumpVel.Mul(b.currPhyState.jumpForce)
		b.vel = b.vel.Add(b.jumpVel)

		for _, seg := range *collisionSeg {
			game.fractions = append(game.fractions, seg.closestPoint)
			// *fraction = append(*fraction, seg.closestPoint)
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

	if b.vel.Y < -20 {
		b.vel.Y = -20
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

	// if b.pos.Y < game.getCurrentLevel().MaxY+b.radius {
	// 	b.pos.Y = game.getCurrentLevel().MaxY + b.radius
	// 	b.vel.Y = 0
	// }
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

	b.CheckCollisions(collisionSeg, ground, lastX, game)

	return nil
}

func (b *Ball) CheckCollisions(gameCollSeg *[]Segment, ground []*Segment, lastX *float64, game *Game) {
	// average normal
	avgNormal := Vector{0, 0}
	collisionSeg := []Segment{}
	var penetrationSum float64
	wallThickness := 3.0 // to avoid falling into a segment

	if !isCircleRectangleColl(b.pos, b.radius, *game.borderSquare) {
		b.vel = Vector{}
		if game.getCurrentLevel().SavePoint != nil {
			if isCircleRectangleColl(game.getCurrentLevel().SavePoint.Position, b.radius, *game.borderSquare) {
				b.pos = game.getCurrentLevel().SavePoint.Position
			} else {
				b.pos = getStartPositionPtr(ground)
			}
		} else {
			b.pos = getStartPositionPtr(ground)
		}
	}

	for _, seg := range ground {
		// current position
		closest := closestPointOnSegment(seg.A, seg.B, b.pos)
		distVec := b.pos.Sub(closest)
		dist := distVec.Len()

		// true - collision with segment
		if dist < b.radius+wallThickness {
			*lastX = seg.A.X

			// Push the wheel out of the ground
			normal := distVec.Normalize()

			// params
			seg.closestPoint = closest
			seg.normal = normal
			collisionSeg = append(collisionSeg, *seg)
			penetration := b.radius + wallThickness - dist
			penetrationSum += penetration

			// if segment is red then minus score
			if seg.isRed && game.getCurrentLevel().Score > 0 {
				game.getCurrentLevel().Score--
			}

			// die
			if seg.isMovingWall {
				b.isDied = true
			}
		}

		// check collision with save point
		if seg.savePoint != nil {
			if circleToCircle(b.pos, b.radius, seg.savePoint.Position, seg.savePoint.Radius) {
				game.getCurrentLevel().SavePoint = seg.savePoint
				// b.onGround = true
				seg.savePoint = nil

				game.getCurrentLevel().Score += savePointScore

				// collision with finish
				if game.getCurrentLevel().SavePoint.IsFinish {
					game.getCurrentLevel().Score += savePointScore * 5
					game.getCurrentLevel().Finished = true
				}
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
			// friction
			// b.vel = b.vel.Sub(avgNormal.Mul(velDot)).Mul(b.currPhyState.friction)

			// Reflect velocity along the collision normal, friction
			reflected := b.vel.Sub(avgNormal.Mul(velDot))
			b.vel = reflected.Mul(b.currPhyState.bounceFactor).Mul(b.currPhyState.friction)
		}

		// to avoid falling between two segments
		if avgPenetration > 2 {
			b.vel = b.vel.Add(Vector{1, -1})
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
