package game

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"math/rand"
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
	// multiplyChartX = 50
	// multiplyChartY = -20
	multiplyChartX = 10
	multiplyChartY = -50
	// groundBuffSize - buffer consist of two slices of ground, groundBuffSize is size of one slice
	groundBuffSize = 40
	// savePointSpawn - how often save points spawns
	savePointSpawn = 10
	// savePointScore - add points after collision with save point
	savePointScore = 300
	// GameFilesDir - files related with game and levels
	GameFilesDir = "gameFiles"
	// scoreFileName - file with user score
	scoreFileName = "score"
	// defaultScore - score at first start
	defaultScore = 10000
	// max wall height
	wallHeight = 2000.0
	// redSegmentSpawn how often red segment spawns
	redSegmentSpawn = 50

	// levelFirstScore first score at start level
	levelFirstScore = 0

	// movingWall speed
	movWallSpeedHight = 15
	movWallSpeedSlow  = 2
)

// Draw variables
var playBackground = color.RGBA{0, 0, 0, 255}
var wallColor = color.RGBA{200, 10, 60, 255}
var savePointColor = color.RGBA{130, 255, 130, 255}
var groundColor = color.RGBA{10, 60, 60, 255}
var groundColorHover = color.RGBA{30, 90, 90, 255}
var ballColor = color.RGBA{70, 150, 70, 255}
var ballColorBig = color.RGBA{90, 180, 90, 200}

var segmentWidth float32 = 5
var fractionsRadius = 10

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
	enemyBall    *Ball
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
	movingWall   *Segment

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
			return returnToSelectLevel(g)
		}

		// game logic here
		return g.gameUpdate()
	}
	return err
}

func (g *Game) gameUpdate() error {
	// delete old fractions by timer
	g.updateFrame()
	// update MovingWall
	g.updateMovingWall()

	// update savePoint position
	updateSavePointPosition(g.groundBuff[0])
	updateSavePointPosition(g.groundBuff[1])

	// update enemy
	if g.enemyBall != nil {
		g.enemyBall.pos = g.enemyBall.pos.Add(g.enemyBall.vel)
	}
	// fill Ground slice
	groundFromBuff, lenBuff, middleSegment, lastXbuff := g.fillGround()

	err := g.ball.Update(&g.collisionSeg, groundFromBuff, g)
	if err != nil {
		return err
	}

	if g.ball.isDied {
		g.getCurrentLevel().Score = 0
		g.getCurrentLevel().SavePoint = nil
		return returnToSelectLevel(g)

	}

	if g.getCurrentLevel().Finished {
		return returnToSelectLevel(g)
	}

	g.updateGroundBuffer(middleSegment, lenBuff, lastXbuff)

	// Update camera
	g.camera.Update(g.ball.pos.X, g.ball.pos.Y)

	return nil
}

func (g *Game) updateGroundBuffer(middleSegment *Segment, lenBuff int, lastXbuff float64) {
	// update groundBuff if next chunk
	if g.ball.pos.X > middleSegment.B.X && lenBuff >= groundBuffSize*2 {
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

		// update wall
		if g.movingWall.A.X > g.ball.pos.X {
			g.movingWall.A.X = g.groundBuff[0][0].A.X
			g.movingWall.B.X = g.groundBuff[0][0].A.X
		}
	}
}

func (g *Game) fillGround() (groundFromBuff []*Segment, lenBuff int, middleSegment *Segment, lastXbuff float64) {

	// fill unite slice from g.groundBuff[0] and g.groundBuff[1]
	groundFromBuff = make([]*Segment, 0, len(g.groundBuff[0])+len(g.groundBuff[1]))
	groundFromBuff = append(groundFromBuff, g.groundBuff[0]...)
	groundFromBuff = append(groundFromBuff, g.groundBuff[1]...)

	lenBuff = len(groundFromBuff)
	middleSegment = groundFromBuff[int(float64(len(groundFromBuff))/1.5)]
	lastXbuff = groundFromBuff[len(groundFromBuff)-1].B.X

	borderSquare := newBorderSquare(groundFromBuff)
	g.borderSquare = &borderSquare
	groundFromBuff = append(groundFromBuff, &borderSquare.left)
	groundFromBuff = append(groundFromBuff, &borderSquare.right)
	groundFromBuff = append(groundFromBuff, &borderSquare.top)
	groundFromBuff = append(groundFromBuff, g.movingWall)

	return groundFromBuff, lenBuff, middleSegment, lastXbuff
}

// updateMovingWall update MovingWall
func (g *Game) updateMovingWall() {
	// increse speed if movingWall too far
	distanceWallBall := math.Abs(g.ball.pos.X - g.movingWall.A.X)
	if distanceWallBall > ScreenWidth {
		g.movingWall.A.X += movWallSpeedHight
		g.movingWall.B.X += movWallSpeedHight
	} else {
		g.movingWall.A.X += movWallSpeedSlow
		g.movingWall.B.X += movWallSpeedSlow
	}
}

func (g *Game) updateFrame() {
	g.frameTimer.Update()
	if g.frameTimer.IsReady() {
		g.frameTimer.Reset()

		if len(g.fractions) > 20 {
			g.fractions = g.fractions[20:len(g.fractions)]
		} else if len(g.fractions) > 0 {
			g.fractions = g.fractions[1:len(g.fractions)]
		}
	}
}

// saveCurrentLevel marshals level to json and save it in file
func (g *Game) saveCurrentLevel() error {
	// save level to file
	level := g.getCurrentLevel()
	level.MovingWall = g.movingWall

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
	err = g.initializeLevelState(segments, maxX, maxY)
	if err != nil {
		return err
	}

	// add maxX maxY to file if 0
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

	redCount := 100

	for i := 0; i < len(segments); i++ {
		seg := Segment{
			A: points[i],
			B: points[i+1],
		}

		if i%savePointSpawn == 0 {
			startPosition := seg.GetPosWithMinY()
			startPosition = startPosition.Sub(seg.Normal().Mul(15))

			pos := startPosition
			randInt := float64(rand.Intn(200))
			pos.Y -= randInt

			seg.savePoint = &SavePoint{
				Position: Vector{
					X: pos.X,
					Y: pos.Y,
				},
				startPosition: startPosition,
				Radius:        20,
			}
		}

		// set red segment
		if i%redSegmentSpawn == 0 && i > groundBuffSize {
			redCount = 0
		}

		if redCount < 4 {
			seg.isRed = true
			redCount++
		}

		segments[i] = seg

		if seg.MinY() < maxY {
			maxY = seg.MinY()
		}
	}

	// create last save point
	pos := Vector{
		X: segments[len(segments)-1].A.X,
		Y: segments[len(segments)-1].MinY() - 100,
	}

	segments[len(segments)-1].savePoint = &SavePoint{
		Position:      pos,
		startPosition: pos,
		IsFinish:      true,
		Radius:        50,
	}

	return segments, segments[len(segments)-1].B.X, maxY
}

func (g *Game) initializeLevelState(segments []Segment, lastX, maxY float64) error {
	g.ground = segments

	// get spawn position
	savePoint := g.getCurrentLevel().SavePoint

	fmt.Println("-----------------------------:")
	fmt.Println("savePoint:", savePoint)

	// if the level was launched for the first time
	if savePoint == nil {
		savePoint = &SavePoint{}

		g.groundBuff = [2][]*Segment{
			makeSegments(segments[:groundBuffSize]),
			makeSegments(segments[groundBuffSize : groundBuffSize*2]),
		}

		savePoint.Position = getStartPosition(g.groundBuff[0])

		// set moving wall
		g.movingWall = &Segment{
			A:            Vector{-ScreenWidth, 0},
			B:            Vector{-ScreenWidth, maxY - wallHeight},
			isMovingWall: true,
		}
	} else {
		savePointIndex := int(savePoint.Position.X) / multiplyChartX
		groundIndex := savePointIndex - groundBuffSize

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
		segments[savePointIndex].savePoint = nil

		if g.getCurrentLevel().MovingWall != nil {
			g.movingWall = g.getCurrentLevel().MovingWall
		}
		if g.movingWall.A.X > savePoint.Position.X || g.movingWall.B.X > savePoint.Position.X || g.movingWall.B.Y > maxY-wallHeight {
			g.movingWall = &Segment{
				A:            Vector{g.groundBuff[0][0].A.X, 0},
				B:            Vector{g.groundBuff[0][0].A.X, maxY - wallHeight},
				isMovingWall: true,
			}
		}
	}

	g.ball = NewBall(savePoint.Position)
	g.currentState = StatePlaying
	spawnEnemy := g.groundBuff[0][len(g.groundBuff[0])-1].A
	spawnEnemy.Y -= 400
	spawnEnemy.X -= 600
	g.enemyBall = NewBall(spawnEnemy)
	g.enemyBall.vel = Vector{-10, 1}

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
	screen.Fill(playBackground)

	// Draw borderSquare
	borderSquare := g.borderSquare
	if borderSquare != nil {
		vector.StrokeLine(screen,
			float32(g.borderSquare.top.A.X-g.camera.X),
			float32(g.borderSquare.top.A.Y-g.camera.Y),
			float32(g.borderSquare.top.B.X-g.camera.X),
			float32(g.borderSquare.top.B.Y-g.camera.Y),
			segmentWidth, wallColor, false)
		vector.StrokeLine(screen,
			float32(g.borderSquare.drawRight.A.X-g.camera.X),
			float32(g.borderSquare.drawRight.A.Y-g.camera.Y),
			float32(g.borderSquare.drawRight.B.X-g.camera.X),
			float32(g.borderSquare.drawRight.B.Y-g.camera.Y),
			segmentWidth, wallColor, false)
		vector.StrokeLine(screen,
			float32(g.borderSquare.bottom.A.X-g.camera.X),
			float32(g.borderSquare.bottom.A.Y-g.camera.Y),
			float32(g.borderSquare.bottom.B.X-g.camera.X),
			float32(g.borderSquare.bottom.B.Y-g.camera.Y),
			segmentWidth, wallColor, false)
		vector.StrokeLine(screen,
			float32(g.borderSquare.drawLeft.A.X-g.camera.X),
			float32(g.borderSquare.drawLeft.A.Y-g.camera.Y),
			float32(g.borderSquare.drawLeft.B.X-g.camera.X),
			float32(g.borderSquare.drawLeft.B.Y-g.camera.Y),
			segmentWidth, wallColor, false)
	}

	// Draw moving Wall
	vector.StrokeLine(screen,
		float32(g.movingWall.A.X-g.camera.X),
		float32(g.movingWall.A.Y-g.camera.Y),
		float32(g.movingWall.B.X-g.camera.X),
		float32(g.movingWall.B.Y-g.camera.Y),
		segmentWidth, wallColor, false)

	// Draw ground
	// for _, seg := range g.ground {
	// 	ebitenutil.DrawLine(screen, seg.a.X-g.camera.X, seg.a.Y-g.camera.Y, seg.b.X-g.camera.X, seg.b.Y-g.camera.Y, color.RGBA{100, 200, 100, 255})
	// }

	drawGround(screen, g.groundBuff[0], g.camera)
	drawGround(screen, g.groundBuff[1], g.camera)

	// Draw ball
	ballColor := ballColor
	if g.ball.currPhyState.state == phyStateB {
		ballColor = ballColorBig
	}
	vector.DrawFilledCircle(
		screen,
		float32(g.ball.pos.X-g.camera.X),
		float32(g.ball.pos.Y-g.camera.Y),
		float32(g.ball.radius), ballColor, false)

	// Draw enemy
	if g.enemyBall != nil {
		vector.DrawFilledCircle(
			screen,
			float32(g.enemyBall.pos.X-g.camera.X),
			float32(g.enemyBall.pos.Y-g.camera.Y),
			float32(g.enemyBall.radius), wallColor, false)
	}

	vector.StrokeLine(screen,
		float32(g.enemyBall.pos.X-g.camera.X),
		float32(g.camera.Y-g.camera.Y+ScreenHeight-100),
		float32(g.enemyBall.pos.X-g.camera.X),
		float32(g.camera.Y-g.camera.Y+ScreenHeight),
		2, wallColor, false)

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
			float32(fractionsRadius), ballColor, false)
	}

	// ebitenutil.DebugPrintAt(screen, "isOnGround:"+strconv.Itoa(g.score), 10, 45)
	// Draw score
	text.Draw(screen, fmt.Sprintf("Score: %d$", g.score), g.titleFont, 10, 50, color.White)
	text.Draw(screen, fmt.Sprintf("Level score: %d$", g.getCurrentLevel().Score), g.buttonFont, 10, 100, color.White)
}

func drawGround(screen *ebiten.Image, ground []*Segment, camera *Camera) {
	for _, seg := range ground {
		if seg.savePoint != nil {
			vector.DrawFilledCircle(screen,
				float32(seg.savePoint.Position.X-camera.X),
				float32(seg.savePoint.Position.Y-camera.Y),
				float32(seg.savePoint.Radius),
				savePointColor, false)
			if seg.savePoint.IsFinish {
				vector.DrawFilledCircle(screen,
					float32(seg.savePoint.Position.X-camera.X),
					float32(seg.savePoint.Position.Y-camera.Y),
					float32(seg.savePoint.Radius*0.7),
					color.Black, false)
			}
		}

		color := groundColor
		if seg.isRed {
			color = wallColor
		}

		vector.StrokeLine(screen,
			float32(seg.A.X-camera.X),
			float32(seg.A.Y-camera.Y),
			float32(seg.B.X-camera.X),
			float32(seg.B.Y-camera.Y),
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
	menuBg.Fill(playBackground)

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
		color.White)

	// Create and draw buttons
	buttons := []Button{
		{
			X: ScreenWidth/2 - 200, Y: 200, Width: 400, Height: 60,
			Text: "PLAY", Color: ballColor,
			HoverColor: ballColorBig,
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
		color.White)

	// Draw score
	score := fmt.Sprintf("Score: %s$", strconv.Itoa(g.score))
	// boundScore := text.BoundString(g.titleFont, score)
	text.Draw(screen, score, g.buttonFont,
		200, 120,
		color.White)

	// Draw levels
	levelButtons := make([]Button, len(g.levels))
	sellLevel := make([]Button, len(g.levels))
	for i, level := range g.levels {
		// levelColor := groundColor
		// levelColorHover := groundColor
		// levelColorHover.R += 20
		// levelColorHover.G += 20
		// levelColorHover.B += 20

		levelButtons[i] = Button{
			X: 200, Y: 150 + float64(i)*80, Width: 200, Height: 60,
			Text:       level.Name,
			Color:      groundColor,
			HoverColor: groundColorHover,
			Action: func(lvlIdx int) func() {
				return func() {
					if !g.levels[lvlIdx].Finished {
						g.currentLevel = lvlIdx
						g.currentState = StateLoadingLevel
					}
				}
			}(i),
		}

		sellLevelBtnCol := groundColor
		sellLevelBtnColHover := groundColorHover
		if level.Finished {
			sellLevelBtnCol = ballColor
			sellLevelBtnColHover = ballColorBig
		}

		sellLevel[i] = Button{
			X: levelButtons[i].X + levelButtons[i].Width + 10, Y: 150 + float64(i)*80, Width: 150, Height: 60,
			Text:       fmt.Sprintf("%d$", level.Score),
			Color:      sellLevelBtnCol,
			HoverColor: sellLevelBtnColHover,
			Action: func(lvlIdx int) func() {
				return func() {
					err := resetLevel(g.levels[lvlIdx], g)
					g.drawError = err
				}
			}(i),
		}

		// Draw level button
		g.drawProgressButton(screen, &levelButtons[i], *level)
		if levelButtons[i].IsClicked() {
			levelButtons[i].Action()
		}

		// Draw sell level
		g.drawButton(screen, &sellLevel[i])
		if sellLevel[i].IsClicked() {
			sellLevel[i].Action()
		}
	}

	// Draw return button

	returnBtn := Button{
		X: 10, Y: ScreenHeight - 70, Width: 60, Height: 60,
		Text:       "return",
		Color:      groundColor,
		HoverColor: groundColorHover,
		Action: func() {
			g.currentState = StateMenu
		},
	}
	g.drawReturnButton(screen, &returnBtn)

	if returnBtn.IsClicked() {
		returnBtn.Action()
	}

	// g.drawReturnButton(screen, &returnBtn)
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

func (g *Game) drawProgressButton(screen *ebiten.Image, btn *Button, level Level) {
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
	// Draw progress
	progressWidth := btn.Width * float64(calculateLevelProgress(level)) / 100
	ebitenutil.DrawRect(screen, btn.X, btn.Y, progressWidth, btn.Height, ballColor)

	// Draw button text
	bounds := text.BoundString(g.buttonFont, btn.Text)
	textX := btn.X + (btn.Width-float64(bounds.Dx()))/2
	textY := btn.Y + (btn.Height)/2 + float64(bounds.Dy())/2
	text.Draw(screen, btn.Text, g.buttonFont, int(textX), int(textY), color.White)
}

func (g *Game) drawReturnButton(screen *ebiten.Image, btn *Button) {
	// Check hover state
	mx, my := ebiten.CursorPosition()
	hover := float64(mx) > btn.X && float64(mx) < btn.X+btn.Width &&
		float64(my) > btn.Y && float64(my) < btn.Y+btn.Height

	// Choose color
	btnColor := btn.Color
	var arrowColor color.RGBA = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if hover {
		btnColor = btn.HoverColor
		arrowColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}

	// Draw button
	ebitenutil.DrawRect(screen, btn.X, btn.Y, btn.Width, btn.Height, btnColor)

	tex := ebiten.NewImage(1, 1)
	for y := 0; y < 1; y++ {
		for x := 0; x < 1; x++ {
			tex.Set(x, y, arrowColor) // Light stripe
		}
	}

	arrowSize := float32(btn.Width / 3) // Size of the arrow
	arrowX := float32(btn.X + btn.Width/2 - (btn.Width / 6))
	arrowY := float32(btn.Y + btn.Height/2)

	// Create triangle vertices (left-pointing arrow)
	vertices := []ebiten.Vertex{
		// Tip of the arrow (left point)
		{
			DstX:   arrowX,
			DstY:   arrowY,
			ColorR: float32(arrowColor.R),
			ColorG: float32(arrowColor.G),
			ColorB: float32(arrowColor.B),
			ColorA: float32(arrowColor.A),
		},
		// Top right point
		{
			DstX:   arrowX + arrowSize,
			DstY:   arrowY - arrowSize/2,
			ColorR: float32(arrowColor.R),
			ColorG: float32(arrowColor.G),
			ColorB: float32(arrowColor.B),
			ColorA: float32(arrowColor.A),
		},
		// Bottom right point
		{
			DstX:   arrowX + arrowSize,
			DstY:   arrowY + arrowSize/2,
			ColorR: float32(arrowColor.R),
			ColorG: float32(arrowColor.G),
			ColorB: float32(arrowColor.B),
			ColorA: float32(arrowColor.A),
		},
	}

	// Triangle indices (order to draw vertices)
	indices := []uint16{0, 1, 2}

	// Draw the triangle
	screen.DrawTriangles(vertices, indices, tex, &ebiten.DrawTrianglesOptions{
		AntiAlias: true, // Smooth edges
	})

}

func (g *Game) getCurrentLevel() *Level {
	return g.levels[g.currentLevel]
}
