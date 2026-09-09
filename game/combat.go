package game

import (
	"fmt"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// ВСПОМОГАТЕЛЬНАЯ ФУНКЦИЯ ДЛЯ СТОПОК
// =============================================================================
//
// addToInventoryWithStack — добавляет предмет в инвентарь с учётом стопок.
// Если в инвентаре уже есть такой предмет — увеличивает его Count.
// Иначе добавляет новый предмет с Count=1.
//
// Работает для всех типов предметов, кроме золота (которое сразу в кошелёк).
// Мечи и щиты тоже складываются в стопки (по имени и типу).
//
// 🆕 Реликвии и Амулет НЕ стакаются (см. IsStackable в item.go).
func (g *Game) addToInventoryWithStack(item *Item) {
	if item == nil || g.player == nil {
		return
	}

	// 🆕 Реликвии и Амулет не стакаются — добавляем как отдельный предмет
	if item.IsStackable() {
		// Для стакающихся предметов проверяем, есть ли уже такой в инвентаре
		for _, inv := range g.player.Inventory {
			if inv != nil && inv.Name == item.Name && inv.Type == item.Type {
				// Нашли такой же предмет — увеличиваем стопку
				inv.Count++
				g.logAndSync("ITEM_STACK: %s добавлен в стопку (всего: %d)",
					item.Name, inv.Count)
				return
			}
		}
	}

	// Не нашли такой предмет (или предмет не стакамый) — добавляем новый
	item.Count = 1
	g.player.Inventory = append(g.player.Inventory, item)
}

// =============================================================================
// БОЙ И ХОДЫ
// =============================================================================

// abs — модуль числа (вспомогательная функция)
//
// ⚠️ Также используется в level_fov.go (hasLineOfSight) и level_boss.go
// (FindFreeSpotNearStairsDown) — все файлы в одном пакете `game`.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// monsterIsAdjacent — проверяет, стоит ли монстр рядом с игроком (включая диагонали)
func (g *Game) monsterIsAdjacent(m *Monster) bool {
	if g.player == nil || m == nil {
		return false
	}
	return abs(m.X-g.player.X) <= 1 && abs(m.Y-g.player.Y) <= 1
}

// checkPlayerDeath — проверяет, погиб ли игрок, и переводит игру в состояние смерти
func (g *Game) checkPlayerDeath() bool {
	if g.player == nil || g.player.HP > 0 {
		return false
	}
	// Если уже выходим или уже на экране смерти — не делаем ничего
	if g.quit {
		return true
	}
	if g.state == StateDeathMenu {
		return true
	}
	g.player.HP = 0
	g.logAndSync("GAME_OVER: Игрок погиб")
	g.state = StateDeathMenu
	return true
}

// processTurn — обрабатывает один ход игры (движение игрока + ходы монстров)
//
// 🆕 ОСОБЫЕ СВОЙСТВА БОССОВ:
// Для каждого живого босса вызывается processBossAbilities, которая
// активирует регенерацию или призыв миньонов.
//
// 🆕 ЭТАП 3: АГРЕССИВНЫЕ МОНСТРЫ С АМУЛЕТОМ:
// Когда игрок несёт Амулет Бездны, монстры видят дальше (10 клеток вместо 6).
// Это передаётся в AIUpdate через параметр `aggressive`.
func (g *Game) processTurn(dx, dy int) {
	if g.level == nil || g.player == nil {
		return
	}

	// Если игрок мёртв — не обрабатываем ходы
	if g.player.HP <= 0 {
		return
	}

	// Фаза 1: действие игрока
	if dx != 0 || dy != 0 {
		newX := g.player.X + dx
		newY := g.player.Y + dy

		// Если на целевой клетке монстр — атакуем его
		if monster := g.level.GetMonsterAt(newX, newY); monster != nil {
			g.logAndSync("COMBAT: Игрок атакует %s на (%d, %d)",
				monster.Name, newX, newY)
			g.attackMonster(monster)
			if g.checkPlayerDeath() {
				return
			}
		} else if g.level.CanMoveTo(newX, newY) {
			// Если клетка свободна — перемещаемся
			g.logAndSync("MOVE: Игрок идет на (%d, %d)", newX, newY)
			g.player.Move(dx, dy)
			if g.checkPlayerDeath() {
				return
			}
			// Если на клетке предмет — подбираем его
			if item := g.level.GetItemAt(newX, newY); item != nil {
				g.logAndSync("ITEM: Игрок наступает на %s", item.Name)
				g.pickupItem(item)
			}
		} else {
			// Клетка непроходима
			g.addMessage("Туда нельзя пройти!")
			return
		}
	}

	// Фаза 2: ходы монстров (идём с конца, чтобы можно было удалять мёртвых)
	// 🆕 ЭТАП 3: передаём g.player.HasAmulet для агрессивности монстров
	for i := len(g.level.Monsters) - 1; i >= 0; i-- {
		if g.checkPlayerDeath() {
			break
		}
		m := g.level.Monsters[i]
		if m == nil {
			continue
		}
		// Удаляем мёртвых монстров
		if m.HP <= 0 {
			g.logAndSync("DEATH: %s погибает и удаляется", m.Name)
			g.level.RemoveMonster(m)
			continue
		}

		// 🆕 Особые свойства боссов (регенерация, призыв миньонов)
		// Вызываются каждый ход для каждого живого босса
		if m.IsBoss {
			g.processBossAbilities(m)
			if g.checkPlayerDeath() {
				break
			}
		}

		// Если монстр рядом с игроком — атакует его
		if g.monsterIsAdjacent(m) {
			g.logAndSync("COMBAT: %s атакует игрока!", m.Name)
			g.monsterAttacksPlayer(m)
			if g.checkPlayerDeath() {
				break
			}
			continue
		}

		// Иначе монстр перемещается по своему ИИ
		// 🆕 ЭТАП 3: передаём g.player.HasAmulet для агрессивности
		m.AIUpdate(g.player.X, g.player.Y, g.level, g.player.HasAmulet)
	}

	g.checkPlayerDeath()
}

// =============================================================================
// ОСОБЫЕ СВОЙСТВА БОССОВ
// =============================================================================
//
// processBossAbilities — обрабатывает особые свойства босса каждый ход.
// Вызывается из processTurn для каждого живого босса.
//
// Свойства:
//   - Регенерация: восстанавливает 3 HP каждый ход (не выше MaxHP)
//   - Призыв миньонов: каждые 5 ходов призывает монстра рядом с боссом
//   - Двойная атака: обрабатывается в monsterAttacksPlayer
func (g *Game) processBossAbilities(boss *Monster) {
	if boss == nil || boss.HP <= 0 || g.level == nil {
		return
	}

	switch boss.BossAbility {
	case BossAbilityRegen:
		// Регенерация: восстанавливаем 3 HP каждый ход
		if boss.HP < boss.MaxHP {
			regen := 3
			oldHP := boss.HP
			boss.HP += regen
			if boss.HP > boss.MaxHP {
				boss.HP = boss.MaxHP
			}
			g.logAndSync("BOSS_REGEN: %s восстанавливает %d HP. HP: %d -> %d",
				boss.Name, boss.HP-oldHP, oldHP, boss.HP)
			// Показываем сообщение только если восстановление существенное
			// и босс ещё не полностью здоров
			if boss.HP-oldHP > 0 && boss.HP < boss.MaxHP {
				g.addMessage(fmt.Sprintf("%s регенерирует здоровье...", boss.Name))
			}
		}

	case BossAbilitySummon:
		// Призыв миньонов: каждые 5 ходов
		boss.SummonCounter++
		if boss.SummonCounter >= 5 {
			boss.SummonCounter = 0
			g.summonMinion(boss)
		}
	}
}

// summonMinion — призывает миньона рядом с боссом.
// Миньон — случайный обычный монстр с характеристиками текущего уровня.
// Вызывается из processBossAbilities для боссов со способностью призыва.
func (g *Game) summonMinion(boss *Monster) {
	if boss == nil || g.level == nil {
		return
	}

	// Типы миньонов (базовые характеристики)
	minionTypes := []struct {
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
	}

	// Выбираем случайный тип миньона
	mt := minionTypes[rand.Intn(len(minionTypes))]

	// Масштабируем характеристики по глубине (как в spawnMonsters)
	depth := g.level.Depth
	hp := mt.hp * (1 + depth/2)
	attack := mt.attack * (1 + depth/3)
	gold := mt.gold * depth
	xp := mt.xp + depth*2

	// Ищем свободную клетку рядом с боссом
	// Функция определена в level_boss.go
	x, y := g.level.FindFreeSpotNear(boss.X, boss.Y)

	// Если не нашли место — не призываем
	if x < 0 || y < 0 {
		return
	}

	minion := NewMonster(x, y, mt.name, hp, attack, gold, xp, mt.symbol, mt.color)
	minion.SetLogger(g.logger)
	g.level.Monsters = append(g.level.Monsters, minion)

	g.addMessage(fmt.Sprintf("%s призывает %s!", boss.Name, mt.name))
	g.logAndSync("BOSS_SUMMON: %s призвал %s на (%d, %d)", boss.Name, mt.name, x, y)
}

// =============================================================================
// ПОДБОР ПРЕДМЕТОВ
// =============================================================================
//
// pickupItem — подбирает предмет с пола.
//
// МЕХАНИКА СТОПОК:
//   - Золото сразу добавляется в кошелёк (не занимает место в инвентаре)
//   - ВСЕ остальные предметы (зелья, еда, оружие, броня) складываются в стопки:
//     если в инвентаре уже есть такой предмет — увеличиваем его Count,
//     иначе добавляем новый предмет с Count=1
//
// 🆕 ЭТАП 3: ПОДБОР АМУЛЕТА БЕЗДНЫ:
// Когда игрок подбирает Амулет Бездны, устанавливается player.HasAmulet = true.
// Монстры становятся агрессивнее. Игрок должен вернуться на уровень 1 для победы.
func (g *Game) pickupItem(item *Item) {
	if item == nil || g.player == nil || g.level == nil {
		return
	}

	if item.Type == ItemTypeGold {
		// Золото сразу в кошелёк
		g.player.Gold += item.Value
		g.logAndSync("GOLD: Подобрано %d золота. Всего: %d",
			item.Value, g.player.Gold)
		g.addMessage(fmt.Sprintf("Подобрано %s (%d золота)",
			item.Name, item.Value))
		g.level.RemoveItem(item)
		return
	}

	// 🆕 ЭТАП 3: Подбор Амулета Бездны
	// Амулет не добавляется в инвентарь, а устанавливает флаг HasAmulet
	if item.Type == ItemTypeAmulet {
		g.player.HasAmulet = true
		g.logAndSync("AMULET: Игрок подобрал Амулет Бездны!")
		g.addMessage("Вы подобрали АМУЛЕТ БЕЗДНЫ! Монстры стали агрессивнее!")
		g.addMessage("Вернитесь на уровень 1, чтобы победить!")
		g.level.RemoveItem(item)
		return
	}

	// Добавляем предмет в инвентарь с учётом стопок (для всех типов)
	// Реликвии и Амулет не стакаются (см. IsStackable в item.go)
	g.addToInventoryWithStack(item)
	g.logAndSync("ITEM_PICKUP: Подобран предмет %s", item.Name)
	g.level.RemoveItem(item)
	g.addMessage(fmt.Sprintf("Подобрано %s", item.Name))
}

// =============================================================================
// АТАКА МОНСТРА
// =============================================================================
//
// attackMonster — игрок атакует монстра
//
// 🆕 Особое сообщение для боссов: когда босс повержен, показываем
// "⚔ БОСС ПОВЕРЖЕН! Путь к лестнице открыт!"
//
// 🆕 ЭТАП 3: СПАВН АМУЛЕТА ПОСЛЕ УБИЙСТВА КОРОЛЯ БЕЗДНЫ:
// Когда Король Бездны повержен, на его месте спавнится Амулет Бездны.
// Игрок должен подобрать Амулет и вернуться на уровень 1 для победы.
func (g *Game) attackMonster(monster *Monster) {
	if monster == nil || g.player == nil || g.level == nil {
		return
	}

	damage := g.player.Attack()
	monster.TakeDamage(damage)
	g.addMessage(fmt.Sprintf("Вы атаковали %s на %d урона!", monster.Name, damage))

	// Если монстр погиб — удаляем его и начисляем награду
	if monster.HP <= 0 {
		g.level.RemoveMonster(monster)
		g.player.Gold += monster.GoldValue
		leveledUp := g.player.GainXP(monster.XPValue)
		g.logAndSync("KILL: %s убит. Gold +%d, XP +%d",
			monster.Name, monster.GoldValue, monster.XPValue)

		// 🆕 ЭТАП 3: Спавн Амулета Бездны после убийства Короля Бездны
		// Амулет спавнится на месте Короля Бездны
		if monster.Name == "Король Бездны" {
			amulet := NewItem(monster.X, monster.Y, "Амулет Бездны", ItemTypeAmulet, 0, '&', tcell.ColorYellow)
			g.level.Items = append(g.level.Items, amulet)
			g.addMessage("⚔ КОРОЛЬ БЕЗДНЫ ПОВЕРЖЕН!")
			g.addMessage("Амулет Бездны появился на его месте! Подберите его!")
			g.logAndSync("FINAL_BOSS_KILLED: Амулет Бездны заспавнен на (%d, %d)",
				monster.X, monster.Y)
		} else if monster.IsBoss {
			// 🆕 Особое сообщение для обычных боссов
			g.addMessage(fmt.Sprintf("⚔ %s ПОВЕРЖЕН! Путь к лестнице открыт!", monster.Name))
		} else if leveledUp {
			g.addMessage(fmt.Sprintf(
				"%s погиб! +%d золота, +%d опыта. Уровень повышен до %d!",
				monster.Name, monster.GoldValue, monster.XPValue, g.player.Level))
		} else {
			g.addMessage(fmt.Sprintf(
				"%s погиб! +%d золота, +%d опыта.",
				monster.Name, monster.GoldValue, monster.XPValue))
		}
	}
}

// =============================================================================
// АТАКА МОНСТРА ПО ИГРОКУ
// =============================================================================
//
// monsterAttacksPlayer — монстр атакует игрока
//
// 🆕 ОСОБОЕ СВОЙСТВО БОССА — ДВОЙНАЯ АТАКА:
// Если босс имеет способность BossAbilityDoubleAttack (Древний Дракон),
// он наносит урон дважды за один ход.
func (g *Game) monsterAttacksPlayer(monster *Monster) {
	if monster == nil || g.player == nil || monster.HP <= 0 {
		return
	}

	monsterDamage := monster.Attack()

	// Защита игрока уменьшает урон (минимум 1)
	actualDamage := monsterDamage - g.player.Defense
	if actualDamage < 1 {
		actualDamage = 1
	}

	oldHP := g.player.HP
	g.player.HP -= actualDamage

	g.logAndSync("DAMAGE_TAKEN: Игрок получает %d урона от %s. HP: %d -> %d",
		actualDamage, monster.Name, oldHP, g.player.HP)
	g.addMessage(fmt.Sprintf("%s атакует вас на %d урона!",
		monster.Name, actualDamage))

	// 🆕 ДВОЙНАЯ АТАКА БОССА (Древний Дракон)
	// Босс наносит урон второй раз за тот же ход
	// Проверяем, что игрок ещё жив после первого удара
	if monster.IsBoss && monster.BossAbility == BossAbilityDoubleAttack && g.player.HP > 0 {
		secondDamage := monsterDamage - g.player.Defense
		if secondDamage < 1 {
			secondDamage = 1
		}
		oldHP2 := g.player.HP
		g.player.HP -= secondDamage

		g.logAndSync("DAMAGE_TAKEN: Двойная атака! Игрок получает ещё %d урона. HP: %d -> %d",
			secondDamage, oldHP2, g.player.HP)
		g.addMessage(fmt.Sprintf("%s наносит ВТОРОЙ удар на %d урона!",
			monster.Name, secondDamage))
	}
}

// =============================================================================
// 🆕 ЭТАП 3: ЭФФЕКТЫ СВИТКОВ
// =============================================================================
//
// Эти функции вызываются из useItem в input.go при использовании свитка.
// Каждый свиток — одноразовый предмет. После использования свиток удаляется.

// useScrollMap — свиток карты: открывает весь этаж (снимает туман войны).
// Все клетки становятся исследованными (Explored = true).
func (g *Game) useScrollMap() {
	if g.level == nil {
		return
	}
	for y := 0; y < g.level.Height; y++ {
		if y >= len(g.level.Tiles) {
			continue
		}
		for x := 0; x < g.level.Width; x++ {
			if x >= len(g.level.Tiles[y]) {
				continue
			}
			g.level.Tiles[y][x].Explored = true
		}
	}
	g.addMessage("Свиток карты открывает весь этаж!")
	g.logAndSync("SCROLL_MAP: Весь этаж открыт")
}

// useScrollTeleport — свиток телепортации: случайное перемещение по уровню.
// Игрок перемещается на случайную свободную клетку.
func (g *Game) useScrollTeleport() {
	if g.level == nil || g.player == nil {
		return
	}
	x, y := g.level.FindFreeSpot()
	g.player.X = x
	g.player.Y = y
	g.addMessage("Свиток телепортации переносит вас в другое место!")
	g.logAndSync("SCROLL_TELEPORT: Игрок перемещён на (%d, %d)", x, y)
}

// useScrollLightning — свиток молнии: наносит 20 урона всем монстрам на уровне.
// Мёртвые монстры удаляются, начисляется награда.
func (g *Game) useScrollLightning() {
	if g.level == nil || g.player == nil {
		return
	}

	damage := 20
	killedCount := 0

	// Идём с конца, чтобы можно было удалять мёртвых монстров
	for i := len(g.level.Monsters) - 1; i >= 0; i-- {
		m := g.level.Monsters[i]
		if m == nil {
			continue
		}
		m.TakeDamage(damage)
		if m.HP <= 0 {
			// Монстр погиб от молнии — начисляем награду
			g.player.Gold += m.GoldValue
			g.player.GainXP(m.XPValue)
			g.level.RemoveMonster(m)
			killedCount++
		}
	}

	g.addMessage(fmt.Sprintf("Свиток молнии поражает всех монстров! Убито: %d", killedCount))
	g.logAndSync("SCROLL_LIGHTNING: Убито %d монстров", killedCount)
}

// useScrollBanishment — свиток изгнания: уничтожает случайного монстра на уровне.
// Монстр удаляется без начисления награды (магия изгнания не даёт опыта).
func (g *Game) useScrollBanishment() {
	if g.level == nil || len(g.level.Monsters) == 0 {
		g.addMessage("Свиток изгнания не находит цели!")
		return
	}

	// Выбираем случайного монстра
	idx := rand.Intn(len(g.level.Monsters))
	m := g.level.Monsters[idx]
	if m == nil {
		g.addMessage("Свиток изгнания не находит цели!")
		return
	}

	banishedName := m.Name
	g.level.RemoveMonster(m)

	g.addMessage(fmt.Sprintf("Свиток изгнания уничтожает %s!", banishedName))
	g.logAndSync("SCROLL_BANISHMENT: %s изгнан", banishedName)
}