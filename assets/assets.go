package assets

import (
	"embed"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed *
var assets embed.FS

var ScoreFace = mustLoadFace("Fonts/Kenney Mini.ttf", 32)
var ScoreFaceBig = mustLoadFace("Fonts/Kenney Mini.ttf", 42)
var ScoreFont = mustLoadFont("Fonts/Kenney Mini.ttf", 32)

func mustLoadFace(name string, size float64) text.Face {
	return text.NewGoXFace(mustLoadFont(name, size))
}

func mustLoadFont(name string, size float64) font.Face {
	fontdata, err := assets.ReadFile(name)
	if err != nil {
		panic(err)
	}

	sfntFont, err := opentype.Parse(fontdata)
	if err != nil {
		panic(err)
	}
	opentypeFace, err := opentype.NewFace(sfntFont, &opentype.FaceOptions{
		Size: size,
		DPI:  72,
	})
	if err != nil {
		panic(err)
	}

	return opentypeFace
}
