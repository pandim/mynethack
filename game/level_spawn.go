package game

import (
	"math/rand/v2"

	"github.com/gdamore/tcell/v2"
)

const (
	MaxSpawnAttempts = 100
)

func (l *Level) spawnMonsters(count int) {
	if l == nil || count <= 0 || l.Width < 3 || l.Height < 3 {
		return
	}
	monsterTypes := []struct {
		name           string
		hp, attack, gold, xp int
		symbol         rune
		color          tcell.Color
	}{
		{"Гоблин", 8, 2, 5, 8, 'g', tcell.ColorGreen},
		{"Орк", 12, 3, 10, 15, 'o', tcell.ColorDarkRed},
		{"Скелет", 10, 2, 8, 12, 's', tcell.ColorWhite},
		{"Крыса", 4, 1, 2, 3, 'r', tcell.ColorBrown},
	}
	depth := l.Depth
	for i := 0; i < count; i++ {
		var x, y int
		attempts := 0
		for {
			x = 1 + rand.IntN(l.Width-2)
			y = 1 + rand.IntN(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor && !l.hasMonsterAt(x, y) && !l.hasItemAt(x, y) && !l.isStairsAt(x, y) {
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
	}
}

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
			if l.Tiles[y][x].Type == TileFloor && !l.hasItemAt(x, y) && !l.hasMonsterAt(x, y) && !l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > MaxSpawnAttempts {
				return
			}
		}
		it := itemTypes[rand.IntN(len(itemTypes))]
		l.Items = append(l.Items, NewItem(x, y, it.name, it.itype, it.value, it.symbol, it.color))
	}
}

func (l *Level) spawnMerchants(depth int) {
	if l == nil {
		return
	}
	// 🆕 Торговцы на уровнях: 2, 5, 8, 11, 14
	if depth != 2 && depth != 5 && depth != 8 && depth != 11 && depth != 14 {
		return
	}
	var x, y int
	attempts := 0
	for {
		x = 1 + rand.IntN(l.Width-2)
		y = 1 + rand.IntN(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor && !l.hasMonsterAt(x, y) && !l.hasItemAt(x, y) && !l.isStairsAt(x, y) {
			break
		}
		attempts++
		if attempts > MaxSpawnAttempts {
			return
		}
	}
	merchant := NewMerchant(x, y, depth)
	l.Merchants = append(l.Merchants, merchant)
}

func (l *Level) spawnAltars() {
	if l == nil {
		return
	}
	var x, y int
	attempts := 0
	for {
		x = 1 + rand.IntN(l.Width-2)
		y = 1 + rand.IntN(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor && !l.hasMonsterAt(x, y) && !l.hasItemAt(x, y) && !l.isStairsAt(x, y) {
			break
		}
		attempts++
		if attempts > MaxSpawnAttempts {
			return
		}
	}
	altar := NewAltar(x, y)
	l.Altars = append(l.Altars, altar)
}

func (l *Level) spawnChests() {
	if l == nil {
		return
	}
	var x, y int
	attempts := 0
	for {
		x = 1 + rand.IntN(l.Width-2)
		y = 1 + rand.IntN(l.Height-2)
		if l.Tiles[y][x].Type == TileFloor && !l.hasMonsterAt(x, y) && !l.hasItemAt(x, y) && !l.isStairsAt(x, y) && !l.hasChestAt(x, y) {
			break
		}
		attempts++
		if attempts > MaxSpawnAttempts {
			return
		}
	}
	
	// 🆕 ИСПРАВЛЕНИЕ: Используем NewGoldenChest для золотых сундуков, 
	// чтобы поле Contents корректно устанавливалось в "golden".
	// В оригинальном коде использовался NewChest, который задавал "potion"/"food"/"monster",
	// из-за чего логика ценного лута в interact.go никогда не срабатывала.
	isGolden := rand.IntN(100) < 20
	var chest *Chest
	if isGolden {
		chest = NewGoldenChest(x, y)
	} else {
		chest = NewChest(x, y)
	}
	
	l.Chests = append(l.Chests, chest)
}

func (l *Level) respawnMonsters() {
	if l == nil {
		return
	}
	l.Monsters = make([]*Monster, 0)
	count := (5 + l.Depth) / 2
	if count < 2 {
		count = 2
	}
	monsterTypes := []struct {
		name           string
		hp, attack, gold, xp int
		symbol         rune
		color          tcell.Color
	}{
		{"Гоблин", 8, 2, 5, 8, 'g', tcell.ColorGreen},
		{"Орк", 12, 3, 10, 15, 'o', tcell.ColorDarkRed},
		{"Скелет", 10, 2, 8, 12, 's', tcell.ColorWhite},
		{"Крыса", 4, 1, 2, 3, 'r', tcell.ColorBrown},
	}
	strengthMultiplier := 1 + (l.VisitCount / 2)
	for i := 0; i < count; i++ {
		var x, y int
		attempts := 0
		for {
			x = 1 + rand.IntN(l.Width-2)
			y = 1 + rand.IntN(l.Height-2)
			if l.Tiles[y][x].Type == TileFloor && !l.hasMonsterAt(x, y) && !l.hasItemAt(x, y) && !l.isStairsAt(x, y) {
				break
			}
			attempts++
			if attempts > MaxSpawnAttempts {
				return
			}
		}
		mt := monsterTypes[rand.IntN(len(monsterTypes))]
		hp := mt.hp * (1 + l.Depth/2) * strengthMultiplier
		attack := mt.attack * (1 + l.Depth/3) * strengthMultiplier
		gold := mt.gold * l.Depth * strengthMultiplier
		xp := mt.xp + l.Depth*2
		m := NewMonster(x, y, mt.name, hp, attack, gold, xp, mt.symbol, mt.color)
		m.SetLogger(l.logger)
		l.Monsters = append(l.Monsters, m)
	}
}

func newMerchantItems(depth, visitCount int) []*Item {
	basePrices := []struct {
		name         string
		itype        ItemType
		value, price int
		symbol       rune
		color        tcell.Color
	}{
		{"Зелье здоровья", ItemTypePotion, 10, 30, '!', tcell.ColorRed},
		{"Еда", ItemTypePotion, 0, 15, '%', tcell.ColorPurple},
		{"Меч", ItemTypeWeapon, 5, 50, '/', tcell.ColorYellow},
		{"Щит", ItemTypeArmor, 3, 40, '[', tcell.ColorBlue},
	}
	items := make([]*Item, 0)
	priceMultiplier := visitCount
	for _, bp := range basePrices {
		if rand.IntN(100) < 70 {
			price := (bp.price + depth*5) * priceMultiplier
			item := NewItem(0, 0, bp.name, bp.itype, bp.value, bp.symbol, bp.color)
			item.Price = price
			items = append(items, item)
		}
	}
	return items
}