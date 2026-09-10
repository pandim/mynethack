package game

import (
	"math/rand/v2"

	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// КОНСТАНТЫ СПАВНА
// =============================================================================
const (
	MaxSpawnAttempts = 100 // Максимальное количество попыток заспавнить объект на уровне
)

// =============================================================================
// СПАВН МОНСТРОВ И ПРЕДМЕТОВ
// =============================================================================
//
// spawnMonsters — создаёт монстров на уровне.
// Количество и характеристики монстров масштабируются по глубине уровня.
//
// Масштабирование:
//   - Количество монстров: 5 + depth
//   - HP: базовое * (1 + depth/2)
//   - Атака: базовая * (1 + depth/3)
//   - Золото: базовое * depth
//   - Опыт: базовый + depth*2
func (l *Level) spawnMonsters(count int) {
	if l == nil || count <= 0 || l.Width < 3 || l.Height < 3 {
		return
	}
	// Типы монстров с базовыми характеристиками
	monsterTypes := []struct {
		name   string
		hp     int
		attack int
		gold   int
		xp     int
		symbol rune
		color  tcell.Color
	}{
		{"Гоблин", 8, 2, 5, 8, 'g', tcell.ColorGreen},
		{"Орк", 12, 3, 10, 15, 'o', tcell.ColorDarkRed},
		{"Скелет", 10, 2, 8, 12, 's', tcell.ColorWhite},
		{"Крыса", 4, 1, 2, 3, 'r', tcell.ColorBrown},
	}

	depth := l.Depth
	for i := 0; i < count; i++ {
		// Ищем свободную клетку пола (не на лестнице, без монстров и предметов)
		var x, y int
		attempts := 0
		for {
			x = 1 + rand.IntN(l.Width-2)
			y = 1 + rand.IntN(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor &&
				!l.hasMonsterAt(x, y) &&
				!l.hasItemAt(x, y) &&
				!l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > MaxSpawnAttempts {
				return
			}
		}

		mt := monsterTypes[rand.IntN(len(monsterTypes))]
		hp := mt.hp * (1 + depth/2)
		attack := mt.attack * (1 + depth/3)
		gold := mt.gold * depth
		xp := mt.xp + depth*2

		m := NewMonster(x, y, mt.name, hp, attack, gold, xp, mt.symbol, mt.color)
		m.SetLogger(l.logger)
		l.Monsters = append(l.Monsters, m)

		if l.logger != nil {
			l.logger.Printf("SPAWN_MONSTER: %s (HP=%d ATK=%d Gold=%d XP=%d) на (%d, %d)",
				mt.name, hp, attack, gold, xp, x, y)
		}
	}
}

// spawnItems — создаёт предметы на уровне.
// Предметы размещаются на случайных клетках пола.
func (l *Level) spawnItems(count int) {
	if l == nil || count <= 0 || l.Width < 3 || l.Height < 3 {
		return
	}
	itemTypes := []struct {
		name   string
		itype  ItemType
		value  int
		symbol rune
		color  tcell.Color
	}{
		{"Зелье здоровья", ItemTypePotion, 10, '!', tcell.ColorRed},
		{"Меч", ItemTypeWeapon, 5, '/', tcell.ColorYellow},
		{"Щит", ItemTypeArmor, 3, '[', tcell.ColorBlue},
		{"Мешок золота", ItemTypeGold, 20, '$', tcell.ColorYellow},
		{"Еда", ItemTypePotion, 0, '%', tcell.ColorPurple},
	}

	for i := 0; i < count; i++ {
		var x, y int
		attempts := 0
		for {
			x = 1 + rand.IntN(l.Width-2)
			y = 1 + rand.IntN(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor &&
				!l.hasItemAt(x, y) &&
				!l.hasMonsterAt(x, y) &&
				!l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > MaxSpawnAttempts {
				return
			}
		}

		it := itemTypes[rand.IntN(len(itemTypes))]
		l.Items = append(l.Items, NewItem(x, y, it.name, it.itype, it.value, it.symbol, it.color))

		if l.logger != nil {
			l.logger.Printf("SPAWN_ITEM: %s на (%d, %d)", it.name, x, y)
		}
	}
}

// =============================================================================
// 🆕 ЭТАП 1: СПАВН ТОРГОВЦЕВ, АЛТАРЕЙ И СУНДУКОВ
// =============================================================================

// spawnMerchants — создаёт торговцев на каждом 5-м уровне.
func (l *Level) spawnMerchants(depth int) {
	if l == nil || depth%5 != 0 {
		return
	}

	var x, y int
	attempts := 0
	for {
		x = 1 + rand.IntN(l.Width-2)
		y = 1 + rand.IntN(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor &&
			!l.hasMonsterAt(x, y) &&
			!l.hasItemAt(x, y) &&
			!l.isStairsAt(x, y) {
			break
		}
		attempts++
		if attempts > MaxSpawnAttempts {
			return
		}
	}

	merchant := NewMerchant(x, y, depth)
	l.Merchants = append(l.Merchants, merchant)

	if l.logger != nil {
		l.logger.Printf("SPAWN_MERCHANT: на уровне %d в точке (%d, %d)", depth, x, y)
	}
}

// spawnAltars — создаёт алтари на каждом уровне.
func (l *Level) spawnAltars() {
	if l == nil {
		return
	}

	var x, y int
	attempts := 0
	for {
		x = 1 + rand.IntN(l.Width-2)
		y = 1 + rand.IntN(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor &&
			!l.hasMonsterAt(x, y) &&
			!l.hasItemAt(x, y) &&
			!l.isStairsAt(x, y) {
			break
		}
		attempts++
		if attempts > MaxSpawnAttempts {
			return
		}
	}

	altar := NewAltar(x, y)
	l.Altars = append(l.Altars, altar)

	if l.logger != nil {
		l.logger.Printf("SPAWN_ALTAR: на уровне %d в точке (%d, %d)", l.Depth, x, y)
	}
}

// spawnChests — создаёт сундуки на каждом уровне.
// 🆕 ЭТАП 1: 20% шанс, что сундук будет золотым (требует ключ).
func (l *Level) spawnChests() {
	if l == nil {
		return
	}

	var x, y int
	attempts := 0
	for {
		x = 1 + rand.IntN(l.Width-2)
		y = 1 + rand.IntN(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor &&
			!l.hasMonsterAt(x, y) &&
			!l.hasItemAt(x, y) &&
			!l.isStairsAt(x, y) &&
			!l.hasChestAt(x, y) {
			break
		}
		attempts++
		if attempts > MaxSpawnAttempts {
			return
		}
	}

	// 🆕 ЭТАП 1: 20% шанс на золотой сундук
	isGolden := rand.IntN(100) < 20

	// ИСПРАВЛЕНО: создаем сундук с 2 аргументами, затем устанавливаем флаг
	chest := NewChest(x, y)
	chest.IsGolden = isGolden 
	l.Chests = append(l.Chests, chest)

	if l.logger != nil {
		chestType := "обычный"
		if isGolden {
			chestType = "золотой"
		}
		l.logger.Printf("SPAWN_CHEST: %s сундук на уровне %d в точке (%d, %d)", chestType, l.Depth, x, y)
	}
}