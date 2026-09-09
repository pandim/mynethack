package game

import (
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// КОНСТАНТЫ ФИНАЛЬНОГО БОССА
// =============================================================================
//
// 🆕 ЭТАП 3: Король Бездны — финальный босс на 25 уровне.
// После его убийства спавнится Амулет Бездны (см. combat.go → attackMonster).
const (
	FinalBossDepth = 25 // глубина, на которой спавнится Король Бездны
)

// =============================================================================
// СПАВН БОССОВ
// =============================================================================
//
// spawnBoss — спавнит босса на каждом 6-м уровне (6, 12, 18...)
// и Короля Бездны на 25 уровне.
// Босс охраняет лестницу вниз: пока он жив, игрок не может спуститься.
//
// Типы боссов чередуются по глубине:
//   - Глубина 6:  Вождь Гоблинов (без особого свойства)
//   - Глубина 12: Некромант (регенерация 3 HP/ход)
//   - Глубина 18: Древний Дракон (двойная атака)
//   - Глубина 24: Повелитель Бездны (призыв миньонов каждые 5 ходов)
//   - Глубина 25: 🆕 Король Бездны (финальный босс, призыв миньонов)
//   - Глубина 30+: цикл повторяется с удвоенными характеристиками
//
// Босс спавнится рядом с лестницей вниз (в пределах 2 клеток), но не на самой
// лестнице, чтобы игрок мог на неё встать после победы.
func (l *Level) spawnBoss(depth int) {
	if l == nil {
		return
	}

	// 🆕 ЭТАП 3: Король Бездны на 25 уровне
	// Спавним его ДО проверки кратности 6, чтобы он появился именно на 25 уровне
	if depth == FinalBossDepth {
		l.spawnFinalBoss()
		return
	}

	// Обычные боссы только на каждом 6-м уровне
	if depth%6 != 0 {
		return
	}

	// Типы боссов с базовыми характеристиками
	bossTypes := []struct {
		name    string
		hp      int
		attack  int
		gold    int
		xp      int
		symbol  rune
		color   tcell.Color
		ability BossAbility
	}{
		{"Вождь Гоблинов", 100, 10, 100, 50, 'G', tcell.ColorGreen, BossAbilityNone},
		{"Некромант", 150, 15, 200, 100, 'N', tcell.ColorFuchsia, BossAbilityRegen},
		{"Древний Дракон", 200, 20, 300, 150, 'D', tcell.ColorRed, BossAbilityDoubleAttack},
		{"Повелитель Бездны", 250, 25, 500, 200, 'B', tcell.ColorWhite, BossAbilitySummon},
	}

	// Определяем, какой босс по счёту и какой цикл
	// Для глубины 6:  bossTier=1, bossIndex=0, cycle=0, multiplier=1
	// Для глубины 12: bossTier=2, bossIndex=1, cycle=0, multiplier=1
	// Для глубины 30: bossTier=5, bossIndex=0, cycle=1, multiplier=2
	bossTier := depth / 6
	bossIndex := (bossTier - 1) % len(bossTypes)
	cycle := (bossTier - 1) / len(bossTypes)
	multiplier := 1 + cycle // удвоение характеристик каждый цикл

	bt := bossTypes[bossIndex]

	// Масштабируем характеристики по циклу
	hp := bt.hp * multiplier
	attack := bt.attack * multiplier
	gold := bt.gold * multiplier
	xp := bt.xp * multiplier

	// Ищем свободную клетку рядом с лестницей вниз
	x, y := l.FindFreeSpotNearStairsDown()

	boss := NewBoss(x, y, bt.name, hp, attack, gold, xp, bt.symbol, bt.color, bt.ability)
	boss.SetLogger(l.logger)
	l.Monsters = append(l.Monsters, boss)

	if l.logger != nil {
		l.logger.Printf("SPAWN_BOSS: %s (HP=%d ATK=%d Gold=%d XP=%d Ability=%d) на (%d, %d)",
			bt.name, hp, attack, gold, xp, bt.ability, x, y)
	}
}

// =============================================================================
// 🆕 ЭТАП 3: СПАВН КОРОЛЯ БЕЗДНЫ
// =============================================================================
//
// spawnFinalBoss — спавнит Короля Бездны на 25 уровне.
// Король Бездны — финальный босс. После его убийства спавнится Амулет Бездны
// (см. combat.go → attackMonster).
//
// Характеристики Короля Бездны:
//   - HP: 500 (в 2 раза больше, чем у Повелителя Бездны)
//   - Атака: 50
//   - Золото: 1000
//   - Опыт: 500
//   - Способность: призыв миньонов (как у Повелителя Бездны)
//
// Король Бездны спавнится рядом с лестницей вниз.
// Пока он жив, игрок не может спуститься (блокировка лестницы).
//
// Символ 'K' (от "King"), красный цвет.
func (l *Level) spawnFinalBoss() {
	if l == nil {
		return
	}

	// Ищем свободную клетку рядом с лестницей вниз
	// Функция определена ниже в этом файле
	x, y := l.FindFreeSpotNearStairsDown()

	// Король Бездны: финальный босс с призывом миньонов
	// Константа BossAbilitySummon определена в monster.go
	boss := NewBoss(x, y, "Король Бездны", 500, 50, 1000, 500, 'K', tcell.ColorRed, BossAbilitySummon)
	boss.SetLogger(l.logger)
	l.Monsters = append(l.Monsters, boss)

	if l.logger != nil {
		l.logger.Printf("SPAWN_FINAL_BOSS: Король Бездны (HP=500 ATK=50 Gold=1000 XP=500) на (%d, %d)",
			x, y)
	}
}

// =============================================================================
// ПОИСК МЕСТА ДЛЯ БОССА И МИНЬОНОВ
// =============================================================================
//
// FindFreeSpotNearStairsDown — находит свободную клетку рядом с лестницей вниз.
// Ищет в пределах 2 клеток от лестницы. Если не находит — использует
// FindFreeSpot как запасной вариант.
// Используется для спавна босса рядом с лестницей.
func (l *Level) FindFreeSpotNearStairsDown() (int, int) {
	if l == nil {
		return 0, 0
	}

	sx, sy := l.StairsDownX, l.StairsDownY

	// Если лестница не размещена — запасной вариант
	if sx < 0 || sy < 0 {
		return l.FindFreeSpot()
	}

	// Ищем в пределах 2 клеток от лестницы (кольцами от близких к дальним)
	for radius := 1; radius <= 2; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				// Пропускаем клетки вне текущего "кольца"
				// (чтобы идти от ближних к дальним)
				// Примечание: функция abs определена в combat.go
				if abs(dx) < radius && abs(dy) < radius {
					continue
				}
				nx, ny := sx+dx, sy+dy
				if l.CanMoveTo(nx, ny) &&
					!l.hasMonsterAt(nx, ny) &&
					!l.hasItemAt(nx, ny) &&
					!l.isStairsAt(nx, ny) {
					return nx, ny
				}
			}
		}
	}

	// Запасной вариант: любое свободное место
	// Функция определена в level_stairs.go
	return l.FindFreeSpot()
}

// FindFreeSpotNear — находит свободную клетку рядом с точкой (x, y).
// Ищет в пределах 1 клетки. Если не находит — возвращает (-1, -1).
// Используется для призыва миньонов рядом с боссом (в combat.go → summonMinion).
func (l *Level) FindFreeSpotNear(x, y int) (int, int) {
	if l == nil {
		return -1, -1
	}

	// 8 направлений вокруг точки (включая диагонали)
	// Тип point определён в level.go
	dirs := []point{
		{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1},
		{X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1},
	}

	// Перемешиваем направления для случайности
	// Функция rand определена в пакете math/rand (импортирован выше)
	for i := len(dirs) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		dirs[i], dirs[j] = dirs[j], dirs[i]
	}

	for _, d := range dirs {
		nx, ny := x+d.X, y+d.Y
		if l.CanMoveTo(nx, ny) &&
			!l.hasMonsterAt(nx, ny) &&
			!l.hasItemAt(nx, ny) &&
			!l.isStairsAt(nx, ny) {
			return nx, ny
		}
	}

	return -1, -1 // не нашли свободное место
}

// =============================================================================
// ЗАПРОСЫ БОССА
// =============================================================================
//
// HasAliveBoss — проверяет, есть ли на уровне живой босс.
// Используется для блокировки лестницы вниз, пока босс не повержен
// (в input.go → handleMovement).
func (l *Level) HasAliveBoss() bool {
	if l == nil {
		return false
	}
	for _, m := range l.Monsters {
		if m != nil && m.IsBoss && m.HP > 0 {
			return true
		}
	}
	return false
}

// GetBossName — возвращает имя живого босса на уровне (для сообщений).
// Если босса нет, возвращает пустую строку.
// Используется в input.go → handleMovement для сообщения о блокировке лестницы.
func (l *Level) GetBossName() string {
	if l == nil {
		return ""
	}
	for _, m := range l.Monsters {
		if m != nil && m.IsBoss && m.HP > 0 {
			return m.Name
		}
	}
	return ""
}

// GetAliveBoss — возвращает живого босса на уровне (для полоски здоровья).
// Если босса нет, возвращает nil.
// Используется в render.go → renderBossHealthBar.
func (l *Level) GetAliveBoss() *Monster {
	if l == nil {
		return nil
	}
	for _, m := range l.Monsters {
		if m != nil && m.IsBoss && m.HP > 0 {
			return m
		}
	}
	return nil
}