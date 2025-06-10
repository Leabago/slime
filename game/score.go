package game

import (
	"image/color"
	"path/filepath"
)

// difficulty
const (
	Easy = iota
	Medium
	Difficult
)

type Score struct {
	CurrentDifficulty int         `json:"currentDifficulty"`
	Difficulty        map[int]int `json:"difficulty"`
}

func newScore() *Score {
	return &Score{
		Difficulty:        map[int]int{Easy: defaultScore, Medium: defaultScore, Difficult: defaultScore},
		CurrentDifficulty: Easy,
	}
}

func (s *Score) getScore() int {
	return s.Difficulty[s.CurrentDifficulty]
}

func (s *Score) setScore(score int) {
	s.Difficulty[s.CurrentDifficulty] = score
}

func (s *Score) plusScore(plus int) {
	s.Difficulty[s.CurrentDifficulty] = s.Difficulty[s.CurrentDifficulty] + plus
}

func (s *Score) minusScore(minus int) {
	s.Difficulty[s.CurrentDifficulty] = s.Difficulty[s.CurrentDifficulty] - minus
	if s.Difficulty[s.CurrentDifficulty] < 0 {
		s.Difficulty[s.CurrentDifficulty] = 0
	}
}

func (s *Score) setDifficulty(difficulty int) {
	s.CurrentDifficulty = difficulty
}

func getDifficultName(difficulty int) string {
	switch difficulty {
	case Easy:
		return "EASY"
	case Medium:
		return "MEDIUM"
	case Difficult:
		return "DIFFICULT"
	default:
		return "UNKNOWN"
	}
}

func getDifficultColor(difficulty int) color.RGBA {
	switch difficulty {
	case Easy:
		return ballColor
	case Medium:
		return yellowColor
	case Difficult:
		return wallColor
	default:
		return playBackground
	}
}

func getDifficultColorHower(difficulty int) color.RGBA {
	switch difficulty {
	case Easy:
		return ballColorBig
	case Medium:
		return yellowColorHover
	case Difficult:
		return wallColorHover
	default:
		return playBackground
	}
}

func (s *Score) changeDifficulty() (int, error) {
	switch s.CurrentDifficulty {
	case Easy:
		s.CurrentDifficulty = Medium
	case Medium:
		s.CurrentDifficulty = Difficult
	case Difficult:
		s.CurrentDifficulty = Easy
	default:
		s.CurrentDifficulty = Easy
	}

	scoreFilePath := filepath.Join(GameFilesDir, scoreFileName)
	return s.CurrentDifficulty, saveBinary(s, scoreFilePath)
}
