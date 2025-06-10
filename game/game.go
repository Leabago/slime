package game

import (
	"ball/assets"
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

	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

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

	// GameFilesDir - files related with game and levels
	GameFilesDir = "gameFiles"
	// scoreFileName - file with user score
	scoreFileName = "score"
	// defaultScore - score at first start
	defaultScore = 0
	// max wall height
	wallHeight = 2000.0
)

// EASY
const (
	groundBuffSize_EASY     = 80
	savePointSpawn_EASY     = 7
	savePointScore_EASY     = 300
	savePointWidthMove_EASY = 20.0
	redSegmentSpawn_EASY    = 60
	movWallSpeedHight_EASY  = 15.0
	movWallSpeedSlow_EASY   = 2.0
	enemyBallSlow_EASY      = 0.5
)

// MEDIUM
const (
	groundBuffSize_MEDIUM     = 50
	savePointSpawn_MEDIUM     = 11
	savePointScore_MEDIUM     = 200
	savePointWidthMove_MEDIUM = 60.0
	redSegmentSpawn_MEDIUM    = 50
	movWallSpeedHight_MEDIUM  = 16.0
	movWallSpeedSlow_MEDIUM   = 4.0
	enemyBallSlow_MEDIUM      = 0.7
)

// DIFFICULT
const (
	groundBuffSize_DIFFICULT     = 35
	savePointSpawn_DIFFICULT     = 15
	savePointScore_DIFFICULT     = 300
	savePointWidthMove_DIFFICULT = 120.0
	redSegmentSpawn_DIFFICULT    = 30
	movWallSpeedHight_DIFFICULT  = 17.0
	movWallSpeedSlow_DIFFICULT   = 5.0
	enemyBallSlow_DIFFICULT      = 0.8
)

var (
	// groundBuffSize - buffer consist of two slices of ground, groundBuffSize is size of one slice
	groundBuffSize = groundBuffSize_EASY
	// savePointSpawn - how often save points spawns
	savePointSpawn = savePointSpawn_EASY
	// savePointScore - add points after collision with save point
	savePointScore = savePointScore_EASY
	// savePointWidthMove amplitude of upward movement
	savePointWidthMove = savePointWidthMove_EASY
	// redSegmentSpawn how often red segment spawns
	redSegmentSpawn = redSegmentSpawn_EASY
	// movWallSpeedHight speed Hight
	movWallSpeedHight = movWallSpeedHight_EASY
	// movWallSpeedSlow speed Slow
	movWallSpeedSlow = movWallSpeedSlow_EASY
	// enemyBallSlow measure of slowing down, the smaller the slower
	enemyBallSlow = enemyBallSlow_EASY
)

// Draw variables
var (
	playBackground           = color.RGBA{0, 0, 0, 255}
	wallColor                = color.RGBA{200, 10, 60, 255}
	wallColorHover           = color.RGBA{220, 30, 90, 255}
	savePointColor           = color.RGBA{130, 255, 130, 255}
	groundColor              = color.RGBA{10, 60, 60, 255}
	groundColorHover         = color.RGBA{30, 90, 90, 255}
	ballColor                = color.RGBA{70, 150, 70, 255}
	ballColorBig             = color.RGBA{90, 180, 90, 200}
	yellowColor              = color.RGBA{200, 100, 0, 255}
	yellowColorHover         = color.RGBA{220, 120, 20, 255}
	segmentWidth     float32 = 5
	fractionsRadius          = 10
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
	enemyBall    *Ball
	collisionSeg []Segment
	camera       *Camera
	score        *Score
	fractions    []Vector
	frameTimer   *Timer

	// Game data
	levels       []*Level
	currentLevel int

	// Menu
	menuFont     font.Face
	titleFont    font.Face
	buttonFont   font.Face
	currentState int
	menuBg       *ebiten.Image

	// wall
	borderSquare *BorderSquare
	movingWall   *Segment

	drawError error

	difficulty int
}

func (g *Game) Update() error {
	//  return error from Draw
	if g.drawError != nil {
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
			g.getCurrentLevel().setMovingWall(g.movingWall)
			g.getCurrentLevel().setEnemyBallPos(&g.enemyBall.pos)

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

	// update player
	g.ball.Update(groundFromBuff, g)
	// update enemy
	g.updateEnemy()
	// check collisions and move objects
	g.CheckCollisions(&g.collisionSeg, groundFromBuff)

	// return if player is died
	if g.ball.isDied {
		g.getCurrentLevel().resetLevel()

		err := saveLevel(g.getCurrentLevel())
		if err != nil {
			return err
		}

		return returnToSelectLevel(g)

	}
	// return if player is finished
	if g.getCurrentLevel().getFinished() {
		return returnToSelectLevel(g)
	}
	// update Ground Buffer if player reached middle
	g.updateGroundBuffer(middleSegment, lenBuff, lastXbuff)

	// Update camera
	g.camera.Update(g.ball.pos.X, g.ball.pos.Y)

	return nil
}

// updateGroundBuffer update Ground Buffer if player reached middle
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

	levelJson, err := json.Marshal(level)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(GameFilesDir, getJsonName(level.Ticker)), levelJson, 0644)
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
	savePoint := g.getCurrentLevel().getSavePoint()

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
			IsMovingWall: true,
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

		if g.getCurrentLevel().getMovingWall() != nil {
			g.movingWall = g.getCurrentLevel().getMovingWall()
		}
		if g.movingWall.A.X > savePoint.Position.X || g.movingWall.B.X > savePoint.Position.X || g.movingWall.B.Y > maxY-wallHeight {
			g.movingWall = &Segment{
				A:            Vector{g.groundBuff[0][0].A.X, 0},
				B:            Vector{g.groundBuff[0][0].A.X, maxY - wallHeight},
				IsMovingWall: true,
			}
		}

		// set enemy position

	}

	// set ball and enemy
	g.ball = NewBall(savePoint.Position)
	g.currentState = StatePlaying

	g.enemyBall = NewEnemyBall()
	// set position if exist

	if g.getCurrentLevel().getEnemyBallPos() != nil {
		g.enemyBall.pos = *g.getCurrentLevel().getEnemyBallPos()
	}

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
			segmentWidth, yellowColor, false)
		vector.StrokeLine(screen,
			float32(g.borderSquare.drawRight.A.X-g.camera.X),
			float32(g.borderSquare.drawRight.A.Y-g.camera.Y),
			float32(g.borderSquare.drawRight.B.X-g.camera.X),
			float32(g.borderSquare.drawRight.B.Y-g.camera.Y),
			segmentWidth, yellowColor, false)
		vector.StrokeLine(screen,
			float32(g.borderSquare.bottom.A.X-g.camera.X),
			float32(g.borderSquare.bottom.A.Y-g.camera.Y),
			float32(g.borderSquare.bottom.B.X-g.camera.X),
			float32(g.borderSquare.bottom.B.Y-g.camera.Y),
			segmentWidth, yellowColor, false)
		vector.StrokeLine(screen,
			float32(g.borderSquare.drawLeft.A.X-g.camera.X),
			float32(g.borderSquare.drawLeft.A.Y-g.camera.Y),
			float32(g.borderSquare.drawLeft.B.X-g.camera.X),
			float32(g.borderSquare.drawLeft.B.Y-g.camera.Y),
			segmentWidth, yellowColor, false)
	}

	// Draw moving Wall
	vector.StrokeLine(screen,
		float32(g.movingWall.A.X-g.camera.X),
		float32(g.movingWall.A.Y-g.camera.Y),
		float32(g.movingWall.B.X-g.camera.X),
		float32(g.movingWall.B.Y-g.camera.Y),
		segmentWidth, wallColor, false)

	// Draw ground
	for _, seg := range g.ground {
		vector.StrokeLine(screen,
			float32(seg.A.X-g.camera.X),
			float32(seg.A.Y-g.camera.Y),
			float32(seg.B.X-g.camera.X),
			float32(seg.B.Y-g.camera.Y),
			1, groundColor, false)
	}

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

	// Draw score
	options := &text.DrawOptions{}
	options.GeoM.Translate(10, 10)
	options.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, fmt.Sprintf("Level score: %d$", g.getCurrentLevel().Score.getScore()), assets.ScoreFace, options)

	// Draw return button
	g.drawReturnButton(screen, StateLevelSelect)
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
			color = yellowColor
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

	// load score from file or use default score
	score, err := LoadScore()
	if err != nil {
		return nil, err
	}

	// set variables depending on the Difficulty
	setDifficultyVars(score.CurrentDifficulty)

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

			if level.Score == nil {
				level.Score = newScore()
			}

			if len(level.LevelEntities) == 0 {
				level.LevelEntities = NewLevelEntities()
			}

			level.setDifficulty(score.CurrentDifficulty)

			levels = append(levels, level)
		}
	}

	game := &Game{
		frameTimer: NewTimer(80 * time.Millisecond),
		score:      score,

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
	}

	return game, nil
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	screen.DrawImage(g.menuBg, nil)

	// Draw title
	title := "STOCK JUMPER"
	w, h := text.Measure(title, assets.ScoreFaceBig, 0)
	options := &text.DrawOptions{}
	options.GeoM.Translate((ScreenWidth)/2-w/2, 130-h/2)
	options.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, title, assets.ScoreFaceBig, options)

	// Create and draw buttons
	buttons := []Button{
		{
			X: ScreenWidth/2 - 200, Y: 200, Width: 200, Height: 60,
			Text:       "PLAY",
			Color:      ballColor,
			HoverColor: ballColorBig,
			Action:     func() { g.currentState = StateLevelSelect },
		},
		{
			X: ScreenWidth / 2, Y: 200, Width: 200, Height: 60,
			Text:       getDifficultName(g.score.CurrentDifficulty),
			Color:      getDifficultColor(g.score.CurrentDifficulty),
			HoverColor: getDifficultColorHower(g.score.CurrentDifficulty),
			Action: func() {
				g.drawError = g.changeDifficulty()
			},
		},
		{
			X: ScreenWidth/2 - 200, Y: 200, Width: 200, Height: 60,
			Text:       "PLAY",
			Color:      ballColor,
			HoverColor: ballColorBig,
			Action:     func() { g.currentState = StateLevelSelect },
		},
		{
			X: ScreenWidth/2 - 100, Y: 280, Width: 200, Height: 60,
			Text:       "QUIT",
			Color:      wallColor,
			HoverColor: wallColorHover,
			Action:     func() { g.currentState = StateTermination },
		},
	}

	for i, btn := range buttons {
		drawButtonText(screen, &buttons[i])
		if btn.IsClicked() {
			btn.Action()
		}
	}

}

func (g *Game) drawLevelSelect(screen *ebiten.Image) {
	screen.DrawImage(g.menuBg, nil)
	// screen.Fill(color.RGBA{R: 20, G: 20, B: 40, A: 255})

	// Draw title

	title := fmt.Sprintf("SELECT LEVEL curr diff: %d$", g.score.CurrentDifficulty)
	// title := "SELECT LEVEL"
	w, h := text.Measure(title, assets.ScoreFaceBig, 0)
	options := &text.DrawOptions{}
	options.GeoM.Translate(ScreenWidth/2-w/2, 50-h/2)
	options.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, title, assets.ScoreFaceBig, options)

	options2 := &text.DrawOptions{}
	options2.GeoM.Translate(200, 70)
	options2.ColorScale.ScaleWithColor(color.White)

	// Draw score
	score := fmt.Sprintf("Score: %s$", strconv.Itoa(g.score.getScore()))
	text.Draw(screen, score, assets.ScoreFace, options2)

	// Draw levels
	levelButtons := make([]Button, len(g.levels))
	sellLevel := make([]Button, len(g.levels))
	for i, level := range g.levels {
		levelButtons[i] = Button{
			X: 200, Y: 150 + float64(i)*80, Width: 400, Height: 60,
			Text:       level.Name,
			Color:      groundColor,
			HoverColor: groundColorHover,
			Action: func(lvlIdx int) func() {
				return func() {
					if !g.levels[lvlIdx].getFinished() {
						g.currentLevel = lvlIdx
						g.currentState = StateLoadingLevel
					}
				}
			}(i),
		}

		sellLevelBtnCol := groundColor
		sellLevelBtnColHover := groundColorHover
		if level.getFinished() {
			sellLevelBtnCol = ballColor
			sellLevelBtnColHover = ballColorBig
		}

		sellLevel[i] = Button{
			X: levelButtons[i].X + levelButtons[i].Width + 10, Y: 150 + float64(i)*80, Width: 450, Height: 60,
			Text:       fmt.Sprintf("%d$", level.Score.getScore()),
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
		drawProgressButton(screen, &levelButtons[i], level)
		if levelButtons[i].IsClicked() {
			levelButtons[i].Action()
		}

		// Draw sell level
		drawButtonText(screen, &sellLevel[i])
		if sellLevel[i].IsClicked() {
			sellLevel[i].Action()
		}
	}

	// Draw return button
	g.drawReturnButton(screen, StateMenu)
}

func (g *Game) drawReturnButton(screen *ebiten.Image, returnState int) {
	btn := Button{
		X: 10, Y: ScreenHeight - 70, Width: 60, Height: 60,
		Color:      groundColor,
		HoverColor: groundColorHover,
		Action: func() {
			g.currentState = returnState
		},
	}

	// Check hover state
	mx, my := ebiten.CursorPosition()
	hover := float64(mx) > btn.X && float64(mx) < btn.X+btn.Width &&
		float64(my) > btn.Y && float64(my) < btn.Y+btn.Height

	// Choose color
	btnColor := btn.Color
	arrowColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if hover {
		btnColor = btn.HoverColor
		arrowColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}

	// Draw button
	vector.DrawFilledRect(screen,
		float32(btn.X),
		float32(btn.Y),
		float32(btn.Width),
		float32(btn.Height),
		btnColor, false)

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

	// action
	if btn.IsClicked() {
		btn.Action()
	}
}

func (g *Game) getCurrentLevel() *Level {
	return g.levels[g.currentLevel]
}

func (g *Game) updateEnemy() {
	g.enemyBall.vel.X *= enemyBallSlow
	g.enemyBall.vel.Y *= enemyBallSlow
}

// CheckCollisions check collisions and move objects
func (game *Game) CheckCollisions(gameCollSeg *[]Segment, ground []*Segment) {
	// average normal
	avgNormal := Vector{0, 0}
	collisionSeg := []Segment{}
	var penetrationSum float64
	wallThickness := 3.0 // to avoid falling into a segment
	minVec := math.MaxFloat64
	velEnemy := Vector{}

	if !isCircleRectangleColl(game.ball.pos, game.ball.radius, *game.borderSquare) {
		game.ball.vel = Vector{}
		if game.getCurrentLevel().getSavePoint() != nil {
			if isCircleRectangleColl(game.getCurrentLevel().getSavePoint().Position, game.ball.radius, *game.borderSquare) {
				game.ball.pos = game.getCurrentLevel().getSavePoint().Position
			} else {
				game.ball.pos = getStartPositionPtr(ground)
			}
		} else {
			game.ball.pos = getStartPositionPtr(ground)
		}
	}

	// check collision ball with emeny
	if circleToCircle(game.ball.pos, game.ball.radius, game.enemyBall.pos, game.enemyBall.radius) {
		game.ball.isDied = true
	}

	for _, seg := range ground {
		// current position
		closest := closestPointOnSegment(seg.A, seg.B, game.ball.pos)
		distVec := game.ball.pos.Sub(closest)
		dist := distVec.Len()

		// // current position enemy
		closestEnemy := closestPointOnSegment(seg.A, seg.B, game.enemyBall.pos)
		distVecEnemy := game.enemyBall.pos.Sub(closestEnemy)
		distEnemy := distVecEnemy.Len()

		// check collision enemy with ground
		if !seg.IsMovingWall && !seg.isBorder {
			if distEnemy < minVec {
				minVec = distEnemy

				vec := seg.A.Sub(seg.B).Normalize()
				vec = vec.Add(seg.A.Sub(game.enemyBall.pos).Normalize())
				velEnemy = vec
			}
		}

		// true - collision ball with segment
		if dist < game.ball.radius+wallThickness {

			// Push the wheel out of the ground
			normal := distVec.Normalize()

			// params
			seg.closestPoint = closest
			seg.normal = normal
			collisionSeg = append(collisionSeg, *seg)
			penetration := game.ball.radius + wallThickness - dist
			penetrationSum += penetration

			// if segment is red then minus score
			if seg.isRed && game.getCurrentLevel().Score.getScore() > 0 {
				game.getCurrentLevel().Score.setScore(game.getCurrentLevel().Score.getScore() - 1)
			}

			// die if collision with moving wall
			if seg.IsMovingWall {
				game.ball.isDied = true
			}
		}

		// check collision with save point
		if seg.savePoint != nil {
			if circleToCircle(game.ball.pos, game.ball.radius, seg.savePoint.Position, seg.savePoint.Radius) {
				game.getCurrentLevel().setSavePoint(seg.savePoint)
				// b.onGround = true
				seg.savePoint = nil

				game.getCurrentLevel().Score.plusScore(savePointScore)

				// collision with finish
				if game.getCurrentLevel().getSavePoint().IsFinish {
					game.getCurrentLevel().Score.plusScore(savePointScore * 5)
					game.getCurrentLevel().setFinished(true)
				}
			}
		}
	}

	// add velocity to enemy
	if minVec != math.MaxFloat64 {
		game.enemyBall.vel = game.enemyBall.vel.Add(velEnemy)
	}

	// respawn enemy
	closestEnemy := closestPointOnSegment(game.borderSquare.left.A, game.borderSquare.left.B, game.enemyBall.pos)
	distVecEnemy := game.enemyBall.pos.Sub(closestEnemy)
	distEnemy := distVecEnemy.Len()

	if distEnemy < game.enemyBall.radius || !isCircleRectangleColl(game.enemyBall.pos, game.enemyBall.radius, *game.borderSquare) {
		game.enemyBall.pos = game.borderSquare.drawRight.B
	}

	// add velocity to ball
	if len(collisionSeg) > 0 {
		for _, n := range collisionSeg {
			avgNormal = avgNormal.Add(n.normal)
		}
		avgNormal = avgNormal.Normalize()

		// Apply averaged correction
		avgPenetration := penetrationSum / float64(len(collisionSeg))
		game.ball.pos = game.ball.pos.Add(avgNormal.Mul(avgPenetration))

		// Handle velocity response
		velDot := game.ball.vel.Dot(avgNormal)
		if velDot < 0 {

			// friction
			game.ball.vel = game.ball.vel.Sub(avgNormal.Mul(velDot)).Mul(game.ball.currPhyState.friction)

			// Reflect velocity along the collision normal, friction
			// reflected := b.vel.Sub(avgNormal.Mul(velDot))
			// b.vel = reflected.Mul(b.currPhyState.bounceFactor)
		}

		// to avoid falling between two segments
		if avgPenetration > 2 {
			game.ball.vel = game.ball.vel.Add(Vector{1, -1})
		}

		game.ball.onGround = true
		game.ball.doubleJump = 0
	}

	// get average angle
	angle := SlopeAngleFromNormal(avgNormal)

	// if state "A" then the ball cannot climb a high slope
	if game.ball.currPhyState.state == phyStateA {
		if angle > 70 {
			game.ball.jumpVel = avgNormal.Add(game.ball.currPhyState.scrambleWall)
		} else {
			game.ball.jumpVel = game.ball.currPhyState.jump
		}
	}
	// if state "B" then the ball can slide a slope
	if game.ball.currPhyState.state == phyStateB {
		game.ball.jumpVel = game.ball.currPhyState.jump
	}
	game.ball.jumpVel = game.ball.jumpVel.Mul(game.ball.currPhyState.jumpForce)
	*gameCollSeg = collisionSeg
}

// changeDifficulty change difficulty for score and all levels
func (g *Game) changeDifficulty() error {
	difficulty, err := g.score.changeDifficulty()
	if err != nil {
		return err
	}

	for _, l := range g.levels {
		l.setDifficulty(difficulty)

		err := saveLevel(l)
		if err != nil {
			return err
		}
	}

	setDifficultyVars(difficulty)

	return nil
}

// setDifficultyVars set variables depending on the Difficulty
func setDifficultyVars(difficulty int) {
	switch difficulty {
	case Easy:
		groundBuffSize = groundBuffSize_EASY
		savePointSpawn = savePointSpawn_EASY
		savePointScore = savePointScore_EASY
		savePointWidthMove = savePointWidthMove_EASY
		redSegmentSpawn = redSegmentSpawn_EASY
		movWallSpeedHight = movWallSpeedHight_EASY
		movWallSpeedSlow = movWallSpeedSlow_EASY
	case Medium:
		groundBuffSize = groundBuffSize_MEDIUM
		savePointSpawn = savePointSpawn_MEDIUM
		savePointScore = savePointScore_MEDIUM
		savePointWidthMove = savePointWidthMove_MEDIUM
		redSegmentSpawn = redSegmentSpawn_MEDIUM
		movWallSpeedHight = movWallSpeedHight_MEDIUM
		movWallSpeedSlow = movWallSpeedSlow_MEDIUM
	case Difficult:
		groundBuffSize = groundBuffSize_DIFFICULT
		savePointSpawn = savePointSpawn_DIFFICULT
		savePointScore = savePointScore_DIFFICULT
		savePointWidthMove = savePointWidthMove_DIFFICULT
		redSegmentSpawn = redSegmentSpawn_DIFFICULT
		movWallSpeedHight = movWallSpeedHight_DIFFICULT
		movWallSpeedSlow = movWallSpeedSlow_DIFFICULT
	}
}
