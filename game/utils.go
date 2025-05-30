package game

import (
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

// getJsonName adds .json at the end
func getJsonName(fileName string) string {
	return fileName + ".json"
}

// calculateLevelProgress calculate percentage progress level
func calculateLevelProgress(level Level) int {
	if level.Finished {
		return 100
	} else if level.SavePoint == nil {
		return 0
	} else {
		if level.MaxX == 0 {
			return 0
		} else {
			return int(level.SavePoint.Position.X * 100 / level.MaxX)
		}
	}
}

// makeSegments copy array of Segments
func makeSegments(segments []Segment) []*Segment {
	pointers := make([]*Segment, len(segments))
	for i := range segments {
		pointers[i] = &segments[i]
	}
	return pointers
}

// circleToCircle check circle to circle collision
func circleToCircle(posA Vector, rA float64, posB Vector, rB float64) bool {
	distX := posB.X - posA.X
	distY := posB.Y - posA.Y
	distance := math.Sqrt((distX * distX) + (distY * distY))

	if distance <= rA+rB {
		return true
	}
	return false
}

// isCircleRectangleColl check what ball inside BorderSquare
func isCircleRectangleColl(circle Vector, radius float64, borderSquare BorderSquare) bool {
	testX := circle.X
	testY := (circle.Y)

	if circle.X < borderSquare.leftX() { // left
		testX = borderSquare.leftX()
	} else if circle.X > borderSquare.rightX() { // right
		testX = borderSquare.rightX()
	}

	if math.Abs(circle.Y) > math.Abs(borderSquare.topY()) { //top
		testY = math.Abs(borderSquare.topY())
	} else if math.Abs(circle.Y) < math.Abs(borderSquare.bottomY()) { // bottom
		testY = math.Abs(borderSquare.bottomY())
	}

	distX := circle.X - testX
	distY := math.Abs(circle.Y) - math.Abs(testY)
	distance := math.Sqrt(distX*distX + distY*distY)

	if distance < radius {
		return true
	}

	return false
}

// findMinMaxY return minY and maxY
func findMinMaxY(segments []*Segment) (float64, float64) {
	minY := segments[0].a.Y
	maxY := segments[0].a.Y

	for _, s := range segments {
		if s.MinY() < minY {
			minY = s.MinY()
		}

		if s.MaxY() > maxY {
			maxY = s.MaxY()
		}
	}

	return minY, maxY
}

// LoadScore loads the score from file or initializes with default value
func LoadScore() (*int, error) {
	// load score from file or use default score
	score := defaultScore

	scoreFilePath := filepath.Join(GameFilesDir, scoreFileName)
	err := loadBinary(&score, scoreFilePath)

	switch {
	case err == nil:
		// Successfully loaded existing score
		return &score, nil
	case errors.Is(err, os.ErrNotExist):
		// File doesn't exist - create with default
		err = saveBinary(score, scoreFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize score file: %w", err)
		}
		return &score, nil

	default:
		// Other errors (permission, corruption, etc.)
		return nil, fmt.Errorf("failed to load score: %w", err)
	}
}

func readLevelCSV(filename string) ([]Vector, error) {
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

// getStartPositionPtr calculate start position
func getStartPositionPtr(segments []*Segment) Vector {
	minY := segments[0].MinY() - ballPhysicA.radius
	avrX := segments[0].AvrX()
	return Vector{avrX, minY}
}
func getStartPosition(segments []Segment) Vector {
	minY := segments[0].MinY() - ballPhysicA.radius
	avrX := segments[0].AvrX()
	return Vector{avrX, minY}
}

// resetLevel set level score to 0 and clean savePoint
func resetLevel(level *Level, game *Game) error {
	if !level.Finished {
		game.score += level.Score
	} else {
		game.score += level.Score * 2
	}

	level.Score = 0
	level.Finished = false
	level.SavePoint = nil

	// save in files
	err := saveLevel(*level)
	if err != nil {
		return err
	}
	err = saveBinary(game.score, filepath.Join(GameFilesDir, scoreFileName))
	if err != nil {
		return err
	}

	return nil
}
