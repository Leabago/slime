package game

import (
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
)

type Level struct {
	Name       string     `json:"name"`
	Ticker     string     `json:"ticker"`
	ChartFile  string     `json:"chartFile"`
	Number     int        `json:"number"`
	Finished   bool       `json:"finished"`
	Score      int        `json:"score"`
	SavePoint  *SavePoint `json:"savePoint,omitempty"`
	MovingWall *Segment   `json:"movingWall,omitempty"`
	MaxX       float64    `json:"maxX,omitempty"`
	MaxY       float64    `json:"maxY,omitempty"` // MaxY the graph crosses zero and becomes negative. Keep as negative value
}

func saveBinary(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(data)
}

func loadBinary(data interface{}, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	return decoder.Decode(data)
}

// saveLevel marshals level to json and save it in file
func saveLevel(level Level) error {
	// save level to file
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
