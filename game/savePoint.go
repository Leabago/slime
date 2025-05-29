package game

// SavePoint is the place where the game automatically saves the user
type SavePoint struct {
	Position Vector  `json:"position"`
	Radius   float64 `json:"radius"`
	IsFinish bool    `json:"isFinish"`
}
