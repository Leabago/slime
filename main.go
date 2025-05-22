package main

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 1300
	screenHeight = 800

	gravityA   = 0.98
	frictionA  = 0.98
	radiusA    = 30.0
	speedRunA  = 1.0
	jumpA      = -1.0
	jumpForceA = 13.0

	gravityB  = 0.5
	frictionB = 0.1
	radiusB   = 40
	speedRunB = 2
	// jumpB      = -0.2
	jumpB      = 0.0
	jumpForceB = 10
)

// var (
// 	gravity   = gravityA
// 	friction  = frictionA
// 	radius    = radiusA
// 	speedRun  = speedRunA
// 	jump      = jumpA
// 	jumpForce = jumpForceA
// )

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

type PhyState struct {
	name         string
	gravity      float64
	friction     float64
	radius       float64
	speedRun     float64
	jump         Vector
	jumpForce    float64
	scrambleWall Vector

	jumpVelFunc func(jumpVel, normal Vector) Vector
}

func (p *PhyState) getJumpVel(jumpVel, normal Vector) Vector {
	return p.jumpVelFunc(jumpVel, normal)
}

// Camera struct
type Camera struct {
	X, Y          float64
	Width, Height float64
}

func (c *Camera) Update(playerX, playerY float64) {
	targetX := playerX - c.Width/2
	targetY := playerY - c.Height/2

	// Smooth camera movement
	c.X += (targetX - c.X) * 0.1
	c.Y += (targetY - c.Y) * 0.1

	// Keep camera within reasonable bounds
	// if c.X < 0 {
	// 	c.X = 0
	// }
	// if c.Y < 0 {
	// 	c.Y = 0
	// }
}

type Segment struct {
	a, b         Vector
	closestPoint Vector
	normal       Vector
}

func (s Segment) Normal() Vector {
	dx := s.b.x - s.a.x
	dy := s.b.y - s.a.y
	return Vector{-dy, dx}.Normalize()
}

type phyState int

const (
	phyStateA phyState = iota
	phyStateB
)

type Wheel struct {
	pos         Vector
	prevPos     Vector
	vel         Vector
	radius      float64
	onGround    bool
	facingRight bool

	// phyStateA iota
}

type Game struct {
	ground       []Segment
	wheel        *Wheel
	collisionSeg []Segment
	jumpVel      Vector

	camera       *Camera
	currPhyState *PhyState
	phyStateA    PhyState
	phyStateB    PhyState

	score float64

	fraction []Vector

	frameTimer *Timer
}

func (g *Game) Update() error {

	// delete old fractions
	// if len(g.fraction) > 10 {
	// 	g.fraction = g.fraction[1 : len(g.fraction)-1]
	// }

	//	 delete old fractions by timer
	g.frameTimer.Update()
	if g.frameTimer.IsReady() {
		g.frameTimer.Reset()

		if len(g.fraction) > 1 {
			g.fraction = g.fraction[1:len(g.fraction)]
		} else {
			g.fraction = g.fraction[0:0]
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyO) {
		g.currPhyState = &g.phyStateA
	}

	if ebiten.IsKeyPressed(ebiten.KeyP) {
		g.currPhyState = &g.phyStateB
	}

	w := g.wheel
	w.radius = g.currPhyState.radius

	// Gravity
	w.vel.y += g.currPhyState.gravity

	// Move left/right
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		w.vel.x += g.currPhyState.speedRun
		w.facingRight = false

		// g.jumpVel = g.jumpVel.Add(Vector{2, 0})
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		w.vel.x -= g.currPhyState.speedRun
		w.facingRight = true

		// g.jumpVel = g.jumpVel.Add(Vector{-2, 0})
	}

	// Jump if on ground

	if ebiten.IsKeyPressed(ebiten.KeySpace) && w.onGround && g.score > 0 {
		g.score -= 1

		// if g.currPhyState.name == "A" {
		g.jumpVel = g.jumpVel.Mul(g.currPhyState.jumpForce)
		w.vel = w.vel.Add(g.jumpVel)
		w.onGround = false
		// isJump = true
		// }

		// if g.currPhyState.name == "B" {
		// 	g.jumpVel = g.jumpVel.Mul(g.currPhyState.jumpForce)
		// 	w.vel = w.vel.Add(g.jumpVel)
		// 	w.onGround = false
		// 	isJump = true
		// }

		for _, seg := range g.collisionSeg {
			g.fraction = append(g.fraction, seg.closestPoint)
		}

	}

	// if ebiten.IsKeyPressed(ebiten.KeySpace) {
	// 	for _, seg := range g.collisionSeg {
	// 		g.fraction = append(g.fraction, seg.closestPoint)
	// 	}
	// }

	if w.vel.y > 20 {
		w.vel.y = 20
	}

	if w.vel.x > 10 {
		w.vel.x = 10
	}

	if w.vel.x < -10 {
		w.vel.x = -10
	}

	if w.pos.x < 0 {
		w.pos.x = 0
		w.vel.x = 0
	}

	// Apply velocity
	w.pos.x += w.vel.x
	w.pos.y += w.vel.y

	// prePosX := w.pos.x
	// prePosY := w.pos.y

	// Dampen
	w.vel.x *= 0.995
	w.vel.y *= 0.995

	// Handle collision with ground segments
	w.onGround = false

	g.collisionSeg = []Segment{}

	var penetrationSum float64

	for _, seg := range g.ground {
		// current position
		closest := closestPointOnSegment(seg.a, seg.b, w.pos)
		distVec := w.pos.Sub(closest)
		dist := distVec.Len()

		// prevclosest := closestPointOnSegment(seg.a, seg.b, Vector{prePosX, prePosY})
		// prevdistVec := w.pos.Sub(prevclosest)
		// prevdist := prevdistVec.Len()

		// fmt.Println("prevdist: ", dist)
		// fmt.Println("isJump: ", isJump)

		// if prevdist <= w.radius && isJump {

		// 	g.fraction = append(g.fraction, prevclosest)

		// }

		if dist < w.radius {

			// Push the wheel out of the ground
			normal := distVec.Normalize()

			// params
			seg.closestPoint = closest
			seg.normal = normal

			g.collisionSeg = append(g.collisionSeg, seg)

			//

			penetration := w.radius - dist

			penetrationSum += penetration

		}
	}

	minMormal := 1.0
	// normal_ := Vector{}
	avgNormal := Vector{0, 0}

	// for _, seg := range g.collisionSeg {
	// 	if math.Abs(seg.normal.x) < minMormal {
	// 		minMormal = math.Abs(seg.normal.x)
	// 		normal_ = seg.normal
	// 	}
	// }

	if len(g.collisionSeg) > 0 {

		for _, n := range g.collisionSeg {
			avgNormal = avgNormal.Add(n.normal)
		}
		avgNormal = avgNormal.Normalize()

		// Apply averaged correction
		avgPenetration := penetrationSum / float64(len(g.collisionSeg))

		w.pos = w.pos.Add(avgNormal.Mul(avgPenetration))

		// Handle velocity response
		velDot := w.vel.Dot(avgNormal)
		if velDot < 0 {
			w.vel = w.vel.Sub(avgNormal.Mul(velDot)).Mul(g.currPhyState.friction)
		}

		w.onGround = true
	}

	fmt.Println("minMormal:", minMormal)
	fmt.Println("avgNormal:", avgNormal)

	angle := SlopeAngleFromNormal(avgNormal)
	fmt.Println("angle: ", angle)

	if angle > 75 {
		// w.onGround = false
		// g.jumpForce = 1
		// g.jumpVel = normal_

		// g.jumpVel = normal_.Add(Vector{0, g.currPhyState.scrambleWall})
		// g.jumpVel = normal_.Add(g.currPhyState.scrambleWall)

		// g.jumpVel = g.currPhyState.getJumpVel(g.jumpVel, avgNormal)

		if g.currPhyState.name == "A" {
			g.jumpVel = avgNormal.Add(g.currPhyState.scrambleWall)
		}
		// if g.currPhyState.name == "B" {
		// 	// g.jumpVel = g.currPhyState.scrambleWall
		// 	g.jumpVel = g.currPhyState.jump
		// }
	} else {

		if g.currPhyState.name == "A" {
			g.jumpVel = g.currPhyState.jump
		}

	}

	if g.currPhyState.name == "B" {
		g.jumpVel = Vector{0, 0}
		g.jumpVel = g.jumpVel.Add(Vector{0, -0.4})
	}

	// cursX, cursY := ebiten.CursorPosition()
	// w.pos.x = float64(cursX)
	// w.pos.y = float64(cursY)

	// Update camera
	g.camera.Update(g.wheel.pos.x, g.wheel.pos.y)

	w.prevPos = w.pos

	return nil
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

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{220, 220, 255, 255})

	// Draw ground
	for _, seg := range g.ground {
		ebitenutil.DrawLine(screen, seg.a.x-g.camera.X, seg.a.y-g.camera.Y, seg.b.x-g.camera.X, seg.b.y-g.camera.Y, color.RGBA{100, 200, 100, 255})

	}

	// Draw wheel
	w := g.wheel

	if w.facingRight {
		ebitenutil.DrawCircle(screen, w.pos.x-g.camera.X, w.pos.y-g.camera.Y, w.radius, color.Black)

	} else {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(w.pos.x-g.camera.X, w.pos.y-g.camera.Y)
		ebitenutil.DrawCircle(screen, w.pos.x-g.camera.X, w.pos.y-g.camera.Y, w.radius, color.Black)
	}

	// Show direction line
	dirX := w.pos.x + w.radius*math.Cos(0)
	dirY := w.pos.y + w.radius*math.Sin(0)
	ebitenutil.DrawLine(screen, w.pos.x-g.camera.X, w.pos.y-g.camera.Y, dirX-g.camera.X, dirY-g.camera.Y, color.RGBA{255, 0, 0, 255})

	// Draw ground
	for _, seg := range g.collisionSeg {
		// r := uint8(rand.Intn(255))
		// g := uint8(rand.Intn(255))
		// b := uint8(rand.Intn(255))
		// ebitenutil.DrawLine(screen, seg.a.x, seg.a.y, seg.b.x, seg.b.y, color.RGBA{r, g, b, 50})

		ebitenutil.DrawLine(screen, seg.a.x-g.camera.X, seg.a.y-g.camera.Y, seg.b.x-g.camera.X, seg.b.y-g.camera.Y, color.RGBA{1, 1, 1, 255})

		ebitenutil.DrawCircle(screen, seg.closestPoint.x-g.camera.X, seg.closestPoint.y-g.camera.Y, 5, color.RGBA{255, 0, 0, 255})

		ebitenutil.DrawCircle(screen, seg.closestPoint.x-g.camera.X, seg.closestPoint.y-g.camera.Y, 5, color.RGBA{255, 0, 0, 255})

	}

	fmt.Println("seg.fraction:", len(g.fraction))
	for _, fr := range g.fraction {
		ebitenutil.DrawCircle(screen, fr.x-g.camera.X, fr.y-g.camera.Y, 5, color.RGBA{100, 100, 0, 255})
	}

	ebitenutil.DebugPrintAt(screen, "isOnGround:"+strconv.FormatFloat(g.score, 'f', 8, 64), 10, 45)

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func closestPointOnSegment(a, b, p Vector) Vector {
	ap := p.Sub(a)
	ab := b.Sub(a)
	t := ap.Dot(ab) / ab.Dot(ab)
	t = math.Max(0, math.Min(1, t))
	return a.Add(ab.Mul(t))
}

func main() {

	// Open the CSV file
	file, err := os.Open("game_AAPL3.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read all records at once
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	groundPoints := []Vector{}

	// Print each record
	maxY := 0.0
	for _, record := range records {

		x := record[0]
		y := record[1]

		xStr, err := strconv.ParseFloat(x, 64)
		if err != nil {
			panic(err)
		}

		yStr, err := strconv.ParseFloat(y, 64)
		if err != nil {
			panic(err)
		}

		// xStr *= 30
		// yStr *= (-10)
		groundPoints = append(groundPoints, Vector{xStr * 50, yStr * 20 * (-1)})

		if math.Abs(yStr) > maxY {
			maxY = math.Abs(yStr)
		}
	}

	fmt.Println("maxY: ", maxY)

	for _, g := range groundPoints {
		g.y += maxY
	}

	// maxY *= 5
	// buffY := radius

	segments := make([]Segment, len(groundPoints)-1)
	for i := 0; i < len(groundPoints)-1; i++ {
		segments[i] = Segment{
			a: groundPoints[i],
			b: groundPoints[i+1],
		}

		fmt.Println(segments[i].a.x, "/", segments[i].a.y)
	}

	game := &Game{
		frameTimer: NewTimer(50 * time.Millisecond),
		score:      1000,
		ground:     segments,
		wheel: &Wheel{
			pos:     Vector{400, segments[3].a.y - radiusA - 100},
			prevPos: Vector{400, segments[3].a.y - radiusA - 100},
			vel:     Vector{0, 0},
			radius:  radiusA,
		},
		jumpVel: Vector{0, -20},
		camera: &Camera{
			Width:  float64(screenWidth),
			Height: float64(screenHeight),
		},
		phyStateA: PhyState{
			name:         "A",
			gravity:      gravityA,
			friction:     frictionA,
			radius:       radiusA,
			speedRun:     speedRunA,
			jump:         Vector{0, jumpA},
			jumpForce:    jumpForceA,
			scrambleWall: Vector{0, -1},
			jumpVelFunc: func(jumpVel, normal Vector) Vector {
				return normal.Add(Vector{0, -1})
			},
		},

		phyStateB: PhyState{
			name:         "B",
			gravity:      gravityB,
			friction:     frictionB,
			radius:       radiusB,
			speedRun:     speedRunB,
			jump:         Vector{0, jumpB},
			jumpForce:    jumpForceB,
			scrambleWall: Vector{0, -0.5},
			jumpVelFunc: func(jumpVel, normal Vector) Vector {
				return Vector{0, -0.5}
			},
		},
		currPhyState: &PhyState{
			name:         "A",
			gravity:      gravityA,
			friction:     frictionA,
			radius:       radiusA,
			speedRun:     speedRunA,
			jump:         Vector{0, jumpA},
			jumpForce:    jumpForceA,
			scrambleWall: Vector{0, -1},
			jumpVelFunc: func(jumpVel, normal Vector) Vector {
				return normal.Add(Vector{0, -1})
			},
		},
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Polygonal Ground with Rolling Wheel")
	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

// Timer
type Timer struct {
	currentTicks int
	targetTicks  int
}

func NewTimer(d time.Duration) *Timer {
	return &Timer{
		currentTicks: 0,
		targetTicks:  int(d.Milliseconds()) * ebiten.TPS() / 1000,
	}
}

func (t *Timer) Update() {
	if t.currentTicks < t.targetTicks {
		t.currentTicks++
	}
}

func (t *Timer) IsReady() bool {
	return t.currentTicks >= t.targetTicks
}

func (t *Timer) Reset() {
	t.currentTicks = 0
}

//-------------------------------------------------------
