package game

import (
	"encoding/gob"
	"os"
)

type Level struct {
	Name   string
	Ticker string
	Number int
	Locked bool
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
