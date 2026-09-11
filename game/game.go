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

// =============================================================================
// КОНСТАНТЫ ИГРЫ
// =============================================================================
const (
	screenWidth   = 80              // ширина игрового экрана в символах
	screenHeight  = 24              // высота игрового экрана в символах
	mapWidth      = 78              // ширина карты (с запасом на боковые границы)
	mapHeight     = 20              // высота карты (с запасом на статус-бар и сообщения)
	messageHeight = 3               // количество строк для отображения сообщений
	saveFile      = "savegame.json" // файл сохранения игры
	logFile       = "nethack.log"   // файл логов для отладки
	musicFile     = "music.mp3"     // фоновая музыка
	saveVersion   = 4               // версия формата сохранения (для обратной совместимости)
)

// =============================================================================
// СОСТОЯНИЯ ИГРЫ
// =============================================================================
type GameState int

const (
	StatePlaying     GameState = iota // обычный игровой процесс
	StateStartMenu                    // стартовое меню
	StateQuitConfirm                  // подтверждение выхода
	StateDeathMenu                    // экран смерти игрока
	StateHelp                         // экран помощи
	StateMerchant                     // экран торговли
	StateVictory                      // экран победы
	StatePopup                        // 🆕 попап с сообщением
)

// =============================================================================
// СТРАНИЦЫ ЭКРАНА ПОМОЩИ
// Экран помощи разбит на страницы, так как вся информация не помещается
// на один экран (высота 24 строки). Перелистывание: стрелки ← → или A/D.
// =============================================================================
const (
	helpPageControls  = 0 // страница 1: управление (клавиши)
	helpPageSymbols   = 1 // страница 2: символы (легенда карты)
	helpPageMechanics = 2 // страница 3: механики (свитки, реликвии, боссы)
	helpPageCount     = 3 // общее количество страниц
)

// =============================================================================
// ОСНОВНАЯ СТРУКТУРА ИГРЫ
// =============================================================================
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

	// Торговец
	currentMerchant     *Merchant 
		// 🆕 Попапы и предупреждения
	popupMessage      string // сообщение для попапа (если != "" — показываем)
	wasInCriticalHP   bool   // 🆕 был ли игрок в критическом состоянии (HP ≤ 5)
	wasTooFull        bool // 🆕 был ли игрок слишком сыт (Hunger < 200)	
	pendingRelicSellIndex int     // 🆕 Индекс реликвии, ожидающей подтверждения продажи (-1 если нет)

	// Музыка
	musicStreamer beep.StreamSeekCloser 
	musicVolume   *linearVolume         
	musicCtrl     *beep.Ctrl            
	musicFileH    *os.File              
	musicEnabled  bool                  
	musicLevel    float64               
}

// =============================================================================
// СОБЫТИЕ МИГАНИЯ
// =============================================================================
type blinkEvent struct {
	when time.Time
}

func (e *blinkEvent) When() time.Time {
	return e.when
}

func stringWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// =============================================================================
// СОЗДАНИЕ НОВОЙ ИГРЫ
// =============================================================================
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
		pendingRelicSellIndex: -1, // 🆕 Инициализация флага подтверждения
		popupMessage:      "",  // 🆕
		wasInCriticalHP: false,
		wasTooFull:        false, // 🆕
	}
}

func (g *Game) logAndSync(format string, v ...interface{}) {
	if g.logger != nil {
		g.logger.Printf(format, v...)
	}
	if g.logFileHandle != nil {
		g.logFileHandle.Sync()
	}
}

// =============================================================================
// ТАЙМЕР МИГАНИЯ
// =============================================================================
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
				if g.screen != nil {
					_ = g.screen.PostEvent(&blinkEvent{when: time.Now()})
				}
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

// =============================================================================
// ГЛАВНЫЙ ЦИКЛ ИГРЫ
// =============================================================================
func (g *Game) Run() error {
	if g.logFileHandle != nil {
		defer g.logFileHandle.Close()
	}

	var err error
	g.screen, err = tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("не удалось создать экран: %w", err)
	}
	if err := g.screen.Init(); err != nil {
		return fmt.Errorf("не удалось инициализировать экран: %w", err)
	}
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