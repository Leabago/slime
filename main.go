package main

import (
	"ball/game"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

func createDirIfNotExist(dirPath string) error {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Create the directory with 0755 permissions (rwxr-xr-x)
		err := os.Mkdir(dirPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

func main() {
	err := createDirIfNotExist(game.GameFilesDir)
	if err != nil {
		panic(err)
	}

	g, err := game.NewGame()
	if err != nil {
		panic(err)
	}

	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("Slime")

	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}
