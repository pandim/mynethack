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
// Игра может находиться в одном из состояний, и в зависимости от состояния
// обрабатываются разные экраны и ввод пользователя.
// =============================================================================
type GameState int

const (
	StatePlaying     GameState = iota // обычный игровой процесс
	StateStartMenu                    // стартовое меню (показывается всегда)
	StateQuitConfirm                  // подтверждение выхода
	StateDeathMenu                    // экран смерти игрока
	StateHelp                         // экран помощи (вызывается клавишей ?)
	StateMerchant                     // экран торговли с торговцем
)

// =============================================================================
// СТРАНИЦЫ ЭКРАНА ПОМОЩИ
// Экран помощи разбит на страницы, так как вся информация не помещается
// на один экран (высота 24 строки). Перелистывание: стрелки ← → или A/D.
// =============================================================================
const (
	helpPageControls = 0 // страница 1: управление (клавиши)
	helpPageSymbols  = 1 // страница 2: символы (легенда карты)
	helpPageCount    = 2 // общее количество страниц
)

// =============================================================================
// ОСНОВНАЯ СТРУКТУРА ИГРЫ
// =============================================================================
type Game struct {
	// Экран и отображение
	screen        tcell.Screen     // терминальный экран (библиотека tcell)
	player        *Player          // игрок
	level         *Level           // текущий уровень подземелья
	levels        map[int]*Level   // кэш всех уровней (чтобы не генерировать заново при возврате)
	depth         int              // текущая глубина (номер уровня)
	messages      []string         // последние сообщения для отображения
	quit          bool             // флаг выхода из игры
	showInventory bool             // открыт ли инвентарь
	helpPage      int              // текущая страница экрана помощи (0 = управление, 1 = символы)
	logger        *log.Logger      // логгер для отладки
	logFileHandle *os.File         // файловый дескриптор лога
	state         GameState        // текущее состояние игры

	// Торговец
	currentMerchant *Merchant // текущий торговец (для экрана торговли)

	// Музыка
	musicStreamer beep.StreamSeekCloser // стример аудио (нужен для закрытия)
	musicVolume   *linearVolume         // наша обёртка громкости (определена в audio.go)
	musicCtrl     *beep.Ctrl            // контроллер паузы/возобновления
	musicFileH    *os.File              // файловый дескриптор mp3
	musicEnabled  bool                  // включена ли музыка
	musicLevel    float64               // 0.0..1.0 (линейная громкость для пользователя)
}

// =============================================================================
// СОБЫТИЕ МИГАНИЯ
// Служебное событие для периодической перерисовки экрана (мигание символа @).
// Генерируется таймером каждые 250 мс.
// =============================================================================
type blinkEvent struct {
	when time.Time
}

func (e *blinkEvent) When() time.Time {
	return e.when
}

// stringWidth — считает видимую ширину строки (учитывает многобайтовые символы)
func stringWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// =============================================================================
// СОЗДАНИЕ НОВОЙ ИГРЫ
// =============================================================================
func NewGame() *Game {
	var f *os.File
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Пытаемся открыть файл логов. Если не получится — пишем в stderr.
	fh, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Ошибка открытия файла логов:", err)
	} else {
		f = fh
		logger = log.New(f, "", log.LstdFlags)
	}

	logger.Println("=== Запуск игры ===")

	return &Game{
		messages:      make([]string, 0),
		depth:         1,
		logger:        logger,
		logFileHandle: f,
		state:         StatePlaying,
		musicEnabled:  true,
		musicLevel:    0.3, // 30% — комфортная фоновая громкость по умолчанию
	}
}

// logAndSync — записывает в лог и сразу сбрасывает буфер на диск.
// Важно для отладки: если игра упадёт, лог не потеряется.
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
// Периодически отправляет события перерисовки для мигания символа игрока (@).
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

	// Возвращаем функцию остановки (можно вызвать только один раз благодаря `sync.Once`)
	return func() {
		once.Do(func() { close(stopCh) })
		<-doneCh // ждём завершения горутины
	}
}

// =============================================================================
// ГЛАВНЫЙ ЦИКЛ ИГРЫ
// =============================================================================
func (g *Game) Run() error {
	if g.logFileHandle != nil {
		defer g.logFileHandle.Close()
	}

	// Создаём и инициализируем терминальный экран
	var err error
	g.screen, err = tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("не удалось создать экран: %w", err)
	}
	if err := g.screen.Init(); err != nil {
		return fmt.Errorf("не удалось инициализировать экран: %w", err)
	}
	defer g.screen.Fini()

	// Запускаем музыку (и гарантированно останавливаем при выходе)
	// Функции initMusic/stopMusic определены в audio.go
	g.initMusic()
	defer g.stopMusic()

	// Запускаем таймер мигания
	stopBlink := g.startBlinkTicker()
	defer stopBlink()

	g.screen.SetStyle(tcell.StyleDefault)
	g.screen.Clear()

	// 🆕 Всегда показываем стартовое меню с ASCII-арт заголовком.
	// Меню проверяет наличие сохранения и показывает соответствующие опции.
	// Функция startNewGame вызывается из handleStartMenuInput при выборе [N].
	g.state = StateStartMenu

	// Главный игровой цикл
	for !g.quit {
		func() {
			// Защита от паник: если что-то пошло не так, логируем и выходим
			defer func() {
				if r := recover(); r != nil {
					g.logAndSync("CRITICAL ERROR (Recovered): %v", r)
					g.addMessage("КРИТИЧЕСКАЯ ОШИБКА! Смотрите лог.")
					g.quit = true
				}
			}()

			// В зависимости от состояния обрабатываем разные экраны
			// Функции обработки определены в render.go, input.go, interact.go
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
				// Экран торговли с торговцем (функции в interact.go)
				g.renderMerchantScreen()
				g.handleMerchantInput()
			}
		}()
	}

	return nil
}