package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type Sprite interface {
	Render(*sdl.Renderer)
	Update()
	Event(sdl.Event)
	Destroy()
}

type Screen struct {
	title                  string
	window                 *sdl.Window
	renderer               *sdl.Renderer
	width, height          int32
	font                   *ttf.Font
	flags                  uint32
	running                bool
	bg, fg, bgMenu, fgMenu sdl.Color
	fpsCountTime, fpsCount uint32
	sprites                []Sprite
	menuLine               *MenuLine
	statusLine             *StatusLine
	statusLineMessage      string
}

func NewScreen(title string, window *sdl.Window, renderer *sdl.Renderer, width, height int32) *Screen {
	return &Screen{
		title:    title,
		window:   window,
		renderer: renderer,
		width:    width,
		height:   height,
		bg:       sdl.Color{192, 192, 192, 0},
		fg:       sdl.Color{0, 0, 0, 255},
		fgMenu:   sdl.Color{0, 0, 0, 255},
		bgMenu:   sdl.Color{128, 128, 128, 0},
	}
}

func (s *Screen) setup() {
	var err error
	fontSize := int(float64(s.height) * 0.03) // Главная константа перерисовки экрана
	s.font, err = ttf.OpenFont("assets/Roboto-Regular.ttf", fontSize)
	if err != nil {
		panic(err)
	}

	s.menuLine = NewMenuLine(s.title, sdl.Rect{0, 0, s.width, int32(float64(fontSize) * 1.5)}, s.fgMenu, s.bgMenu, s.renderer, s.font, s.quit)
	s.sprites = append(s.sprites, s.menuLine)
	s.statusLine = NewStatusLine("test message", sdl.Rect{0, s.height - int32(float64(fontSize)*1.5), s.width, int32(float64(fontSize) * 1.5)}, s.fgMenu, s.bgMenu, s.renderer, s.font)
	s.statusLineMessage = "test message"
	s.sprites = append(s.sprites, s.statusLine)
}

func (s *Screen) setMode() {
	if s.flags == 0 {
		s.flags = sdl.WINDOW_FULLSCREEN_DESKTOP
		mode, err := sdl.GetCurrentDisplayMode(0)
		if err != nil {
			panic(err)
		}
		s.width, s.height = mode.W, mode.H
	} else {
		s.flags = 0
		s.width, s.height = 800, 600
	}
	s.window.SetFullscreen(s.flags)
	s.window.SetSize(s.width, s.height)
	s.Destroy()
	s.setup()
}

func (s *Screen) Event() {
	event := sdl.WaitEventTimeout(3)
	switch t := event.(type) {
	case *sdl.QuitEvent:
		s.quit()
	case *sdl.KeyboardEvent:
		if t.Keysym.Sym == sdl.K_ESCAPE && t.State == sdl.RELEASED {
			s.quit()
		}
		if t.Keysym.Sym == sdl.K_F11 && t.State == sdl.RELEASED {
			s.setMode()
		}
	case *sdl.WindowEvent:
		switch t.Event {
		case sdl.WINDOWEVENT_RESIZED:
			s.width, s.height = t.Data1, t.Data2
			s.Destroy()
			s.setup()
			// fmt.Println("window resized", s.width, s.height)
		case sdl.WINDOWEVENT_FOCUS_GAINED:
			// fmt.Println("window focus gained", s.width, s.height)
		case sdl.WINDOWEVENT_FOCUS_LOST:
			// fmt.Println("window focus lost", s.width, s.height)
		case sdl.WINDOW_MINIMIZED:
			s.Destroy()
		case sdl.WINDOWEVENT_RESTORED:
			s.setup()
		}
	}
	for _, sprite := range s.sprites {
		sprite.Event(event)
	}
}

func (s *Screen) Update() {
	for _, sprite := range s.sprites {
		sprite.Update()
	}
	if sdl.GetTicks()-s.fpsCountTime > 999 {
		s.window.SetTitle(fmt.Sprintf("%s fps:%v", s.title, s.fpsCount))
		s.statusLine.SetMessage(fmt.Sprintf("fps:%v : %v", s.fpsCount, s.statusLineMessage))
		s.fpsCount = 0
		s.fpsCountTime = sdl.GetTicks()
	}
}

func (s *Screen) Render() {
	setColor(s.renderer, s.bg)
	s.renderer.Clear()
	for _, sprite := range s.sprites {
		sprite.Render(s.renderer)
	}
	s.renderer.Present()
	s.fpsCount++
}

func (s *Screen) Init() {
	s.setup()
}

func (s *Screen) Run() {
	s.Init()
	frameRate := uint32(1000 / 60)
	lastTime := sdl.GetTicks()
	s.running = true
	for s.running {
		now := sdl.GetTicks()
		if now >= lastTime {
			i := 0
			for {
				s.Event()
				s.Update()
				lastTime += frameRate
				now = sdl.GetTicks()
				if lastTime > now {
					break
				}
				i++
				if i >= 3 {
					lastTime = now + frameRate
					break
				}
			}
			s.Render()
		} else {
			sdl.Delay(lastTime - now)
		}
	}
}

func (s *Screen) Destroy() {
	for _, sprite := range s.sprites {
		sprite.Destroy()
	}
	s.sprites = s.sprites[:0]
	s.font.Close()
}

func (s *Screen) quit() { s.running = false }
