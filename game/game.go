package game

import (
	"encoding/csv"
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
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
	ScreenWidth  = 1300
	ScreenHeight = 800
	// expand chart by x and y
	multiplyChartX = 50
	multiplyChartY = -100
	// groundBuffSize - buffer consist of two slices of ground, groundBuffSize is size of one slice
	groundBuffSize = 10
	// savePointSpawn - how often save points are spawn
	savePointSpawn = 10
	// savePointScore - add points after collision with save point
	savePointScore = 20.0
	// GameFilesDir - files related with game and levels
	GameFilesDir = "gameFiles"
	// scoreFileName - file with user score
	scoreFileName = "score"
	// defaultScore - score at first start
	defaultScore = 10000
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
	groundBuff [2][]*Segment

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
	var err error
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
			err = saveBinary(g.score, filepath.Join(GameFilesDir, scoreFileName))
			if err != nil {
				return err
			}

			// save savePoint to file
			if g.ball.savePoint != nil {
				err = saveBinary(*g.ball.savePoint, filepath.Join(GameFilesDir, g.levels[g.currentLevel].Ticker))
				if err != nil {
					return err
				}
			}
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

		// fill unite slice from g.groundBuff[0] and g.groundBuff[1]
		groundFromBuff := make([]*Segment, 0, len(g.groundBuff[0])+len(g.groundBuff[1]))
		groundFromBuff = append(groundFromBuff, g.groundBuff[0]...)
		groundFromBuff = append(groundFromBuff, g.groundBuff[1]...)

		wallLeftX := g.groundBuff[0][0].a
		wallRightX := g.groundBuff[1][len(g.groundBuff[1])-1].b
		wallHeight := 2000.0
		wallLeft := &Segment{
			a:     Vector{wallLeftX.X, wallLeftX.Y},
			b:     Vector{wallLeftX.X, wallLeftX.Y - wallHeight},
			isRed: true,
		}
		wallRight := &Segment{
			a:     Vector{wallRightX.X, wallRightX.Y},
			b:     Vector{wallRightX.X, wallRightX.Y - wallHeight},
			isRed: true,
		}
		groundFromBuff = append(groundFromBuff, wallLeft)
		groundFromBuff = append(groundFromBuff, wallRight)

		lastBuffX := 0.0
		lastXSecBuff := g.groundBuff[1][len(g.groundBuff[1])-1].b.X
		err := g.ball.Update(&g.score, &g.collisionSeg, &g.fractions, groundFromBuff, &lastBuffX, wallLeftX.X, lastXSecBuff)
		if err != nil {
			return err
		}

		// update groundBuff if next chunk
		if lastBuffX >= g.groundBuff[1][0].b.X && g.lastXinChart > lastXSecBuff && g.groundBuff[1][0].b.X < g.ball.pos.X-g.ball.radius {
			secondBuffI := int(lastXSecBuff) / multiplyChartX

			// Swap buffers
			g.groundBuff[0], g.groundBuff[1] = g.groundBuff[1], g.groundBuff[0]

			// Calculate safe copy size
			copySize := min(groundBuffSize, len(g.ground)-secondBuffI)

			// Reset and populate the new buffer
			g.groundBuff[1] = make([]*Segment, copySize)
			for i := 0; i < copySize; i++ {
				g.groundBuff[1][i] = &g.ground[secondBuffI+i]
			}
		}

		// Update camera
		g.camera.Update(g.ball.pos.X, g.ball.pos.Y)

	}
	return err
}
func (g *Game) uploadLevel() error {
	level := g.levels[g.currentLevel]

	// Read and parse CSV data
	groundPoints, err := g.readLevelCSV(filepath.Join(GameFilesDir, (level.Ticker + ".csv")))
	if err != nil {
		return fmt.Errorf("failed to read level data: %w", err)
	}

	// Create segments with save points
	segments, lastX, _ := g.createSegments(groundPoints)

	if (len(segments)) <= groundBuffSize*2 {
		return fmt.Errorf("too small points for level")
	}
	// Initialize game state
	err = g.initializeLevelState(segments, lastX, level)
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) readLevelCSV(filename string) ([]Vector, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}

	points := make([]Vector, 0, len(records))
	for _, record := range records {
		x, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid X coordinate: %w", err)
		}

		y, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid Y coordinate: %w", err)
		}

		points = append(points, Vector{
			X: x * multiplyChartX,
			Y: y * multiplyChartY,
		})
	}

	return points, nil
}

func (g *Game) createSegments(points []Vector) ([]Segment, float64, float64) {
	segments := make([]Segment, len(points)-1)
	minY := 0.0

	for i := 0; i < len(segments); i++ {
		seg := Segment{
			a: points[i],
			b: points[i+1],
		}

		if i%savePointSpawn == 0 {
			seg.savePoint = &SavePoint{
				Position: Vector{
					X: seg.AvrX(),
					Y: seg.MinY() - 100,
				},
				Radius: 20,
			}
		}

		if i%3 == 0 {
			seg.isRed = true
		}

		if seg.MinY() > minY {
			minY = seg.MinY()
		}

		segments[i] = seg
	}

	return segments, points[len(points)-1].X, minY
}

func (g *Game) initializeLevelState(segments []Segment, lastX float64, level Level) error {
	g.ground = segments

	// get spawn position
	var savePoint SavePoint
	savePointP := &savePoint

	firstStart := false

	err := loadBinary(savePointP, filepath.Join(GameFilesDir, level.Ticker))
	if err != nil {
		// if ErrNotExist then skip err
		if errors.Is(err, os.ErrNotExist) {
			firstStart = true
		} else {
			return err
		}
	}

	if firstStart {
		minY := min(segments[0].a.Y, segments[0].b.Y) - ballPhysicA.radius
		savePointP.Position = Vector{segments[0].b.X, minY}

		g.groundBuff = [2][]*Segment{
			makePointers(segments[:groundBuffSize]),
			makePointers(segments[groundBuffSize : groundBuffSize*2]),
		}
	} else {
		groundIndex := int(savePointP.Position.X) / multiplyChartX

		// Calculate safe copy size
		copySize := min(groundBuffSize, len(g.ground)-(groundIndex))

		// if save point at the end of the chart
		if copySize <= groundBuffSize {
			g.groundBuff = [2][]*Segment{
				makePointers(segments[groundIndex : groundIndex+copySize]),
				makePointers(segments[groundIndex : groundIndex+copySize]),
			}
		} else {
			g.groundBuff = [2][]*Segment{
				makePointers(segments[groundIndex : groundIndex+groundBuffSize]),
				makePointers(segments[groundIndex+groundBuffSize : groundIndex+groundBuffSize*2]),
			}
		}

		// delete savePoint from spawn
		segments[groundIndex].savePoint = nil
	}

	g.ball = NewBall(savePointP.Position)
	g.lastXinChart = lastX
	g.currentState = StatePlaying

	return nil
}

func makePointers(segments []Segment) []*Segment {
	pointers := make([]*Segment, len(segments))
	for i := range segments {
		pointers[i] = &segments[i]
	}
	return pointers
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

		// Draw wall
		wallLeftX := g.groundBuff[0][0].a
		wallRightX := g.groundBuff[1][len(g.groundBuff[1])-1].b
		wallHeight := 2000.0
		wallColor := color.RGBA{100, 200, 100, 255}
		ebitenutil.DrawLine(screen, wallLeftX.X-g.camera.X, wallLeftX.Y-g.camera.Y, wallLeftX.X-g.camera.X, wallLeftX.Y-wallHeight-g.camera.Y, wallColor)
		ebitenutil.DrawLine(screen, wallRightX.X-g.camera.X, wallRightX.Y-g.camera.Y, wallRightX.X-g.camera.X, wallRightX.Y-wallHeight-g.camera.Y, wallColor)

		// Draw ground
		for _, seg := range g.ground {

			if seg.isRed {
				ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y, color.RGBA{255, 0, 0, 255})

			} else {
				ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y, color.RGBA{100, 200, 100, 255})

			}
		}

		for _, seg := range g.groundBuff[0] {
			ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y-5, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y-5, color.RGBA{255, 200, 100, 255})

			if seg.savePoint != nil {
				ebitenutil.DrawCircle(screen, seg.savePoint.Position.X-g.camera.X, seg.savePoint.Position.Y-g.camera.Y, seg.savePoint.Radius, color.Black)
			}
		}

		for _, seg := range g.groundBuff[1] {
			ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y+5, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y+5, color.RGBA{0, 100, 200, 255})

			if seg.savePoint != nil {
				ebitenutil.DrawCircle(screen, seg.savePoint.Position.X-g.camera.X, seg.savePoint.Position.Y-g.camera.Y, seg.savePoint.Radius, color.Black)
			}
		}

		for _, seg := range g.gameParallelSeg {
			ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y, color.RGBA{100, 100, 255, 255})
		}

		// Draw wheel
		w := g.ball

		if w.facingRight {
			ebitenutil.DrawCircle(screen, w.pos.X-g.camera.X, w.pos.Y-g.camera.Y, w.radius, color.Black)

		} else {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(w.pos.X-g.camera.X, w.pos.Y-g.camera.Y)
			ebitenutil.DrawCircle(screen, w.pos.X-g.camera.X, w.pos.Y-g.camera.Y, w.radius, color.Black)
		}

		// Show direction line
		dirX := w.pos.X + w.radius*math.Cos(0)
		dirY := w.pos.Y + w.radius*math.Sin(0)
		ebitenutil.DrawLine(screen, w.pos.X-g.camera.X, w.pos.Y-g.camera.Y, dirX-g.camera.X, dirY-g.camera.Y, color.RGBA{255, 0, 0, 255})

		// Draw ground
		for _, seg := range g.collisionSeg {
			// r := uint8(rand.Intn(255))
			// g := uint8(rand.Intn(255))
			// b := uint8(rand.Intn(255))
			// ebitenutil.DrawLine(screen, seg.a.x, seg.a.y, seg.b.x, seg.b.y, color.RGBA{r, g, b, 50})

			ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y, color.RGBA{1, 1, 1, 255})

			ebitenutil.DrawCircle(screen, seg.closestPoint.X-g.camera.X, seg.closestPoint.Y-g.camera.Y, 5, color.RGBA{255, 0, 0, 255})

			ebitenutil.DrawCircle(screen, seg.closestPoint.X-g.camera.X, seg.closestPoint.Y-g.camera.Y, 5, color.RGBA{255, 0, 0, 255})

		}

		for _, fr := range g.fractions {
			ebitenutil.DrawCircle(screen, fr.X-g.camera.X, fr.Y-g.camera.Y, 5, color.RGBA{100, 100, 0, 255})
		}

		ebitenutil.DebugPrintAt(screen, "isOnGround:"+strconv.Itoa(g.score), 10, 45)

		// g.drawGame(screen)
	}

}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func NewGame() (*Game, error) {

	// Menu
	// Load fonts
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		return nil, err
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
		{Name: "apple0", Ticker: "game_AAPL0", Number: 1, Locked: false},
		{Name: "apple1", Ticker: "game_AAPL1", Number: 2, Locked: false},
		{Name: "apple2", Ticker: "game_AAPL2", Number: 3, Locked: false},
		{Name: "apple3", Ticker: "game_AAPL3", Number: 4, Locked: false},
		{Name: "Final Boss", Number: 5, Locked: true},
		{Name: "Final Boss", Number: 6, Locked: true},
	}

	scores := make(map[int]int)
	scores[0] = 850  // Tutorial high score
	scores[1] = 1200 // Forest high score

	var defScore int = defaultScore
	score := &defScore

	scoreFilePath := filepath.Join(GameFilesDir, scoreFileName)

	err = loadBinary(score, scoreFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("'%s' file not exist, create default score file", scoreFilePath)
			err = saveBinary(score, scoreFilePath)
			if err != nil {
				return nil, err
			}
		}
	}

	game := &Game{
		frameTimer: NewTimer(80 * time.Millisecond),
		score:      *score,

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

	return game, nil
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
