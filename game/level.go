package game

import (
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
)

type Level struct {
	Name      string     `json:"name"`
	Ticker    string     `json:"ticker"`
	Number    int        `json:"number"`
	Finished  bool       `json:"finished"`
	SavePoint *SavePoint `json:"savePoint,omitempty"`
	ChartFile string     `json:"chartFile"`
	Score     int        `json:"score"`
	MaxX      float64    `json:"maxX"`
	// MaxY the graph crosses zero and becomes negative. Keep as negative value
	MaxY float64 `json:"maxY"`
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
