package game

import (
	"ball/assets"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

func drawButtonText(screen *ebiten.Image, btn *Button) {
	drawButton(screen, btn)
	drawText(screen, btn)
}

func drawButton(screen *ebiten.Image, btn *Button) {
	// Check hover state
	mx, my := ebiten.CursorPosition()
	hover := float64(mx) > btn.X && float64(mx) < btn.X+btn.Width &&
		float64(my) > btn.Y && float64(my) < btn.Y+btn.Height

	// Choose color
	btnColor := btn.Color
	if hover {
		btnColor = btn.HoverColor
	}

	// Draw button
	vector.DrawFilledRect(screen,
		float32(btn.X),
		float32(btn.Y),
		float32(btn.Width),
		float32(btn.Height),
		btnColor, false)
}

// drawText draw button text
func drawText(screen *ebiten.Image, btn *Button) {
	w, h := text.Measure(btn.Text, assets.ScoreFace, 0)
	options := &text.DrawOptions{}
	options.GeoM.Translate(btn.X+(btn.Width-w)/2, btn.Y+(btn.Height-h)/2)
	options.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, btn.Text, assets.ScoreFace, options)
}

// drawProgressButton draw button with level progress
func drawProgressButton(screen *ebiten.Image, btn *Button, level *Level) {
	drawButton(screen, btn)
	progressWidth := btn.Width * float64(calculateLevelProgress(*level)) / 100
	vector.DrawFilledRect(screen,
		float32(btn.X),
		float32(btn.Y),
		float32(progressWidth),
		float32(btn.Height),
		ballColor, false)

	drawText(screen, btn)
}
