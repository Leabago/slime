package game

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Ball struct {
	pos         Vector
	vel         Vector
	radius      float64
	onGround    bool
	facingRight bool

	jumpVel      Vector
	currPhyState *BallPhysic

	// check if a moving wall collision has occurred
	isDied bool

	doubleJump int
}

func NewBall(spawnPos Vector) *Ball {

	ball := &Ball{
		pos:          Vector{spawnPos.X, spawnPos.Y},
		vel:          Vector{0, 0},
		radius:       ballPhysicA.radius,
		currPhyState: &ballPhysicA,
		doubleJump:   0,
	}

	return ball
}

func (b *Ball) Update(collisionSeg *[]Segment, ground []*Segment, game *Game) error {

	// change state
	curState := *b.currPhyState
	b.currPhyState = &ballPhysicA
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		b.currPhyState = &ballPhysicB
	} else if curState.state == phyStateB {
		b.pos.Y += math.Abs(ballPhysicB.radius - ballPhysicA.radius)
	}

	// Move left/right
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		b.vel.X += b.currPhyState.speedRun
		b.facingRight = false

		// if b.vel.X < 0 {
		// 	b.vel.X = 0
		// }
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		b.vel.X -= b.currPhyState.speedRun
		b.facingRight = true

		// if b.vel.X > 0 {
		// 	b.vel.X = 0
		// }
	}

	// fmt.Println(" b.vel.Y: ", b.vel.Y)
	// if b.onGround == false && b.vel.Y > 0 {
	// 	b.doubleJump = true
	// }

	// if inpututil.IsKeyJustPressed(ebiten.KeySpace) && b.doubleJump < 1 && b.onGround == false {
	// 	b.vel = b.vel.Add(b.jumpVel)
	// 	b.doubleJump++
	// }

	// Jump if on ground

	if b.currPhyState == &ballPhysicA {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) && game.getCurrentLevel().Score > 0 {

			if b.doubleJump < 1 && b.onGround == false {
				b.doubleJump++
				b.vel = b.vel.Add(b.jumpVel)
			}

			if b.onGround {
				b.onGround = false
				game.getCurrentLevel().Score--
				b.vel = b.vel.Add(b.jumpVel)
				for _, seg := range *collisionSeg {
					game.fractions = append(game.fractions, seg.closestPoint)
					// *fraction = append(*fraction, seg.closestPoint)
				}
			}
		}
	}

	if b.currPhyState == &ballPhysicB {
		if ebiten.IsKeyPressed(ebiten.KeySpace) && game.getCurrentLevel().Score > 0 {

			if b.onGround {
				b.onGround = false
				game.getCurrentLevel().Score--
				b.vel = b.vel.Add(b.jumpVel)
				for _, seg := range *collisionSeg {
					game.fractions = append(game.fractions, seg.closestPoint)
					// *fraction = append(*fraction, seg.closestPoint)
				}
			}
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) && !b.onGround {
		b.vel = b.vel.Sub(b.jumpVel)

	}

	// chane radius
	b.radius = b.currPhyState.radius

	// Gravity
	b.vel.Y += b.currPhyState.gravity

	// if b.currPhyState.state == phyStateB && !b.onGround {
	// 	b.vel.Y += (b.currPhyState.gravity * 10)
	// }

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

	b.CheckCollisions(collisionSeg, ground, game)

	// Gravity
	// game.enemyBall.pos = game.enemyBall.pos.Add(game.enemyBall.vel)

	game.enemyBall.vel.X *= 0.5
	game.enemyBall.vel.Y *= 0.5

	// if game.enemyBall.onGround {
	// 	game.enemyBall.vel.Y -= 0.5
	// }

	// if game.enemyBall.vel.Y > -10 {
	// 	game.enemyBall.vel.Y = -10
	// }

	// game.enemyBall.vel.X *= 0.995
	// game.enemyBall.vel.Y *= 0.995

	return nil
}

func (b *Ball) CheckCollisions(gameCollSeg *[]Segment, ground []*Segment, game *Game) {
	// average normal
	avgNormal := Vector{0, 0}
	collisionSeg := []Segment{}
	var penetrationSum float64
	wallThickness := 3.0 // to avoid falling into a segment

	minVec := 1000.0
	velEnemy := Vector{}

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

	if circleToCircle(b.pos, b.radius, game.enemyBall.pos, game.enemyBall.radius) {
		b.isDied = true
	}

	for _, seg := range ground {
		// current position
		closest := closestPointOnSegment(seg.A, seg.B, b.pos)
		distVec := b.pos.Sub(closest)
		dist := distVec.Len()

		// // current position enemy
		closestEnemy := closestPointOnSegment(seg.A, seg.B, game.enemyBall.pos)
		distVecEnemy := game.enemyBall.pos.Sub(closestEnemy)
		distEnemy := distVecEnemy.Len()

		if !seg.isMovingWall && !seg.isBorder {

			if distEnemy < minVec {
				minVec = distEnemy

				vec := seg.A.Sub(seg.B).Normalize()
				vec = vec.Add(seg.A.Sub(game.enemyBall.pos).Normalize())
				velEnemy = vec

			}
		}

		// true - collision with segment
		if dist < b.radius+wallThickness {

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
				fmt.Println("seg.isMovingWall")
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

	if minVec != 1000.0 {
		game.enemyBall.vel = game.enemyBall.vel.Add(velEnemy)
	}

	if !isCircleRectangleColl(game.enemyBall.pos, game.enemyBall.radius, *game.borderSquare) {
		fmt.Println("restart1")
		game.enemyBall.pos = game.borderSquare.drawRight.B
	}

	closestEnemy := closestPointOnSegment(game.borderSquare.left.A, game.borderSquare.left.B, game.enemyBall.pos)
	distVecEnemy := game.enemyBall.pos.Sub(closestEnemy)
	distEnemy := distVecEnemy.Len()

	if distEnemy < game.enemyBall.radius {
		fmt.Println("restart2")
		game.enemyBall.pos = game.borderSquare.drawRight.B
	}

	// vecNorm := closestE.Sub(game.enemyBall.pos).Normalize()

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
			b.vel = b.vel.Sub(avgNormal.Mul(velDot)).Mul(b.currPhyState.friction)

			// Reflect velocity along the collision normal, friction
			// reflected := b.vel.Sub(avgNormal.Mul(velDot))
			// b.vel = reflected.Mul(b.currPhyState.bounceFactor)
		}

		// to avoid falling between two segments
		if avgPenetration > 2 {
			b.vel = b.vel.Add(Vector{1, -1})
		}

		b.onGround = true
		b.doubleJump = 0
	}

	// get average angle
	angle := SlopeAngleFromNormal(avgNormal)

	// if state "A" then the ball cannot climb a high slope
	if b.currPhyState.state == phyStateA {
		if angle > 70 {
			b.jumpVel = avgNormal.Add(b.currPhyState.scrambleWall)
		} else {
			b.jumpVel = b.currPhyState.jump
		}
	}
	// if state "B" then the ball can slide a slope
	if b.currPhyState.state == phyStateB {
		b.jumpVel = b.currPhyState.jump
	}
	b.jumpVel = b.jumpVel.Mul(b.currPhyState.jumpForce)
	*gameCollSeg = collisionSeg
}
