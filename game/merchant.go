package game

import (
	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// СТРУКТУРА ТОРГОВЦА
// =============================================================================
//
// NPC-торговец, который появляется на каждом 5-м уровне (5, 10, 15...).
// Продаёт предметы за золото.
type Merchant struct {
	X, Y  int     // координаты на карте
	Items []*Item // товары на продажу
}

// 🆕 ЭТАП 2: Новый товар торговца
//
// Создаёт новый набор товаров для торговца.
// Используется при возрождении торговцев (в respawnLevel в save.go).
//
// Цены масштабируются по глубине уровня И по количеству посещений.
// При каждом повторном посещении цены удваиваются.
//
// ⚠️ ВАЖНО: цена хранится в поле `Price`, а эффект предмета — в поле `Value`.
// Это позволяет зелью лечить на 10, даже если его цена 35 золота.
//
// 🆕 Параметр visitCount: количество посещений уровня
// VisitCount = 1 — первое посещение (обычные цены)
// VisitCount = 2 — второе посещение (цены ×2)
// VisitCount = 3 — третье посещение (цены ×4)
// и т.д.
func newMerchantItems(depth int, visitCount int) []*Item {
	basePrices := []struct {
		name   string
		itype  ItemType
		value  int
		price  int
		symbol rune
		color  tcell.Color
	}{
		{"Зелье здоровья", ItemTypePotion, 10, 30, '!', tcell.ColorRed},
		{"Еда", ItemTypePotion, 0, 15, '%', tcell.ColorPurple},
		{"Меч", ItemTypeWeapon, 5, 50, '/', tcell.ColorYellow},
		{"Щит", ItemTypeArmor, 3, 40, '[', tcell.ColorBlue},
	}

	items := make([]*Item, 0)
	for _, bp := range basePrices {
		// Базовая цена зависит от глубины
		basePrice := bp.price + depth*5
		
		// 🆕 Удваиваем цену при каждом повторном посещении
		// visitCount=1: множитель=1 (первое посещение)
		// visitCount=2: множитель=2 (второе посещение)
		// visitCount=3: множитель=4 (третье посещение)
		multiplier := 1
		for i := 1; i < visitCount; i++ {
			multiplier *= 2
		}
		price := basePrice * multiplier
		
		item := NewItem(0, 0, bp.name, bp.itype, bp.value, bp.symbol, bp.color)
		item.Price = price
		items = append(items, item)
	}

	return items
}

// =============================================================================
// ОТРИСОВКА ТОРГОВЦА
// =============================================================================
func (m *Merchant) Render(screen tcell.Screen, offsetX, offsetY int) {
	if m == nil || screen == nil {
		return
	}
	style := tcell.StyleDefault.
		Foreground(tcell.ColorAqua).
		Background(tcell.ColorBlack)
	screen.SetContent(m.X+offsetX, m.Y+offsetY, 'M', nil, style)
}