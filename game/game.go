package game

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	ScreenWidth    = 1300
	ScreenHeight   = 800
	multiplyChartX = 50
	multiplyChartY = -100
	groundBuffSize = 40
)

// Game states
const (
	StateMenu = iota
	StateLevelSelect
	StatePlaying
	StateLoadingLevel
	StateTermination
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

	// Game data
	levels       []Level
	currentLevel int
	scores       map[int]int // level index -> high score

	// Menu
	menuFont     font.Face
	titleFont    font.Face
	buttonFont   font.Face
	currentState int
	menuBg       *ebiten.Image

	// Termination checks if the exit button is pressed in the main menu
	termination bool
}

func (g *Game) Update() error {

	switch g.currentState {
	case StateMenu:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			return ebiten.Termination
		}
	case StateTermination:
		{
			return ebiten.Termination
		}
	case StateLevelSelect:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.currentState = StateMenu
		}
	case StateLoadingLevel:
		// upload level
		return g.uploadLevel()
	case StatePlaying:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.currentState = StateLevelSelect
		}
		// game logic here

		// delete old fractions by timer
		g.frameTimer.Update()
		if g.frameTimer.IsReady() {
			g.frameTimer.Reset()

			if len(g.fractions) > 0 {
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

	}
	return nil

}

func (g *Game) uploadLevel() error {

	currentLevel := g.levels[g.currentLevel]

	fmt.Println("uploadLevel: ", currentLevel)

	// Open the CSV file
	file, err := os.Open(currentLevel.Ticker)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read all records at once
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	groundPoints := []Vector{}

	// Print each record
	for _, record := range records {
		x := record[0]
		y := record[1]

		xStr, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return err
		}

		yStr, err := strconv.ParseFloat(y, 64)
		if err != nil {
			return err
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

	g.ground = segments
	g.groundBuff = buff
	g.ball = NewBall(segments[0])
	g.lastXinChart = lastX
	g.currentState = StatePlaying
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	switch g.currentState {
	case StateMenu:
		g.drawMenu(screen)
	case StateLevelSelect:
		g.drawLevelSelect(screen)
	case StateLoadingLevel:
		// draw loading
	case StatePlaying:
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

		// g.drawGame(screen)
	}

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func NewGame() *Game {

	// Menu
	// Load fonts
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	titleFont, _ := opentype.NewFace(tt, &opentype.FaceOptions{
		Size: 48, DPI: 72, Hinting: font.HintingFull,
	})
	buttonFont, _ := opentype.NewFace(tt, &opentype.FaceOptions{
		Size: 28, DPI: 72, Hinting: font.HintingFull,
	})
	menuFont, _ := opentype.NewFace(tt, &opentype.FaceOptions{
		Size: 24, DPI: 72, Hinting: font.HintingFull,
	})

	// Create menu background
	menuBg := ebiten.NewImage(ScreenWidth, ScreenHeight)
	menuBg.Fill(color.RGBA{R: 230, G: 230, B: 230, A: 230})

	// Initialize game data
	levels := []Level{
		{Name: "apple0", Ticker: "game_AAPL0.csv", Number: 1, Locked: false},
		{Name: "apple1", Ticker: "game_AAPL1.csv", Number: 2, Locked: false},
		{Name: "apple2", Ticker: "game_AAPL2.csv", Number: 3, Locked: false},
		{Name: "apple3", Ticker: "game_AAPL3.csv", Number: 4, Locked: false},
		{Name: "Final Boss", Number: 5, Locked: true},
		{Name: "Final Boss", Number: 6, Locked: true},
	}

	scores := make(map[int]int)
	scores[0] = 850  // Tutorial high score
	scores[1] = 1200 // Forest high score

	game := &Game{
		frameTimer: NewTimer(80 * time.Millisecond),
		score:      1000,

		camera: &Camera{
			Width:  float64(ScreenWidth),
			Height: float64(ScreenHeight),
		},

		// menu
		menuFont:     menuFont,
		titleFont:    titleFont,
		buttonFont:   buttonFont,
		currentState: StateMenu,
		menuBg:       menuBg,
		levels:       levels,
		scores:       scores,
	}

	return game
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	screen.DrawImage(g.menuBg, nil)

	// Draw title
	title := "STOCK JUMPER"
	bounds := text.BoundString(g.titleFont, title)
	text.Draw(screen, title, g.titleFont,
		(ScreenWidth-bounds.Dx())/2, 150,
		color.RGBA{R: 0, G: 0, B: 0, A: 255})

	// Create and draw buttons
	buttons := []Button{
		{
			X: ScreenWidth/2 - 200, Y: 200, Width: 400, Height: 60,
			Text: "PLAY", Color: color.RGBA{R: 80, G: 200, B: 100, A: 255},
			HoverColor: color.RGBA{R: 100, G: 220, B: 110, A: 255},
			Action:     func() { g.currentState = StateLevelSelect },
		},
		{
			X: ScreenWidth/2 - 100, Y: 280, Width: 200, Height: 60,
			Text: "QUIT", Color: color.RGBA{R: 230, G: 40, B: 40, A: 255},
			HoverColor: color.RGBA{R: 255, G: 50, B: 50, A: 255},
			Action:     func() { g.currentState = StateTermination },
		},
	}

	for i, btn := range buttons {
		g.drawButton(screen, &buttons[i])
		if btn.IsClicked() {
			btn.Action()
		}
	}
}

func (g *Game) drawLevelSelect(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 20, B: 40, A: 255})

	// Draw title
	title := "SELECT LEVEL"
	bounds := text.BoundString(g.titleFont, title)
	text.Draw(screen, title, g.titleFont,
		(ScreenWidth-bounds.Dx())/2, 80,
		color.White)

	// Draw levels
	levelButtons := make([]Button, len(g.levels))
	for i, level := range g.levels {
		levelButtons[i] = Button{
			X: 200, Y: 150 + float64(i)*80, Width: 400, Height: 60,
			Text:       level.Name,
			Color:      color.RGBA{R: 70, G: 70, B: 180, A: 255},
			HoverColor: color.RGBA{R: 100, G: 100, B: 220, A: 255},
			Action: func(lvlIdx int) func() {
				return func() {
					if !g.levels[lvlIdx].Locked {
						g.currentLevel = lvlIdx
						g.currentState = StateLoadingLevel
					}
				}
			}(i),
		}

		// Draw level button
		g.drawButton(screen, &levelButtons[i])
		if levelButtons[i].IsClicked() {
			levelButtons[i].Action()
		}

		// Draw level info (number and score)
		infoText := ""
		if level.Locked {
			infoText = "LOCKED"
		} else if score, exists := g.scores[i]; exists {
			infoText = fmt.Sprintf("High Score: %d", score)
		} else {
			infoText = "Not Played"
		}

		text.Draw(screen, infoText, g.menuFont,
			620, int(180+float64(i)*80),
			color.White)
	}
}

func (g *Game) drawButton(screen *ebiten.Image, btn *Button) {
	// Check hover state
	mx, my := ebiten.CursorPosition()
	hover := float64(mx) > btn.X && float64(mx) < btn.X+btn.Width &&
		float64(my) > btn.Y && float64(my) < btn.Y+btn.Height

	// Choose color
	btnColor := btn.Color
	if hover {
		btnColor = btn.HoverColor
	}

	// Draw button
	ebitenutil.DrawRect(screen, btn.X, btn.Y, btn.Width, btn.Height, btnColor)

	// Draw button text
	bounds := text.BoundString(g.buttonFont, btn.Text)
	textX := btn.X + (btn.Width-float64(bounds.Dx()))/2
	textY := btn.Y + (btn.Height)/2 + float64(bounds.Dy())/2
	text.Draw(screen, btn.Text, g.buttonFont, int(textX), int(textY), color.White)
}

func (b *Button) IsClicked() bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		return float64(mx) > b.X && float64(mx) < b.X+b.Width &&
			float64(my) > b.Y && float64(my) < b.Y+b.Height
	}
	return false
}
