package main

import (
	"fmt"
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
	// Ячейка поля
	cellStateType int32
	Cell          struct {
		pos   sdl.Point
		state cellStateType
		cell  int8
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
	}
	// Наблюдатель строка меню
	StatusLine struct {
		buttons      []buttonsData
		btnInstances []interface{}
		newGameBoard boardConfig
	}
	// Наблюдатель поле игры
	GameBoard struct {
		rect                  sdl.Rect
		btnInstances          []interface{}
		colors                []sdl.Color
		gameBoardSize         boardConfig
		cellWidth, cellHeight int32
		mousePressedAtButton  int32
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
		row, column, mines, high, low int32
	}
	// UI для sdl2
	// Метка умеет выводить текст
	Label struct {
		pos      sdl.Point
		font     *ttf.Font
		fontSize int
		text     string
		color    sdl.Color
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
)

// Перечень событий
const (
	NilEvent Event = iota
	TickEvent
	QuitEvent
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

// состояния ячеек
const (
	closed cellStateType = iota
	flagged
	questionable
	opened
	mined
	saved
	firstMined
	empty
	wrongMines
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

// константы размеров поля
const (
	minRow    = 5
	maxRow    = 30
	minColumn = 5
	maxColumn = 16
	minMines  = 5
	maxMines  = 40
)

var (
	mn                   int32 = 3
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

func (t *Button) Quit() {
	t.font.Close()
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

func (s *Arrow) GetNumber() (value int) {
	arr := strings.Split(s.GetLabel(), ":")
	num := arr[1]
	value, err := strconv.Atoi(num)
	if err != nil {
		panic(err)
	}
	return value
}

func (s *Arrow) SetNumber(value int) {
	arr := strings.Split(s.GetLabel(), ":")
	text := arr[0]
	text += ":" + strconv.Itoa(value)
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
func (s *StatusLine) Setup() (err error) {
	s.newGameBoard = boardConfig{row: 8, column: 8, mines: 10}
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

func (s *StatusLine) GetGameBoardSize() boardConfig {
	return s.newGameBoard
}

func (s *StatusLine) Update(event Event) error {
	switch event {
	case NewGameEvent:
		fmt.Printf("start new game:%v", s.newGameBoard)
	case IncRowEvent: // Replace game board size by arrows
		s.newGameBoard.row = int32(s.btnInstances[4].(*Arrow).GetNumber())
	case DecRowEvent:
		s.newGameBoard.row = int32(s.btnInstances[4].(*Arrow).GetNumber())
	case IncColumnEvent:
		s.newGameBoard.column = int32(s.btnInstances[5].(*Arrow).GetNumber())
	case DecColumnEvent:
		s.newGameBoard.column = int32(s.btnInstances[5].(*Arrow).GetNumber())
	case IncMinesEvent:
		s.newGameBoard.mines = int32(s.btnInstances[6].(*Arrow).GetNumber())
	case DecMinesEvent:
		s.newGameBoard.mines = int32(s.btnInstances[6].(*Arrow).GetNumber())
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
			if n < maxRow {
				n++
			}
		case buttonCol:
			if n < maxColumn {
				n++
			}
		case buttonMines:
			if n < maxMines {
				n++
			}
		}
	case "dec":
		switch name {
		case buttonRow:
			if n > minRow {
				n--
			}
		case buttonCol:
			if n > minColumn {
				n--
			}
		case buttonMines:
			if n > minMines {
				n--
			}
		}
	}
	instance.SetNumber(n)
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
func (s *GameBoard) New(b boardConfig) (err error) {
	s.gameBoardSize = boardConfig{row: b.row, column: b.column, mines: b.mines}
	s.Setup()
	return nil
}

func (s *GameBoard) Setup() (err error) {
	var x, y, w, h, dx, dy, boardColumn int32
	s.colors = []sdl.Color{sdl.Color{192, 192, 192, 255}, sdl.Color{0, 0, 255, 255}, sdl.Color{0, 128, 0, 255}, sdl.Color{255, 0, 0, 255}, sdl.Color{0, 0, 128, 255}, sdl.Color{128, 0, 0, 255}, sdl.Color{0, 128, 128, 255}, sdl.Color{0, 0, 0, 255}, sdl.Color{128, 128, 128, 255}}
	w, h = int32(float64(WinHeight)/1.1), int32(float64(WinHeight)/1.1)
	x, y = (WinHeight-w)/2, (WinHeight-h)/2+StatusLineHeight/2
	s.rect = sdl.Rect{x, y, w, h}
	if s.gameBoardSize.row > s.gameBoardSize.column {
		boardColumn = s.gameBoardSize.row
	} else {
		boardColumn = s.gameBoardSize.column
	}
	s.cellWidth, s.cellHeight = w/boardColumn, (h-StatusLineHeight*2)/boardColumn
	cellFontSize := s.cellHeight - 3
	if len(s.btnInstances) > 0 {
		s.btnInstances = nil
	}
	for dy = 0; dy < s.gameBoardSize.column; dy++ {
		for dx = 0; dx < s.gameBoardSize.row; dx++ {
			x = s.rect.X + dx*s.cellWidth
			y = s.rect.Y + dy*s.cellHeight
			w = s.cellWidth
			h = s.cellHeight
			b := &Button{}
			if err := b.New(sdl.Rect{x, y, w, h}, " ", s.colors[7], s.colors[0], cellFontSize); err != nil {
				panic(err)
			}
			s.btnInstances = append(s.btnInstances, b)
		}
	}
	arr := []string{"M00/F00", "00:00"}
	for dx = 0; dx < int32(len(arr)); dx++ {
		w = (s.rect.H / int32((len(arr) + 1)))
		x = s.rect.X + dx*w + w
		y = s.rect.H - StatusLineHeight/2
		l := &Label{}
		if err = l.New(sdl.Point{x, y}, arr[dx], s.colors[1], StatusLineFontSize); err != nil {
			panic(err)
		}
		s.btnInstances = append(s.btnInstances, l)
	}
	return nil
}

func (s *GameBoard) SetBoard(board []int8) {
	for idx, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			if board[idx] == 0 {
				s.btnInstances[idx].(*Button).SetLabel(" ")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			} else if board[idx] == 1 {
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[1])
			} else if board[idx] == 2 {
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[2])
			} else if board[idx] == 3 {
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[3])
			} else if board[idx] == 4 {
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[4])
			} else if board[idx] == 5 {
				s.btnInstances[idx].(*Button).SetLabel(strconv.Itoa(int(board[idx])))
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[5])
			} else if board[idx] == -9 {
				s.btnInstances[idx].(*Button).SetLabel("*")
				s.btnInstances[idx].(*Button).SetBackground(s.colors[0])
				s.btnInstances[idx].(*Button).SetForeground(s.colors[7])
			}
		}
	}
}

func (s *GameBoard) Update(event Event) error {
	for idx, button := range s.btnInstances {
		switch button.(type) {
		case *Button:
			s.btnInstances[idx].(*Button).Update()
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
				if ok := button.(*Button).Event(event); ok == MouseButtonLeftReleased {
					fmt.Printf("%v Left Released At:%v %v %v\n", idx, t.X, t.Y, button)
					s.mousePressedAtButton = int32(idx)
					return MouseButtonLeftReleasedEvent, nil
				}
			}
		}
	}
	return NilEvent, nil
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
	s.cell = 0
	return nil
}

func (s *Cell) GetState() cellStateType {
	return s.state
}
func (s *Cell) SetState(value cellStateType) {
	s.state = value
}
func (s *Cell) GetCell() int8 {
	return s.cell
}
func (s *Cell) GetMines() bool {
	return s.cell == -9
}
func (s *Cell) SetMines() {
	s.cell = -9
}
func (s *Cell) GetFirstMines() bool {
	return s.cell == -10
}
func (s *Cell) SetFirstMines() {
	s.cell = -10
}
func (s *Cell) GetSavedMines() bool {
	return s.cell == -11
}
func (s *Cell) SetSavedMines() {
	s.cell = -11
}
func (s *Cell) GetBlownMines() bool {
	return s.cell == -12
}
func (s *Cell) SetBlownMines() {
	s.cell = -12
}
func (s *Cell) GetWrongMines() bool {
	return s.cell == -s.cell
}
func (s *Cell) SetWrongMines() {
	s.cell = -s.cell
}
func (s *Cell) GetNumber() int8 {
	return s.cell
}
func (s *Cell) SetNumber(value int8) {
	s.cell = value
	fmt.Println(s.cell)
}

func (s *Cell) Open() {
	if s.state == closed || s.state == questionable {
		s.state = opened
	}
}

func (s *Cell) Mark() {
	if s.state == closed {
		s.state = flagged
	} else if s.state == flagged {
		s.state = questionable
	} else if s.state == questionable {
		s.state = closed
	}
}

func (s *Cell) String() string {
	return fmt.Sprintf("Cell x:%v y:%v state:%v cell:%v\n", s.pos.X, s.pos.Y, s.state, s.cell)
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
	if s.boardSize.row > s.boardSize.column {
		s.boardSize.high = s.boardSize.row
		s.boardSize.low = s.boardSize.column
	} else {
		s.boardSize.high = s.boardSize.column
		s.boardSize.low = s.boardSize.row
	}
	if len(s.field) > 0 {
		s.field = nil
	}
	var column, row, x, y int32
	for column = 0; column < boardSize.column; column++ {
		for row = 0; row < boardSize.row; row++ {
			cell := Cell{}
			x = row % boardSize.row
			y = column / boardSize.column
			cell.New(sdl.Point{x, y})
			s.field = append(s.field, cell)
		}
	}
	return nil
}

func (s *Field) Setup(firstMoveIdx int32) {
	var mines, x, y int32
	firstMovePos, _ := s.getPos(firstMoveIdx)
	for mines < s.boardSize.mines {
		x, y = int32(rand.Intn(int(s.boardSize.row))), int32(rand.Intn(int(s.boardSize.column)))
		if x == firstMovePos.X && y == firstMovePos.Y {
			fmt.Println("get rand again")
			continue
		}
		_, cell := s.getIdxOfCell(x, y)
		if !cell.GetMines() {
			cell.SetMines()
			mines++
		}
	}
	for idx, cell := range s.field {
		var count int8
		if !cell.GetMines() {
			neighbours := s.getNeighbours(int32(idx))
			for _, cell := range neighbours {
				if cell.GetMines() {
					fmt.Println("count:", count)
					count++
				}
			}
			s.field[idx].SetNumber(count)
			fmt.Printf("s:%v,cell:%v,count:%v\n", s, cell, count)
		}
	}
}

func (s *Field) isEdge(x, y int32) bool {
	return x < 0 || x > s.boardSize.row-1 || y < 0 || y > s.boardSize.column-1
}

func (s *Field) getNeighbours(idx int32) (cells []*Cell) {
	var dx, dy, nx, ny int32
	pos, _ := s.getPos(idx)
	for dy = -1; dy < 2; dy++ {
		for dx = -1; dx < 2; dx++ {
			nx = pos.X + dx
			ny = pos.Y + dy
			if !s.isEdge(nx, ny) {
				// fmt.Printf("pos:%v,dx:%v,dy:%v,nx:%v,ny:%v\n", pos, dx, dy, nx, ny)
				_, newCell := s.getIdxOfCell(nx, ny)
				cells = append(cells, newCell)
			}
		}
	}
	fmt.Println(cells, len(cells))
	return cells
}
func (s *Field) getIdxOfCell(x, y int32) (idx int32, cell *Cell) {
	// fmt.Printf("x:%v,y:%v\n", x, y)
	idx = y*s.boardSize.row + x
	cell = &s.field[idx]
	return idx, cell
}
func (s *Field) getPos(idx int32) (pos sdl.Point, cell *Cell) {
	pos.X, pos.Y = idx%s.boardSize.row, idx/s.boardSize.column
	cell = &s.field[idx]
	return pos, cell
}

func (s *Field) GetFieldValues() (board []int8) {
	for _, cell := range s.field {
		board = append(board, cell.cell)
	}
	return board
}

func (s *Field) String() string {
	var x, y int32
	board := ""
	for y = 0; y < s.boardSize.column; y++ {
		board += "\n"
		for x = 0; x < s.boardSize.row; x++ {
			board += fmt.Sprintf("%2v", s.field[y*s.boardSize.row+x].cell)
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
	// fmt.Println(e)
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
	if s.window, err = sdl.CreateWindow("Mines", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, WinWidth, WinHeight, sdl.WINDOW_SHOWN); err != nil {
		panic(err)
	}
	if s.renderer, err = sdl.CreateRenderer(s.window, -1, sdl.RENDERER_ACCELERATED); err != nil {
		panic(err)
	}
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")
	s.pushTime = 100
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
	defaultSize := boardConfig{row: 2, column: 3, mines: 1}
	rand.Seed(time.Now().UTC().UnixNano())
	s.mines = Mines{}
	s.mines.New(defaultSize)
	if err := v.Setup(); err != nil {
		panic(err)
	}
	statusLine := &StatusLine{}
	statusLine.Setup()
	s.mines.Attach(statusLine)
	board := &GameBoard{}
	board.New(defaultSize)
	s.mines.Attach(board)
	firstMove := true
	dirty := true
	running := true
	for running {
		for _, event := range v.GetEvents(s.mines.GetSubscribers()) {
			switch event {
			case QuitEvent:
				running = false
			case TickEvent:
				dirty = true
			case NewGameEvent:
				fmt.Printf("start new game:%v", statusLine.newGameBoard)
				s.mines.field.New(statusLine.newGameBoard)
				board.New(statusLine.newGameBoard)
				firstMove = true
			case MouseButtonLeftReleasedEvent:
				if firstMove {
					s.mines.field.Setup(board.mousePressedAtButton)
					board.SetBoard(s.mines.field.GetFieldValues())
					firstMove = false
				}
				fmt.Println(board.mousePressedAtButton)
				// board
			case IncRowEvent: // Replace game board size by arrows
				statusLine.newGameBoard.row = int32(statusLine.btnInstances[4].(*Arrow).GetNumber())
			case DecRowEvent:
				statusLine.newGameBoard.row = int32(statusLine.btnInstances[4].(*Arrow).GetNumber())
			case IncColumnEvent:
				statusLine.newGameBoard.column = int32(statusLine.btnInstances[5].(*Arrow).GetNumber())
			case DecColumnEvent:
				statusLine.newGameBoard.column = int32(statusLine.btnInstances[5].(*Arrow).GetNumber())
			case IncMinesEvent:
				statusLine.newGameBoard.mines = int32(statusLine.btnInstances[6].(*Arrow).GetNumber())
			case DecMinesEvent:
				statusLine.newGameBoard.mines = int32(statusLine.btnInstances[6].(*Arrow).GetNumber())
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
