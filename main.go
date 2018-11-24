package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

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
	Mines struct {
		subsribers []Observers
		field      Field
	}
	// Ячейка минного поля
	Cell struct {
		pos     sdl.Point
		state   int32
		mined   bool
		counter int32
	}
	// Минное поле
	minesStateType int32
	Field          struct {
		field     []Cell
		state     minesStateType
		boardSize boardConfig
	}
	// Волчок Контроллер
	Spinner struct{ mines Mines }
	// Вид Представление
	View struct {
		window                 *sdl.Window
		renderer               *sdl.Renderer
		event                  sdl.Event
		pushTime, lastPushTime uint32
		flags                  uint32
	}
	// Наблюдатель строка меню
	StatusLine struct {
		buttons       []buttonsData
		btnInstances  []interface{}
		gameBoardSize boardConfig
	}
	// Наблюдатель поле игры
	GameBoard struct {
		rect                  sdl.Rect
		btnInstances          []interface{}
		colors                []sdl.Color
		gameBoardSize         boardConfig
		cellWidth, cellHeight int32
		mousePressedAtButton  int32
		messageBox            *MessageBox
		start                 bool
	}
	// Кнопки строки статуса
	buttonsType int
	buttonsData struct {
		name  buttonsType
		rect  sdl.Rect
		text  string
		event []Event
	}
	// События
	Event int
	// Размеры минного поля
	boardConfig struct {
		row, column, mines, minesPercent int32
	}
	// UI для sdl2
	// Метка умеет выводить текст
	Label struct {
		pos      sdl.Point
		font     *ttf.Font
		fontSize int
		text     string
		color    sdl.Color
		surface  *sdl.Surface
	}
	// Кнопка умеет откликься на нажатия и отжатия левой и правой кнопки
	Button struct {
		rect             sdl.Rect
		font             *ttf.Font
		fontSize         int32
		text             string
		fgColor, bgColor sdl.Color
		focus, hide      bool
		cursor           MouseCursor
	}
	// Стрелки умеет отпралять события нажатия и уже другие наблюдатели на эти события реагируют
	Arrow struct {
		rect             sdl.Rect
		text             string
		fgColor, bgColor sdl.Color
		buttons          []buttonsData
		btnInstances     []interface{}
		count            int
	}
	// Указатель мыши нужен для обработки нажатий кнопки
	MouseCursor struct {
		sdl.Point
		button uint32
	}
	// Умеет выводить окно сообщения
	MessageBox struct {
		rect         sdl.Rect
		title        string
		titleLabel   Label
		message      string
		messageLabel Label
		okButton     Button
		Hide         bool
		fg, bg       sdl.Color
	}
)

// Перечень событий
const (
	NilEvent Event = iota
	TickEvent
	QuitEvent
	WindowResized
	FullScreenToggleEvent
	NewGameEvent
	PauseEvent
	ResetGameEvent
	IncRowEvent
	DecRowEvent
	IncColumnEvent
	DecColumnEvent
	IncMinesEvent
	DecMinesEvent
	IncButtonEvent
	DecButtonEvent
	MouseButtonLeftPressedEvent
	MouseButtonLeftReleasedEvent
	MouseButtonRightPressedEvent
	MouseButtonRightReleasedEvent
)

// перечень кнопок строки статуса
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

// события от кнопок мышки
const (
	MouseButtonLeftPressed int = iota
	MouseButtonLeftReleased
	MouseButtonRightPressed
	MouseButtonRightReleased
)

// состояния ячеек минного поля
const (
	closed int32 = (iota + 10)
	flagged
	questionable
	opened
	mined
	saved
	blown
	firstMined
	empty
	wrongMines
	play
	won
	lost
	marked
)

// состояния игры
const (
	gameStart minesStateType = iota
	gamePlay
	gamePause
	gameWin
	gameOver
)

// константы размеров минного поля
const (
	minRow    = 5
	maxRow    = 30
	minColumn = 5
	maxColumn = 16
	minMines  = 5
	maxMines  = 999
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
o            8             8
8            8             8
8     .oPYo. 8oPYo. .oPYo. 8
8     .oooo8 8    8 8oooo8 8
8     8    8 8    8 8.     8
8oooo `YooP8 `YooP' `Yooo' 8
......:.....::.....::.....:..
:::::::::::::::::::::::::::::
:::::::::::::::::::::::::::::*/

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
	var texture *sdl.Texture
	if t.surface, err = t.font.RenderUTF8Blended(t.text, t.color); err != nil {
		return err
	}
	defer t.surface.Free()
	if texture, err = renderer.CreateTextureFromSurface(t.surface); err != nil {
		return err
	}
	_, _, width, height, _ := texture.Query()
	defer texture.Destroy()
	renderer.Copy(texture, nil, &sdl.Rect{t.pos.X, t.pos.Y, width, height})
	return nil
}

func (t *Label) Destroy() {
	t.font.Close()
}

/*
o     o                             .oPYo.
8b   d8                             8    8
8`b d'8 .oPYo. o    o .oPYo. .oPYo. 8      o    o oPYo. .oPYo. .oPYo. oPYo.
8 `o' 8 8    8 8    8 Yb..   8oooo8 8      8    8 8  `' Yb..   8    8 8  `'
8     8 8    8 8    8   'Yb. 8.     8    8 8    8 8       'Yb. 8    8 8
8     8 `YooP' `YooP' `YooP' `Yooo' `YooP' `YooP' 8     `YooP' `YooP' 8
..::::..:.....::.....::.....::.....::.....::.....:..:::::.....::.....:..::::
::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::
::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::*/
func (m *MouseCursor) Update() (int32, int32, uint32) {
	m.X, m.Y, m.button = sdl.GetMouseState()
	return m.X, m.Y, m.button
}

func (m MouseCursor) String() string {
	return fmt.Sprintf("Mouse x:%v y:%v button:%v", m.X, m.Y, m.button)
}

/*
.oPYo.          o    o
 8   `8          8    8
o8YooP' o    o  o8P  o8P .oPYo. odYo.
 8   `b 8    8   8    8  8    8 8' `8
 8    8 8    8   8    8  8    8 8   8
 8oooP' `YooP'   8    8  `YooP' 8   8
:......::.....:::..:::..::.....:..::..
::::::::::::::::::::::::::::::::::::::
::::::::::::::::::::::::::::::::::::::*/
func (t *Button) New(rect sdl.Rect, text string, fgColor, bgColor sdl.Color, fontSize int32) (err error) {
	t.rect = rect
	t.fontSize = fontSize
	t.text = text
	t.fgColor = fgColor
	t.bgColor = bgColor
	t.focus = false
	t.hide = false
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

func (t *Button) GetLabel() string {
	return t.text
}

func (t *Button) SetLabel(text string) {
	t.text = text
}

func (t *Button) SetBackground(color sdl.Color) {
	t.bgColor = color
}

func (t *Button) SetForeground(color sdl.Color) {
	t.fgColor = color
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
	} else {
		t.focus = false
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
	if !t.focus {
		t.paint(renderer, t.fgColor, t.bgColor)
	} else {
		t.paint(renderer, t.bgColor, t.fgColor)
	}
	return nil
}

func (t *Button) Destroy() {
	t.font.Close()
}

/*

o     o                                            .oPYo.
8b   d8                                            8   `8
8`b d'8 .oPYo. .oPYo. .oPYo. .oPYo. .oPYo. .oPYo. o8YooP' .oPYo. `o  o'
8 `o' 8 8oooo8 Yb..   Yb..   .oooo8 8    8 8oooo8  8   `b 8    8  `bd'
8     8 8.       'Yb.   'Yb. 8    8 8    8 8.      8    8 8    8  d'`b
8     8 `Yooo' `YooP' `YooP' `YooP8 `YooP8 `Yooo'  8oooP' `YooP' o'  `o
..::::..:.....::.....::.....::.....::....8 :.....::......::.....:..:::..
::::::::::::::::::::::::::::::::::::::ooP'.:::::::::::::::::::::::::::::
::::::::::::::::::::::::::::::::::::::...:::::::::::::::::::::::::::::::
*/

func (b *MessageBox) New(rect sdl.Rect, title, message string, fg, bg sdl.Color) (err error) {
	b.rect = rect
	b.title = title
	b.message = message
	b.fg = fg
	b.bg = bg
	err = b.titleLabel.New(sdl.Point{b.rect.X + 5, b.rect.Y + 3}, b.title, b.fg, 10)
	if err != nil {
		panic(err)
	}
	err = b.messageLabel.New(sdl.Point{b.rect.X + 30, b.rect.Y + 50}, b.message, b.fg, 30)
	if err != nil {
		panic(err)
	}
	err = b.okButton.New(sdl.Rect{b.rect.X + (b.rect.W-100)/2, b.rect.Y + b.rect.H - 25, 100, 20}, "Ok", b.fg, b.bg, 20)
	if err != nil {
		panic(err)
	}
	b.Hide = false
	return nil
}

func (b *MessageBox) Update() (err error) {
	b.okButton.Update()
	return nil
}

func (b *MessageBox) SetText(value string) {
	b.messageLabel.SetLabel(value)
}

func (b *MessageBox) Render(renderer *sdl.Renderer) (err error) {
	renderer.SetDrawColor(b.bg.R, b.bg.G, b.bg.B, b.bg.A)
	renderer.FillRect(&b.rect)
	renderer.SetDrawColor(b.fg.R, b.fg.G, b.fg.B, b.fg.A)
	renderer.DrawRect(&sdl.Rect{b.rect.X, b.rect.Y, b.rect.W, 20})
	renderer.DrawRect(&sdl.Rect{b.rect.X, b.rect.Y, b.rect.W, b.rect.H})
	b.titleLabel.Render(renderer)
	b.messageLabel.Render(renderer)
	b.okButton.Render(renderer)
	return nil
}

func (b *MessageBox) Event(event sdl.Event) (pressed bool) {
	pressed = false
	switch event.(type) {
	case *sdl.MouseButtonEvent:
		if ok := b.okButton.Event(event); ok == MouseButtonLeftReleased && !b.Hide {
			pressed = true
		}
	}
	return pressed
}

func (b *MessageBox) Destroy() {
	b.titleLabel.Destroy()
	b.messageLabel.Destroy()
	b.okButton.Destroy()
}

/*
.oo
    .P 8
   .P  8 oPYo. oPYo. .oPYo. o   o   o .oPYo.
  oPooo8 8  `' 8  `' 8    8 Y. .P. .P Yb..
 .P    8 8     8     8    8 `b.d'b.d'   'Yb.
.P     8 8     8     `YooP'  `Y' `Y'  `YooP'
..:::::....::::..:::::.....:::..::..:::.....:
:::::::::::::::::::::::::::::::::::::::::::::
:::::::::::::::::::::::::::::::::::::::::::::*/
func (t *Arrow) New(rect sdl.Rect, text string, fgColor, bgColor sdl.Color, fontSize int32) (err error) {
	t.rect = rect
	t.text = text
	t.fgColor = fgColor
	t.bgColor = bgColor
	t.buttons = []buttonsData{
		{name: buttonDec, rect: sdl.Rect{t.rect.X, t.rect.Y, t.rect.H, t.rect.H}, text: "<", event: []Event{DecButtonEvent}},
		{name: label, rect: sdl.Rect{t.rect.X + t.rect.H, t.rect.Y, t.rect.W / t.rect.H, t.rect.H}, text: t.text, event: []Event{NilEvent}},
		{name: buttonInc, rect: sdl.Rect{t.rect.X + t.rect.H*(t.rect.W/t.rect.H), t.rect.Y, t.rect.H, t.rect.H}, text: ">", event: []Event{IncButtonEvent}}}
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

func (s *Arrow) GetNumber() (value []int) {
	var num, percent string
	arr := strings.Split(s.GetLabel(), ":")
	num = arr[1]
	if len(arr) > 2 {
		percent = arr[3]
	} else {
		percent = "0"
	}
	valueNum, err := strconv.Atoi(num)
	if err != nil {
		panic(err)
	}
	valuePerc, err := strconv.Atoi(percent)
	if err != nil {
		panic(err)
	}
	value = append(value, valueNum, valuePerc)
	return value
}

func (s *Arrow) SetNumber(value []int) {
	arr := strings.Split(s.GetLabel(), ":")
	text := arr[0]
	text += ":" + strconv.Itoa(value[0])
	if len(arr) > 2 {
		text += ":%:" + strconv.Itoa(value[1])
	}
	s.SetLabel(text)
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
							return DecButtonEvent, nil
						case IncButtonEvent:
							return IncButtonEvent, nil
						}
					}
				}
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

func (s *Arrow) Destroy() {
	for i := range s.btnInstances {
		switch s.btnInstances[i].(type) {
		case *Button:
			s.btnInstances[i].(*Button).Destroy()
		case *Label:
			s.btnInstances[i].(*Label).Destroy()
		}
	}
}

/*
.oPYo.   o           o                o      o
8        8           8                8
`Yooo.  o8P .oPYo.  o8P o    o .oPYo. 8     o8 odYo. .oPYo.
    `8   8  .oooo8   8  8    8 Yb..   8      8 8' `8 8oooo8
     8   8  8    8   8  8    8   'Yb. 8      8 8   8 8.
`YooP'   8  `YooP8   8  `YooP' `YooP' 8oooo  8 8   8 `Yooo'
:.....:::..::.....:::..::.....::.....:......:....::..:.....:
::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::
::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::*/
func (s *StatusLine) New(b boardConfig) {
	s.gameBoardSize = b
	s.Setup()
}
func (s *StatusLine) Setup() (err error) {
	if len(s.btnInstances) > 0 {
		s.Destroy()
		s.btnInstances = nil
	}
	s.buttons = []buttonsData{
		{name: buttonQuit, rect: sdl.Rect{0, 0, StatusLineHeight, StatusLineHeight}, text: "<-", event: []Event{QuitEvent}},
		{name: buttonPause, rect: sdl.Rect{StatusLineHeight, 0, StatusLineHeight * 3, StatusLineHeight}, text: "Pause", event: []Event{PauseEvent}},
		{name: buttonReset, rect: sdl.Rect{StatusLineHeight * 4, 0, StatusLineHeight * 3, StatusLineHeight}, text: "Reset", event: []Event{ResetGameEvent}},
		{name: buttonNew, rect: sdl.Rect{StatusLineHeight * 7, 0, StatusLineHeight * 2, StatusLineHeight}, text: "New", event: []Event{NewGameEvent}},
		{name: buttonRow, rect: sdl.Rect{StatusLineHeight * 9, 0, StatusLineHeight * 5, StatusLineHeight}, text: "Rows:" + strconv.Itoa(int(s.gameBoardSize.row)), event: []Event{IncRowEvent, DecRowEvent}},
		{name: buttonCol, rect: sdl.Rect{StatusLineHeight * 15, 0, StatusLineHeight * 6, StatusLineHeight}, text: "Columns:" + strconv.Itoa(int(s.gameBoardSize.column)), event: []Event{IncRowEvent, DecRowEvent}},
		{name: buttonMines, rect: sdl.Rect{StatusLineHeight * 22, 0, StatusLineHeight * 7, StatusLineHeight}, text: "Mines:" + strconv.Itoa(int(s.gameBoardSize.mines)) + ":%:" + strconv.Itoa(int(s.gameBoardSize.minesPercent)), event: []Event{IncRowEvent, DecRowEvent}}}
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

func (s *StatusLine) GetGameBoardSize() boardConfig {
	return s.gameBoardSize
}

func (s *StatusLine) Update(event Event) error {
	switch event {
	case NewGameEvent:
		fmt.Printf("start new game:%v", s.gameBoardSize)
	case IncRowEvent: // Replace game board size by arrows
		s.gameBoardSize.row = int32(s.btnInstances[4].(*Arrow).GetNumber()[0])
		s.gameBoardSize.minesPercent = s.gameBoardSize.mines * 100 / (s.gameBoardSize.row * s.gameBoardSize.column)
	case DecRowEvent:
		s.gameBoardSize.row = int32(s.btnInstances[4].(*Arrow).GetNumber()[0])
		s.gameBoardSize.minesPercent = s.gameBoardSize.mines * 100 / (s.gameBoardSize.row * s.gameBoardSize.column)
	case IncColumnEvent:
		s.gameBoardSize.column = int32(s.btnInstances[5].(*Arrow).GetNumber()[0])
		s.gameBoardSize.minesPercent = s.gameBoardSize.mines * 100 / (s.gameBoardSize.row * s.gameBoardSize.column)
	case DecColumnEvent:
		s.gameBoardSize.column = int32(s.btnInstances[5].(*Arrow).GetNumber()[0])
		s.gameBoardSize.minesPercent = s.gameBoardSize.mines * 100 / (s.gameBoardSize.row * s.gameBoardSize.column)
	case IncMinesEvent:
		s.gameBoardSize.mines = int32(s.btnInstances[6].(*Arrow).GetNumber()[0])
		s.gameBoardSize.minesPercent = s.gameBoardSize.mines * 100 / (s.gameBoardSize.row * s.gameBoardSize.column)
	case DecMinesEvent:
		s.gameBoardSize.mines = int32(s.btnInstances[6].(*Arrow).GetNumber()[0])
		s.gameBoardSize.minesPercent = s.gameBoardSize.mines * 100 / (s.gameBoardSize.row * s.gameBoardSize.column)
	case WindowResized:
		StatusLineHeight = WinHeight / 20
		StatusLineFontSize = StatusLineHeight - 3
		s.Setup()
	}
	for idx, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			s.btnInstances[idx].(*Button).Update()
			if s.buttons[idx].name == buttonNew {
			}
		case *Arrow:
			s.btnInstances[idx].(*Arrow).Update(event)
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
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "inc")
							return IncRowEvent, nil
						case DecButtonEvent:
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "dec")
							return DecRowEvent, nil
						}
					case buttonCol:
						switch ev {
						case IncButtonEvent:
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "inc")
							return IncColumnEvent, nil
						case DecButtonEvent:
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "dec")
							return DecColumnEvent, nil
						}
					case buttonMines:
						switch ev {
						case IncButtonEvent:
							s.calc(s.buttons[idx].name, s.btnInstances[idx].(*Arrow), "inc")
							return IncMinesEvent, nil
						case DecButtonEvent:
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
	n := instance.GetNumber()
	switch op {
	case "inc":
		switch name {
		case buttonRow:
			if n[0] < maxRow {
				n[0]++
				n[1] = int(s.gameBoardSize.minesPercent)
			}
		case buttonCol:
			if n[0] < maxColumn {
				n[0]++
				n[1] = int(s.gameBoardSize.minesPercent)
			}
		case buttonMines:
			if n[0] < maxMines {
				n[0]++
				n[1] = int(s.gameBoardSize.minesPercent)
			}
		}
	case "dec":
		switch name {
		case buttonRow:
			if n[0] > minRow {
				n[0]--
				n[1] = int(s.gameBoardSize.minesPercent)
			}
		case buttonCol:
			if n[0] > minColumn {
				n[0]--
				n[1] = int(s.gameBoardSize.minesPercent)
			}
		case buttonMines:
			if n[0] > minMines {
				n[0]--
				n[1] = int(s.gameBoardSize.minesPercent)
			}
		}
	}
	instance.SetNumber(n)
	m := s.btnInstances[6].(*Arrow).GetNumber()
	m[1] = n[1]
	s.btnInstances[6].(*Arrow).SetNumber(m)
}

func (s *StatusLine) Destroy() {
	for _, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			button.(*Button).Destroy()
		case *Arrow:
			button.(*Arrow).Destroy()
		case *Label:
			button.(*Label).Destroy()
		}

	}
}

/*
.oPYo.                        .oPYo.                          8
8    8                        8   `8                          8
8      .oPYo. ooYoYo. .oPYo. o8YooP' .oPYo. .oPYo. oPYo. .oPYo8
8   oo .oooo8 8' 8  8 8oooo8  8   `b 8    8 .oooo8 8  `' 8    8
8    8 8    8 8  8  8 8.      8    8 8    8 8    8 8     8    8
`YooP8 `YooP8 8  8  8 `Yooo'  8oooP' `YooP' `YooP8 8     `YooP'
:....8 :.....:..:..:..:.....::......::.....::.....:..:::::.....:
:::::8 :::::::::::::::::::::::::::::::::::::::::::::::::::::::::
:::::..:::::::::::::::::::::::::::::::::::::::::::::::::::::::::*/
func (s *GameBoard) New(b boardConfig, start bool) (err error) {
	s.start = start
	s.gameBoardSize = b
	s.colors = []sdl.Color{sdl.Color{192, 192, 192, 255}, sdl.Color{0, 0, 255, 255}, sdl.Color{0, 128, 0, 255}, sdl.Color{255, 0, 0, 255}, sdl.Color{0, 0, 128, 255}, sdl.Color{128, 0, 0, 255}, sdl.Color{0, 128, 128, 255}, sdl.Color{0, 0, 0, 255}, sdl.Color{128, 128, 128, 255}}
	s.Setup()
	return nil
}

func (s *GameBoard) Setup() (err error) {
	var (
		x, y, w, h, dx, dy, idx int32
		board                   []string
	)
	w, h = int32(float64(WinHeight)/1.1), int32(float64(WinHeight)/1.1)
	x, y = (WinHeight-w)/2, (WinHeight-h)/2+StatusLineHeight/2
	s.rect = sdl.Rect{x, y, w, h}
	s.cellWidth, s.cellHeight = w/s.gameBoardSize.row, (h-StatusLineHeight*2)/s.gameBoardSize.column
	cellFontSize := s.cellHeight - 3
	if len(s.btnInstances) > 0 {
		fmt.Printf("clear: %v\n", s.btnInstances)
		s.Destroy()
		tail := len(s.btnInstances) - 1
		for i := 0; i <= tail; i++ {
			switch s.btnInstances[i].(type) {
			case *Button:
				board = append(board, s.btnInstances[i].(*Button).GetLabel())
			}
			s.btnInstances = append(s.btnInstances[:i], s.btnInstances[i+1:tail+1]...)
			tail--
			i--
		}
		fmt.Printf("clear: %v\n", s.btnInstances, board)
	}

	for dy = 0; dy < s.gameBoardSize.column; dy++ {
		for dx = 0; dx < s.gameBoardSize.row; dx++ {
			x = s.rect.X + dx*s.cellWidth
			y = s.rect.Y + dy*s.cellHeight
			w = s.cellWidth
			h = s.cellHeight
			b := &Button{}
			text := " "
			if len(board) > 0 && !s.start {
				fmt.Println("len:", len(board), idx)
				text = board[idx]
			}
			if err := b.New(sdl.Rect{x, y, w, h}, text, s.colors[7], s.colors[8], cellFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, b)
			idx++
		}
	}
	board = nil

	s.messageBox = &MessageBox{}
	if err = s.messageBox.New(sdl.Rect{WinWidth/2 - 300/2, WinHeight/2 - 150/2, 300, 150}, "Message", "Test Message", s.colors[1], s.colors[8]); err != nil {
		panic(err)
	}
	s.messageBox.Hide = true
	s.btnInstances = append(s.btnInstances, s.messageBox)

	arr := []string{"M00/F00", "00:00"}
	for dx = 0; dx < int32(len(arr)); dx++ {
		w = (s.rect.H / int32((len(arr) + 1)))
		x = s.rect.X + dx*w + w
		y = s.rect.H - StatusLineHeight/2
		lbl := &Label{}
		if err = lbl.New(sdl.Point{x, y}, arr[dx], s.colors[1], StatusLineFontSize); err != nil {
			panic(err)
		}
		s.btnInstances = append(s.btnInstances, lbl)
	}
	s.start = false
	return nil
}

func (s *GameBoard) SetBoard(board []int32) {
	for idx, button := range s.btnInstances {
		// fmt.Println("set", idx, button, board[idx])
		switch button.(type) {
		case *Button:
			switch board[idx] {
			case 0:
				s.btnInstances[idx].(*Button).SetLabel(" ")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[8])
			case 1:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[1])
			case 2:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[2])
			case 3:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[3])
			case 4:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[4])
			case 5:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[5])
			case 6:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[6])
			case 7:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			case 8:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[8])
			case mined:
				s.btnInstances[idx].(*Button).SetLabel("*")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[8])
			case firstMined:
				s.btnInstances[idx].(*Button).SetLabel("*")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[3])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[8])
			case closed:
				s.btnInstances[idx].(*Button).SetLabel(" ")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[8])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			case flagged:
				s.btnInstances[idx].(*Button).SetLabel("F")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			case questionable:
				s.btnInstances[idx].(*Button).SetLabel("?")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			case saved:
				s.btnInstances[idx].(*Button).SetLabel("V")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			case blown:
				s.btnInstances[idx].(*Button).SetLabel("b")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			case wrongMines:
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			}
		case *MessageBox:
			switch board[idx] {
			case play:
				fmt.Println("play", idx)
				s.btnInstances[idx].(*MessageBox).Hide = true
			case won:
				s.btnInstances[idx].(*MessageBox).SetText("You Win")
				s.btnInstances[idx].(*MessageBox).Hide = false
				fmt.Println("win", idx)
			case lost:
				s.btnInstances[idx].(*MessageBox).SetText("Game Over")
				s.btnInstances[idx].(*MessageBox).Hide = false
				fmt.Println("game over", idx)
			}
		}
	}
}

func (s *GameBoard) Update(event Event) error {
	if event == WindowResized {
		fmt.Println("resize gameBoard")
		s.Setup()
	} else if event == NewGameEvent {

	}
	for idx, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			s.btnInstances[idx].(*Button).Update()
		case *MessageBox:
			s.btnInstances[idx].(*MessageBox).Update()
		}
	}
	return nil
}

func (s *GameBoard) Render(renderer *sdl.Renderer) (err error) {
	renderer.SetDrawColor(s.colors[0].R, s.colors[0].G, s.colors[0].B, s.colors[0].A)
	renderer.DrawRect(&s.rect)
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
		case *MessageBox:
			if !button.(*MessageBox).Hide {
				if err = button.(*MessageBox).Render(renderer); err != nil {
					panic(err)
				}
			}
		}
	}
	return nil
}

func (s *GameBoard) Event(event sdl.Event) (e Event, err error) {
	for idx, button := range s.btnInstances {
		switch t := event.(type) {
		case *sdl.MouseButtonEvent:
			switch button.(type) {
			case *Button:
				if ok := button.(*Button).Event(event); ok == MouseButtonLeftReleased && s.messageBox.Hide {
					s.mousePressedAtButton = int32(idx)
					return MouseButtonLeftReleasedEvent, nil
				}
				if ok := button.(*Button).Event(event); ok == MouseButtonRightReleased && s.messageBox.Hide {
					s.mousePressedAtButton = int32(idx)
					return MouseButtonRightReleasedEvent, nil
				}
			case *MessageBox:
				if ok := button.(*MessageBox).Event(event); ok {
					fmt.Printf("%v MessageBox ok released:%v %v %v\n", idx, t.X, t.Y, button)
					s.btnInstances[idx].(*MessageBox).Hide = true
				}
			}
		}
	}
	return NilEvent, nil
}

func (s *GameBoard) Destroy() {
	for _, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			button.(*Button).Destroy()
		case *Label:
			button.(*Label).Destroy()
		case *MessageBox:
			button.(*MessageBox).Destroy()
		}
	}
}

/*
.oPYo.        8 8
8    8        8 8
8      .oPYo. 8 8
8      8oooo8 8 8
8    8 8.     8 8
`YooP' `Yooo' 8 8
:.....::.....:....
::::::::::::::::::
::::::::::::::::::*/

func (s *Cell) New(pos sdl.Point) (err error) {
	s.pos = pos
	s.state = closed
	s.mined = false
	s.counter = -1
	return nil
}

func (s *Cell) GetState() int32 {
	return s.state
}
func (s *Cell) SetState(value int32) {
	s.state = value
}
func (s *Cell) GetMines() bool {
	return s.mined
}
func (s *Cell) GetMined() bool {
	return s.state == mined
}
func (s *Cell) SetMines() {
	s.mined = true
}
func (s *Cell) GetFirstMines() bool {
	return s.state == firstMined
}
func (s *Cell) SetFirstMines() {
	s.state = firstMined
}
func (s *Cell) GetSavedMines() bool {
	return s.state == saved
}
func (s *Cell) SetSavedMines() {
	s.state = saved
}
func (s *Cell) GetBlownMines() bool {
	return s.state == blown
}
func (s *Cell) SetBlownMines() {
	s.state = blown
}
func (s *Cell) GetWrongMines() bool {
	return s.state == wrongMines
}
func (s *Cell) SetWrongMines() {
	s.state = wrongMines
}
func (s *Cell) GetNumber() int32 {
	return s.counter
}
func (s *Cell) SetNumber(value int32) {
	s.counter = value
}
func (s *Cell) GetClosed() bool {
	return s.state == closed
}
func (s *Cell) SetClosed() {
	s.state = closed
}
func (s *Cell) GetOpened() bool {
	return s.state == opened
}
func (s *Cell) GetFlagged() bool {
	return s.state == flagged
}
func (s *Cell) SetFlagged() {
	s.state = flagged
}
func (s *Cell) GetQuestioned() bool {
	return s.state == questionable
}
func (s *Cell) SetQuestioned() {
	s.state = questionable
}

func (s *Cell) Open() {
	if s.state == closed || s.state == questionable {
		s.state = opened
	}
}

func (s *Cell) MarkFlag() {
	if s.state == closed {
		s.state = flagged
	} else if s.state == flagged {
		s.state = questionable
	} else if s.state == questionable {
		s.state = closed
	}
}

func (s *Cell) String() string {
	var state string
	switch s.state {
	case closed:
		state = "closed"
	case flagged:
		state = "flagged"
	case questionable:
		state = "questionable"
	case opened:
		state = "opened"
	case mined:
		state = "mined"
	case saved:
		state = "saved"
	case blown:
		state = "blown"
	case firstMined:
		state = "first mined"
	case empty:
		state = "empty"
	case wrongMines:
		state = "wrong mined"
	case marked:
		state = "marked"
	}
	return fmt.Sprintf("Cell x:%v y:%v state:%v count:%v mined:%v\n", s.pos.X, s.pos.Y, state, s.counter, s.mined)
}

/*
 ooooo  o        8      8
 8               8      8
o8oo   o8 .oPYo. 8 .oPYo8
 8      8 8oooo8 8 8    8
 8      8 8.     8 8    8
 8      8 `Yooo' 8 `YooP'
:..:::::..:.....:..:.....:
::::::::::::::::::::::::::
::::::::::::::::::::::::::*/
func (s *Field) New(boardSize boardConfig) (err error) {
	s.boardSize = boardSize
	if len(s.field) > 0 {
		s.field = nil
	}
	var column, row int32
	for column = 0; column < s.boardSize.column; column++ {
		for row = 0; row < s.boardSize.row; row++ {
			cell := Cell{}
			cell.New(sdl.Point{row, column})
			s.field = append(s.field, cell)
		}
	}
	s.state = gameStart
	return nil
}

func (s *Field) Setup(firstMoveIdx int32) {
	var mines, x, y int32
	firstMovePos, _ := s.getPosOfCell(firstMoveIdx)
	for mines < s.boardSize.mines {
		x, y = int32(rand.Intn(int(s.boardSize.row))), int32(rand.Intn(int(s.boardSize.column)))
		if x == firstMovePos.X && y == firstMovePos.Y {
			continue
		}
		_, cell := s.getIdxOfCell(x, y)
		if !cell.GetMines() {
			cell.SetMines()
			mines++
		}
	}
	for idx, cell := range s.field {
		var count int32
		if !cell.GetMines() {
			pos, _ := s.getPosOfCell(int32(idx))
			neighbours := s.getNeighbours(pos.X, pos.Y)
			for _, cell := range neighbours {
				if cell.GetMines() {
					count++
				}
			}
			s.field[idx].SetNumber(count)
		}
	}
	s.state = gamePlay
}

func (s *Field) isFieldEdge(x, y int32) bool {
	return x < 0 || x > s.boardSize.row-1 || y < 0 || y > s.boardSize.column-1
}

func (s *Field) getNeighbours(x, y int32) (cells []*Cell) {
	var dx, dy, nx, ny int32
	for dy = -1; dy < 2; dy++ {
		for dx = -1; dx < 2; dx++ {
			nx = x + dx
			ny = y + dy
			if !s.isFieldEdge(nx, ny) {
				_, newCell := s.getIdxOfCell(nx, ny)
				cells = append(cells, newCell)
			}
		}
	}
	return cells
}
func (s *Field) getIdxOfCell(x, y int32) (idx int32, cell *Cell) {
	if !s.isFieldEdge(x, y) {
		idx = y*s.boardSize.row + x
		cell = &s.field[idx]
		return idx, cell
	}
	return -1, nil
}
func (s *Field) getPosOfCell(idx int32) (pos sdl.Point, cell *Cell) {
	pos.X, pos.Y = idx%s.boardSize.row, idx/s.boardSize.row
	cell = &s.field[idx]
	return pos, cell
}

func (s *Field) Open(x, y int32) {
	if s.isFieldEdge(x, y) {
		return
	}
	_, cell := s.getIdxOfCell(x, y)
	if cell.GetFlagged() || cell.GetOpened() {
		return
	}
	cell.Open()
	if cell.GetMines() {
		cell.SetFirstMines()
		s.state = gameOver
		return
	}
	if cell.GetNumber() > 0 {
		return
	}
	for _, nCell := range s.getNeighbours(x, y) {
		s.Open(nCell.pos.X, nCell.pos.Y)
	}
}

func (s *Field) autoMarkFlags(x, y int32) {
	var countFlags, countClosed, countOpened int32
	_, cell := s.getIdxOfCell(x, y)
	if cell.GetOpened() {
		neighbours := s.getNeighbours(x, y)
		for _, cell := range neighbours {
			if cell.GetFlagged() {
				countFlags++
			} else if cell.GetClosed() {
				countClosed++
			} else if cell.GetOpened() {
				countOpened++
			}
		}
	}
	if countClosed+countFlags == cell.GetNumber() {
		for _, nCell := range s.getNeighbours(x, y) {
			if nCell.GetClosed() {
				nCell.SetFlagged()
			}
		}
	} else if countFlags == cell.GetNumber() {
		for _, nCell := range s.getNeighbours(x, y) {
			s.Open(nCell.pos.X, nCell.pos.Y)
		}
	}
}

func (s *Field) MarkFlag(idx int32) {
	pos, cell := s.getPosOfCell(idx)
	if s.isFieldEdge(pos.X, pos.Y) {
		return
	}
	cell.MarkFlag()
}

func (s *Field) isWin() bool {
	var count int32
	for _, cell := range s.field {
		if cell.GetOpened() {
			count++
		}
	}
	if count+s.boardSize.mines == s.boardSize.row*s.boardSize.column {
		for idx, cell := range s.field {
			if cell.GetMines() {
				s.field[idx].SetSavedMines()
			}
		}
		s.state = gameWin
		return true
	}
	return false
}

func (s *Field) isGameOver() bool {
	if s.state == gameOver {
		for idx, cell := range s.field[:] {
			if cell.GetMines() && cell.GetClosed() {
				s.field[idx].Open()
				s.field[idx].SetBlownMines()
			} else if cell.GetFlagged() && cell.GetMines() {
				s.field[idx].SetSavedMines()
			}
		}
	} else {
		return false
	}
	return true
}

func (s *Field) GetFieldValues() (board []int32) {
	for _, cell := range s.field {
		if cell.state == closed || cell.state == flagged || cell.state == questionable {
			board = append(board, cell.state)
		} else if cell.state >= opened {
			if cell.GetFirstMines() {
				board = append(board, firstMined)
			} else if cell.GetMined() {
				board = append(board, mined)
			} else if cell.GetSavedMines() {
				board = append(board, saved)
			} else if cell.GetBlownMines() {
				board = append(board, blown)
			} else {
				board = append(board, cell.counter)
			}
		}
	}
	if s.state == gameWin {
		board = append(board, won)
	} else if s.state == gameOver {
		board = append(board, lost)
	} else {
		board = append(board, play)
	}
	fmt.Println("send board:", board)
	return board
}

func (s *Field) String() string {
	var x, y int32
	board := ""
	for y = 0; y < s.boardSize.column; y++ {
		board += "\n"
		for x = 0; x < s.boardSize.row; x++ {
			_, cell := s.getIdxOfCell(x, y)
			if cell.counter >= 0 {
				board += fmt.Sprintf("%3v", cell.counter)
			} else if cell.mined {
				board += fmt.Sprintf("%3v", "*")
			}
		}
	}
	return board
}

/*
o     o  o
8b   d8
8`b d'8 o8 odYo. .oPYo. .oPYo.
8 `o' 8  8 8' `8 8oooo8 Yb..
8     8  8 8   8 8.       'Yb.
8     8  8 8   8 `Yooo' `YooP'
..::::..:....::..:.....::.....:
:::::::::::::::::::::::::::::::
:::::::::::::::::::::::::::::::*/
func (s *Mines) New(size boardConfig) {
	s.field = Field{}
	s.field.New(size)
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
		if err := subscriber.Update(e); err != nil {
			panic(err)
		}
	}
}

func (s Mines) GetSubscribers() []Observers {
	return s.subsribers
}

/*
o     o  o
8     8
8     8 o8 .oPYo. o   o   o
`b   d'  8 8oooo8 Y. .P. .P
 `b d'   8 8.     `b.d'b.d'
  `8'    8 `Yooo'  `Y' `Y'
:::..::::..:.....:::..::..::
::::::::::::::::::::::::::::
::::::::::::::::::::::::::::*/
func (s *View) Setup() (err error) {
	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	if s.window, err = sdl.CreateWindow("Mines", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, WinWidth, WinHeight, sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE); err != nil {
		panic(err)
	}
	if s.renderer, err = sdl.CreateRenderer(s.window, -1, sdl.RENDERER_ACCELERATED); err != nil {
		panic(err)
	}
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")
	s.pushTime = 100
	s.lastPushTime = sdl.GetTicks()
	s.flags = 0
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
			log.Printf("SEND Quit")
			return events
		case *sdl.KeyboardEvent:
			if t.Keysym.Sym == sdl.K_ESCAPE && t.State == sdl.RELEASED {
				events = append(events, QuitEvent)
				log.Printf("SEND Quit by escape")
				return events
			} else if t.Keysym.Sym == sdl.K_F11 && t.State == sdl.RELEASED {
				events = append(events, FullScreenToggleEvent)
				log.Printf("SEND window resize by F11")
				return events
			}
		case *sdl.WindowEvent:
			if t.Event == sdl.WINDOWEVENT_RESIZED {
				WinWidth, WinHeight = t.Data1, t.Data2
				events = append(events, WindowResized)
				log.Printf("SEND Window Resized")
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

/*
.oPYo.         o
8
`Yooo. .oPYo. o8 odYo. odYo. .oPYo. oPYo.
    `8 8    8  8 8' `8 8' `8 8oooo8 8  `'
     8 8    8  8 8   8 8   8 8.     8
`YooP' 8YooP'  8 8   8 8   8 `Yooo' 8
:.....:8 ....::....::....::..:.....:..::::
:::::::8 :::::::::::::::::::::::::::::::::
:::::::..:::::::::::::::::::::::::::::::::*/
func (s *Spinner) Run(m Mines, v View) {
	defaultSize := boardConfig{row: 5, column: 5, mines: 5}
	rand.Seed(time.Now().UTC().UnixNano())
	s.mines = m
	s.mines.New(defaultSize)
	if err := v.Setup(); err != nil {
		panic(err)
	}
	statusLine := &StatusLine{}
	statusLine.New(defaultSize)
	s.mines.Attach(statusLine)
	board := &GameBoard{}
	board.New(defaultSize, true)
	s.mines.Attach(board)
	dirty := true
	running := true
	for running {
		for _, event := range v.GetEvents(s.mines.GetSubscribers()) {
			switch event {
			case NewGameEvent:
				s.mines.field.New(statusLine.gameBoardSize)
				board.New(statusLine.gameBoardSize, true)
				s.mines.field.state = gameStart
			case MouseButtonLeftReleasedEvent:
				if s.mines.field.state == gameStart {
					s.mines.field.Setup(board.mousePressedAtButton)
					pos, cell := s.mines.field.getPosOfCell(board.mousePressedAtButton)
					if cell.GetClosed() {
						s.mines.field.Open(pos.X, pos.Y)
					}
					board.SetBoard(s.mines.field.GetFieldValues())
				} else if s.mines.field.state == gamePlay {
					pos, cell := s.mines.field.getPosOfCell(board.mousePressedAtButton)
					if cell.GetClosed() {
						s.mines.field.Open(pos.X, pos.Y)
					} else if cell.GetOpened() {
						s.mines.field.autoMarkFlags(pos.X, pos.Y)
					}
					s.mines.field.isWin()
					s.mines.field.isGameOver()
					board.SetBoard(s.mines.field.GetFieldValues())
				}
			case MouseButtonRightReleasedEvent:
				if s.mines.field.state == gamePlay {
					s.mines.field.MarkFlag(board.mousePressedAtButton)
					board.SetBoard(s.mines.field.GetFieldValues())
				}
				// board
			case IncRowEvent: // Replace game board size by arrows
				statusLine.gameBoardSize.row = int32(statusLine.btnInstances[4].(*Arrow).GetNumber()[0])
			case DecRowEvent:
				statusLine.gameBoardSize.row = int32(statusLine.btnInstances[4].(*Arrow).GetNumber()[0])
			case IncColumnEvent:
				statusLine.gameBoardSize.column = int32(statusLine.btnInstances[5].(*Arrow).GetNumber()[0])
			case DecColumnEvent:
				statusLine.gameBoardSize.column = int32(statusLine.btnInstances[5].(*Arrow).GetNumber()[0])
			case IncMinesEvent:
				statusLine.gameBoardSize.mines = int32(statusLine.btnInstances[6].(*Arrow).GetNumber()[0])
			case DecMinesEvent:
				statusLine.gameBoardSize.mines = int32(statusLine.btnInstances[6].(*Arrow).GetNumber()[0])
			case FullScreenToggleEvent:
				if v.flags == 0 {
					v.flags = sdl.WINDOW_FULLSCREEN_DESKTOP
				} else {
					v.flags = 0
				}
				v.window.SetFullscreen(v.flags)
				v.window.SetSize(WinWidth, WinHeight)
				log.Printf("GOT screen toggle")
			case WindowResized:
				log.Printf("GOT Resized")
			case QuitEvent:
				running = false
			case TickEvent:
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

/*
o     o         o
8b   d8
8`b d'8 .oPYo. o8 odYo.
8 `o' 8 .oooo8  8 8' `8
8     8 8    8  8 8   8
8     8 `YooP8  8 8   8
..::::..:.....::....::..
::::::::::::::::::::::::
::::::::::::::::::::::::*/
func main() {
	m := Mines{}
	v := View{}
	c := Spinner{}
	c.Run(m, v)
}
