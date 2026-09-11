package game

import (
	"fmt"
	"math/rand/v2"
	"os"

	"github.com/gdamore/tcell/v2"
)

func (g *Game) handleHelpInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()
	keyEvent, ok := ev.(*tcell.EventKey)
	if !ok {
		return
	}
	if keyEvent.Key() == tcell.KeyLeft {
		if g.helpPage > helpPageControls {
			g.helpPage--
		}
		return
	}
	if keyEvent.Key() == tcell.KeyRight {
		if g.helpPage < helpPageCount-1 {
			g.helpPage++
		}
		return
	}
	r := keyEvent.Rune()
	if r == 'a' || r == 'A' {
		if g.helpPage > helpPageControls {
			g.helpPage--
		}
		return
	}
	if r == 'd' || r == 'D' {
		if g.helpPage < helpPageCount-1 {
			g.helpPage++
		}
		return
	}
	g.state = StatePlaying
}

func (g *Game) handleVictoryInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()
	switch ev.(type) {
	case *tcell.EventKey:
		g.quit = true
	}
}

func (g *Game) handleInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()
	switch ev := ev.(type) {
	case *blinkEvent:
		return
	case *tcell.EventKey:
		// 🆕 Если показан попап — закрываем по любой клавише (без return, чтобы клавиша сработала)
		if g.popupMessage != "" {
			g.popupMessage = ""
		}

		isEscape := ev.Key() == tcell.KeyEscape || ev.Rune() == 27
		isQuit := ev.Key() == tcell.KeyCtrlC || ev.Key() == tcell.KeyCtrlQ
		if isEscape || isQuit {
			if g.showInventory {
				g.showInventory = false
				g.addMessage("Инвентарь закрыт")
				return
			}
			g.state = StateQuitConfirm
			return
		}
		if g.showInventory {
			g.handleInventoryInput(ev.Rune())
		} else {
			g.handleMovement(ev.Rune(), ev.Key())
		}
	case *tcell.EventResize:
		g.screen.Sync()
	}
}

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
			g.loadGameFromMenu()
		} else if r == 'n' || r == 'N' {
			if _, err := os.Stat(saveFile); err == nil {
				os.Remove(saveFile)
			}
			g.startNewGame()
		} else if r == 'm' || r == 'M' {
			g.toggleMusic()
		}
	}
}

func (g *Game) handleQuitConfirmInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()
	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEscape || ev.Rune() == 27 || ev.Rune() == 'n' || ev.Rune() == 'N' {
			g.state = StatePlaying
			return
		}
		if ev.Rune() == 'y' || ev.Rune() == 'Y' || ev.Key() == tcell.KeyEnter {
			g.quit = true
			return
		}
	}
}

func (g *Game) handleDeathInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()
	switch ev := ev.(type) {
	case *tcell.EventKey:
		if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC ||
			ev.Rune() == 'y' || ev.Rune() == 'Y' ||
			ev.Rune() == 'q' || ev.Rune() == 'Q' {
			g.quit = true
			return
		}
		if ev.Rune() == 'n' || ev.Rune() == 'N' || ev.Key() == tcell.KeyEnter {
			if _, err := os.Stat(saveFile); err == nil {
				os.Remove(saveFile)
			}
			g.startNewGame()
			return
		}
	}
}

func (g *Game) handleInventoryInput(key rune) {
	if g.player == nil {
		return
	}
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
		g.useItem(index)
		return
	}
	if key == 27 || key == 'q' || key == 'Q' {
		g.showInventory = false
		g.addMessage("Инвентарь закрыт")
		return
	}
}

// =============================================================================
// 🆕 ИСПОЛЬЗОВАНИЕ ПРЕДМЕТОВ (С УМНЫМИ ПРОВЕРКАМИ И КОРРЕКТНЫМ АПГРЕЙДОМ)
// =============================================================================
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
			// 🆕 УМНАЯ ПРОВЕРКА СЫТОСТИ (нельзя есть, если сытость > 80%, то есть Hunger < 200)
			if g.player.Hunger < 200 {
				g.addMessage("Вам не надо есть, вы можете лопнуть!")
				return // не тратим еду и не тратим ход
			}
			// Уменьшаем голод на 50%
			g.player.Hunger -= g.player.Hunger / 2
			g.addMessage("Вы поели. Голод уменьшился.")
			g.logAndSync("ITEM_USE: Съедена еда. Голод: %d", g.player.Hunger)
		} else {
			// 🆕 УМНАЯ ПРОВЕРКА ЗДОРОВЬЯ (нельзя пить зелье, если HP > 90%)
			if g.player.MaxHP > 0 {
				hpPercent := float64(g.player.HP) / float64(g.player.MaxHP)
				if hpPercent > 0.90 {
					g.addMessage("Вы чувствуете себя отлично, зелье не требуется!")
					return // не тратим зелье и не тратим ход
				}
			}
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
		g.processTurn(0, 0)

	case ItemTypeWeapon:
		// ✅ АПГРЕЙД ИЛИ ЭКИПИРОВКА
		// Метод EquipWeapon в player.go сам проверяет:
		// - если слот пуст -> экипирует предмет (создает копию)
		// - если слот занят -> улучшает текущее оружие на +1
		g.player.EquipWeapon(item)
		
		// Безопасно расходуем предмет из инвентаря (уменьшаем Count или удаляем)
		g.consumeItem(index)
		
		g.addMessage(fmt.Sprintf("Оружие обработано: %s (Текущий ATK бонус: %d)", item.Name, g.player.EquippedWeapon.Value))
		g.logAndSync("ITEM_EQUIP/UPGRADE: Оружие %s", item.Name)
		g.processTurn(0, 0)

	case ItemTypeArmor:
		// ✅ АПГРЕЙД ИЛИ ЭКИПИРОВКА
		// Метод EquipArmor в player.go сам проверяет:
		// - если слот пуст -> экипирует предмет (создает копию)
		// - если слот занят -> улучшает текущую броню на +1
		g.player.EquipArmor(item)
		
		// Безопасно расходуем предмет из инвентаря
		g.consumeItem(index)
		
		g.addMessage(fmt.Sprintf("Броня обработана: %s (Текущий DEF бонус: %d)", item.Name, g.player.EquippedArmor.Value))
		g.logAndSync("ITEM_EQUIP/UPGRADE: Броня %s", item.Name)
		g.processTurn(0, 0)

	case ItemTypeScroll:
		g.useScroll(index)

	default:
		g.addMessage("Этот предмет нельзя использовать напрямую.")
	}
}

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
		g.player.Inventory = append(g.player.Inventory[:index], g.player.Inventory[index+1:]...)
	}
}

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
		if g.level != nil {
			newX, newY := g.level.FindFreeSpot()
			g.player.X = newX
			g.player.Y = newY
			g.addMessage("Вас телепортировало в другое место!")
			g.logAndSync("SCROLL: Телепортация на (%d, %d)", newX, newY)
		}
	case ScrollLightning:
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
		if g.level != nil && len(g.level.Monsters) > 0 {
			idx := rand.IntN(len(g.level.Monsters))
			m := g.level.Monsters[idx]
			if m != nil {
				g.level.RemoveMonster(m)
				g.player.Gold += m.GoldValue
				g.player.GainXP(m.XPValue)
				g.addMessage(fmt.Sprintf("Монстр %s изгнан в небытие!", m.Name))
				g.logAndSync("SCROLL: Изгнан монстр %s", m.Name)
			}
		} else {
			g.addMessage("На этом уровне нет монстров.")
		}
	}
	g.consumeItem(index)
	g.processTurn(0, 0)
}

func (g *Game) handleMovement(key rune, specialKey tcell.Key) {
	if g.player == nil || g.level == nil {
		return
	}
	dx, dy := 0, 0
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
	if key == ' ' {
		g.logAndSync("ACTION: Игрок ждет ход (пробел)")
		g.processTurn(0, 0)
		return
	}
	if dx == 0 && dy == 0 {
		switch key {
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
			g.logAndSync("ACTION: Игрок ждет ход (s)")
			g.processTurn(0, 0)
			return
		case 'i', 'I':
			g.showInventory = true
			g.logAndSync("UI: Открыт инвентарь")
			g.addMessage("Открыт инвентарь")
			return
		case 'm', 'M':
			g.toggleMusic()
			return
		case '+', '=':
			g.changeMusicVolume(0.1)
			return
		case '-', '_':
			g.changeMusicVolume(-0.1)
			return
		case '?':
			g.state = StateHelp
			g.helpPage = helpPageControls
			return
		case '>':
			if g.level.StairsDown &&
				g.player.X == g.level.StairsDownX &&
				g.player.Y == g.level.StairsDownY {
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
			if g.level.StairsUp &&
				g.player.X == g.level.StairsUpX &&
				g.player.Y == g.level.StairsUpY {
				g.prevLevel()
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
			g.saveGame()
			return
		case 'n', 'N':
			if _, err := os.Stat(saveFile); err == nil {
				os.Remove(saveFile)
			}
			g.startNewGame()
			return
		case 't', 'T':
			if merchant := g.level.GetMerchantAt(g.player.X, g.player.Y); merchant != nil {
				g.currentMerchant = merchant
				g.state = StateMerchant
				g.logAndSync("UI: Открыт экран торговли")
				return
			}
			g.addMessage("Здесь нет торговца.")
			return
		case 'P', 'p':
			if altar := g.level.GetAltarAt(g.player.X, g.player.Y); altar != nil {
				g.showAltarUI(altar)
				g.processTurn(0, 0)
				return
			}
			g.addMessage("Здесь нет алтаря.")
			return
		case 'O', 'o':
			if chest := g.level.GetChestAt(g.player.X, g.player.Y); chest != nil {
				g.openChest(chest)
				g.processTurn(0, 0)
				return
			}
			g.addMessage("Здесь нет сундука.")
			return
		}
	}
	if dx != 0 || dy != 0 {
		g.logAndSync("MOVE: Игрок движется на dx=%d, dy=%d", dx, dy)
		g.processTurn(dx, dy)
	}
}