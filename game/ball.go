package game

import (
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

func (b *Ball) Update(ground []*Segment, game *Game) {

	// change state
	b.currPhyState = &ballPhysicA
	b.radius = b.currPhyState.radius

	// process user clicks
	b.updateControls(game)

	// update radius
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
	if b.vel.X > 7 {
		b.vel.X = 7
	}
	if b.vel.X < -7 {
		b.vel.X = -7
	}

	// limit edge X
	if b.pos.X < 5+b.currPhyState.radius {
		b.pos.X = 5 + b.currPhyState.radius
		b.vel.X = 1
	}

	// Apply velocity
	b.pos = b.pos.Add(b.vel)

	// Dampen velocity
	b.vel.X *= 0.995
	b.vel.Y *= 0.995

}

// updateControls process user clicks
func (b *Ball) updateControls(game *Game) {
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		b.currPhyState = &ballPhysicB
	} else if b.currPhyState.state == phyStateB {
		b.pos.Y += math.Abs(ballPhysicB.radius - ballPhysicA.radius)
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
	if b.currPhyState == &ballPhysicA {
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) && game.getCurrentLevel().Score > 0 {

			if b.doubleJump < 1 && !b.onGround {
				b.doubleJump++
				b.vel = b.vel.Add(b.jumpVel)
			}

			if b.onGround {
				b.onGround = false
				game.getCurrentLevel().Score--
				b.vel = b.vel.Add(b.jumpVel)
				for _, seg := range game.collisionSeg {
					game.fractions = append(game.fractions, seg.closestPoint)
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
				for _, seg := range game.collisionSeg {
					game.fractions = append(game.fractions, seg.closestPoint)
				}
			}
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) && !b.onGround {
		b.vel = b.vel.Sub(b.jumpVel)
	}
}
