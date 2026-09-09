package game

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// ЭКРАН ТОРГОВЛИ
// =============================================================================
//
// Открывается при нажатии T на торговце. Показывает список товаров
// и позволяет купить предмет по номеру.

// renderMerchantScreen — отрисовка экрана торговли
//
// 🆕 ЭТАП 1: В нижней части экрана показываются реликвии из инвентаря игрока.
// Игрок может продать реликвию, нажав соответствующую букву (A-E).
// Цена продажи хранится в поле Value реликвии.
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
		if y >= screenHeight-8 { // оставляем место для секции продажи
			break
		}
	}

	// 🆕 ЭТАП 1: Секция продажи реликций
	// Собираем реликвии из инвентаря
	// Константа ItemTypeRelic определена в item.go
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
			if i >= 5 { // максимум 5 реликций (A-E)
				break
			}
			if y >= screenHeight-4 {
				break
			}
			letter := rune('A' + i)
			text := fmt.Sprintf("%c. %s", letter, relic.Name)
			g.drawString(5, y, text, style)
			// Цена продажи реликвии хранится в Value
			priceText := fmt.Sprintf("%d золота", relic.Value)
			g.drawString(55, y, priceText, priceStyle)
			y++
		}
	}

	if len(relics) > 0 {
		g.drawCentered(screenHeight-2, "Цифра — купить, буква — продать реликвию, ESC — возврат", style)
	} else {
		g.drawCentered(screenHeight-2, "Нажмите цифру для покупки, ESC для возврата", style)
	}

	g.screen.Show()
}

// handleMerchantInput — обработка ввода на экране торговли
//
// 🆕 ЭТАП 1: Буквы A-E (или a-e) — продажа реликвии
// Цифры 1-9 — покупка предмета
// ESC или Q — возврат в игру
func (g *Game) handleMerchantInput() {
	if g.screen == nil {
		return
	}
	ev := g.screen.PollEvent()

	switch ev := ev.(type) {
	case *tcell.EventKey:
		// ESC или Q — возврат в игру
		if ev.Key() == tcell.KeyEscape || ev.Rune() == 'q' || ev.Rune() == 'Q' {
			g.state = StatePlaying
			g.currentMerchant = nil
			return
		}
		// Цифры 1-9 — покупка предмета
		if ev.Rune() >= '1' && ev.Rune() <= '9' {
			index := int(ev.Rune() - '1')
			g.buyItem(index)
			return
		}
		// 🆕 ЭТАП 1: Буквы A-E — продажа реликвии
		// Нормализуем регистр: приводим к нижнему
		r := ev.Rune()
		if r >= 'A' && r <= 'E' {
			r = r - 'A' + 'a'
		}
		if r >= 'a' && r <= 'e' {
			relicListIndex := int(r - 'a') // индекс реликвии в списке (0-4)
			g.sellRelic(relicListIndex)
			return
		}
	}
}

// buyItem — покупка предмета у торговца по номеру
//
// ⚠️ Цена берётся из поля Price (не из Value, чтобы не путать с эффектом).
// Купленный предмет добавляется в инвентарь с учётом стопок.
// Функция стопок определена в combat.go → addToInventoryWithStack.
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

	g.addMessage(fmt.Sprintf("Куплено %s за %d золота.", item.Name, item.Price))
	g.logAndSync("MERCHANT: Куплено %s за %d золота", item.Name, item.Price)

	if len(g.currentMerchant.Items) == 0 {
		g.state = StatePlaying
		g.currentMerchant = nil
		g.addMessage("Товары закончились!")
	}
}

// 🆕 ЭТАП 1: Продажа реликвии
//
// Продаёт реликвию из инвентаря торговцу.
// Реликвия выбирается по индексу в списке реликвий (0-4).
// Цена продажи хранится в поле Value реликвии.
//
// После продажи реликвия удаляется из инвентаря,
// а золото добавляется в кошелёк игрока.
//
// Вызывается из handleMerchantInput при нажатии буквы A-E.
func (g *Game) sellRelic(relicListIndex int) {
	if g.player == nil {
		return
	}

	// Находим реликвию по индексу в списке реликвий
	// (не по индексу в инвентаре, а по порядку среди реликвий)
	// Константа ItemTypeRelic определена в item.go
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
	sellPrice := relic.Value // цена продажи хранится в Value

	// Продаём реликвию
	g.player.Gold += sellPrice
	// Удаляем реликвию из инвентаря
	// Реликвии не стакаются, так что просто удаляем
	g.player.Inventory = append(g.player.Inventory[:relicInventoryIndex],
		g.player.Inventory[relicInventoryIndex+1:]...)

	g.addMessage(fmt.Sprintf("Продано %s за %d золота.", relic.Name, sellPrice))
	g.logAndSync("MERCHANT: Продано %s за %d золота", relic.Name, sellPrice)
}

// =============================================================================
// ВЗАИМОДЕЙСТВИЕ С АЛТАРЁМ И СУНДУКОМ
// =============================================================================

// showAltarUI — благословение экипировки на алтаре.
// Тратит 100 золота на улучшение оружия или брони на +2.
//
// ⚠️ АЛТАРЬ ОДНОРАЗОВЫЙ: после использования устанавливается Used = true,
// и повторное благословение невозможно. Это предотвращает бесконечное улучшение.
func (g *Game) showAltarUI(altar *Altar) {
	if altar == nil {
		return
	}

	if altar.Used {
		g.addMessage("Алтарь уже использован. Он потускнел...")
		return
	}

	cost := 100

	if g.player.Gold < cost {
		g.addMessage(fmt.Sprintf("Нужно %d золота для благословения!", cost))
		return
	}

	if g.player.EquippedWeapon != nil {
		g.player.Gold -= cost
		g.player.EquippedWeapon.Value += 2
		g.player.AttackVal += 2
		altar.Used = true
		g.addMessage(fmt.Sprintf("Оружие благословлено! ATK +%d (потрачено %d золота)",
			g.player.EquippedWeapon.Value, cost))
		g.logAndSync("ALTAR: Оружие благословлено. ATK +%d", g.player.EquippedWeapon.Value)
	} else if g.player.EquippedArmor != nil {
		g.player.Gold -= cost
		g.player.EquippedArmor.Value += 2
		g.player.Defense += 2
		altar.Used = true
		g.addMessage(fmt.Sprintf("Броня благословлена! DEF +%d (потрачено %d золота)",
			g.player.EquippedArmor.Value, cost))
		g.logAndSync("ALTAR: Броня благословлена. DEF +%d", g.player.EquippedArmor.Value)
	} else {
		g.addMessage("Сначала экипируйте оружие или броню!")
	}
}

// openChest — открытие сундука.
// Содержимое может быть: зелье, еда, монстр-ловушка.
//
// 🆕 ЭТАП 1: Золотые сундуки требуют ключ для открытия.
// Золотой сундук содержит: реликвию (50%), свиток (30%), золото (20%).
//
// ⚠️ Зелья и еда из сундука добавляются в инвентарь С УЧЁТОМ СТОПОК.
// Функция стопок определена в combat.go → addToInventoryWithStack.
//
// 🆕 Для золотых сундуков:
//   - Проверяем наличие ключа в инвентаре (тип ItemTypeKey)
//   - Если ключ есть — расходуем его и открываем сундук
//   - Если ключа нет — показываем сообщение и не открываем
//
// Содержимое золотого сундука определяется случайно:
//   - 50% шанс: реликвия (случайная из 5 типов)
//   - 30% шанс: свиток (случайный из 4 типов)
//   - 20% шанс: большое количество золота (50-100 монет)
func (g *Game) openChest(chest *Chest) {
	if chest == nil {
		return
	}
	if chest.Opened {
		g.addMessage("Сундук уже открыт.")
		return
	}

	// 🆕 ЭТАП 1: Золотой сундук требует ключ для открытия
	// Константа ItemTypeKey определена в item.go
	if chest.IsGolden {
		// Ищем ключ в инвентаре
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

		// Расходуем ключ
		// Ключи стакаются, так что уменьшаем Count
		g.player.Inventory[keyIndex].Count--
		if g.player.Inventory[keyIndex].Count <= 0 {
			g.player.Inventory = append(g.player.Inventory[:keyIndex],
				g.player.Inventory[keyIndex+1:]...)
		}
		g.logAndSync("CHEST: Ключ использован для открытия золотого сундука")
	}

	// Открываем сундук
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
		// Спавним монстра-ловушку
		monster := NewMonster(
			chest.X, chest.Y,
			"Ловушка",
			15, 5, 10, 20,
			't', tcell.ColorRed,
		)
		monster.SetLogger(g.logger)
		g.level.Monsters = append(g.level.Monsters, monster)
		g.addMessage("В сундуке была ловушка! Монстр атакует!")
		g.logAndSync("CHEST: Спавнен монстр-ловушка на (%d, %d)", chest.X, chest.Y)

	// 🆕 ЭТАП 1: Золотой сундук с ценным лутом
	// Содержимое определяется случайно:
	//   - 50% шанс: реликвия (случайная из 5 типов)
	//   - 30% шанс: свиток (случайный из 4 типов)
	//   - 20% шанс: большое количество золота (50-100 монет)
	//
	// Константы RelicDiamond, ScrollMap и т.д. определены в item.go.
	// Конструкторы NewRelic, NewScroll определены в item.go.
	case "golden":
		// Используем math/rand для случайного выбора содержимого
		// (импортирован в level_spawn.go, но доступен во всём пакете)
		roll := rand.Intn(100)

		if roll < 50 {
			// 50% шанс: реликвия
			// Выбираем случайную реликвию из 5 типов
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
			rd := relicData[rand.Intn(len(relicData))]
			relic := NewRelic(0, 0, rd.relicID, rd.name, rd.sellPrice, rd.symbol, rd.color)
			// Реликвии не стакаются — добавляем напрямую
			// (не через addToInventoryWithStack)
			g.player.Inventory = append(g.player.Inventory, relic)
			g.addMessage(fmt.Sprintf("В золотом сундуке найдена реликвия: %s!", rd.name))
			g.logAndSync("CHEST: Найдена реликвия %s в золотом сундуке", rd.name)

		} else if roll < 80 {
			// 30% шанс: свиток
			// Выбираем случайный свиток из 4 типов
			scrollData := []struct {
				scrollType int
				name       string
				symbol     rune
				color      tcell.Color
			}{
				{ScrollMap, "Свиток карты", '?', tcell.ColorWhite},
				{ScrollTeleport, "Свиток телепортации", '?', tcell.ColorFuchsia},
				{ScrollLightning, "Свиток молнии", '?', tcell.ColorYellow},
				{ScrollBanishment, "Свиток изгнания", '?', tcell.ColorRed},
			}
			sd := scrollData[rand.Intn(len(scrollData))]
			scroll := NewScroll(0, 0, sd.scrollType, sd.name, sd.symbol, sd.color)
			// Свитки стакаются — используем addToInventoryWithStack
			g.addToInventoryWithStack(scroll)
			g.addMessage(fmt.Sprintf("В золотом сундуке найден свиток: %s!", sd.name))
			g.logAndSync("CHEST: Найден свиток %s в золотом сундуке", sd.name)

		} else {
			// 20% шанс: большое количество золота (50-100 монет)
			goldAmount := 50 + rand.Intn(51) // 50-100
			g.player.Gold += goldAmount
			g.addMessage(fmt.Sprintf("В золотом сундуке найдено %d золота!", goldAmount))
			g.logAndSync("CHEST: Найдено %d золота в золотом сундуке", goldAmount)
		}
	}
}