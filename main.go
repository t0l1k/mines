package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type (
	// Интерфейс наблюдателей
	Observers interface {
		Setup() error
		Update(int) error
		Render(*sdl.Renderer) error
		Event(sdl.Event) (Event, error)
	}
	// Модель
	Mines struct {
		subsribers []Observers
	}
	// Волчок Контроллер
	Spinner struct {
		mines   Mines
		running bool
	}
	// Вид Представление
	View struct {
		window                 *sdl.Window
		renderer               *sdl.Renderer
		event                  sdl.Event
		pushTime, lastPushTime uint32
	}
	// Наблюдатели
	ButtonQuit struct {
		button Button
	}
	// События
	Event int
	// Метка для sdl2
	Label struct {
		pos      sdl.Point
		font     *ttf.Font
		fontSize int
		text     string
		color    sdl.Color
	}
	Button struct {
		pos                sdl.Point
		size               sdl.Point
		rect               sdl.Rect
		font               *ttf.Font
		fontSize           int
		text               string
		fgColor, bgColor   sdl.Color
		focus, hide, dirty bool
		cursor             MouseCursor
	}
	MouseCursor struct {
		sdl.Point
		button uint32
	}
)

const (
	NilEvent Event = iota
	TickEvent
	QuitEvent
)

const (
	ButtonLeftPressed int = iota
	ButtonLeftReleased
	ButtonRightPressed
	ButtonRightReleased
)

var (
	mn                  int32 = 2
	WinWidth, WinHeight int32 = 320 * mn, 180 * mn
	FontName                  = "data/Roboto-Regular.ttf"
	Background                = sdl.Color{0, 129, 110, 255}
	Foreground                = sdl.Color{223, 225, 81, 255}
	Foreground2               = sdl.Color{206, 22, 97, 255}
)

func (t *Label) New(pos sdl.Point, text string, color sdl.Color, fontSize int) (err error) {
	t.pos = pos
	t.fontSize = fontSize
	t.text = text
	t.color = color
	err = ttf.Init()
	if err != nil {
		panic(err)
	}
	if t.font, err = ttf.OpenFont(FontName, t.fontSize); err != nil {
		panic(err)
	}
	return nil
}

// SetLabel replace text
func (t *Label) SetLabel(text string) {
	t.text = text
}

//Render text to surface renderer
func (t *Label) Render(renderer *sdl.Renderer) (err error) {
	var textSurf *sdl.Surface
	var texture *sdl.Texture
	if textSurf, err = t.font.RenderUTF8Blended(t.text, t.color); err != nil {
		return err
	}
	defer textSurf.Free()
	if texture, err = renderer.CreateTextureFromSurface(textSurf); err != nil {
		return err
	}
	_, _, width, height, _ := texture.Query()
	defer texture.Destroy()
	renderer.Copy(texture, nil, &sdl.Rect{t.pos.X, t.pos.Y, width, height})
	return nil
}

// Quit uninit
func (t *Label) Quit() {
	t.font.Close()
}

func (m *MouseCursor) Update() (int32, int32, uint32) {
	m.X, m.Y, m.button = sdl.GetMouseState()
	return m.X, m.Y, m.button
}

func (m MouseCursor) String() string {
	return fmt.Sprintf("Mouse x:%v y:%v button:%v", m.X, m.Y, m.button)
}

func (t *Button) New(pos sdl.Point, size sdl.Point, text string, fgColor sdl.Color, bgColor sdl.Color, fontSize int) (err error) {
	t.pos = pos
	t.size = size
	t.rect = sdl.Rect{t.pos.X, t.pos.Y, t.size.X, t.size.Y}
	t.fontSize = fontSize
	t.text = text
	t.fgColor = fgColor
	t.bgColor = bgColor
	t.focus = false
	t.hide = false
	t.dirty = true
	err = ttf.Init()
	if err != nil {
		panic(err)
	}
	if t.font, err = ttf.OpenFont(FontName, t.fontSize); err != nil {
		panic(err)
	}
	t.cursor = MouseCursor{}
	return nil
}

// SetLabel replace text
func (t *Button) SetLabel(text string) {
	t.text = text
	t.dirty = true
}

func (b *Button) Event(event sdl.Event) int {
	switch t := event.(type) {
	case *sdl.MouseButtonEvent:
		if b.focus && t.Button == sdl.BUTTON_LEFT && t.State == 1 {
			return ButtonLeftPressed
		} else if b.focus && t.Button == sdl.BUTTON_LEFT && t.State == 0 {
			return ButtonLeftReleased
		} else if b.focus && t.Button == sdl.BUTTON_RIGHT && t.State == 1 {
			return ButtonRightPressed
		} else if b.focus && t.Button == sdl.BUTTON_RIGHT && t.State == 0 {
			return ButtonRightReleased
		}
	}
	return -1
}

func (t *Button) Update() {
	t.cursor.Update()
	if t.cursor.InRect(&t.rect) {
		t.focus = true
		t.dirty = true
	} else {
		t.focus = false
		t.dirty = true
	}
}

func (t *Button) paint(renderer *sdl.Renderer, fg, bg sdl.Color) (err error) {
	var textSurf *sdl.Surface
	var texture *sdl.Texture
	if textSurf, err = t.font.RenderUTF8Blended(t.text, fg); err != nil {
		return err
	}
	defer textSurf.Free()
	if texture, err = renderer.CreateTextureFromSurface(textSurf); err != nil {
		return err
	}
	_, _, width, height, _ := texture.Query()
	defer texture.Destroy()
	x := (t.size.X-width)/2 + t.pos.X
	y := (t.size.Y-height)/2 + t.pos.Y
	renderer.SetDrawColor(bg.R, bg.G, bg.B, bg.A)
	renderer.FillRect(&sdl.Rect{t.pos.X, t.pos.Y, t.size.X, t.size.Y})
	renderer.SetDrawColor(fg.R, fg.G, fg.B, fg.A)
	renderer.DrawRect(&sdl.Rect{t.pos.X, t.pos.Y, t.size.X, t.size.Y})
	renderer.Copy(texture, nil, &sdl.Rect{x, y, width, height})
	return nil
}

//Render text to surface renderer
func (t *Button) Render(renderer *sdl.Renderer) (err error) {
	if t.dirty {
		if t.focus {
			t.paint(renderer, t.fgColor, t.bgColor)
		} else {
			t.paint(renderer, t.bgColor, t.fgColor)
		}
	}
	t.dirty = false
	return nil
}

// Quit uninit
func (t *Button) Quit() {
	t.font.Close()
}

func (s *ButtonQuit) Setup() (err error) {
	err = s.button.New(sdl.Point{0, 0}, sdl.Point{15, 15}, "<-", Background, Foreground, 12)
	if err != nil {
		panic(err)
	}
	return nil
}

func (s *ButtonQuit) Update(data int) error {
	s.button.Update()
	return nil
}

func (s ButtonQuit) Render(renderer *sdl.Renderer) (err error) {
	if err = s.button.Render(renderer); err != nil {
		panic(err)
	}
	return nil
}

func (s *ButtonQuit) Event(event sdl.Event) (e Event, err error) {
	switch event.(type) {
	case *sdl.MouseButtonEvent:
		if ok := s.button.Event(event); ok == ButtonLeftReleased {
			return QuitEvent, nil
		}
	}
	return NilEvent, nil
}

func (s *Mines) Attach(o Observers) {
	s.subsribers = append(s.subsribers, o)
}

func (s *Mines) Dettach(o Observers) {
	var idx int
	for i, subscriber := range s.subsribers {
		if subscriber == o {
			idx = i
			break
		}
		s.subsribers = append(s.subsribers[:idx], s.subsribers[idx+1:]...)
	}
}

func (s *Mines) Notify(e Event) {
	for _, subscriber := range s.subsribers {
		if err := subscriber.Update(0); err != nil {
			panic(err)
		}
	}
}

func (s Mines) GetSubscribers() []Observers {
	return s.subsribers
}

func (s *View) Setup() (err error) {
	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	if s.window, err = sdl.CreateWindow("Template", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, WinWidth, WinHeight, sdl.WINDOW_SHOWN); err != nil {
		panic(err)
	}
	if s.renderer, err = sdl.CreateRenderer(s.window, -1, sdl.RENDERER_ACCELERATED); err != nil {
		panic(err)
	}
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")
	s.pushTime = 10
	s.lastPushTime = sdl.GetTicks()
	return nil
}

func (s *View) Render(o []Observers) (err error) {
	s.renderer.SetDrawColor(Background.R, Background.G, Background.B, Background.A)
	s.renderer.Clear()
	for _, subscriber := range o {
		subscriber.Render(s.renderer)
	}
	s.renderer.Present()
	return nil
}

func (s *View) GetEvents(o []Observers) (events []Event) {
	for s.event = sdl.PollEvent(); s.event != nil; s.event = sdl.PollEvent() {
		switch t := s.event.(type) {
		case *sdl.QuitEvent:
			events = append(events, QuitEvent)
			return events
		case *sdl.KeyboardEvent:
			if t.Keysym.Sym == sdl.K_ESCAPE && t.State == sdl.RELEASED {
				events = append(events, QuitEvent)
				return events
			}
		}
		for _, subscriber := range o {
			event, err := subscriber.Event(s.event)
			if err != nil {
				panic(err)
			}
			if event != NilEvent {
				events = append(events, event)
				return events
			}
		}
	}
	if s.lastPushTime+s.pushTime < sdl.GetTicks() {
		s.lastPushTime = sdl.GetTicks()
		events = append(events, TickEvent)
	}
	return events
}

func (s *Spinner) Run(m Mines, v View) {
	s.mines = Mines{}
	if err := v.Setup(); err != nil {
		panic(err)
	}
	btn := &ButtonQuit{}
	btn.Setup()
	s.mines.Attach(btn)
	dirty := true
	s.running = true
	for s.running {
		for _, event := range v.GetEvents(s.mines.GetSubscribers()) {
			if event == QuitEvent {
				s.running = false
			} else if event == TickEvent {
				s.mines.Notify(TickEvent)
				dirty = true
			}
		}
		if dirty {
			if err := v.Render(s.mines.GetSubscribers()); err != nil {
				panic(err)
			}
		}
	}
}

func main() {
	m := Mines{}
	v := View{}
	c := Spinner{}
	c.Run(m, v)
}
