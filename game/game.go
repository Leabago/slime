package game

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	groundBuffSize = 5
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
	// max wall height
	wallHeight = 200.0
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

	// lastXinChart    float64
	gameParallelSeg []Segment

	// Game data
	levels       []*Level
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

	// wall
	borderSquare *BorderSquare
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
			g.fractions = []Vector{}
			err = saveBinary(g.score, filepath.Join(GameFilesDir, scoreFileName))
			if err != nil {
				return err
			}

			// save level to file
			g.saveLevel()
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

		lenBuff := len(groundFromBuff)
		middleSegment := groundFromBuff[len(groundFromBuff)/2]
		lastXbuff := groundFromBuff[len(groundFromBuff)-1].b.X

		minY, maxY := findMinMaxY(groundFromBuff)
		borderSquare := newBorderSquare(groundFromBuff[0].a.X, groundFromBuff[len(groundFromBuff)-1].b.X, minY-wallHeight, maxY)
		g.borderSquare = &borderSquare
		groundFromBuff = append(groundFromBuff, &borderSquare.left)
		groundFromBuff = append(groundFromBuff, &borderSquare.right)
		groundFromBuff = append(groundFromBuff, &borderSquare.top)
		groundFromBuff = append(groundFromBuff, &borderSquare.bottom)

		lastXcollision := 0.0

		err := g.ball.Update(&g.collisionSeg, &g.fractions, groundFromBuff, &lastXcollision, g)
		if err != nil {
			return err
		}

		if g.getCurrentLevel().Finished {
			fmt.Println("win level")
			g.currentState = StateLevelSelect
			g.saveLevel()
			return nil
		}

		// update groundBuff if next chunk
		// if lastXcollision >= g.groundBuff[1][0].b.X && g.getCurrentLevel().MaxX > lastXbuff && g.groundBuff[1][0].b.X < g.ball.pos.X-(g.ball.radius*2) {
		// 	secondBuffI := int(lastXbuff) / multiplyChartX

		if g.ball.pos.X > middleSegment.b.X && lenBuff >= groundBuffSize*2 {
			// Swap buffers
			g.groundBuff[0], g.groundBuff[1] = g.groundBuff[1], g.groundBuff[0]

			// Calculate safe copy size
			secondBuffI := int(lastXbuff) / multiplyChartX
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

// saveLevel marshals level to json and save it in file
func (g *Game) saveLevel() error {
	// save level to file

	level := g.getCurrentLevel()

	levelJson, err := json.Marshal(level)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(GameFilesDir, getJsonName(level.Ticker)), levelJson, 644)
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) uploadLevel() error {

	// Read and parse CSV data
	groundPoints, err := readLevelCSV(filepath.Join(GameFilesDir, (g.getCurrentLevel().ChartFile)))
	if err != nil {
		return fmt.Errorf("failed to read level data: %w", err)
	}

	// Create segments with save points
	segments, maxX, maxY := g.createSegments(groundPoints)

	if (len(segments)) <= groundBuffSize*2 {
		return fmt.Errorf("too small points for level")
	}
	// Initialize game state
	err = g.initializeLevelState(segments, maxX)
	if err != nil {
		return err
	}

	if g.getCurrentLevel().MaxX == 0 {
		g.getCurrentLevel().MaxX = maxX
		g.getCurrentLevel().MaxY = maxY
		err = g.saveLevel()
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) createSegments(points []Vector) ([]Segment, float64, float64) {
	segments := make([]Segment, len(points)-1)
	maxY := 0.0

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

		segments[i] = seg

		if seg.MinY() < maxY {
			maxY = seg.MinY()
		}
	}

	// create last save point
	segments[len(segments)-1].savePoint = &SavePoint{
		Position: Vector{
			X: segments[len(segments)-1].a.X,
			Y: segments[len(segments)-1].MinY() - 100,
		},

		IsFinish: true,
		Radius:   40,
	}

	return segments, points[len(points)-1].X, maxY
}

func (g *Game) initializeLevelState(segments []Segment, lastX float64) error {
	g.ground = segments

	// get spawn position
	savePoint := g.getCurrentLevel().SavePoint

	fmt.Println("savePoint: ", savePoint)

	// if the level was launched for the first time
	if savePoint == nil {
		savePoint = &SavePoint{}
		savePoint.Position = getStartPosition2(segments)

		g.groundBuff = [2][]*Segment{
			makeSegments(segments[:groundBuffSize]),
			makeSegments(segments[groundBuffSize : groundBuffSize*2]),
		}
	} else {
		groundIndex := int(savePoint.Position.X) / multiplyChartX
		groundIndex -= groundBuffSize
		if groundIndex < 0 {
			groundIndex = 0
		}

		// Calculate safe copy size
		safeLeftSize := min(groundBuffSize, len(g.ground)-(groundIndex))
		safeRightSize := min(groundBuffSize, len(g.ground)-(groundIndex+safeLeftSize))
		if safeRightSize < 0 {
			safeRightSize = 0
		}

		// if save point at the end of the chart

		g.groundBuff = [2][]*Segment{
			makeSegments(segments[groundIndex : groundIndex+safeLeftSize]),
			makeSegments(segments[groundIndex+safeLeftSize : groundIndex+safeLeftSize+safeRightSize]),
		}

		// if safeLeftSize < groundBuffSize {
		// 	g.groundBuff = [2][]*Segment{
		// 		makeSegments(segments[groundIndex : groundIndex+safeLeftSize]),
		// 		makeSegments(segments[groundIndex : groundIndex+safeLeftSize]),
		// 	}
		// } else {

		// 	copySize := min(groundBuffSize, len(g.ground)-(groundIndex+groundBuffSize))

		// 	g.groundBuff = [2][]*Segment{
		// 		makeSegments(segments[groundIndex : groundIndex+groundBuffSize]),
		// 		makeSegments(segments[groundIndex+copySize : groundIndex+groundBuffSize+copySize]),
		// 	}
		// }
		// else {
		// 	g.groundBuff = [2][]*Segment{
		// 		makeSegments(segments[groundIndex : groundIndex+groundBuffSize]),
		// 		makeSegments(segments[groundIndex+groundBuffSize : groundIndex+groundBuffSize*2]),
		// 	}
		// }

		// delete savePoint from spawn
		segments[groundIndex].savePoint = nil
	}

	g.ball = NewBall(savePoint.Position)
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

		// Draw wall
		// wallLeftX := g.groundBuff[0][0].a
		// wallRightX := g.groundBuff[1][len(g.groundBuff[1])-1].b
		wallColor := color.RGBA{255, 0, 0, 255}

		// borderSquare
		borderSquare := g.borderSquare
		if borderSquare != nil {
			ebitenutil.DrawLine(screen, g.borderSquare.top.a.X-g.camera.X, g.borderSquare.top.a.Y-g.camera.Y, g.borderSquare.top.b.X-g.camera.X, g.borderSquare.top.b.Y-g.camera.Y, wallColor)
			ebitenutil.DrawLine(screen, g.borderSquare.right.a.X-g.camera.X, g.borderSquare.right.a.Y-g.camera.Y, g.borderSquare.right.b.X-g.camera.X, g.borderSquare.right.b.Y-g.camera.Y, wallColor)
			ebitenutil.DrawLine(screen, g.borderSquare.bottom.a.X-g.camera.X, g.borderSquare.bottom.a.Y-g.camera.Y, g.borderSquare.bottom.b.X-g.camera.X, g.borderSquare.bottom.b.Y-g.camera.Y, wallColor)
			ebitenutil.DrawLine(screen, g.borderSquare.left.a.X-g.camera.X, g.borderSquare.left.a.Y-g.camera.Y, g.borderSquare.left.b.X-g.camera.X, g.borderSquare.left.b.Y-g.camera.Y, wallColor)
		}

		// Draw ground
		for _, seg := range g.ground {
			ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y, color.RGBA{100, 200, 100, 255})

		}

		for _, seg := range g.groundBuff[0] {
			// ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y-5, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y-5, color.RGBA{255, 200, 100, 255})

			if seg.savePoint != nil {
				ebitenutil.DrawCircle(screen, seg.savePoint.Position.X-g.camera.X, seg.savePoint.Position.Y-g.camera.Y, seg.savePoint.Radius, color.Black)
			}

			if seg.isRed {
				vector.StrokeLine(screen, float32(seg.a.X-g.camera.X), float32(seg.a.Y-g.camera.Y-5), float32(seg.b.X-g.camera.X), float32(seg.b.Y-g.camera.Y-5), 1, color.RGBA{255, 0, 0, 255}, false)
			} else {
				vector.StrokeLine(screen, float32(seg.a.X-g.camera.X), float32(seg.a.Y-g.camera.Y-5), float32(seg.b.X-g.camera.X), float32(seg.b.Y-g.camera.Y-5), 1, color.RGBA{255, 200, 100, 255}, false)
			}
		}

		for _, seg := range g.groundBuff[1] {
			if seg.isRed {
				vector.StrokeLine(screen, float32(seg.a.X-g.camera.X), float32(seg.a.Y-g.camera.Y+5), float32(seg.b.X-g.camera.X), float32(seg.b.Y-g.camera.Y+5), 5, color.RGBA{255, 0, 0, 255}, false)
			} else {
				vector.StrokeLine(screen, float32(seg.a.X-g.camera.X), float32(seg.a.Y-g.camera.Y+5), float32(seg.b.X-g.camera.X), float32(seg.b.Y-g.camera.Y+5), 5, color.RGBA{0, 100, 200, 255}, false)
			}
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

	dirEntry, err := os.ReadDir(GameFilesDir)
	if err != nil {
		return nil, err
	}

	// Initialize game data
	levels := []*Level{}
	scores := make(map[int]int)
	for _, e := range dirEntry {
		// find levels
		if strings.Contains(e.Name(), ".json") {
			file, err := os.ReadFile(filepath.Join(GameFilesDir, e.Name()))
			if err != nil {
				return nil, err
			}
			level := &Level{}
			err = json.Unmarshal(file, level)
			if err != nil {
				return nil, err
			}
			levels = append(levels, level)

			scores[level.Number] = level.Score
		}
	}

	// load score from file or use default score
	score, err := LoadScore()
	if err != nil {
		return nil, err
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
		color.Black)

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
	screen.DrawImage(g.menuBg, nil)
	// screen.Fill(color.RGBA{R: 20, G: 20, B: 40, A: 255})

	// Draw title
	title := "SELECT LEVEL"
	bounds := text.BoundString(g.titleFont, title)
	text.Draw(screen, title, g.titleFont,
		(ScreenWidth-bounds.Dx())/2, 80,
		color.Black)

	// Draw score
	score := fmt.Sprintf("You current score is %s $", strconv.Itoa(g.score))
	// boundScore := text.BoundString(g.titleFont, score)
	text.Draw(screen, score, g.buttonFont,
		200, 120,
		color.Black)

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
					if !g.levels[lvlIdx].Finished {
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
		if level.Finished {
			infoText = "FINISHED"
		}

		infoText += " | " + fmt.Sprintf("Score: %d", level.Score) + "$"
		infoText += " | " + strconv.Itoa(calculateLevelProgress(*level)) + "%"

		text.Draw(screen, infoText, g.menuFont,
			620, int(180+float64(i)*80),
			color.Black)
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

func (g *Game) getCurrentLevel() *Level {
	return g.levels[g.currentLevel]
}
