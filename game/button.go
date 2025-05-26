package game

import "image/color"

type Button struct {
	X, Y          float64
	Width, Height float64
	Text          string
	Action        func()
	Color         color.RGBA
	HoverColor    color.RGBA
}
