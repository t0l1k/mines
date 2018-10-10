package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type (
	// Интерфейс наблюдателей
	Observers interface {
		Setup() error
		Update(Event) error
		Render(*sdl.Renderer) error
		Event(sdl.Event) (Event, error)
	}
	// Модель
	Mines struct{ subsribers []Observers }
	// Волчок Контроллер
	Spinner struct{ mines Mines }
	// Вид Представление
	View struct {
		window                 *sdl.Window
		renderer               *sdl.Renderer
		event                  sdl.Event
		pushTime, lastPushTime uint32
	}
	// Наблюдатели
	StatusLine struct {
		buttons      []buttonsData
		btnInstances []interface{}
	}
	// кнопки строки статуса
	buttonsType int
	buttonsData struct {
		name  buttonsType
		rect  sdl.Rect
		text  string
		event []Event
	}
	// События
	Event int
	// UI для sdl2
	Label struct {
		pos      sdl.Point
		font     *ttf.Font
		fontSize int
		text     string
		color    sdl.Color
	}

	Button struct {
		rect               sdl.Rect
		font               *ttf.Font
		fontSize           int32
		text               string
		fgColor, bgColor   sdl.Color
		focus, hide, dirty bool
		cursor             MouseCursor
	}

	Arrow struct {
		rect             sdl.Rect
		text             string
		fgColor, bgColor sdl.Color
		buttons          []buttonsData
		btnInstances     []interface{}
		count            int
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
	PauseEvent
	ResetGameEvent
	NewGameEvent
	IncRowEvent
	DecRowEvent
	IncColumnEvent
	DecColumnEvent
	IncMinesEvent
	DecMinesEvent
	IncButtonEvent
	DecButtonEvent
)

const (
	buttonQuit buttonsType = iota
	buttonPause
	buttonReset
	buttonNew
	buttonRow
	buttonCol
	buttonMines
	buttonHistory
	buttonDec
	buttonInc
	label
)

const (
	MouseButtonLeftPressed int = iota
	MouseButtonLeftReleased
	MouseButtonRightPressed
	MouseButtonRightReleased
	ButtonIncPressed
	ButtonDecPressed
	ButtonIncReleased
	ButtonDecReleased
)
const (
	minRow    = 5
	maxRow    = 30
	minColumn = 5
	maxColumn = 16
	minMines  = 5
	maxMines  = 99
)

var (
	mn                   int32 = 2
	WinWidth, WinHeight  int32 = 320 * mn, 180 * mn
	row, column, mines   int   = 8, 8, 10
	FontName                   = "data/Roboto-Regular.ttf"
	Background                 = sdl.Color{0, 129, 110, 255}
	Foreground                 = sdl.Color{223, 225, 81, 255}
	BackgroundStatusLine       = sdl.Color{0, 64, 32, 255}
	ForegroundStatusLine       = sdl.Color{255, 0, 64, 255}
	StatusLineFontSize   int32 = StatusLineHeight - 3
	StatusLineHeight     int32 = WinHeight / 20
)

/*
     ***** *                    *                  ***
  ******  *                   **                    ***
 **   *  *                    **                     **
*    *  *                     **                     **
    *  *                      **                     **
   ** **              ****    ** ****       ***      **
   ** **             * ***  * *** ***  *   * ***     **
   ** **            *   ****  **   ****   *   ***    **
   ** **           **    **   **    **   **    ***   **
   ** **           **    **   **    **   ********    **
   *  **           **    **   **    **   *******     **
      *            **    **   **    **   **          **
  ****           * **    **   **    **   ****    *   **
 *  *************   ***** **   *****      *******    *** *
*     *********      ***   **   ***        *****      ***
*
 **
*/

func (t *Label) New(pos sdl.Point, text string, color sdl.Color, fontSize int32) (err error) {
	t.pos = pos
	t.fontSize = int(fontSize)
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

func (t *Label) SetLabel(text string) {
	t.text = text
}

func (t *Label) GetLabel() string {
	return t.text
}

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

/*
   ***** **
  ******  ***                     *         *
 **   *  * **                    **        **
*    *  *  **                    **        **
    *  *   *    **   ****      ********  ********    ****
   ** **  *      **    ***  * ********  ********    * ***  * ***  ****
   ** ** *       **     ****     **        **      *   ****   **** **** *
   ** ***        **      **      **        **     **    **     **   ****
   ** ** ***     **      **      **        **     **    **     **    **
   ** **   ***   **      **      **        **     **    **     **    **
   *  **     **  **      **      **        **     **    **     **    **
      *      **  **      **      **        **     **    **     **    **
  ****     ***    ******* **     **        **      ******      **    **
 *  ********       *****   **     **        **      ****       ***   ***
*     ****                                                      ***   ***
*
 **
*/

func (t *Button) New(rect sdl.Rect, text string, fgColor sdl.Color, bgColor sdl.Color, fontSize int32) (err error) {
	t.rect = rect
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
	if t.font, err = ttf.OpenFont(FontName, int(t.fontSize)); err != nil {
		panic(err)
	}
	t.cursor = MouseCursor{}
	return nil
}

func (t *Button) SetLabel(text string) {
	t.text = text
	t.dirty = true
}

func (b *Button) Event(event sdl.Event) int {
	switch t := event.(type) {
	case *sdl.MouseButtonEvent:
		if b.focus && t.Button == sdl.BUTTON_LEFT && t.State == 1 {
			return MouseButtonLeftPressed
		} else if b.focus && t.Button == sdl.BUTTON_LEFT && t.State == 0 {
			return MouseButtonLeftReleased
		} else if b.focus && t.Button == sdl.BUTTON_RIGHT && t.State == 1 {
			return MouseButtonRightPressed
		} else if b.focus && t.Button == sdl.BUTTON_RIGHT && t.State == 0 {
			return MouseButtonRightReleased
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
	x := (t.rect.W-width)/2 + t.rect.X
	y := (t.rect.H-height)/2 + t.rect.Y
	renderer.SetDrawColor(bg.R, bg.G, bg.B, bg.A)
	renderer.FillRect(&sdl.Rect{t.rect.X, t.rect.Y, t.rect.W, t.rect.H})
	renderer.SetDrawColor(fg.R, fg.G, fg.B, fg.A)
	renderer.DrawRect(&sdl.Rect{t.rect.X, t.rect.Y, t.rect.W, t.rect.H})
	renderer.Copy(texture, nil, &sdl.Rect{x, y, width, height})
	return nil
}

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

func (t *Button) Quit() {
	t.font.Close()
}

/* **
     *****
    *  ***                                             **
       ***                                             **
      *  **       ***  ****    ***  ****       ****     **    ***    ****        ****
      *  **        **** **** *  **** **** *   * ***  *   **    ***     ***  *   * **** *
     *    **        **   ****    **   ****   *   ****    **     ***     ****   **  ****
     *    **        **           **         **    **     **      **      **   ****
    *      **       **           **         **    **     **      **      **     ***
    *********       **           **         **    **     **      **      **       ***
   *        **      **           **         **    **     **      **      **         ***
   *        **      **           **         **    **     **      **      *     ****  **
  *****      **     ***          ***         ******       ******* *******     * **** *
 *   ****    ** *    ***          ***         ****         *****   *****         ****
*     **      **
*
 ** */

func (t *Arrow) New(rect sdl.Rect, text string, fgColor, bgColor sdl.Color, fontSize int32) (err error) {
	t.rect = rect
	t.text = text
	t.fgColor = fgColor
	t.bgColor = bgColor
	t.buttons = []buttonsData{
		{name: buttonDec, rect: sdl.Rect{t.rect.X, t.rect.Y, t.rect.H, t.rect.H}, text: "<", event: []Event{DecButtonEvent}},
		{name: label, rect: sdl.Rect{t.rect.X + t.rect.H, t.rect.Y, t.rect.H * 6, t.rect.H}, text: t.text, event: []Event{NilEvent}},
		{name: buttonInc, rect: sdl.Rect{t.rect.X + t.rect.H*6, t.rect.Y, t.rect.H, t.rect.H}, text: ">", event: []Event{IncButtonEvent}}}
	for _, button := range t.buttons {
		switch button.name {
		case buttonDec:
			btn := &Button{}
			if err = btn.New(button.rect, button.text, t.bgColor, sdl.Color{0, 0, 0, 255}, fontSize); err != nil {
				panic(err)
			}
			t.btnInstances = append(t.btnInstances, btn)
		case buttonInc:
			btn := &Button{}
			if err = btn.New(button.rect, button.text, t.bgColor, sdl.Color{0, 0, 0, 255}, fontSize); err != nil {
				panic(err)
			}
			t.btnInstances = append(t.btnInstances, btn)
		case label:
			lbl := &Label{}
			if err = lbl.New(sdl.Point{button.rect.X, button.rect.Y}, button.text, t.fgColor, fontSize); err != nil {
				panic(err)
			}
			t.btnInstances = append(t.btnInstances, lbl)
		}
	}
	return nil
}

func (t *Arrow) SetLabel(text string) {
	t.text = text
	t.btnInstances[1].(*Label).SetLabel(t.text)
}

func (t *Arrow) GetLabel() string {
	return t.text
}

func (s *Arrow) Event(event sdl.Event) (e Event, err error) {
	for idx, button := range s.btnInstances {
		switch event.(type) {
		case *sdl.MouseButtonEvent:
			switch button.(type) {
			case *Button:
				if ok := button.(*Button).Event(event); ok == MouseButtonLeftReleased {
					for i := 0; i < len(s.buttons[idx].event); i++ {
						switch s.buttons[idx].event[i] {
						case DecButtonEvent:
							// fmt.Println("DecButtonEvent!", button)
							return DecButtonEvent, nil
						case IncButtonEvent:
							// fmt.Println("IncButtonEvent!", button)
							return IncButtonEvent, nil
						}
					}
				}
			case *Label:
			}
		}
	}
	return NilEvent, nil
}

func (s *Arrow) Update(event Event) {
	for i := range s.btnInstances {
		switch s.btnInstances[i].(type) {
		case *Button:
			s.btnInstances[i].(*Button).Update()
		case *Label:
		}
	}
}

func (s *Arrow) Render(renderer *sdl.Renderer) (err error) {
	for _, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			if err = button.(*Button).Render(renderer); err != nil {
				panic(err)
			}
		case *Label:
			if err = button.(*Label).Render(renderer); err != nil {
				panic(err)
			}
		}
	}
	return nil
}

/********
    *       ***      *                    *
   *         **     **                   **
   **        *      **                   **
    ***           ********             ******** **   ****        ****
   ** ***        ********     ****    ********   **    ***  *   * **** *
    *** ***         **       * ***  *    **      **     ****   **  ****
      *** ***       **      *   ****     **      **      **   ****
        *** ***     **     **    **      **      **      **     ***
          ** ***    **     **    **      **      **      **       ***
           ** **    **     **    **      **      **      **         ***
            * *     **     **    **      **      **      **    ****  **
  ***        *      **     **    **      **       ******* **  * **** *
 *  *********        **     ***** **      **       *****   **    ****
*     *****                  ***   **
*
 ***/
func (s *StatusLine) Setup() (err error) {
	s.buttons = []buttonsData{
		{name: buttonQuit, rect: sdl.Rect{0, 0, StatusLineHeight, StatusLineHeight}, text: "<-", event: []Event{QuitEvent}},
		{name: buttonPause, rect: sdl.Rect{StatusLineHeight, 0, StatusLineHeight * 3, StatusLineHeight}, text: "Pause", event: []Event{PauseEvent}},
		{name: buttonReset, rect: sdl.Rect{StatusLineHeight * 4, 0, StatusLineHeight * 3, StatusLineHeight}, text: "Reset", event: []Event{ResetGameEvent}},
		{name: buttonNew, rect: sdl.Rect{StatusLineHeight * 7, 0, StatusLineHeight * 2, StatusLineHeight}, text: "New", event: []Event{NewGameEvent}},
		{name: buttonRow, rect: sdl.Rect{StatusLineHeight * 9, 0, StatusLineHeight * 6, StatusLineHeight}, text: "Rows:" + strconv.Itoa(row), event: []Event{IncRowEvent, DecRowEvent}},
		{name: buttonCol, rect: sdl.Rect{StatusLineHeight * 16, 0, StatusLineHeight * 6, StatusLineHeight}, text: "Columns:" + strconv.Itoa(column), event: []Event{IncRowEvent, DecRowEvent}},
		{name: buttonMines, rect: sdl.Rect{StatusLineHeight * 23, 0, StatusLineHeight * 6, StatusLineHeight}, text: "Mines:" + strconv.Itoa(mines), event: []Event{IncRowEvent, DecRowEvent}}}
	for _, button := range s.buttons {
		switch button.name {
		case buttonQuit:
			btn := &Button{}
			if err = btn.New(button.rect, button.text, BackgroundStatusLine, ForegroundStatusLine, StatusLineFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, btn)
		case buttonPause:
			btn := &Button{}
			if err = btn.New(button.rect, button.text, BackgroundStatusLine, ForegroundStatusLine, StatusLineFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, btn)
		case buttonReset:
			btn := &Button{}
			if err = btn.New(button.rect, button.text, BackgroundStatusLine, ForegroundStatusLine, StatusLineFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, btn)
		case buttonNew:
			btn := &Button{}
			if err = btn.New(button.rect, button.text, BackgroundStatusLine, ForegroundStatusLine, StatusLineFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, btn)
		case buttonRow:
			btn := &Arrow{}
			if err = btn.New(button.rect, button.text, BackgroundStatusLine, ForegroundStatusLine, StatusLineFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, btn)
		case buttonCol:
			btn := &Arrow{}
			if err = btn.New(button.rect, button.text, BackgroundStatusLine, ForegroundStatusLine, StatusLineFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, btn)
		case buttonMines:
			btn := &Arrow{}
			if err = btn.New(button.rect, button.text, BackgroundStatusLine, ForegroundStatusLine, StatusLineFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, btn)
		}
	}
	return nil
}

func (s *StatusLine) Update(event Event) error {
	for i, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			s.btnInstances[i].(*Button).Update()
		case *Arrow:
			s.btnInstances[i].(*Arrow).Update(event)
		}
	}
	return nil
}

func (s *StatusLine) Render(renderer *sdl.Renderer) (err error) {
	for _, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			if err = button.(*Button).Render(renderer); err != nil {
				panic(err)
			}
		case *Arrow:
			if err = button.(*Arrow).Render(renderer); err != nil {
				panic(err)
			}
		}

	}
	return nil
}

func (s *StatusLine) Event(event sdl.Event) (e Event, err error) {
	for idx, button := range s.btnInstances {
		switch event.(type) {
		case *sdl.MouseButtonEvent:
			switch button.(type) {
			case *Button:
				if ok := button.(*Button).Event(event); ok == MouseButtonLeftReleased {
					for i := 0; i < len(s.buttons[idx].event); i++ {
						switch s.buttons[idx].event[i] {
						case QuitEvent:
							fmt.Println("Get QuitEvent", s.buttons[idx].name)
							return QuitEvent, nil
						case PauseEvent:
							fmt.Println("Get PauseEvent", s.buttons[idx].name)
							return PauseEvent, nil
						case ResetGameEvent:
							fmt.Println("Get ResetEvent", s.buttons[idx].name)
							return ResetGameEvent, nil
						case NewGameEvent:
							fmt.Println("Get NewEvent", s.buttons[idx].name)
							return NewGameEvent, nil
						}
					}
				}

			case *Arrow:
				if ev, _ := button.(*Arrow).Event(event); ev != NilEvent {
					switch s.buttons[idx].name {
					case buttonRow:
						switch ev {
						case IncButtonEvent:
							fmt.Println("Get inc row event", s.buttons[idx].name, ev)
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "inc")
							return IncRowEvent, nil
						case DecButtonEvent:
							fmt.Println("Get dec row event", s.buttons[idx].name, ev)
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "dec")
							return DecRowEvent, nil
						}
					case buttonCol:
						switch ev {
						case IncButtonEvent:
							fmt.Println("Get inc column event", s.buttons[idx].name, ev)
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "inc")
							return IncColumnEvent, nil
						case DecButtonEvent:
							fmt.Println("Get dec column event", s.buttons[idx].name, ev)
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "dec")
							return DecColumnEvent, nil

						}
					case buttonMines:
						switch ev {
						case IncButtonEvent:
							fmt.Println("Get inc mines event", s.buttons[idx].name, ev)
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "inc")
							return IncMinesEvent, nil
						case DecButtonEvent:
							fmt.Println("Get dec mines event", s.buttons[idx].name, ev)
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "dec")
							return DecMinesEvent, nil
						}
					}
				}
			}
		}
	}
	return NilEvent, nil
}

func (s *StatusLine) calc(name buttonsType, instance *Arrow, op string) {
	text := instance.GetLabel()
	textArr := strings.Split(text, ":")
	text, num := textArr[0], textArr[1]
	n, err := strconv.Atoi(num)
	if err != nil {
		panic(err)
	}
	if name == buttonRow || name == buttonCol || name == buttonMines {

		if n < maxMines {
			if op == "inc" {
				n++
			}
		}
		if n > minMines {
			if op == "dec" {
				n--
			}
		}
	}
	text += ":" + strconv.Itoa(n)
	fmt.Println("update", text, textArr, num, n, op, name)
	instance.SetLabel(text)
}

/*    *****   **    **
  ******  ***** *****     *
 **   *  *  ***** *****  ***
*    *  *   * **  * **    *
    *  *    *     *                                       ****
   ** **    *     *     ***     ***  ****       ***      * **** *
   ** **    *     *      ***     **** **** *   * ***    **  ****
   ** **    *     *       **      **   ****   *   ***  ****
   ** **    *     *       **      **    **   **    ***   ***
   ** **    *     **      **      **    **   ********      ***
   *  **    *     **      **      **    **   *******         ***
      *     *      **     **      **    **   **         ****  **
  ****      *      **     **      **    **   ****    * * **** *
 *  *****           **    *** *   ***   ***   *******     ****
*     **                   ***     ***   ***   *****
*
 ** */
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
		if err := subscriber.Update(e); err != nil {
			panic(err)
		}
	}
}

func (s Mines) GetSubscribers() []Observers {
	return s.subsribers
}

/* ***** *      **
  ******  *    *****     *
 **   *  *       *****  ***              **
*    *  **       * **    *               **
    *  ***      *                         **    ***    ****
   **   **      *      ***        ***      **    ***     ***  *
   **   **      *       ***      * ***     **     ***     ****
   **   **     *         **     *   ***    **      **      **
   **   **     *         **    **    ***   **      **      **
   **   **     *         **    ********    **      **      **
    **  **    *          **    *******     **      **      **
     ** *     *          **    **          **      **      *
      ***     *          **    ****    *    ******* *******
       *******           *** *  *******      *****   *****
         ***              ***    ******/
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
	s.pushTime = 1
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

/********
    *       ***               *
   *         **              ***
   **        *                *
    ***             ****                                                ***  ****
   ** ***          * ***  * ***     ***  ****    ***  ****       ***     **** **** *
    *** ***       *   ****   ***     **** **** *  **** **** *   * ***     **   ****
      *** ***    **    **     **      **   ****    **   ****   *   ***    **
        *** ***  **    **     **      **    **     **    **   **    ***   **
          ** *** **    **     **      **    **     **    **   ********    **
           ** ** **    **     **      **    **     **    **   *******     **
            * *  **    **     **      **    **     **    **   **          **
  ***        *   *******      **      **    **     **    **   ****    *   ***
 *  *********    ******       *** *   ***   ***    ***   ***   *******     ***
*     *****      **            ***     ***   ***    ***   ***   *****
*                **
 **              **
                  ***/
func (s *Spinner) Run(m Mines, v View) {
	s.mines = Mines{}
	if err := v.Setup(); err != nil {
		panic(err)
	}
	st := &StatusLine{}
	st.Setup()
	s.mines.Attach(st)
	dirty := true
	running := true
	for running {
		for _, event := range v.GetEvents(s.mines.GetSubscribers()) {
			if event == QuitEvent {
				running = false
			} else if event == TickEvent {
				dirty = true
			}
			s.mines.Notify(event)
		}
		if dirty {
			if err := v.Render(s.mines.GetSubscribers()); err != nil {
				panic(err)
			}
		}
	}
}

/******   **    **
  ******  ***** *****                *
 **   *  *  ***** *****             ***
*    *  *   * **  * **               *
    *  *    *     *
   ** **    *     *        ****    ***     ***  ****
   ** **    *     *       * ***  *  ***     **** **** *
   ** **    *     *      *   ****    **      **   ****
   ** **    *     *     **    **     **      **    **
   ** **    *     **    **    **     **      **    **
   *  **    *     **    **    **     **      **    **
      *     *      **   **    **     **      **    **
  ****      *      **   **    **     **      **    **
 *  *****           **   ***** **    *** *   ***   ***
*     **                  ***   **    ***     ***   ***
*
 ***/
func main() {
	m := Mines{}
	v := View{}
	c := Spinner{}
	c.Run(m, v)
}
