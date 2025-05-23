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
	ScreenWidth    = 1300
	ScreenHeight   = 800
	multiplyChartX = 50
	multiplyChartY = -100
	groundBuffSize = 40
)

type Game struct {
	ground     []Segment
	groundBuff [2][]Segment

	ball         *Ball
	collisionSeg []Segment
	camera       *Camera
	score        int
	fractions    []Vector
	frameTimer   *Timer

	lastXinChart    float64
	gameParallelSeg []Segment
}

func (g *Game) Update() error {

	//	 delete old fractions by timer
	g.frameTimer.Update()
	if g.frameTimer.IsReady() {
		g.frameTimer.Reset()

		if len(g.fractions) > 10 {
			g.fractions = g.fractions[1:len(g.fractions)]
		}
	}

	groundFromBuff := make([]Segment, groundBuffSize*2)
	for _, g := range g.groundBuff[0] {
		groundFromBuff = append(groundFromBuff, g)
	}
	for _, g := range g.groundBuff[1] {
		groundFromBuff = append(groundFromBuff, g)
	}

	lastBuffX := 0.0
	lastXSecBuff := g.groundBuff[1][len(g.groundBuff[1])-1].b.x
	err := g.ball.Update(&g.score, &g.collisionSeg, &g.fractions, groundFromBuff, &lastBuffX, g.groundBuff[0][0].a.x, lastXSecBuff)
	if err != nil {
		return err
	}

	if lastBuffX >= (g.groundBuff[1][0]).b.x && g.lastXinChart > lastXSecBuff {
		secondBuffI := int(lastXSecBuff)
		secondBuffI /= multiplyChartX
		g.groundBuff[0] = g.groundBuff[1]
		g.groundBuff[1] = []Segment{}

		plusGroundBuffSize := groundBuffSize
		if secondBuffI+groundBuffSize > len(g.ground)-1 {
			plusGroundBuffSize = len(g.ground) - secondBuffI
		}

		for i := secondBuffI; i < secondBuffI+plusGroundBuffSize; i++ {
			g.groundBuff[1] = append(g.groundBuff[1], g.ground[i])
		}
	}

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

	for _, seg := range g.groundBuff[0] {
		ebitenutil.DrawLine(screen, seg.a.x-g.camera.X, seg.a.y-g.camera.Y-5, seg.b.x-g.camera.X, seg.b.y-g.camera.Y-5, color.RGBA{255, 200, 100, 255})
	}

	for _, seg := range g.groundBuff[1] {
		ebitenutil.DrawLine(screen, seg.a.x-g.camera.X, seg.a.y-g.camera.Y+5, seg.b.x-g.camera.X, seg.b.y-g.camera.Y+5, color.RGBA{0, 100, 200, 255})
	}

	for _, seg := range g.gameParallelSeg {
		ebitenutil.DrawLine(screen, seg.a.x-g.camera.X, seg.a.y-g.camera.Y, seg.b.x-g.camera.X, seg.b.y-g.camera.Y, color.RGBA{100, 100, 255, 255})
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
	file, err := os.Open("game_AAPL.csv")
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

		groundPoints = append(groundPoints, Vector{xStr * multiplyChartX, yStr * multiplyChartY})
	}

	lastX := 0.0
	segments := make([]Segment, len(groundPoints)-1)
	for i := 0; i < len(groundPoints)-1; i++ {
		segments[i] = Segment{
			a: groundPoints[i],
			b: groundPoints[i+1],
		}

		lastX = groundPoints[i+1].x
	}

	buff := [2][]Segment{}

	for i := 0; i < groundBuffSize; i++ {
		buff[0] = append(buff[0], segments[i])
	}

	for i := groundBuffSize; i < groundBuffSize+groundBuffSize; i++ {
		buff[1] = append(buff[1], segments[i])
	}

	game := &Game{
		frameTimer:   NewTimer(80 * time.Millisecond),
		score:        1000,
		ground:       segments,
		groundBuff:   buff,
		ball:         NewBall(segments[0]),
		lastXinChart: lastX,

		camera: &Camera{
			Width:  float64(ScreenWidth),
			Height: float64(ScreenHeight),
		},
	}

	return game
}
