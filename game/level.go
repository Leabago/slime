package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Level struct {
	Name      string `json:"name"`
	Ticker    string `json:"ticker"`
	ChartFile string `json:"chartFile"`
	Number    int    `json:"number"`
	// Finished     bool       `json:"finished"`
	Score *Score `json:"score,omitempty"`
	// SavePoint    *SavePoint `json:"savePoint,omitempty"`
	// MovingWall   *Segment   `json:"movingWall,omitempty"`
	// EnemyBallPos *Vector    `json:"enemyBallPos,omitempty"`
	MaxX float64 `json:"maxX,omitempty"`
	MaxY float64 `json:"maxY,omitempty"` // MaxY the graph crosses zero and becomes negative. Keep as negative value

	CurrentDifficulty int
	LevelEntities     map[int]*LevelEntities `json:"levelEntities,omitempty"`
}

type LevelEntities struct {
	Finished     bool       `json:"finished"`
	SavePoint    *SavePoint `json:"savePoint,omitempty"`
	MovingWall   *Segment   `json:"movingWall,omitempty"`
	EnemyBallPos *Vector    `json:"enemyBallPos,omitempty"`
}

func NewLevelEntities() map[int]*LevelEntities {
	return map[int]*LevelEntities{
		Easy:      &LevelEntities{},
		Medium:    &LevelEntities{},
		Difficult: &LevelEntities{},
	}
}

// setDifficulty set current difficulty
func (l *Level) setDifficulty(difficulty int) {
	l.Score.setDifficulty(difficulty)
	l.CurrentDifficulty = difficulty
}

func (l *Level) getSavePoint() *SavePoint {
	return l.LevelEntities[l.CurrentDifficulty].SavePoint
}
func (l *Level) setSavePoint(savePoint *SavePoint) {
	l.LevelEntities[l.CurrentDifficulty].SavePoint = savePoint
}

func (l *Level) setMovingWall(movingWall *Segment) {
	l.LevelEntities[l.CurrentDifficulty].MovingWall = movingWall
}

func (l *Level) getMovingWall() *Segment {
	return l.LevelEntities[l.CurrentDifficulty].MovingWall
}

func (l *Level) setEnemyBallPos(enemyBallPos *Vector) {
	l.LevelEntities[l.CurrentDifficulty].EnemyBallPos = enemyBallPos
}

func (l *Level) getEnemyBallPos() *Vector {
	return l.LevelEntities[l.CurrentDifficulty].EnemyBallPos
}
func (l *Level) getFinished() bool {
	return l.LevelEntities[l.CurrentDifficulty].Finished
}

func (l *Level) setFinished(finished bool) {
	l.LevelEntities[l.CurrentDifficulty].Finished = finished
}

func (l *Level) resetLevel() {
	l.Score.setScore(defaultScore)
	l.LevelEntities[l.CurrentDifficulty] = &LevelEntities{}
}

// saveLevel marshals level to json and save it in file
func saveLevel(level *Level) error {
	// save level to file
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
