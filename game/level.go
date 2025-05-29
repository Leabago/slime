package game

import (
	"encoding/gob"
	"os"
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

func saveLevel(level Level) {

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
