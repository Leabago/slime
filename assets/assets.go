package assets

import (
	"embed"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed *
var assets embed.FS

var ScoreFace = mustLoadFace("Fonts/Kenney Mini.ttf", 32)
var ScoreFaceBig = mustLoadFace("Fonts/Kenney Mini.ttf", 42)
var ScoreFont = mustLoadFont("Fonts/Kenney Mini.ttf", 32)

var (
	audioContext *audio.Context
	bgmPlayer    *audio.Player
)

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

func CreatePlayer() *audio.Player {
	// Load background music (OGG/Vorbis format recommended)
	file, err := os.Open("assets/Music/menu.ogg")
	if err != nil {
		log.Fatal(err)
	}

	// Decode the audio file
	decoded, err := vorbis.DecodeWithSampleRate(44100, file)
	if err != nil {
		log.Fatal(err)
	}

	// Create audio player
	bgmPlayer, err = audioContext.NewPlayer(decoded)
	if err != nil {
		log.Fatal(err)
	}

	return bgmPlayer
}
