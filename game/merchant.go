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

// =============================================================================
// СОЗДАНИЕ ТОРГОВЦА
// =============================================================================
//
// Создаёт торговца с набором товаров. Цены масштабируются по глубине уровня.
//
// ⚠️ ВАЖНО: цена хранится в поле `Price`, а эффект предмета — в поле `Value`.
// Это позволяет зелью лечить на 10, даже если его цена 35 золота.
func NewMerchant(x, y, depth int) *Merchant {
	basePrices := []struct {
		name   string
		itype  ItemType
		value  int // эффект предмета (лечение, бонус)
		price  int // базовая цена в золоте
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
		// Масштабируем цену по глубине
		price := bp.price + depth*5
		// Создаём предмет с эффектом в Value
		item := NewItem(0, 0, bp.name, bp.itype, bp.value, bp.symbol, bp.color)
		// Цену храним отдельно в Price (не перезаписываем Value!)
		item.Price = price
		items = append(items, item)
	}

	return &Merchant{
		X:     x,
		Y:     y,
		Items: items,
	}
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