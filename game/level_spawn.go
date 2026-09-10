package game

import (
	"math/rand"

	"github.com/gdamore/tcell/v2"
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
			x = 1 + rand.Intn(l.Width-2)
			y = 1 + rand.Intn(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor &&
				!l.hasMonsterAt(x, y) &&
				!l.hasItemAt(x, y) &&
				!l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > 100 {
				return
			}
		}

		mt := monsterTypes[rand.Intn(len(monsterTypes))]
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
			x = 1 + rand.Intn(l.Width-2)
			y = 1 + rand.Intn(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor &&
				!l.hasItemAt(x, y) &&
				!l.hasMonsterAt(x, y) &&
				!l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > 100 {
				return
			}
		}

		it := itemTypes[rand.Intn(len(itemTypes))]
		l.Items = append(l.Items, NewItem(x, y, it.name, it.itype, it.value, it.symbol, it.color))

		if l.logger != nil {
			l.logger.Printf("SPAWN_ITEM: %s на (%d, %d)", it.name, x, y)
		}
	}
}

// =============================================================================
// СПАВН НОВЫХ ОБЪЕКТОВ (ТОРГОВЦЫ, АЛТАРИ, СУНДУКИ)
// =============================================================================

// spawnMerchants — спавнит торговца на каждом 5-м уровне (5, 10, 15...).
// Торговец размещается в центре случайной комнаты.
func (l *Level) spawnMerchants(depth int) {
	if l == nil || depth%5 != 0 {
		return
	}
	if len(l.Rooms) == 0 {
		return
	}
	room := l.Rooms[rand.Intn(len(l.Rooms))]
	x := room.X + room.W/2
	y := room.Y + room.H/2

	merchant := NewMerchant(x, y, depth)
	l.Merchants = append(l.Merchants, merchant)

	if l.logger != nil {
		l.logger.Printf("SPAWN_MERCHANT: Торговец на (%d, %d) глубина %d", x, y, depth)
	}
}

// spawnAltars — спавнит алтарь на каждом уровне.
// Алтарь размещается в центре случайной комнаты.
func (l *Level) spawnAltars() {
	if l == nil || len(l.Rooms) == 0 {
		return
	}
	room := l.Rooms[rand.Intn(len(l.Rooms))]
	x := room.X + room.W/2
	y := room.Y + room.H/2

	altar := NewAltar(x, y)
	l.Altars = append(l.Altars, altar)

	if l.logger != nil {
		l.logger.Printf("SPAWN_ALTAR: Алтарь на (%d, %d)", x, y)
	}
}

// spawnChests — спавнит 1-2 обычных сундука на уровне.
// Сундуки размещаются на случайных клетках пола.
//
// 🆕 Золотые сундуки спавнятся отдельно (см. spawnGoldenChests).
func (l *Level) spawnChests() {
	if l == nil || l.Width < 3 || l.Height < 3 {
		return
	}
	count := 1 + rand.Intn(2)
	for i := 0; i < count; i++ {
		var x, y int
		attempts := 0
		for {
			x = 1 + rand.Intn(l.Width-2)
			y = 1 + rand.Intn(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor &&
				!l.hasItemAt(x, y) &&
				!l.hasMonsterAt(x, y) &&
				!l.isStairsAt(x, y) &&
				!l.hasChestAt(x, y) {
				break
			}
			attempts++
			if attempts > 100 {
				return
			}
		}

		chest := NewChest(x, y)
		l.Chests = append(l.Chests, chest)

		if l.logger != nil {
			l.logger.Printf("SPAWN_CHEST: Сундук на (%d, %d) содержимое: %s", x, y, chest.Contents)
		}
	}
}

// =============================================================================
// 🆕 ЭТАП 1: СПАВН РЕЛИКВИЙ, СВИТКОВ, КЛЮЧЕЙ, ЗОЛОТЫХ СУНДУКОВ
// =============================================================================

// 🆕 ЭТАП 1: Спавн реликвий
//
// Спавнит реликвию на уровне. Шанс спавна зависит от глубины:
//   - Уровни 1-4:  20% шанс
//   - Уровни 5-9:  30% шанс
//   - Уровни 10+:  40% шанс
//
// Тип реликвии выбирается с учётом редкости.
// Более редкие реликвии (Осколок звезды) имеют меньший шанс выпадения.
//
// Вызывается из NewLevel в level.go.
func (l *Level) spawnRelics(depth int) {
	if l == nil || l.Width < 3 || l.Height < 3 {
		return
	}

	// Определяем шанс спавна в зависимости от глубины
	spawnChance := 20
	if depth >= 5 {
		spawnChance = 30
	}
	if depth >= 10 {
		spawnChance = 40
	}

	// Проверяем, спавнить ли реликвию
	if rand.Intn(100) >= spawnChance {
		return
	}

	// Определяем тип реликвии с учётом редкости
	// Веса: Алмаз=30, Кубок=25, Корона=20, Статуэтка=15, Осколок=10
	// Итого: 100
	weights := []int{30, 25, 20, 15, 10}
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}
	roll := rand.Intn(totalWeight)
	relicID := 0
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if roll < cumulative {
			relicID = i
			break
		}
	}

	// Данные реликвий: имя, цена продажи, символ, цвет
	// Константы RelicDiamond и т.д. определены в item.go
	relicData := []struct {
		name      string
		sellPrice int
		symbol    rune
		color     tcell.Color
	}{
		{"Алмаз", 150, '*', tcell.ColorWhite},
		{"Золотой кубок", 200, '!', tcell.ColorYellow},
		{"Корона гоблинов", 300, ']', tcell.ColorGreen},
		{"Древняя статуэтка", 400, '/', tcell.ColorFuchsia},
		{"Осколок звезды", 500, '+', tcell.ColorAqua},
	}

	rd := relicData[relicID]

	// Ищем свободную клетку пола
	var x, y int
	attempts := 0
	for {
		x = 1 + rand.Intn(l.Width-2)
		y = 1 + rand.Intn(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor &&
			!l.hasItemAt(x, y) &&
			!l.hasMonsterAt(x, y) &&
			!l.isStairsAt(x, y) {
			break
		}
		attempts++
		if attempts > 100 {
			return
		}
	}

	// Создаём реликвию и добавляем на уровень
	// Конструктор NewRelic определён в item.go
	relic := NewRelic(x, y, relicID, rd.name, rd.sellPrice, rd.symbol, rd.color)
	l.Items = append(l.Items, relic)

	if l.logger != nil {
		l.logger.Printf("SPAWN_RELIC: %s (цена %d) на (%d, %d) глубина %d",
			rd.name, rd.sellPrice, x, y, depth)
	}
}

// 🆕 ЭТАП 1: Спавн свитков
//
// Спавнит 1-2 свитка на уровне. Тип свитка выбирается случайно из 4 вариантов.
// Свитки — одноразовые предметы с различными эффектами.
//
// Вызывается из NewLevel в level.go.
func (l *Level) spawnScrolls(depth int) {
	if l == nil || l.Width < 3 || l.Height < 3 {
		return
	}

	// Количество свитков: 1-2
	count := 1 + rand.Intn(2)

	// Данные свитков: тип, имя, символ, цвет
	// Константы ScrollMap и т.д. определены в item.go
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

	for i := 0; i < count; i++ {
		var x, y int
		attempts := 0
		for {
			x = 1 + rand.Intn(l.Width-2)
			y = 1 + rand.Intn(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor &&
				!l.hasItemAt(x, y) &&
				!l.hasMonsterAt(x, y) &&
				!l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > 100 {
				return
			}
		}

		// Конструктор NewScroll определён в item.go
		sd := scrollData[rand.Intn(len(scrollData))]
		scroll := NewScroll(x, y, sd.scrollType, sd.name, sd.symbol, sd.color)
		l.Items = append(l.Items, scroll)

		if l.logger != nil {
			l.logger.Printf("SPAWN_SCROLL: %s на (%d, %d) глубина %d",
				sd.name, x, y, depth)
		}
	}
}

// 🆕 ЭТАП 1: Спавн ключей
//
// Спавнит ключ от золотого сундука. Шанс спавна: 50% на уровень.
// Ключ нужен для открытия золотых сундуков (см. interact.go → openChest).
//
// Вызывается из NewLevel в level.go.
func (l *Level) spawnKeys(depth int) {
	if l == nil || l.Width < 3 || l.Height < 3 {
		return
	}

	// 50% шанс спавна ключа
	if rand.Intn(100) >= 50 {
		return
	}

	var x, y int
	attempts := 0
	for {
		x = 1 + rand.Intn(l.Width-2)
		y = 1 + rand.Intn(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor &&
			!l.hasItemAt(x, y) &&
			!l.hasMonsterAt(x, y) &&
			!l.isStairsAt(x, y) {
			break
		}
		attempts++
		if attempts > 100 {
			return
		}
	}

	// Ключ — предмет типа ItemTypeKey, не имеет эффекта (Value = 0)
	// Константа ItemTypeKey определена в item.go
	key := NewItem(x, y, "Ключ от сундука", ItemTypeKey, 0, 'k', tcell.ColorGray)
	l.Items = append(l.Items, key)

	if l.logger != nil {
		l.logger.Printf("SPAWN_KEY: Ключ от сундука на (%d, %d) глубина %d", x, y, depth)
	}
}

// 🆕 ЭТАП 1: Спавн золотых сундуков
//
// Спавнит золотой сундук на уровнях 3+. Шанс: 25% на уровень.
// Золотые сундуки содержат ценный лут, но требуют ключ для открытия.
//
// Вызывается из NewLevel в level.go.
func (l *Level) spawnGoldenChests(depth int) {
	if l == nil || l.Width < 3 || l.Height < 3 {
		return
	}

	// Золотые сундуки появляются только на уровнях 3+
	if depth < 3 {
		return
	}

	// 25% шанс спавна золотого сундука
	if rand.Intn(100) >= 25 {
		return
	}

	var x, y int
	attempts := 0
	for {
		x = 1 + rand.Intn(l.Width-2)
		y = 1 + rand.Intn(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor &&
			!l.hasItemAt(x, y) &&
			!l.hasMonsterAt(x, y) &&
			!l.isStairsAt(x, y) &&
			!l.hasChestAt(x, y) {
			break
		}
		attempts++
		if attempts > 100 {
			return
		}
	}

	// Конструктор NewGoldenChest определён в chest.go
	chest := NewGoldenChest(x, y)
	l.Chests = append(l.Chests, chest)

	if l.logger != nil {
		l.logger.Printf("SPAWN_GOLDEN_CHEST: Золотой сундук на (%d, %d) глубина %d", x, y, depth)
	}
}

// =============================================================================
// 🆕 ЭТАП 2: ВОЗРОЖДЕНИЕ УРОВНЕЙ
// =============================================================================
//
// Эти функции вызываются из respawnLevel в save.go при повторном посещении уровня.

// 🆕 ЭТАП 2: Возрождение монстров
//
// Возрождает монстров на уровне при повторном посещении.
//
// Механика:
//   - Монстров становится МЕНЬШЕ: половина от обычного количества
//   - Но их характеристики УВЕЛИЧИВАЮТСЯ: как на уровне depth+1
//   - Боссы НЕ возрождаются: живые боссы остаются, убитые не возвращаются
//
// Это делает повторное посещение уровней интересным:
// врагов меньше, но они сильнее. Игрок может вернуться за ресурсами,
// но встречает более серьёзное сопротивление.
//
// Пример:
//   Уровень 5, обычное количество монстров: 5 + 5 = 10
//   При возрождении: 10 / 2 = 5 монстров, но с характеристиками уровня 6
func (l *Level) respawnMonsters() {
	if l == nil || l.Width < 3 || l.Height < 3 {
		return
	}

	// Сохраняем живых боссов (они не возрождаются, остаются как есть)
	// Убитые боссы не возвращаются — лестница остаётся разблокированной
	newMonsters := make([]*Monster, 0)
	for _, m := range l.Monsters {
		if m != nil && m.IsBoss && m.HP > 0 {
			newMonsters = append(newMonsters, m)
		}
	}
	l.Monsters = newMonsters

	// Типы монстров с базовыми характеристиками (те же, что в spawnMonsters)
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

	// Количество монстров: половина от обычного (меньше монстров)
	// Обычное количество: 5 + depth
	// Возрождённое количество: (5 + depth) / 2
	count := (5 + l.Depth) / 2
	if count < 1 {
		count = 1 // минимум 1 монстр
	}

	// Характеристики масштабируются как на уровне depth+1 (сильнее)
	// Это делает возрождённых монстров опаснее обычных
	effectiveDepth := l.Depth + 1

	for i := 0; i < count; i++ {
		var x, y int
		attempts := 0
		for {
			x = 1 + rand.Intn(l.Width-2)
			y = 1 + rand.Intn(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor &&
				!l.hasMonsterAt(x, y) &&
				!l.hasItemAt(x, y) &&
				!l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > 100 {
				return
			}
		}

		mt := monsterTypes[rand.Intn(len(monsterTypes))]
		hp := mt.hp * (1 + effectiveDepth/2)
		attack := mt.attack * (1 + effectiveDepth/3)
		gold := mt.gold * effectiveDepth
		xp := mt.xp + effectiveDepth*2

		m := NewMonster(x, y, mt.name, hp, attack, gold, xp, mt.symbol, mt.color)
		m.SetLogger(l.logger)
		l.Monsters = append(l.Monsters, m)

		if l.logger != nil {
			l.logger.Printf("RESPAWN_MONSTER: %s (HP=%d ATK=%d Gold=%d XP=%d) на (%d, %d)",
				mt.name, hp, attack, gold, xp, x, y)
		}
	}
}

// 🆕 ЭТАП 2: Новое содержимое сундука
//
// Возвращает новое случайное содержимое для обычного сундука.
// Используется при возрождении сундуков (в respawnLevel в save.go).
//
// Возвращает одну из строк: "potion", "food", "monster".
// Золотые сундуки НЕ используют эту функцию — их содержимое остаётся "golden".
func newChestContents() string {
	contents := []string{"potion", "food", "monster"}
	return contents[rand.Intn(len(contents))]
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