package game

import (
	"encoding/json"
	"fmt"
	"image/color"
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
	groundBuffSize = 10
	// savePointSpawn - how often save points spawns
	savePointSpawn = 10
	// savePointScore - add points after collision with save point
	savePointScore = 40
	// GameFilesDir - files related with game and levels
	GameFilesDir = "gameFiles"
	// scoreFileName - file with user score
	scoreFileName = "score"
	// defaultScore - score at first start
	defaultScore = 10000
	// max wall height
	wallHeight = 2000.0
	// redSegmentSpawn how often red segment spawns
	redSegmentSpawn = 5
)

// Draw variables
var wallColor = color.RGBA{255, 0, 0, 255}
var segmentWidth float32 = 5
var redColor = color.RGBA{255, 0, 0, 255}
var groundColor = color.RGBA{50, 255, 50, 255}
var ballColor = color.RGBA{50, 255, 50, 255}
var collSegColor = color.RGBA{1, 1, 1, 255}
var fractionsRadius = 5
var fractionsColor = color.RGBA{100, 100, 0, 255}

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

	drawError error
}

func (g *Game) Update() error {
	//  return error from Draw()
	if g.drawError != nil {
		fmt.Println("g.drawError ", g.drawError)
		return g.drawError
	}

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
			g.saveCurrentLevel()
		}
		// game logic here

		// delete old fractions by timer
		g.frameTimer.Update()
		if g.frameTimer.IsReady() {
			g.frameTimer.Reset()

			if len(g.fractions) > 20 {
				g.fractions = g.fractions[20:len(g.fractions)]
			} else if len(g.fractions) > 0 {
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
		borderSquare := newBorderSquare(groundFromBuff, minY-wallHeight, maxY)
		g.borderSquare = &borderSquare
		groundFromBuff = append(groundFromBuff, &borderSquare.left)
		groundFromBuff = append(groundFromBuff, &borderSquare.right)
		groundFromBuff = append(groundFromBuff, &borderSquare.top)
		groundFromBuff = append(groundFromBuff, &borderSquare.bottom)

		lastXcollision := 0.0

		err := g.ball.Update(&g.collisionSeg, groundFromBuff, &lastXcollision, g)
		if err != nil {
			return err
		}

		if g.getCurrentLevel().Finished {
			fmt.Println("win level")
			g.currentState = StateLevelSelect
			g.saveCurrentLevel()
			return nil
		}

		// update groundBuff if next chunk
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

// saveCurrentLevel marshals level to json and save it in file
func (g *Game) saveCurrentLevel() error {
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
		err = g.saveCurrentLevel()
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

		if i%redSegmentSpawn == 0 && i > 10 {
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
		savePoint.Position = getStartPosition(segments)

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
		g.drawPlaying(screen)
	}
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	screen.Fill(color.RGBA{220, 220, 255, 255})

	// Draw borderSquare
	borderSquare := g.borderSquare
	if borderSquare != nil {
		vector.StrokeLine(screen,
			float32(g.borderSquare.top.a.X-g.camera.X),
			float32(g.borderSquare.top.a.Y-g.camera.Y),
			float32(g.borderSquare.top.b.X-g.camera.X),
			float32(g.borderSquare.top.b.Y-g.camera.Y),
			segmentWidth/2, wallColor, false)
		vector.StrokeLine(screen,
			float32(g.borderSquare.drawRight.a.X-g.camera.X),
			float32(g.borderSquare.drawRight.a.Y-g.camera.Y),
			float32(g.borderSquare.drawRight.b.X-g.camera.X),
			float32(g.borderSquare.drawRight.b.Y-g.camera.Y),
			segmentWidth/2, wallColor, false)
		// ebitenutil.DrawLine(screen, g.borderSquare.bottom.a.X-g.camera.X, g.borderSquare.bottom.a.Y-g.camera.Y, g.borderSquare.bottom.b.X-g.camera.X, g.borderSquare.bottom.b.Y-g.camera.Y, wallColor)
		vector.StrokeLine(screen,
			float32(g.borderSquare.drawLeft.a.X-g.camera.X),
			float32(g.borderSquare.drawLeft.a.Y-g.camera.Y),
			float32(g.borderSquare.drawLeft.b.X-g.camera.X),
			float32(g.borderSquare.drawLeft.b.Y-g.camera.Y),
			segmentWidth/2, wallColor, false)
	}

	// Draw ground
	// for _, seg := range g.ground {
	// 	ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y, color.RGBA{100, 200, 100, 255})
	// }

	drawGround(screen, g.groundBuff[0], g.camera)
	drawGround(screen, g.groundBuff[1], g.camera)

	// Draw wheel
	vector.DrawFilledCircle(
		screen,
		float32(g.ball.pos.X-g.camera.X),
		float32(g.ball.pos.Y-g.camera.Y),
		float32(g.ball.radius), ballColor, false)

	// Draw collisions
	// for _, seg := range g.collisionSeg {
	// 	vector.StrokeLine(screen,
	// 		float32(seg.a.X-g.camera.X),
	// 		float32(seg.a.Y-g.camera.Y+5),
	// 		float32(seg.b.X-g.camera.X),
	// 		float32(seg.b.Y-g.camera.Y+5),
	// 		segmentWidth/2, collSegColor, false)

	// 	vector.DrawFilledCircle(screen,
	// 		float32(seg.closestPoint.X-g.camera.X),
	// 		float32(seg.closestPoint.Y-g.camera.Y),
	// 		float32(fractionsRadius), fractionsColor, false)

	// }

	for _, fr := range g.fractions {
		vector.DrawFilledCircle(screen,
			float32(fr.X-g.camera.X),
			float32(fr.Y-g.camera.Y),
			float32(fractionsRadius), fractionsColor, false)
	}

	ebitenutil.DebugPrintAt(screen, "isOnGround:"+strconv.Itoa(g.score), 10, 45)
}

func drawGround(screen *ebiten.Image, ground []*Segment, camera *Camera) {
	for _, seg := range ground {
		if seg.savePoint != nil {
			vector.DrawFilledCircle(screen,
				float32(seg.savePoint.Position.X-camera.X),
				float32(seg.savePoint.Position.Y-camera.Y),
				float32(seg.savePoint.Radius),
				color.Black, false)
		}

		color := groundColor
		if seg.isRed {
			color = redColor
		}

		vector.StrokeLine(screen,
			float32(seg.a.X-camera.X),
			float32(seg.a.Y-camera.Y),
			float32(seg.b.X-camera.X),
			float32(seg.b.Y-camera.Y),
			segmentWidth, color, false)
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
	sellLevel := make([]Button, len(g.levels))
	for i, level := range g.levels {
		levelButtons[i] = Button{
			X: 200, Y: 150 + float64(i)*80, Width: 200, Height: 60,
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

		sellLevel[i] = Button{
			X: levelButtons[i].X + levelButtons[i].Width, Y: 150 + float64(i)*80, Width: 50, Height: 60,
			Text:       "sell",
			Color:      color.RGBA{R: 70, G: 70, B: 180, A: 255},
			HoverColor: color.RGBA{R: 100, G: 100, B: 220, A: 255},
			Action: func(lvlIdx int) func() {
				return func() {
					err := resetLevel(g.levels[lvlIdx], g)
					g.drawError = err
				}
			}(i),
		}

		// Draw level button
		g.drawButton(screen, &levelButtons[i])
		if levelButtons[i].IsClicked() {
			levelButtons[i].Action()
		}

		// Draw sell level
		g.drawButton(screen, &sellLevel[i])
		if sellLevel[i].IsClicked() {
			sellLevel[i].Action()
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
