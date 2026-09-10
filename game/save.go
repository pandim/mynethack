package game

import (
	"encoding/json"
	"fmt"
	"os"
)

// =============================================================================
// ФОРМАТ СОХРАНЕНИЯ ИГРЫ
// =============================================================================
type SaveData struct {
	Version int            // версия формата (для проверки совместимости)
	Player  *Player        // состояние игрока
	Depth   int            // текущая глубина
	Levels  map[int]*Level // все посещённые уровни
}

// =============================================================================
// УПРАВЛЕНИЕ ИГРОЙ
// =============================================================================
//
// startNewGame — начинает новую игру с чистого листа
func (g *Game) startNewGame() {
	g.depth = 1
	g.messages = make([]string, 0)
	g.showInventory = false
	g.quit = false
	g.state = StatePlaying
	g.currentMerchant = nil

	// Создаём кэш уровней и генерируем первый уровень
	g.levels = make(map[int]*Level)
	g.level = g.getOrCreateLevel(g.depth)

	// Создаём игрока и помещаем его на свободное место на уровне
	g.player = NewPlayer(10, 10)
	g.player.SetLogger(g.logger)
	x, y := g.level.FindFreeSpot()
	g.player.X = x
	g.player.Y = y

	g.logAndSync("SPAWN: Игрок создан на уровне %d в точке (%d, %d)", g.depth, x, y)
	g.addMessage("Добро пожаловать в подземелье! Нажмите ? для помощи.")
}

// getOrCreateLevel — возвращает уровень из кэша или генерирует новый.
// Это важно: при возврате на предыдущий уровень мы должны увидеть его
// в том же состоянии (с теми же монстрами и предметами).
//
// 🆕 ЭТАП 2: Если уровень уже в кэше, он возвращается как есть.
// Возрождение происходит в nextLevel/prevLevel после вызова этой функции.
func (g *Game) getOrCreateLevel(depth int) *Level {
	if depth < 1 {
		depth = 1
	}
	if g.levels == nil {
		g.levels = make(map[int]*Level)
	}
	// Проверяем, что уровень в кэше валидный
	if lvl, ok := g.levels[depth]; ok && lvl != nil && isLevelValid(lvl) {
		return lvl
	}
	// Генерируем новый уровень
	lvl := NewLevel(mapWidth, mapHeight, depth, g.logger)
	g.levels[depth] = lvl
	return lvl
}

// isLevelValid — проверяет корректность уровня (защита от повреждённых сохранений)
func isLevelValid(l *Level) bool {
	if l == nil || l.Width <= 0 || l.Height <= 0 {
		return false
	}
	if len(l.Tiles) != l.Height {
		return false
	}
	for _, row := range l.Tiles {
		if len(row) != l.Width {
			return false
		}
	}
	return true
}

// prepareLevelAfterLoad — восстанавливает уровень после загрузки из сохранения.
// Гарантирует, что лестницы существуют и находятся на проходимых клетках.
// Также инициализирует новые поля (торговцы, алтари, сундуки), если они пусты.
func (g *Game) prepareLevelAfterLoad(l *Level, depth int) {
	if l == nil {
		return
	}
	if l.Monsters == nil {
		l.Monsters = make([]*Monster, 0)
	}
	if l.Items == nil {
		l.Items = make([]*Item, 0)
	}
	// Инициализируем новые поля (торговцы, алтари, сундуки)
	if l.Merchants == nil {
		l.Merchants = make([]*Merchant, 0)
	}
	if l.Altars == nil {
		l.Altars = make([]*Altar, 0)
	}
	if l.Chests == nil {
		l.Chests = make([]*Chest, 0)
	}
	l.Depth = depth
	l.SetLogger(g.logger)

	// Гарантируем наличие лестницы вниз
	if !l.StairsDown {
		l.StairsDown = true
		x, y := l.FindFreeSpot()
		l.StairsDownX = x
		l.StairsDownY = y
	}
	// На уровнях глубже первого гарантируем лестницу вверх
	if depth > 1 && !l.StairsUp {
		l.StairsUp = true
		x, y := l.FindFreeSpotExcluding(l.StairsDownX, l.StairsDownY)
		l.StairsUpX = x
		l.StairsUpY = y
	}
	// Если лестница оказалась на непроходимой клетке — перемещаем её
	if l.StairsDown && !l.CanMoveTo(l.StairsDownX, l.StairsDownY) {
		x, y := l.FindFreeSpotExcluding(l.StairsUpX, l.StairsUpY)
		l.StairsDownX = x
		l.StairsDownY = y
	}
	if l.StairsUp && !l.CanMoveTo(l.StairsUpX, l.StairsUpY) {
		x, y := l.FindFreeSpotExcluding(l.StairsDownX, l.StairsDownY)
		l.StairsUpX = x
		l.StairsUpY = y
	}
}

// loadGameFromMenu — загружает игру из сохранения (вызывается из стартового меню)
func (g *Game) loadGameFromMenu() {
	file, err := os.ReadFile(saveFile)
	if err != nil {
		g.logAndSync("LOAD_ERROR: Не удалось прочитать сохранение: %v", err)
		g.startNewGame()
		return
	}

	var data SaveData

	// Тщательная валидация сохранения перед использованием
	if err := json.Unmarshal(file, &data); err != nil ||
		data.Version != saveVersion ||
		data.Player == nil ||
		data.Depth < 1 ||
		data.Player.HP <= 0 ||
		data.Player.MaxHP <= 0 {
		g.logAndSync("LOAD_ERROR: Сохранение повреждено, некорректно или имеет старую версию")
		g.startNewGame()
		return
	}

	if data.Levels == nil || len(data.Levels) == 0 {
		g.logAndSync("LOAD_ERROR: В сохранении нет уровней")
		g.startNewGame()
		return
	}

	// Фильтруем невалидные уровни
	cleanedLevels := make(map[int]*Level)
	for depth, lvl := range data.Levels {
		if depth < 1 || !isLevelValid(lvl) {
			continue
		}
		cleanedLevels[depth] = lvl
	}

	if len(cleanedLevels) == 0 {
		g.logAndSync("LOAD_ERROR: Ни один уровень в сохранении не корректен")
		g.startNewGame()
		return
	}

	// Если текущий уровень отсутствует — генерируем его
	if _, ok := cleanedLevels[data.Depth]; !ok {
		cleanedLevels[data.Depth] = NewLevel(mapWidth, mapHeight, data.Depth, g.logger)
	}

	// Восстанавливаем состояние игры
	g.levels = cleanedLevels
	g.depth = data.Depth
	for depth, lvl := range g.levels {
		g.prepareLevelAfterLoad(lvl, depth)
	}
	g.level = g.levels[g.depth]

	// Восстанавливаем игрока
	// 🆕 ЭТАП 3: Если поле HasAmulet отсутствует в старом сохранении,
	// оно будет инициализировано значением по умолчанию (false)
	g.player = data.Player
	if g.player.Inventory == nil {
		g.player.Inventory = make([]*Item, 0)
	}
	if g.player.HP > g.player.MaxHP {
		g.player.HP = g.player.MaxHP
	}
	g.player.SetLogger(g.logger)

	// Если игрок оказался на непроходимой клетке — перемещаем его
	if !g.level.CanMoveTo(g.player.X, g.player.Y) ||
		g.level.hasMonsterAt(g.player.X, g.player.Y) {
		x, y := g.level.FindFreeSpot()
		g.player.X = x
		g.player.Y = y
	}

	g.messages = make([]string, 0)
	g.showInventory = false
	g.quit = false
	g.state = StatePlaying
	g.currentMerchant = nil

	g.logAndSync("LOAD: Игра загружена. Игрок на (%d, %d)", g.player.X, g.player.Y)
	g.addMessage("Игра загружена.")
}

// saveGame — сохраняет текущее состояние игры в файл
func (g *Game) saveGame() {
	if g.player == nil || g.level == nil {
		return
	}
	if g.levels == nil {
		g.levels = make(map[int]*Level)
	}
	g.levels[g.depth] = g.level

	data := SaveData{
		Version: saveVersion,
		Player:  g.player,
		Depth:   g.depth,
		Levels:  g.levels,
	}

	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		g.logAndSync("ERROR: Ошибка сериализации сохранения: %v", err)
		g.addMessage("Не удалось сохранить игру!")
		return
	}

	if err := os.WriteFile(saveFile, file, 0644); err != nil {
		g.logAndSync("ERROR: Не удалось записать сохранение: %v", err)
		g.addMessage("Не удалось сохранить игру!")
		return
	}

	g.logAndSync("SAVE: Игра сохранена на глубине %d", g.depth)
	g.addMessage("Игра сохранена.")
}

// nextLevel — спуск на следующий уровень (клавиша >)
//
// 🆕 ЭТАП 2: Если уровень уже был посещён, монстры и объекты возрождаются.
// Монстров становится меньше, но они сильнее (характеристики как на уровне+1).
// Алтари оживают, сундуки получают новое содержимое, торговцы — новый товар.
// Боссы НЕ возрождаются: живой босс остаётся, убитый не возвращается.
func (g *Game) nextLevel() {
	if g.player == nil {
		return
	}
	g.depth++

	// 🆕 ЭТАП 2: Проверяем, был ли уровень уже посещён (уже в кэше)
	_, alreadyVisited := g.levels[g.depth]

	g.level = g.getOrCreateLevel(g.depth)

	// 🆕 ЭТАП 2: Если уровень был посещён — возрождаем монстров и объекты
	// Функция respawnLevel определена ниже в этом файле
	// Функции respawnMonsters, newChestContents, newMerchantItems — в level_spawn.go
	if alreadyVisited {
		g.respawnLevel(g.level)
	}

	// Ставим игрока у лестницы вверх нового уровня (логично: он спустился оттуда)
	if g.level.StairsUp &&
		g.level.StairsUpX >= 0 &&
		g.level.StairsUpY >= 0 &&
		g.level.CanMoveTo(g.level.StairsUpX, g.level.StairsUpY) {
		g.player.X = g.level.StairsUpX
		g.player.Y = g.level.StairsUpY
	} else {
		x, y := g.level.FindFreeSpot()
		g.player.X = x
		g.player.Y = y
	}

	g.logAndSync("LEVEL_DOWN: Переход на уровень %d. Игрок на (%d, %d)",
		g.depth, g.player.X, g.player.Y)
	g.addMessage(fmt.Sprintf("Вы спустились на уровень %d.", g.depth))
	g.saveGame() // автосохранение при переходе между уровнями
}

// prevLevel — подъём на предыдущий уровень (клавиша <)
//
// 🆕 ЭТАП 2: Если уровень уже был посещён, монстры и объекты возрождаются.
// Монстров становится меньше, но они сильнее (характеристики как на уровне+1).
// Алтари оживают, сундуки получают новое содержимое, торговцы — новый товар.
// Боссы НЕ возрождаются: живой босс остаётся, убитый не возвращается.
//
// 🆕 ЭТАП 3: ПРОВЕРКА ПОБЕДЫ
// Если игрок на уровне 2 с Амулетом Бездны поднимается на уровень 1 — ПОБЕДА!
// Проверка победы происходит в input.go → handleMovement → case '<'
func (g *Game) prevLevel() {
	if g.player == nil {
		return
	}
	if g.depth <= 1 {
		g.addMessage("Вы уже на поверхности!")
		return
	}
	g.depth--

	// 🆕 ЭТАП 2: Проверяем, был ли уровень уже посещён (уже в кэше)
	_, alreadyVisited := g.levels[g.depth]

	g.level = g.getOrCreateLevel(g.depth)

	// 🆕 ЭТАП 2: Если уровень был посещён — возрождаем монстров и объекты
	// Функция respawnLevel определена ниже в этом файле
	// Функции respawnMonsters, newChestContents, newMerchantItems — в level_spawn.go
	if alreadyVisited {
		g.respawnLevel(g.level)
	}

	// Ставим игрока у лестницы вниз (логично: он поднялся оттуда)
	if g.level.StairsDown &&
		g.level.StairsDownX >= 0 &&
		g.level.StairsDownY >= 0 &&
		g.level.CanMoveTo(g.level.StairsDownX, g.level.StairsDownY) {
		g.player.X = g.level.StairsDownX
		g.player.Y = g.level.StairsDownY
	} else {
		x, y := g.level.FindFreeSpot()
		g.player.X = x
		g.player.Y = y
	}

	g.logAndSync("LEVEL_UP: Подъем на уровень %d. Игрок на (%d, %d)",
		g.depth, g.player.X, g.player.Y)
	g.addMessage(fmt.Sprintf("Вы поднялись на уровень %d.", g.depth))
	g.saveGame()
}

// =============================================================================
// 🆕 ЭТАП 2: ВОЗРОЖДЕНИЕ УРОВНЯ
// =============================================================================
//
// =============================================================================
// 🆕 ЭТАП 2: ВОЗРОЖДЕНИЕ УРОВНЯ
// =============================================================================
//
// respawnLevel — возрождает уровень при повторном посещении.
// Вызывается из nextLevel и prevLevel, если уровень уже был посещён.
//
// Механика возрождения:
//   - Монстры возрождаются в меньшем количестве, но с характеристиками уровня+1
//     (см. respawnMonsters в level_spawn.go)
//   - Алтари оживают (сброс Used)
//   - Сундуки оживают (сброс Opened, новое содержимое для обычных сундуков)
//   - Торговцы получают новый товар
//   - 🆕 VisitCount увеличивается — цены удваиваются при каждом посещении
//
// Это делает повторное посещение уровней осмысленным:
// игрок может вернуться за ресурсами, но встречает более сильных врагов
// и более дорогие товары.
//
// Боссы НЕ возрождаются: если босс жив, он остаётся на месте;
// если босс был убит, он не возвращается.
//
// Предметы на полу остаются на месте (не возрождаются и не удаляются).
// Золотые сундуки сохраняют содержимое "golden" (не меняется на новое).
func (g *Game) respawnLevel(l *Level) {
	if l == nil {
		return
	}

	// 🆕 Увеличиваем счётчик посещений — цены будут удвоены
	l.VisitCount++

	g.logAndSync("RESPAWN: Возрождение уровня %d (посещение #%d)", l.Depth, l.VisitCount)

	// 1. Возрождаем монстров (меньше, но сильнее)
	// Функция определена в level_spawn.go
	l.respawnMonsters()

	// 2. Оживляем алтари (сброс Used)
	// Алтарь снова можно использовать для благословения
	// Поле Used определено в altar.go
	for _, altar := range l.Altars {
		if altar != nil && altar.Used {
			altar.Used = false
		}
	}

	// 3. Оживляем сундуки (сброс Opened, новое содержимое)
	// Сундук снова можно открыть, содержимое определяется заново
	// Функция newChestContents определена в level_spawn.go
	// 🆕 Золотые сундуки сохраняют содержимое "golden" (поле IsGolden в chest.go)
	for _, chest := range l.Chests {
		if chest != nil && chest.Opened {
			chest.Opened = false
			// Золотые сундуки не меняют содержимое
			// Поле IsGolden определено в chest.go
			if !chest.IsGolden {
				chest.Contents = newChestContents() // новое случайное содержимое
			}
		}
	}

	// 4. Оживляем торговцев (новый товар с удвоенными ценами)
	// Торговец снова продаёт товары (возможно, другие)
	// Функция newMerchantItems определена в level_spawn.go
	// 🆕 Цены удваиваются при каждом посещении
	for _, merchant := range l.Merchants {
		if merchant != nil {
			merchant.Items = newMerchantItems(l.Depth, l.VisitCount) // новый товар с удвоенными ценами
		}
	}

	g.addMessage(fmt.Sprintf("Уровень изменился... Монстры вернулись! (посещение #%d)", l.VisitCount))
}