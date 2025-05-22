package main

import (
	"ball/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {

	g := game.NewGame()

	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowTitle("Slime")
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}
