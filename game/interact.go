package game

import (
	"fmt"
	"math/rand/v2"
	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// ЭКРАН ТОРГОВЛИ
// =============================================================================
func (g *Game) renderMerchantScreen() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()

	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	priceStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)

	g.drawCentered(2, "=== ТОРГОВЕЦ ===", titleStyle)
	g.drawCentered(3, fmt.Sprintf("Ваше золото: %d", g.player.Gold), priceStyle)

	if g.currentMerchant == nil || len(g.currentMerchant.Items) == 0 {
		g.drawCentered(5, "Товары закончились!", style)
		g.drawCentered(7, "Нажмите ESC для возврата", style)
		g.screen.Show()
		return
	}

	y := 5
	for i, item := range g.currentMerchant.Items {
		if item == nil {
			continue
		}
		text := fmt.Sprintf("%d. %s", i+1, item.Name)
		switch item.Type {
		case ItemTypePotion:
			if item.Name == "Еда" {
				text += " (утоляет голод)"
			} else {
				text += fmt.Sprintf(" (+%d HP)", item.Value)
			}
		case ItemTypeWeapon:
			text += fmt.Sprintf(" (ATK +%d)", item.Value)
		case ItemTypeArmor:
			text += fmt.Sprintf(" (DEF +%d)", item.Value)
		}
		g.drawString(5, y, text, style)
		priceText := fmt.Sprintf("%d золота", item.Price)
		g.drawString(55, y, priceText, priceStyle)
		y++
		if y >= screenHeight-8 { 
			break
		}
	}

	relics := make([]*Item, 0)
	for _, item := range g.player.Inventory {
		if item != nil && item.Type == ItemTypeRelic {
			relics = append(relics, item)
		}
	}

	if len(relics) > 0 {
		y += 1
		g.drawString(5, y, "--- Ваши реликвии для продажи ---", style)
		y++
		for i, relic := range relics {
			if i >= 5 { 
				break
			}
			if y >= screenHeight-4 {
				break
			}
			letter := rune('A' + i)
			text := fmt.Sprintf("%c. %s", letter, relic.Name)
			g.drawString(5, y, text, style)
			priceText := fmt.Sprintf("%d золота", relic.Value)
			g.drawString(55, y, priceText, priceStyle)
			y++
		}
	}

	// 🆕 НИЖНЯЯ ПОДСКАЗКА С УЧЕТОМ ПОДТВЕРЖДЕНИЯ
	if g.pendingRelicSellIndex >= 0 {
		warnStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
		g.drawCentered(screenHeight-2, "⚠ Продать эту уникальную реликвию? [Y] Да / [N] Нет ⚠", warnStyle)
	} else if len(relics) > 0 {
		g.drawCentered(screenHeight-2, "Цифра — купить, буква — продать реликвию, ESC — возврат", style)
	} else {
		g.drawCentered(screenHeight-2, "Нажмите цифру для покупки, ESC для возврата", style)
	}

	g.screen.Show()
}

// handleMerchantInput — обработка ввода на экране торговли
func (g *Game) handleMerchantInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev := ev.(type) {
	case *tcell.EventKey:
		// 🆕 ОБРАБОТКА ПОДТВЕРЖДЕНИЯ ПРОДАЖИ РЕЛИКВИИ
		if g.pendingRelicSellIndex >= 0 {
			if ev.Key() == tcell.KeyEnter || ev.Rune() == 'y' || ev.Rune() == 'Y' {
				g.sellRelic(g.pendingRelicSellIndex)
				g.pendingRelicSellIndex = -1 // Сбрасываем флаг
				return
			}
			if ev.Key() == tcell.KeyEscape || ev.Rune() == 'n' || ev.Rune() == 'N' {
				g.pendingRelicSellIndex = -1 // Сбрасываем флаг
				g.addMessage("Продажа отменена.")
				return
			}
			return // Игнорируем все остальные клавиши во время подтверждения
		}

		// ESC или Q — возврат в игру
		if ev.Key() == tcell.KeyEscape || ev.Rune() == 'q' || ev.Rune() == 'Q' {
			g.state = StatePlaying
			g.currentMerchant = nil
			g.pendingRelicSellIndex = -1
			return
		}

		// Цифры 1-9 — покупка предмета
		if ev.Rune() >= '1' && ev.Rune() <= '9' {
			index := int(ev.Rune() - '1')
			g.buyItem(index)
			return
		}

		// 🆕 Буквы A-E — ЗАПРОС на продажу реликвии (не мгновенная продажа)
		r := ev.Rune()
		if r >= 'A' && r <= 'E' {
			r = r - 'A' + 'a'
		}
		if r >= 'a' && r <= 'e' {
			relicListIndex := int(r - 'a') 
			g.pendingRelicSellIndex = relicListIndex // Устанавливаем флаг ожидания
			return
		}
	}
}

func (g *Game) buyItem(index int) {
	if g.currentMerchant == nil || index < 0 || index >= len(g.currentMerchant.Items) {
		return
	}
	item := g.currentMerchant.Items[index]
	if item == nil {
		return
	}
	if g.player.Gold < item.Price {
		g.addMessage(fmt.Sprintf("Недостаточно золота! Нужно %d.", item.Price))
		return
	}
	g.player.Gold -= item.Price
	g.addToInventoryWithStack(item)
	g.currentMerchant.Items = append(g.currentMerchant.Items[:index], g.currentMerchant.Items[index+1:]...)
	g.addMessage(fmt.Sprintf("Вы купили %s за %d золота.", item.Name, item.Price))
	g.logAndSync("MERCHANT: Вы купили %s за %d золота", item.Name, item.Price)

	if len(g.currentMerchant.Items) == 0 {
		g.state = StatePlaying
		g.currentMerchant = nil
		g.addMessage("Товары закончились!")
	}
}

func (g *Game) sellRelic(relicListIndex int) {
	if g.player == nil {
		return
	}
	relicCount := 0
	relicInventoryIndex := -1
	for i, item := range g.player.Inventory {
		if item != nil && item.Type == ItemTypeRelic {
			if relicCount == relicListIndex {
				relicInventoryIndex = i
				break
			}
			relicCount++
		}
	}

	if relicInventoryIndex < 0 {
		g.addMessage("Нет реликвии для продажи!")
		return
	}

	relic := g.player.Inventory[relicInventoryIndex]
	sellPrice := relic.Value 

	g.player.Gold += sellPrice
	g.player.Inventory = append(g.player.Inventory[:relicInventoryIndex], g.player.Inventory[relicInventoryIndex+1:]...)
	
	g.addMessage(fmt.Sprintf("Вы продали %s за %d золота.", relic.Name, sellPrice))
	g.logAndSync("MERCHANT: Вы продали %s за %d золота", relic.Name, sellPrice)
}

// =============================================================================
// ВЗАИМОДЕЙСТВИЕ С АЛТАРЁМ И СУНДУКОМ
// =============================================================================
func (g *Game) showAltarUI(altar *Altar) {
	if altar == nil {
		return
	}
	if altar.Used {
		g.addMessage("Алтарь уже использован. Он потускнел...")
		return
	}

	// 🆕 МАСШТАБИРОВАНИЕ ЦЕНЫ АЛТАРЯ ОТ ГЛУБИНЫ
	cost := 100 + g.depth*20 

	if g.player.Gold < cost {
		g.addMessage(fmt.Sprintf("Нужно %d золота для благословения!", cost))
		return
	}

	if g.player.EquippedWeapon != nil {
		g.player.Gold -= cost
		g.player.EquippedWeapon.Value += 2
		g.player.AttackVal += 2
		altar.Used = true
		g.addMessage(fmt.Sprintf("Оружие благословлено! ATK +%d (потрачено %d золота)", g.player.EquippedWeapon.Value, cost))
		g.logAndSync("ALTAR: Оружие благословлено. ATK +%d", g.player.EquippedWeapon.Value)
	} else if g.player.EquippedArmor != nil {
		g.player.Gold -= cost
		g.player.EquippedArmor.Value += 2
		g.player.Defense += 2
		altar.Used = true
		g.addMessage(fmt.Sprintf("Броня благословлена! DEF +%d (потрачено %d золота)", g.player.EquippedArmor.Value, cost))
		g.logAndSync("ALTAR: Броня благословлена. DEF +%d", g.player.EquippedArmor.Value)
	} else {
		g.addMessage("Сначала экипируйте оружие или броню!")
	}
}

func (g *Game) openChest(chest *Chest) {
	if chest == nil {
		return
	}
	if chest.Opened {
		g.addMessage("Сундук уже открыт.")
		return
	}


	if chest.IsGolden {
		g.addMessage("Золотой сундук открыт!") // 🆕 ДОБАВЬТЕ ЭТУ СТРОКУ
		keyIndex := -1
		for i, item := range g.player.Inventory {
			if item != nil && item.Type == ItemTypeKey {
				keyIndex = i
				break
			}
		}
		if keyIndex == -1 {
			g.addMessage("Нужен ключ для открытия золотого сундука!")
			g.logAndSync("CHEST: Попытка открыть золотой сундук без ключа")
			return
		}
		g.player.Inventory[keyIndex].Count--
		if g.player.Inventory[keyIndex].Count <= 0 {
			g.player.Inventory = append(g.player.Inventory[:keyIndex], g.player.Inventory[keyIndex+1:]...)
		}
		g.logAndSync("CHEST: Ключ использован для открытия золотого сундука")
	}

	chest.Opened = true
	switch chest.Contents {
	case "potion":
		potion := NewItem(0, 0, "Зелье здоровья", ItemTypePotion, 15, '!', tcell.ColorRed)
		g.addToInventoryWithStack(potion)
		g.addMessage("В сундуке найдено Зелье здоровья!")
		g.logAndSync("CHEST: Найдено Зелье здоровья")
	case "food":
		food := NewItem(0, 0, "Еда", ItemTypePotion, 0, '%', tcell.ColorPurple)
		g.addToInventoryWithStack(food)
		g.addMessage("В сундуке найдена Еда!")
		g.logAndSync("CHEST: Найдена Еда")
	case "monster":
		monster := NewMonster(chest.X, chest.Y, "Ловушка", 15, 5, 10, 20, 't', tcell.ColorRed)
		monster.SetLogger(g.logger)
		g.level.Monsters = append(g.level.Monsters, monster)
		g.addMessage("В сундуке была ловушка! Монстр атакует!")
		g.logAndSync("CHEST: Спавнен монстр-ловушка на (%d, %d)", chest.X, chest.Y)
	case "golden":
		roll := rand.IntN(100)
		if roll < 50 {
			relicData := []struct {
				relicID   int
				name      string
				sellPrice int
				symbol    rune
				color     tcell.Color
			}{
				{RelicDiamond, "Алмаз", 150, '*', tcell.ColorWhite},
				{RelicChalice, "Золотой кубок", 200, '!', tcell.ColorYellow},
				{RelicCrown, "Корона гоблинов", 300, ']', tcell.ColorGreen},
				{RelicStatuette, "Древняя статуэтка", 400, '/', tcell.ColorFuchsia},
				{RelicStarShard, "Осколок звезды", 500, '+', tcell.ColorAqua},
			}
			rd := relicData[rand.IntN(len(relicData))]
			relic := NewRelic(0, 0, rd.relicID, rd.name, rd.sellPrice, rd.symbol, rd.color)
			g.player.Inventory = append(g.player.Inventory, relic)
			g.addMessage(fmt.Sprintf("В золотом сундуке найдена реликвия: %s!", rd.name))
			g.logAndSync("CHEST: Найдена реликвия %s в золотом сундуке", rd.name)
		} else if roll < 80 {
			// 🆕 ВИЗУАЛ СВИТКОВ: символ '~' и уникальные цвета
			scrollData := []struct {
				scrollType int
				name       string
				symbol     rune
				color      tcell.Color
			}{
				{ScrollMap, "Свиток карты", '~', tcell.ColorWhite},
				{ScrollTeleport, "Свиток телепортации", '~', tcell.ColorAqua},
				{ScrollLightning, "Свиток молнии", '~', tcell.ColorYellow},
				{ScrollBanishment, "Свиток изгнания", '~', tcell.ColorRed},
			}
			sd := scrollData[rand.IntN(len(scrollData))]
			scroll := NewScroll(0, 0, sd.scrollType, sd.name, sd.symbol, sd.color)
			g.addToInventoryWithStack(scroll)
			g.addMessage(fmt.Sprintf("В золотом сундуке найден свиток: %s!", sd.name))
			g.logAndSync("CHEST: Найден свиток %s в золотом сундуке", sd.name)
		} else {
			goldAmount := 50 + rand.IntN(51) 
			g.player.Gold += goldAmount
			g.addMessage(fmt.Sprintf("В золотом сундуке найдено %d золота!", goldAmount))
			g.logAndSync("CHEST: Найдено %d золота в золотом сундуке", goldAmount)
		}
	}
}