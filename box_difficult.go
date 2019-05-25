package main

import (
	"strconv"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type DifficultStr struct {
	name               string
	row, column, mines int
}

type BoxDifficult struct {
	texBg         *sdl.Texture
	rect          sdl.Rect
	relX, relY    int32
	fg, bg        sdl.Color
	sprites       []Sprite
	difficltLines []DifficultStr
	show          bool
}

func NewBoxDifficult(rect sdl.Rect, fg, bg sdl.Color, renderer *sdl.Renderer, font *ttf.Font, fn func()) *BoxDifficult {
	texBg := NewBoxDifficultTexture(rect, fg, bg, renderer, font)
	diffLines := []DifficultStr{
		{name: "Begginer", row: 9, column: 9, mines: 10},
		{name: "Intermediate", row: 16, column: 16, mines: 40},
		{name: "Expert", row: 30, column: 16, mines: 99},
		{name: "Custom", row: 5, column: 5, mines: 5},
	}
	var sprites []Sprite
	titleHeight := rect.H / 5
	for i, line := range diffLines {
		line := line
		btn := NewButton(renderer, line.name, sdl.Rect{rect.X, 1 + rect.Y + (titleHeight * (int32(i) + 1)), rect.W - int32(float64(titleHeight)*2)*3, titleHeight + 1}, fg, bg, font, fn)
		w := int32(float64(titleHeight) * 0.8)
		w1 := int32(float64(titleHeight) * 2)
		lblRow := NewLabel(strconv.Itoa(line.row), sdl.Point{rect.X + (rect.W - w1) - w1*2 + w, rect.Y + titleHeight*int32(i+1)}, fg, renderer, font)
		lblColumn := NewLabel(strconv.Itoa(line.column), sdl.Point{rect.X + (rect.W - w1) - w1 + w, rect.Y + titleHeight*int32(i+1)}, fg, renderer, font)
		lblMines := NewLabel(strconv.Itoa(line.mines), sdl.Point{rect.X + (rect.W - w1) + w, rect.Y + titleHeight*int32(i+1)}, fg, renderer, font)
		sprites = append(sprites, btn, lblRow, lblColumn, lblMines)
	}
	return &BoxDifficult{
		rect:          rect,
		fg:            fg,
		bg:            bg,
		texBg:         texBg,
		difficltLines: diffLines,
		sprites:       sprites,
		show:          true,
	}
}

func NewBoxDifficultTexture(rect sdl.Rect, fg, bg sdl.Color, renderer *sdl.Renderer, font *ttf.Font) *sdl.Texture {
	titleHeight := rect.H / 5
	lblTitle := NewLabel("Select Difficult", sdl.Point{5, 0}, fg, renderer, font)
	defer lblTitle.Destroy()
	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ARGB8888, sdl.TEXTUREACCESS_TARGET, rect.W, rect.H)
	if err != nil {
		panic(err)
	}
	renderer.SetRenderTarget(texture)
	texture.SetBlendMode(sdl.BLENDMODE_BLEND)
	setColor(renderer, bg)
	renderer.Clear()
	setColor(renderer, fg)
	renderer.DrawRect(&sdl.Rect{0, 0, rect.W, rect.H})
	lblTitle.Render(renderer)
	renderer.DrawRect(&sdl.Rect{0, 0, rect.W, titleHeight})
	for i := int32(0); i < 3; i++ {
		renderer.DrawRect(&sdl.Rect{0, titleHeight, rect.W, titleHeight * (i + 1)})
		w := int32(float64(titleHeight) * 2)
		x := (rect.W - w) - w*i
		renderer.DrawLine(x, titleHeight, x, rect.H)
	}
	renderer.SetRenderTarget(nil)
	return texture
}

func (s *BoxDifficult) Show(state bool) { s.show = state }

func (s *BoxDifficult) Render(renderer *sdl.Renderer) {
	if s.show {
		renderer.Copy(s.texBg, nil, &s.rect)
		for _, sprite := range s.sprites {
			sprite.Render(renderer)
		}
	}
}

func (s *BoxDifficult) Update() {
	if s.show {
		for _, sprite := range s.sprites {
			sprite.Update()
		}
	}
}

func (s *BoxDifficult) Event(e sdl.Event) {
	if s.show {
		for _, sprite := range s.sprites {
			sprite.Event(e)
		}
	}
}

func (s *BoxDifficult) Destroy() {
	for _, sprite := range s.sprites {
		sprite.Destroy()
	}
	s.sprites = s.sprites[:0]
}
