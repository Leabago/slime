package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Button struct {
	X, Y          float64
	Width, Height float64
	Text          string
	Action        func()
	Color         color.RGBA
	HoverColor    color.RGBA
}

func (b *Button) IsClicked() bool {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		return float64(mx) > b.X && float64(mx) < b.X+b.Width &&
			float64(my) > b.Y && float64(my) < b.Y+b.Height
	}
	return false
}
