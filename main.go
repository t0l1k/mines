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
		row, column, mines int32
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
func (s *StatusLine) New(b boardConfig) {
	s.gameBoardSize = b
	s.Setup()
}
func (s *StatusLine) Setup() (err error) {
	s.buttons = []buttonsData{
		{name: buttonQuit, rect: sdl.Rect{0, 0, StatusLineHeight, StatusLineHeight}, text: "<-", event: []Event{QuitEvent}},
		{name: buttonPause, rect: sdl.Rect{StatusLineHeight, 0, StatusLineHeight * 3, StatusLineHeight}, text: "Pause", event: []Event{PauseEvent}},
		{name: buttonReset, rect: sdl.Rect{StatusLineHeight * 4, 0, StatusLineHeight * 3, StatusLineHeight}, text: "Reset", event: []Event{ResetGameEvent}},
		{name: buttonNew, rect: sdl.Rect{StatusLineHeight * 7, 0, StatusLineHeight * 2, StatusLineHeight}, text: "New", event: []Event{NewGameEvent}},
		{name: buttonRow, rect: sdl.Rect{StatusLineHeight * 9, 0, StatusLineHeight * 6, StatusLineHeight}, text: "Rows:" + strconv.Itoa(int(s.gameBoardSize.row)), event: []Event{IncRowEvent, DecRowEvent}},
		{name: buttonCol, rect: sdl.Rect{StatusLineHeight * 16, 0, StatusLineHeight * 6, StatusLineHeight}, text: "Columns:" + strconv.Itoa(int(s.gameBoardSize.column)), event: []Event{IncRowEvent, DecRowEvent}},
		{name: buttonMines, rect: sdl.Rect{StatusLineHeight * 23, 0, StatusLineHeight * 6, StatusLineHeight}, text: "Mines:" + strconv.Itoa(int(s.gameBoardSize.mines)), event: []Event{IncRowEvent, DecRowEvent}}}
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
		s.gameBoardSize.row = int32(s.btnInstances[4].(*Arrow).GetNumber())
	case DecRowEvent:
		s.gameBoardSize.row = int32(s.btnInstances[4].(*Arrow).GetNumber())
	case IncColumnEvent:
		s.gameBoardSize.column = int32(s.btnInstances[5].(*Arrow).GetNumber())
	case DecColumnEvent:
		s.gameBoardSize.column = int32(s.btnInstances[5].(*Arrow).GetNumber())
	case IncMinesEvent:
		s.gameBoardSize.mines = int32(s.btnInstances[6].(*Arrow).GetNumber())
	case DecMinesEvent:
		s.gameBoardSize.mines = int32(s.btnInstances[6].(*Arrow).GetNumber())
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
	s.gameBoardSize = b
	s.Setup()
	return nil
}

func (s *GameBoard) Setup() (err error) {
	var x, y, w, h, dx, dy int32
	s.colors = []sdl.Color{sdl.Color{192, 192, 192, 255}, sdl.Color{0, 0, 255, 255}, sdl.Color{0, 128, 0, 255}, sdl.Color{255, 0, 0, 255}, sdl.Color{0, 0, 128, 255}, sdl.Color{128, 0, 0, 255}, sdl.Color{0, 128, 128, 255}, sdl.Color{0, 0, 0, 255}, sdl.Color{128, 128, 128, 255}}
	w, h = int32(float64(WinHeight)/1.1), int32(float64(WinHeight)/1.1)
	x, y = (WinHeight-w)/2, (WinHeight-h)/2+StatusLineHeight/2
	s.rect = sdl.Rect{x, y, w, h}
	s.cellWidth, s.cellHeight = w/s.gameBoardSize.row, (h-StatusLineHeight*2)/s.gameBoardSize.column
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
			if err := b.New(sdl.Rect{x, y, w, h}, " ", s.colors[7], s.colors[8], cellFontSize); err != nil {
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

func (s *GameBoard) SetBoard(board []int32) {
	for idx, button := range s.btnInstances {
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
				if ok := button.(*Button).Event(event); ok == MouseButtonRightReleased {
					fmt.Printf("%v Right Released At:%v %v %v\n", idx, t.X, t.Y, button)
					s.mousePressedAtButton = int32(idx)
					return MouseButtonRightReleasedEvent, nil
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
		fmt.Println("opened", s)
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
			fmt.Println("Init Field:", cell, row, column)
		}
	}
	s.state = gameStart
	fmt.Println(s)
	return nil
}

func (s *Field) Setup(firstMoveIdx int32) {
	var mines, x, y int32
	firstMovePos, _ := s.getPosOfCell(firstMoveIdx)
	for mines < s.boardSize.mines {
		x, y = int32(rand.Intn(int(s.boardSize.row))), int32(rand.Intn(int(s.boardSize.column)))
		if x == firstMovePos.X && y == firstMovePos.Y {
			fmt.Println("get rand again")
			continue
		}
		_, cell, err := s.getIdxOfCell(x, y)
		if err != nil {
			panic(err)
		}
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
	fmt.Println(s)
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
				_, newCell, err := s.getIdxOfCell(nx, ny)
				if err != nil {
					panic(err)
				}
				cells = append(cells, newCell)
				fmt.Printf("x:%v,y:%v,nx:%v,ny:%v,cells:%v\n", x, y, nx, ny, cells)
			}
		}
	}
	fmt.Println("cells:", cells)
	return cells
}
func (s *Field) getIdxOfCell(x, y int32) (idx int32, cell *Cell, err error) {
	if !s.isFieldEdge(x, y) {
		idx = y*s.boardSize.row + x
		cell = &s.field[idx]
		return idx, cell, nil
	}
	return -1, nil, fmt.Errorf("getIdxOfCell:get wrong index x:%v,y:%v,err:%v", x, y, err)
}
func (s *Field) getPosOfCell(idx int32) (pos sdl.Point, cell *Cell) {
	pos.X, pos.Y = idx%s.boardSize.row, idx/s.boardSize.column
	cell = &s.field[idx]
	return pos, cell
}

func (s *Field) Open(x, y int32) {
	if s.isFieldEdge(x, y) {
		fmt.Printf("Open Field Cell Edge x:%v,y:%v", x, y)
		return
	}
	_, cell, err := s.getIdxOfCell(x, y)
	if err != nil {
		panic(err)
	}
	if cell.GetFlagged() || cell.GetOpened() {
		fmt.Printf("Opened or Flagged Field x:%v,y:%v,cell:%v", x, y, cell)
		// s.autoMarkFlags(x,y)
		return
	}
	cell.Open()
	fmt.Printf("Open Field Cell Open x:%v,y:%v,cell:%v", x, y, cell)
	if cell.GetMines() {
		cell.SetFirstMines()
		s.state = gameOver
		fmt.Printf("Open Field Cell Mined x:%v,y:%v,cell:%v", x, y, cell)
		return
	}
	if cell.GetNumber() > 0 {
		fmt.Printf("Open Field Cell Number x:%v,y:%v,cell:%v", x, y, cell)
		return
	}
	for _, nCell := range s.getNeighbours(x, y) {
		s.Open(nCell.pos.X, nCell.pos.Y)
	}
}

func (s *Field) autoMarkFlags(x, y int32) {
	var countFlags, countClosed, countOpened int32
	_, cell, err := s.getIdxOfCell(x, y)
	if err != nil {
		panic(err)
	}
	fmt.Println("begin auto mark")
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
	fmt.Printf("Get Closed:%v Opened:%v Flagged:%v\n\n", countClosed, countOpened, countFlags)
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

func (s *Field) Mark(idx int32) {
	pos, cell := s.getPosOfCell(idx)
	if s.isFieldEdge(pos.X, pos.Y) {
		return
	}
	cell.Mark()
}

func (s *Field) isWin() bool {
	var count int32
	for _, cell := range s.field {
		if cell.GetOpened() {
			count++
		}
	}
	if count+s.boardSize.mines == s.boardSize.row*s.boardSize.column {
		for _, cell := range s.field {
			if cell.GetMines() {
				cell.SetSavedMines()
			}
		}
		fmt.Println("Game Winned", s)
		s.state = gameWin
		return true
	}
	return false
}

func (s *Field) isGameOver() bool {
	if s.state == gameOver {
		for idx, cell := range s.field {
			if cell.GetMines() {
				s.field[idx].Open()
				if cell.GetFlagged() {
					s.field[idx].SetSavedMines()
				} else {
					s.field[idx].SetBlownMines()
				}
			}
		}
	} else {
		return false
	}
	fmt.Println("Game Over", s)
	return true
}

func (s *Field) GetFieldValues() (board []int32) {
	for _, cell := range s.field {
		if cell.state == closed || cell.state == flagged || cell.state == questionable {
			board = append(board, cell.state)
			// fmt.Println("cell state", cell.state, cell)
		} else if cell.state >= opened {
			if cell.GetFirstMines() {
				board = append(board, firstMined)
			} else if cell.GetMined() {
				board = append(board, mined)
			} else if cell.GetSavedMines() {
				board = append(board, mined)
			} else if cell.GetBlownMines() {
				board = append(board, mined)
			} else {
				board = append(board, cell.counter)
			}
		}
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
			_, cell, err := s.getIdxOfCell(x, y)
			if err != nil {
				panic(err)
			}
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
				fmt.Printf("start new game:%v", statusLine.gameBoardSize)
				s.mines.field.New(statusLine.gameBoardSize)
				board.New(statusLine.gameBoardSize)
				firstMove = true
			case MouseButtonLeftReleasedEvent:
				if firstMove {
					s.mines.field.Setup(board.mousePressedAtButton)
					pos, cell := s.mines.field.getPosOfCell(board.mousePressedAtButton)
					if cell.GetClosed() {
						s.mines.field.Open(pos.X, pos.Y)
					}
					board.SetBoard(s.mines.field.GetFieldValues())
					firstMove = false
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
					s.mines.field.Mark(board.mousePressedAtButton)
					board.SetBoard(s.mines.field.GetFieldValues())
				}
				// board
			case IncRowEvent: // Replace game board size by arrows
				statusLine.gameBoardSize.row = int32(statusLine.btnInstances[4].(*Arrow).GetNumber())
			case DecRowEvent:
				statusLine.gameBoardSize.row = int32(statusLine.btnInstances[4].(*Arrow).GetNumber())
			case IncColumnEvent:
				statusLine.gameBoardSize.column = int32(statusLine.btnInstances[5].(*Arrow).GetNumber())
			case DecColumnEvent:
				statusLine.gameBoardSize.column = int32(statusLine.btnInstances[5].(*Arrow).GetNumber())
			case IncMinesEvent:
				statusLine.gameBoardSize.mines = int32(statusLine.btnInstances[6].(*Arrow).GetNumber())
			case DecMinesEvent:
				statusLine.gameBoardSize.mines = int32(statusLine.btnInstances[6].(*Arrow).GetNumber())
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
