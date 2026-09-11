package game

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/faiface/beep"
	"github.com/gdamore/tcell/v2"
)

const (
	screenWidth   = 80
	screenHeight  = 24
	mapWidth      = 78
	mapHeight     = 20
	messageHeight = 3
	saveFile      = "savegame.json"
	logFile       = "nethack.log"
	musicFile     = "music.mp3"
	saveVersion   = 4
)

type GameState int

const (
	StatePlaying     GameState = iota
	StateStartMenu
	StateQuitConfirm
	StateDeathMenu
	StateHelp
	StateMerchant
	StateVictory
	StatePopup
)

const (
	helpPageControls  = 0
	helpPageSymbols   = 1
	helpPageMechanics = 2
	helpPageCount     = 3
)

type Game struct {
	screen        tcell.Screen
	player        *Player
	level         *Level
	levels        map[int]*Level
	depth         int
	messages      []string
	quit          bool
	showInventory bool
	helpPage      int
	logger        *log.Logger
	logFileHandle *os.File
	state         GameState

	deathReason      string
	amuletFlashUntil time.Time

	currentMerchant       *Merchant
	pendingRelicSellIndex int

	popupMessage    string
	wasInCriticalHP bool
	wasTooFull      bool

	musicStreamer beep.StreamSeekCloser
	musicVolume   *linearVolume
	musicCtrl     *beep.Ctrl
	musicFileH    *os.File
	musicEnabled  bool
	musicLevel    float64
}

type blinkEvent struct {
	when time.Time
}

func (e *blinkEvent) When() time.Time { return e.when }
func stringWidth(s string) int        { return utf8.RuneCountInString(s) }

func NewGame() *Game {
	var f *os.File
	logger := log.New(os.Stderr, "", log.LstdFlags)
	fh, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Ошибка открытия файла логов:", err)
	} else {
		f = fh
		logger = log.New(f, "", log.LstdFlags)
	}
	logger.Println("=== Запуск игры ===")

	return &Game{
		messages:              make([]string, 0),
		depth:                 1,
		logger:                logger,
		logFileHandle:         f,
		state:                 StatePlaying,
		musicEnabled:          true,
		musicLevel:            0.3,
		pendingRelicSellIndex: -1,
		popupMessage:          "",
		wasInCriticalHP:       false,
		wasTooFull:            false,
		deathReason:           "",
		amuletFlashUntil:      time.Time{},
	}
}

func (g *Game) logAndSync(format string, v ...interface{}) {
	if g.logger != nil { g.logger.Printf(format, v...) }
	if g.logFileHandle != nil { g.logFileHandle.Sync() }
}

func (g *Game) startBlinkTicker() func() {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	var once sync.Once
	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if g.screen != nil { _ = g.screen.PostEvent(&blinkEvent{when: time.Now()}) }
			case <-stopCh:
				return
			}
		}
	}()
	return func() {
		once.Do(func() { close(stopCh) })
		<-doneCh
	}
}

func (g *Game) Run() error {
	if g.logFileHandle != nil { defer g.logFileHandle.Close() }
	var err error
	g.screen, err = tcell.NewScreen()
	if err != nil { return fmt.Errorf("не удалось создать экран: %w", err) }
	if err := g.screen.Init(); err != nil { return fmt.Errorf("не удалось инициализировать экран: %w", err) }
	defer g.screen.Fini()

	g.initMusic()
	defer g.stopMusic()
	stopBlink := g.startBlinkTicker()
	defer stopBlink()

	g.screen.SetStyle(tcell.StyleDefault)
	g.screen.Clear()
	g.state = StateStartMenu

	for !g.quit {
		func() {
			defer func() {
				if r := recover(); r != nil {
					g.logAndSync("CRITICAL ERROR (Recovered): %v", r)
					g.addMessage("КРИТИЧЕСКАЯ ОШИБКА! Смотрите лог.")
					g.quit = true
				}
			}()
			switch g.state {
			case StatePlaying:
				g.render()
				g.handleInput()
			case StateStartMenu:
				g.renderStartMenu()
				g.handleStartMenuInput()
			case StateQuitConfirm:
				g.renderQuitConfirm()
				g.handleQuitConfirmInput()
			case StateDeathMenu:
				g.renderDeathScreen()
				g.handleDeathInput()
			case StateHelp:
				g.renderHelpScreen()
				g.handleHelpInput()
			case StateMerchant:
				g.renderMerchantScreen()
				g.handleMerchantInput()
			case StateVictory:
				g.renderVictoryScreen()
				g.handleVictoryInput()
			}
		}()
	}
	return nil
}