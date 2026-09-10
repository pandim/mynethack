package game

import (
	"fmt"
	"math/rand/v2" // ДОБАВЛЕНО: для работы rand.IntN
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

	// Любая другая клавиша — возврат в игру
	g.state = StatePlaying
}

// =============================================================================
// 🆕 ЭТАП 3: ОБРАБОТКА ВВОДА НА ЭКРАНЕ ПОБЕДЫ
// =============================================================================
//
// handleVictoryInput — обработка ввода на экране победы.
// Любая клавиша — выход из игры.
// Состояние StateVictory определено в game.go.
// Экран победы отрисовывается в render.go → renderVictoryScreen.
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
		if ev.Key() == tcell.KeyEscape || ev.Rune() == 27 || ev.Rune() == 'n' || ev.Rune() == 'N' {
			// Отмена выхода
			g.state = StatePlaying
			return
		}
		if ev.Rune() == 'y' || ev.Rune() == 'Y' || ev.Key() == tcell.KeyEnter {
			// Подтверждение выхода
			g.quit = true
			return
		}
	}
}

// handleDeathInput — обработка ввода на экране смерти (ИСПРАВЛЕНО ИМЯ)
func (g *Game) handleDeathInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'q' || ev.Rune() == 'Q' {
			g.quit = true
			return
		}
		if ev.Rune() == 'n' || ev.Rune() == 'N' || ev.Key() == tcell.KeyEnter {
			// Новая игра после смерти: удаляем старое сохранение
			if _, err := os.Stat(saveFile); err == nil {
				os.Remove(saveFile)
			}
			g.startNewGame()
			return
		}
	}
}

// handleInventoryInput — обработка ввода в инвентаре
func (g *Game) handleInventoryInput(key rune) {
	if g.player == nil {
		return
	}

	// Цифры 1-9 для использования предметов
	if key >= '1' && key <= '9' {
		index := int(key - '1')
		if index < 0 || index >= len(g.player.Inventory) {
			g.addMessage("Такого предмета нет!")
			return
		}

		item := g.player.Inventory[index]
		if item == nil {
			return
		}

		// Используем предмет
		g.useItem(index)
		return
	}

	// ESC или q — закрыть инвентарь
	if key == 27 || key == 'q' || key == 'Q' {
		g.showInventory = false
		g.addMessage("Инвентарь закрыт")
		return
	}
}

// useItem — использование предмета из инвентаря по индексу
func (g *Game) useItem(index int) {
	if g.player == nil || index < 0 || index >= len(g.player.Inventory) {
		return
	}

	item := g.player.Inventory[index]
	if item == nil {
		return
	}

	switch item.Type {
	case ItemTypePotion:
		if item.Name == "Еда" {
			g.player.Hunger = 0
			g.addMessage("Вы поели. Голод утолен.")
			g.logAndSync("ITEM_USE: Съедена еда")
		} else {
			heal := item.Value
			oldHP := g.player.HP
			g.player.HP += heal
			if g.player.HP > g.player.MaxHP {
				g.player.HP = g.player.MaxHP
			}
			g.addMessage(fmt.Sprintf("Вы выпили %s и восстановили %d HP!", item.Name, g.player.HP-oldHP))
			g.logAndSync("ITEM_USE: Выпито %s, HP восстановлено на %d", item.Name, g.player.HP-oldHP)
		}
		g.consumeItem(index)
		g.processTurn(0, 0) // Ход тратится

	case ItemTypeWeapon:
		// 🆕 МЕХАНИКА УЛУЧШЕНИЯ: если имя совпадает, улучшаем текущее оружие на +1
		if g.player.EquippedWeapon != nil && g.player.EquippedWeapon.Name == item.Name {
			g.player.EquippedWeapon.Value += 1
			g.player.AttackVal += 1
			g.consumeItem(index) // Расходуем 1 предмет из стопки (уменьшаем Count)
			g.addMessage(fmt.Sprintf("Ваше оружие %s улучшено! Теперь ATK +%d", 
				g.player.EquippedWeapon.Name, g.player.EquippedWeapon.Value))
			g.logAndSync("ITEM_UPGRADE: Оружие %s улучшено до ATK +%d", 
				g.player.EquippedWeapon.Name, g.player.EquippedWeapon.Value)
		} else {
			// Если имя не совпадает или ничего не экипировано — обычная замена
			if g.player.EquippedWeapon != nil {
				g.player.AttackVal -= g.player.EquippedWeapon.Value
				g.player.Inventory = append(g.player.Inventory, g.player.EquippedWeapon)
			}
			g.player.EquippedWeapon = item
			g.player.AttackVal += item.Value
			g.player.Inventory = append(g.player.Inventory[:index], g.player.Inventory[index+1:]...)
			g.addMessage(fmt.Sprintf("Вы экипировали %s (ATK +%d)!", item.Name, item.Value))
			g.logAndSync("ITEM_EQUIP: Экипировано оружие %s", item.Name)
		}
		g.processTurn(0, 0)

	case ItemTypeArmor:
		// 🆕 МЕХАНИКА УЛУЧШЕНИЯ: если имя совпадает, улучшаем текущую броню на +1
		if g.player.EquippedArmor != nil && g.player.EquippedArmor.Name == item.Name {
			g.player.EquippedArmor.Value += 1
			g.player.Defense += 1
			g.consumeItem(index) // Расходуем 1 предмет из стопки (уменьшаем Count)
			g.addMessage(fmt.Sprintf("Ваша броня %s улучшена! Теперь DEF +%d", 
				g.player.EquippedArmor.Name, g.player.EquippedArmor.Value))
			g.logAndSync("ITEM_UPGRADE: Броня %s улучшена до DEF +%d", 
				g.player.EquippedArmor.Name, g.player.EquippedArmor.Value)
		} else {
			// Если имя не совпадает или ничего не экипировано — обычная замена
			if g.player.EquippedArmor != nil {
				g.player.Defense -= g.player.EquippedArmor.Value
				g.player.Inventory = append(g.player.Inventory, g.player.EquippedArmor)
			}
			g.player.EquippedArmor = item
			g.player.Defense += item.Value
			g.player.Inventory = append(g.player.Inventory[:index], g.player.Inventory[index+1:]...)
			g.addMessage(fmt.Sprintf("Вы экипировали %s (DEF +%d)!", item.Name, item.Value))
			g.logAndSync("ITEM_EQUIP: Экипирована броня %s", item.Name)
		}
		g.processTurn(0, 0)

	case ItemTypeScroll:
		g.useScroll(index)

	default:
		g.addMessage("Этот предмет нельзя использовать напрямую.")
	}
}

// consumeItem — уменьшает Count предмета или удаляет его из инвентаря
func (g *Game) consumeItem(index int) {
	if g.player == nil || index < 0 || index >= len(g.player.Inventory) {
		return
	}

	item := g.player.Inventory[index]
	if item == nil {
		return
	}

	item.Count--
	if item.Count <= 0 {
		// Удаляем предмет из инвентаря
		g.player.Inventory = append(g.player.Inventory[:index], g.player.Inventory[index+1:]...)
	}
}

// =============================================================================
// 🆕 ЭТАП 1: ИСПОЛЬЗОВАНИЕ СВИТКОВ
// =============================================================================
//
// useScroll — применяет эффект свитка и тратит его.
// Реализованы 4 типа свитков: карта, телепорт, молния, изгнание.
func (g *Game) useScroll(index int) {
	if g.player == nil || index < 0 || index >= len(g.player.Inventory) {
		return
	}

	item := g.player.Inventory[index]
	if item == nil || item.Type != ItemTypeScroll {
		return
	}

	switch item.ScrollType {
	case ScrollMap:
		// Открываем всю карту
		if g.level != nil {
			for y := 0; y < g.level.Height; y++ {
				for x := 0; x < g.level.Width; x++ {
					g.level.Tiles[y][x].Explored = true
					g.level.Tiles[y][x].Visible = true
				}
			}
			g.addMessage("Свиток карты озарил всё подземелье!")
			g.logAndSync("SCROLL: Использован свиток карты")
		}

	case ScrollTeleport:
		// Случайная телепортация на свободную клетку
		if g.level != nil {
			newX, newY := g.level.FindFreeSpot()
			g.player.X = newX
			g.player.Y = newY
			g.addMessage("Вас телепортировало в другое место!")
			g.logAndSync("SCROLL: Телепортация на (%d, %d)", newX, newY)
		}

	case ScrollLightning:
		// Урон всем монстрам на уровне
		if g.level != nil && len(g.level.Monsters) > 0 {
			damage := 15 + g.player.Level*2
			killedCount := 0
			for i := len(g.level.Monsters) - 1; i >= 0; i-- {
				m := g.level.Monsters[i]
				if m != nil {
					m.TakeDamage(damage)
					if m.HP <= 0 {
						g.level.RemoveMonster(m)
						g.player.Gold += m.GoldValue
						g.player.GainXP(m.XPValue)
						killedCount++
					}
				}
			}
			if killedCount > 0 {
				g.addMessage(fmt.Sprintf("Молния поразила всех монстров! Убито: %d", killedCount))
			} else {
				g.addMessage("Молния поразила всех монстров, но никто не погиб!")
			}
			g.logAndSync("SCROLL: Молния убила %d монстров", killedCount)
		} else {
			g.addMessage("На этом уровне нет монстров.")
		}

	case ScrollBanishment:
		// Уничтожает одного случайного монстра на уровне
		if g.level != nil && len(g.level.Monsters) > 0 {
			idx := rand.IntN(len(g.level.Monsters))
			m := g.level.Monsters[idx]
			if m != nil {
				g.level.RemoveMonster(m)
				g.player.Gold += m.GoldValue
				g.player.GainXP(m.XPValue)
				g.addMessage(fmt.Sprintf("Монстр %s был изгнан в небытие!", m.Name))
				g.logAndSync("SCROLL: Изгнан монстр %s", m.Name)
			}
		} else {
			g.addMessage("На этом уровне нет монстров.")
		}
	}

	// Тратим свиток
	g.consumeItem(index)
	g.processTurn(0, 0)
}

// handleMovement — обработка клавиш движения и действий
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
				// БЛОКИРОВКА ЛЕСТНИЦЫ БОССОМ:
				// Пока босс жив, игрок не может спуститься
				// Функции определены в level_boss.go
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

				// Поднимаемся на уровень выше
				g.prevLevel()

				// 🆕 ЭТАП 3: ПРОВЕРКА ПОБЕДЫ ПОСЛЕ ПОДЪЁМА
				// Основной путь к победе: подняться с уровня 2 на уровень 1.
				// Если игрок оказался на уровне 1 с Амулетом — ПОБЕДА!
				if g.depth == 1 && g.player.HasAmulet {
					g.logAndSync("VICTORY: Игрок вернулся на уровень 1 с Амулетом Бездны!")
					g.state = StateVictory
					return
				}

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
			// Функция GetMerchantAt определена в level_queries.go
			if merchant := g.level.GetMerchantAt(g.player.X, g.player.Y); merchant != nil {
				g.currentMerchant = merchant
				g.state = StateMerchant
				g.logAndSync("UI: Открыт экран торговли")
				return
			}
			g.addMessage("Здесь нет торговца.")
			return
		case 'P', 'p':
			// P — молитва на алтаре (если стоим на алтаре)
			if altar := g.level.GetAltarAt(g.player.X, g.player.Y); altar != nil {
				g.showAltarUI(altar)
				g.processTurn(0, 0) // Молитва тратит ход
				return
			}
			g.addMessage("Здесь нет алтаря.")
			return
		case 'O', 'o':
			// O — открыть сундук (если стоим на сундуке)
			if chest := g.level.GetChestAt(g.player.X, g.player.Y); chest != nil {
				g.openChest(chest)
				g.processTurn(0, 0) // Открытие тратит ход
				return
			}
			g.addMessage("Здесь нет сундука.")
			return
		}
	}

	// Если было движение, обрабатываем ход
	if dx != 0 || dy != 0 {
		g.logAndSync("MOVE: Игрок движется на dx=%d, dy=%d", dx, dy)
		g.processTurn(dx, dy)
	}
}