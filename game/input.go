package game

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

// handleHelpInput — обработка ввода на экране помощи.
//
// НАВИГАЦИЯ:
//   - Стрелки ← → или клавиши A/D: перелистывание страниц
//   - Любая другая клавиша: возврат в игру
//
// Перелистывание НЕ циклическое: на первой странице ← ничего не делает,
// на последней странице → ничего не делает.
func (g *Game) handleHelpInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	keyEvent, ok := ev.(*tcell.EventKey)
	if !ok {
		return
	}

	// --- Перелистывание стрелками ---

	// Стрелка влево — предыдущая страница
	if keyEvent.Key() == tcell.KeyLeft {
		if g.helpPage > helpPageControls {
			g.helpPage--
		}
		return
	}

	// Стрелка вправо — следующая страница
	if keyEvent.Key() == tcell.KeyRight {
		if g.helpPage < helpPageCount-1 {
			g.helpPage++
		}
		return
	}

	// --- Перелистывание клавишами A/D ---

	r := keyEvent.Rune()

	// A — предыдущая страница (аналог стрелки влево)
	if r == 'a' || r == 'A' {
		if g.helpPage > helpPageControls {
			g.helpPage--
		}
		return
	}

	// D — следующая страница (аналог стрелки вправо)
	if r == 'd' || r == 'D' {
		if g.helpPage < helpPageCount-1 {
			g.helpPage++
		}
		return
	}

	// --- Любая другая клавиша — возврат в игру ---
	g.state = StatePlaying
}

// =============================================================================
// 🆕 ЭТАП 3: ОБРАБОТКА ВВОДА НА ЭКРАНЕ ПОБЕДЫ
// =============================================================================
//
// handleVictoryInput — обработка ввода на экране победы.
// Любая клавиша — выход из игры.
func (g *Game) handleVictoryInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev.(type) {
	case *tcell.EventKey:
		// Любая клавиша — выход из игры
		g.quit = true
	}
}

// =============================================================================
// ОБРАБОТКА ВВОДА
// =============================================================================

// handleInput — обработка ввода в режиме игры
func (g *Game) handleInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev := ev.(type) {
	case *blinkEvent:
		// Событие мигания — просто перерисовываем экран
		// (нужно для анимации боссов и мигания игрока с Амулетом)
		return
	case *tcell.EventKey:
		isEscape := ev.Key() == tcell.KeyEscape || ev.Rune() == 27
		isQuit := ev.Key() == tcell.KeyCtrlC || ev.Key() == tcell.KeyCtrlQ

		// ESC или Ctrl+C: закрываем инвентарь или показываем диалог выхода
		if isEscape || isQuit {
			if g.showInventory {
				g.showInventory = false
				g.addMessage("Инвентарь закрыт")
				return
			}
			g.state = StateQuitConfirm
			return
		}

		// В зависимости от того, открыт ли инвентарь, обрабатываем разные клавиши
		if g.showInventory {
			g.handleInventoryInput(ev.Rune())
		} else {
			g.handleMovement(ev.Rune(), ev.Key())
		}
	case *tcell.EventResize:
		// При изменении размера терминала синхронизируем экран
		g.screen.Sync()
	}
}

// handleStartMenuInput — обработка ввода в стартовом меню
func (g *Game) handleStartMenuInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
			g.quit = true
			return
		}
		r := ev.Rune()
		if r == 'l' || r == 'L' {
			g.loadGameFromMenu() // загрузить сохранение (функция в save.go)
		} else if r == 'n' || r == 'N' {
			// Новая игра: удаляем старое сохранение
			if _, err := os.Stat(saveFile); err == nil {
				os.Remove(saveFile)
			}
			g.startNewGame() // функция в save.go
		} else if r == 'm' || r == 'M' {
			g.toggleMusic() // вкл/выкл музыку в меню (функция в audio.go)
		}
	}
}

// handleQuitConfirmInput — обработка ввода в диалоге подтверждения выхода
func (g *Game) handleQuitConfirmInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyCtrlC {
			g.quit = true
			return
		}
		r := ev.Rune()
		if r == 'y' || r == 'Y' {
			g.quit = true // подтверждаем выход
		} else if r == 'n' || r == 'N' || ev.Key() == tcell.KeyEscape {
			g.state = StatePlaying // отменяем выход
		}
	}
}

// handleDeathInput — обработка ввода на экране смерти
func (g *Game) handleDeathInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev := ev.(type) {
	case *tcell.EventKey:
		r := ev.Rune()
		// Y или ESC — выход из игры
		if r == 'y' || r == 'Y' ||
			ev.Key() == tcell.KeyEscape ||
			ev.Key() == tcell.KeyCtrlC {
			g.quit = true
			return
		}
		// N — новая игра (удаляем старое сохранение)
		if r == 'n' || r == 'N' {
			if _, err := os.Stat(saveFile); err == nil {
				os.Remove(saveFile)
			}
			g.startNewGame() // функция в save.go
			return
		}
	}
}

// handleInventoryInput — обработка ввода в инвентаре
func (g *Game) handleInventoryInput(key rune) {
	if g.player == nil {
		g.showInventory = false
		return
	}

	switch key {
	case 'q', 'Q':
		// Закрытие инвентаря
		g.showInventory = false
		g.logAndSync("UI: Инвентарь закрыт")
		g.addMessage("Инвентарь закрыт")
	default:
		// Использование предмета по номеру (1-9)
		if key >= '1' && key <= '9' {
			index := int(key - '1')
			if index < len(g.player.Inventory) {
				g.useItem(index)
			} else {
				g.addMessage("Неверный номер предмета!")
			}
		}
	}
}

// useItem — использование предмета из инвентаря.
//
// МЕХАНИКА СТОПОК:
//   - При использовании предмета Count уменьшается на 1
//   - Когда Count достигает 0 — предмет удаляется из инвентаря
//   - Работает для ВСЕХ типов: зелья, еда, оружие, броня, свитки
//
// МЕХАНИКА "УЛУЧШЕНИЯ" для оружия/брони:
//   - Если нет экипированного предмета — экипируем новое
//   - Если есть — улучшаем текущее на +1 (новый предмет расходуется)
//
// 🆕 ЭТАП 3: ЭФФЕКТЫ СВИТКОВ:
//   - Свиток карты: открывает весь этаж
//   - Свиток телепортации: случайное перемещение
//   - Свиток молнии: урон всем монстрам
//   - Свиток изгнания: уничтожает случайного монстра
//
// 🆕 ЭТАП 3: РЕЛИКВИИ И АМУЛЕТ:
//   - Реликвии нельзя использовать (только продать торговцу)
//   - Амулет нельзя использовать (он нужен для победы)
func (g *Game) useItem(index int) {
	if g.player == nil || index < 0 || index >= len(g.player.Inventory) {
		return
	}

	item := g.player.Inventory[index]
	if item == nil {
		return
	}

	g.logAndSync("ACTION: Использование предмета %s (Count=%d)", item.Name, item.Count)

	switch item.Type {
	case ItemTypePotion:
		// Зелья: еда утоляет голод, остальные лечат
		if item.Name == "Еда" {
			g.player.Hunger = 0
			g.logAndSync("STATUS: Голод утолен едой")
			g.addMessage("Вы поели. Голод утолен.")
		} else {
			g.player.Heal(item.Value)
			g.addMessage(fmt.Sprintf("Выпито %s: +%d HP", item.Name, item.Value))
		}

	case ItemTypeWeapon:
		// ⚠️ МЕХАНИКА "УЛУЧШЕНИЯ":
		// Если нет экипированного оружия — экипируем новое
		// Если есть — улучшаем текущее на +1 ATK
		oldATK := 0
		if g.player.EquippedWeapon != nil {
			oldATK = g.player.EquippedWeapon.Value
		}
		g.player.EquipWeapon(item)
		if oldATK == 0 {
			// Первая экипировка оружия
			g.addMessage(fmt.Sprintf("Экипировано %s: ATK +%d", item.Name, item.Value))
		} else {
			// Улучшение существующего оружия
			g.addMessage(fmt.Sprintf("%s улучшен! ATK +%d -> +%d",
				g.player.EquippedWeapon.Name, oldATK, g.player.EquippedWeapon.Value))
		}

	case ItemTypeArmor:
		// ⚠️ МЕХАНИКА "УЛУЧШЕНИЯ":
		// Если нет экипированной брони — экипируем новую
		// Если есть — улучшаем текущую на +1 DEF
		oldDEF := 0
		if g.player.EquippedArmor != nil {
			oldDEF = g.player.EquippedArmor.Value
		}
		g.player.EquipArmor(item)
		if oldDEF == 0 {
			// Первая экипировка брони
			g.addMessage(fmt.Sprintf("Экипировано %s: DEF +%d", item.Name, item.Value))
		} else {
			// Улучшение существующей брони
			g.addMessage(fmt.Sprintf("%s улучшен! DEF +%d -> +%d",
				g.player.EquippedArmor.Name, oldDEF, g.player.EquippedArmor.Value))
		}

	case ItemTypeGold:
		// Золото уже в кошельке (подбирается автоматически)
		g.addMessage("Золото уже в кошельке!")

	// 🆕 ЭТАП 3: Эффекты свитков
	// Функции эффектов определены в combat.go
	case ItemTypeScroll:
		switch item.ScrollType {
		case ScrollMap:
			g.useScrollMap()
		case ScrollTeleport:
			g.useScrollTeleport()
		case ScrollLightning:
			g.useScrollLightning()
		case ScrollBanishment:
			g.useScrollBanishment()
		default:
			g.addMessage("Неизвестный свиток!")
		}

	// 🆕 ЭТАП 3: Реликвии нельзя использовать
	// Реликвии можно только продать торговцу (см. interact.go)
	case ItemTypeRelic:
		g.addMessage("Реликвию можно продать торговцу (клавиша T на торговце).")

	// 🆕 ЭТАП 3: Ключ нельзя использовать напрямую
	// Ключ используется автоматически при открытии золотого сундука
	case ItemTypeKey:
		g.addMessage("Ключ нужен для открытия золотых сундуков (клавиша O на сундуке).")

	// 🆕 ЭТАП 3: Амулет нельзя использовать
	// Амулет нужен для победы — вернитесь на уровень 1
	case ItemTypeAmulet:
		g.addMessage("Амулет Бездны излучает странное сияние... Вернитесь на уровень 1!")

	default:
		g.addMessage("Нельзя использовать этот предмет!")
	}

	// Уменьшаем счётчик стопки (для всех типов предметов)
	// 🆕 Реликвии и Амулет не имеют стопок (Count всегда 1)
	item.Count--
	if item.Count <= 0 {
		// Стопка пуста — удаляем предмет из инвентаря
		g.player.Inventory = append(g.player.Inventory[:index], g.player.Inventory[index+1:]...)
		g.logAndSync("ITEM_USED: %s полностью израсходован", item.Name)
	} else {
		// В стопке ещё есть предметы
		g.addMessage(fmt.Sprintf("Осталось %s: %d шт.", item.Name, item.Count))
	}

	// Если инвентарь пуст — закрываем инвентарь
	if len(g.player.Inventory) == 0 {
		g.showInventory = false
	}
}

// handleMovement — обработка клавиш движения и команд
//
// Функции, вызываемые отсюда и определённые в других файлах:
//   - processTurn, nextLevel, prevLevel → combat.go / save.go
//   - saveGame, startNewGame → save.go
//   - toggleMusic, changeMusicVolume → audio.go
//   - showAltarUI, openChest → interact.go
//   - GetMerchantAt, GetAltarAt, GetChestAt, HasAliveBoss → level.go / level_boss.go
//
// 🆕 ЭТАП 3: ПРОВЕРКА ПОБЕДЫ НА 1 УРОВНЕ:
// Когда игрок на уровне 2 с Амулетом нажимает '<' (подняться по лестнице),
// игра переходит в состояние StateVictory (экран победы).
func (g *Game) handleMovement(key rune, specialKey tcell.Key) {
	if g.player == nil || g.level == nil {
		return
	}

	dx, dy := 0, 0

	// Сначала проверяем специальные клавиши (стрелки)
	switch specialKey {
	case tcell.KeyUp:
		dy = -1
	case tcell.KeyDown:
		dy = 1
	case tcell.KeyLeft:
		dx = -1
	case tcell.KeyRight:
		dx = 1
	}

	// Пробел — ждать ход
	if key == ' ' {
		g.logAndSync("ACTION: Игрок ждет ход (пробел)")
		g.processTurn(0, 0)
		return
	}

	// Если стрелки не использовались, проверяем обычные клавиши
	if dx == 0 && dy == 0 {
		switch key {
		// Движение: WASD для 4 направлений, QEZC для диагоналей
		case 'a', 'A':
			dx = -1
		case 'x', 'X':
			dy = 1
		case 'w', 'W':
			dy = -1
		case 'd', 'D':
			dx = 1
		case 'q', 'Q':
			dx, dy = -1, -1
		case 'e', 'E':
			dx, dy = 1, -1
		case 'z', 'Z':
			dx, dy = -1, 1
		case 'c', 'C':
			dx, dy = 1, 1
		case 's':
			// S — ждать ход (аналог пробела)
			g.logAndSync("ACTION: Игрок ждет ход (s)")
			g.processTurn(0, 0)
			return
		case 'i', 'I':
			// I — открыть инвентарь
			g.showInventory = true
			g.logAndSync("UI: Открыт инвентарь")
			g.addMessage("Открыт инвентарь")
			return
		case 'm', 'M':
			// M — вкл/выкл музыку
			g.toggleMusic()
			return
		case '+', '=':
			// + — громче на 10%
			g.changeMusicVolume(0.1)
			return
		case '-', '_':
			// - — тише на 10%
			g.changeMusicVolume(-0.1)
			return
		case '?':
			// ? — экран помощи
			g.state = StateHelp
			g.helpPage = helpPageControls // всегда открываем с первой страницы
			return
		case '>':
			// > — спуститься по лестнице
			if g.level.StairsDown &&
				g.player.X == g.level.StairsDownX &&
				g.player.Y == g.level.StairsDownY {
				// 🆕 БЛОКИРОВКА ЛЕСТНИЦЫ БОССОМ:
				// Пока босс жив, игрок не может спуститься
				if g.level.HasAliveBoss() {
					bossName := g.level.GetBossName()
					g.addMessage(fmt.Sprintf("%s охраняет лестницу! Сначала победите его!", bossName))
					return
				}
				g.nextLevel()
				return
			}
			g.addMessage("Здесь нет лестницы вниз.")
			return
		case '<':
			// < — подняться по лестнице
			if g.level.StairsUp &&
				g.player.X == g.level.StairsUpX &&
				g.player.Y == g.level.StairsUpY {

				// 🆕 ЭТАП 3: ПРОВЕРКА ПОБЕДЫ
				// Если игрок на уровне 2 с Амулетом и поднимается на уровень 1 — ПОБЕДА!
				// Уровень 2 — последний уровень перед поверхностью.
				// Подъём с уровня 2 с Амулетом = победа.
				if g.depth == 2 && g.player.HasAmulet {
					g.logAndSync("VICTORY: Игрок вернулся на поверхность с Амулетом Бездны!")
					g.state = StateVictory
					return
				}

				g.prevLevel()
				return
			}
			g.addMessage("Здесь нет лестницы вверх.")
			return
		case 'S':
			// Shift+S — сохранить игру
			g.saveGame()
			return
		case 'n', 'N':
			// N — новая игра (удаляем сохранение)
			if _, err := os.Stat(saveFile); err == nil {
				os.Remove(saveFile)
			}
			g.startNewGame()
			return
		case 't', 'T':
			// T — торговля (если стоим на торговце)
			// Открываем экран торговли с выбором по цифре
			if merchant := g.level.GetMerchantAt(g.player.X, g.player.Y); merchant != nil {
				g.currentMerchant = merchant
				g.state = StateMerchant
				return
			}
			g.addMessage("Здесь нет торговца.")
			return
		case 'b', 'B':
			// B — благословение (если стоим на алтаре)
			if altar := g.level.GetAltarAt(g.player.X, g.player.Y); altar != nil {
				g.showAltarUI(altar) // функция в interact.go
				return
			}
			g.addMessage("Здесь нет алтаря.")
			return
		case 'o', 'O':
			// O — открыть сундук (если стоим на сундуке)
			if chest := g.level.GetChestAt(g.player.X, g.player.Y); chest != nil {
				g.openChest(chest) // функция в interact.go
				return
			}
			g.addMessage("Здесь нет сундука.")
			return
		}
	}

	// Если было движение — обрабатываем ход
	if dx != 0 || dy != 0 {
		g.processTurn(dx, dy)
	}
}