package game

import (
	"encoding/csv"
	"image/color"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	ScreenWidth  = 1300
	ScreenHeight = 800
)

type Game struct {
	ground       []Segment
	ball         *Ball
	collisionSeg []Segment
	camera       *Camera
	score        int
	fractions    []Vector
	frameTimer   *Timer
}

func (g *Game) Update() error {

	//	 delete old fractions by timer
	g.frameTimer.Update()
	if g.frameTimer.IsReady() {
		g.frameTimer.Reset()

		if len(g.fractions) > 20 {
			g.fractions = g.fractions[1:len(g.fractions)]
		}
	}

	err := g.ball.Update(&g.score, g.collisionSeg, &g.fractions)
	if err != nil {
		return err
	}

	g.ball.CheckCollisions(&g.collisionSeg, g.ground)

	// Update camera
	g.camera.Update(g.ball.pos.x, g.ball.pos.y)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{220, 220, 255, 255})

	// Draw ground
	for _, seg := range g.ground {
		ebitenutil.DrawLine(screen, seg.a.x-g.camera.X, seg.a.y-g.camera.Y, seg.b.x-g.camera.X, seg.b.y-g.camera.Y, color.RGBA{100, 200, 100, 255})

	}

	// Draw wheel
	w := g.ball

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

	for _, fr := range g.fractions {
		ebitenutil.DrawCircle(screen, fr.x-g.camera.X, fr.y-g.camera.Y, 5, color.RGBA{100, 100, 0, 255})
	}

	ebitenutil.DebugPrintAt(screen, "isOnGround:"+strconv.Itoa(g.score), 10, 45)

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func NewGame() *Game {

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

	for _, g := range groundPoints {
		g.y += maxY
	}

	segments := make([]Segment, len(groundPoints)-1)
	for i := 0; i < len(groundPoints)-1; i++ {
		segments[i] = Segment{
			a: groundPoints[i],
			b: groundPoints[i+1],
		}

	}

	b := NewBall(segments[3])

	game := &Game{
		frameTimer: NewTimer(80 * time.Millisecond),
		score:      1000,
		ground:     segments,
		ball:       b,

		camera: &Camera{
			Width:  float64(ScreenWidth),
			Height: float64(ScreenHeight),
		},
	}

	return game
}
