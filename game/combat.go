package game

import (
	"fmt"
	"math/rand"
	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// ВСПОМОГАТЕЛЬНАЯ ФУНКЦИЯ ДЛЯ СТОПОК
// =============================================================================
func (g *Game) addToInventoryWithStack(item *Item) {
	if item == nil || g.player == nil {
		return
	}

	if item.IsStackable() {
		for _, inv := range g.player.Inventory {
			if inv != nil && inv.Name == item.Name && inv.Type == item.Type {
				inv.Count++
				g.logAndSync("ITEM_STACK: %s добавлен в стопку (всего: %d)", item.Name, inv.Count)
				return
			}
		}
	}

	item.Count = 1
	g.player.Inventory = append(g.player.Inventory, item)
}

// =============================================================================
// БОЙ И ХОДЫ
// =============================================================================
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (g *Game) monsterIsAdjacent(m *Monster) bool {
	if g.player == nil || m == nil {
		return false
	}
	return abs(m.X-g.player.X) <= 1 && abs(m.Y-g.player.Y) <= 1
}

func (g *Game) checkPlayerDeath() bool {
	if g.player == nil || g.player.HP > 0 {
		return false
	}
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

func (g *Game) processTurn(dx, dy int) {
	if g.level == nil || g.player == nil {
		return
	}

	if g.player.HP <= 0 {
		return
	}

	if dx != 0 || dy != 0 {
		newX := g.player.X + dx
		newY := g.player.Y + dy

		if monster := g.level.GetMonsterAt(newX, newY); monster != nil {
			g.logAndSync("COMBAT: Игрок атакует %s на (%d, %d)", monster.Name, newX, newY)
			g.attackMonster(monster)
			if g.checkPlayerDeath() {
				return
			}
		} else if g.level.CanMoveTo(newX, newY) {
			g.logAndSync("MOVE: Игрок идет на (%d, %d)", newX, newY)
			g.player.Move(dx, dy)
			if g.checkPlayerDeath() {
				return
			}
			if item := g.level.GetItemAt(newX, newY); item != nil {
				g.logAndSync("ITEM: Игрок наступает на %s", item.Name)
				g.pickupItem(item)
			}
		} else {
			g.addMessage("Туда нельзя пройти!")
			return
		}
	}

	for i := len(g.level.Monsters) - 1; i >= 0; i-- {
		if g.checkPlayerDeath() {
			break
		}
		m := g.level.Monsters[i]
		if m == nil {
			continue
		}

		if m.HP <= 0 {
			deathVerb := "умер"
			if m.Name == "Ловушка" || m.Name == "Крыса" {
				deathVerb = "умерла"
			}
			g.logAndSync("DEATH: %s %s и удаляется", m.Name, deathVerb)
			g.level.RemoveMonster(m)
			continue
		}

		if m.IsBoss {
			g.processBossAbilities(m)
			if g.checkPlayerDeath() {
				break
			}
		}

		if g.monsterIsAdjacent(m) {
			g.logAndSync("COMBAT: %s атакует игрока!", m.Name)
			g.monsterAttacksPlayer(m)
			if g.checkPlayerDeath() {
				break
			}
			continue
		}

		m.AIUpdate(g.player.X, g.player.Y, g.level, g.player.HasAmulet)
	}
	g.checkPlayerDeath()
}

// =============================================================================
// ОСОБЫЕ СВОЙСТВА БОССОВ
// =============================================================================
func (g *Game) processBossAbilities(boss *Monster) {
	if boss == nil || boss.HP <= 0 || g.level == nil {
		return
	}
	switch boss.BossAbility {
	case BossAbilityRegen:
		if boss.HP < boss.MaxHP {
			regen := 3
			oldHP := boss.HP
			boss.HP += regen
			if boss.HP > boss.MaxHP {
				boss.HP = boss.MaxHP
			}
			g.logAndSync("BOSS_REGEN: %s восстанавливает %d HP. HP: %d -> %d", boss.Name, boss.HP-oldHP, oldHP, boss.HP)
			if boss.HP-oldHP > 0 && boss.HP < boss.MaxHP {
				g.addMessage(fmt.Sprintf("%s регенерирует здоровье...", boss.Name))
			}
		}
	case BossAbilitySummon:
		boss.SummonCounter++
		if boss.SummonCounter >= 5 {
			boss.SummonCounter = 0
			g.summonMinion(boss)
		}
	}
}

func (g *Game) summonMinion(boss *Monster) {
	if boss == nil || g.level == nil {
		return
	}
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
	mt := minionTypes[rand.Intn(len(minionTypes))]
	depth := g.level.Depth
	hp := mt.hp * (1 + depth/2)
	attack := mt.attack * (1 + depth/3)
	gold := mt.gold * depth
	xp := mt.xp + depth*2
	x, y := g.level.FindFreeSpotNear(boss.X, boss.Y)
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
func (g *Game) pickupItem(item *Item) {
	if item == nil || g.player == nil || g.level == nil {
		return
	}
	if item.Type == ItemTypeGold {
		g.player.Gold += item.Value
		g.logAndSync("GOLD: Подобрано %d золота. Всего: %d", item.Value, g.player.Gold)
		g.addMessage(fmt.Sprintf("Подобрано %s (%d золота)", item.Name, item.Value))
		g.level.RemoveItem(item)
		return
	}
	if item.Type == ItemTypeAmulet {
		g.player.HasAmulet = true
		g.logAndSync("AMULET: Игрок подобрал Амулет Бездны!")
		g.addMessage("Вы подобрали АМУЛЕТ БЕЗДНЫ!")
		g.addMessage("Монстры стали агрессивнее! Вернитесь на уровень 1!")
		g.level.RemoveItem(item)
		return
	}
	g.addToInventoryWithStack(item)
	g.logAndSync("ITEM_PICKUP: Подобран предмет %s", item.Name)
	g.level.RemoveItem(item)
	g.addMessage(fmt.Sprintf("Подобрано %s", item.Name))
}

// =============================================================================
// АТАКА МОНСТРА
// =============================================================================
const FinalBossName = "Король Бездны"

func (g *Game) attackMonster(monster *Monster) {
	if monster == nil || g.player == nil || g.level == nil {
		return
	}
	damage := g.player.Attack()
	monster.TakeDamage(damage)
	g.addMessage(fmt.Sprintf("Вы атаковали %s на %d урона!", monster.Name, damage))

	if monster.HP <= 0 {
		g.level.RemoveMonster(monster)
		g.player.Gold += monster.GoldValue
		leveledUp := g.player.GainXP(monster.XPValue)
		g.logAndSync("KILL: %s убит. Gold +%d, XP +%d", monster.Name, monster.GoldValue, monster.XPValue)

		deathVerb := "умер"
		if monster.Name == "Ловушка" || monster.Name == "Крыса" {
			deathVerb = "умерла"
		}

		if monster.Name == FinalBossName {
			amulet := NewItem(monster.X, monster.Y, "Амулет Бездны", ItemTypeAmulet, 0, '&', tcell.ColorYellow)
			g.level.Items = append(g.level.Items, amulet)
			g.addMessage("⚔ КОРОЛЬ БЕЗДНЫ ПОВЕРЖЕН!")
			g.addMessage("Амулет Бездны появился на его месте! Подберите его!")
			g.logAndSync("FINAL_BOSS_KILLED: Амулет Бездны заспавнен на (%d, %d)", monster.X, monster.Y)
		} else if monster.IsBoss {
			g.addMessage(fmt.Sprintf("⚔ %s ПОВЕРЖЕН! Путь к лестнице открыт!", monster.Name))
		} else if leveledUp {
			g.addMessage(fmt.Sprintf("%s %s! +%d золота, +%d опыта. Уровень повышен до %d!", monster.Name, deathVerb, monster.GoldValue, monster.XPValue, g.player.Level))
		} else {
			g.addMessage(fmt.Sprintf("%s %s! +%d золота, +%d опыта.", monster.Name, deathVerb, monster.GoldValue, monster.XPValue))
		}

		// 🆕 ШАНС ВЫПАДЕНИЯ КЛЮЧЕЙ И СВИТКОВ С МОНСТРОВ
		lootRoll := rand.Intn(100)
		if lootRoll < 5 { // 5% шанс на свиток
			scrollTypes := []struct {
				sType int
				name  string
				color tcell.Color
			}{
				{ScrollMap, "Свиток карты", tcell.ColorWhite},
				{ScrollTeleport, "Свиток телепортации", tcell.ColorAqua},
				{ScrollLightning, "Свиток молнии", tcell.ColorYellow},
				{ScrollBanishment, "Свиток изгнания", tcell.ColorRed},
			}
			st := scrollTypes[rand.Intn(len(scrollTypes))]
			// Используем символ '~' и уникальный цвет
			scroll := NewScroll(0, 0, st.sType, st.name, '~', st.color)
			g.addToInventoryWithStack(scroll)
			g.addMessage(fmt.Sprintf("С %s выпал %s!", monster.Name, st.name))
			g.logAndSync("LOOT: С %s выпал %s", monster.Name, st.name)
		} else if lootRoll < 20 { // 15% шанс на ключ (значения 5-19)
			key := NewItem(0, 0, "Ключ", ItemTypeKey, 0, 'k', tcell.ColorAqua)
			g.addToInventoryWithStack(key)
			g.addMessage(fmt.Sprintf("С %s выпал Ключ!", monster.Name))
			g.logAndSync("LOOT: С %s выпал Ключ", monster.Name)
		}
	}
}

// =============================================================================
// АТАКА МОНСТРА ПО ИГРОКУ
// =============================================================================
func (g *Game) monsterAttacksPlayer(monster *Monster) {
	if monster == nil || g.player == nil || monster.HP <= 0 {
		return
	}
	monsterDamage := monster.Attack()
	actualDamage := monsterDamage - g.player.Defense
	if actualDamage < 1 {
		actualDamage = 1
	}
	oldHP := g.player.HP
	g.player.HP -= actualDamage
	g.logAndSync("DAMAGE_TAKEN: Игрок получает %d урона от %s. HP: %d -> %d", actualDamage, monster.Name, oldHP, g.player.HP)
	g.addMessage(fmt.Sprintf("%s атакует вас на %d урона!", monster.Name, actualDamage))

	if monster.IsBoss && monster.BossAbility == BossAbilityDoubleAttack && g.player.HP > 0 {
		secondDamage := monsterDamage - g.player.Defense
		if secondDamage < 1 {
			secondDamage = 1
		}
		oldHP2 := g.player.HP
		g.player.HP -= secondDamage
		g.logAndSync("DAMAGE_TAKEN: Двойная атака! Игрок получает ещё %d урона. HP: %d -> %d", secondDamage, oldHP2, g.player.HP)
		g.addMessage(fmt.Sprintf("%s наносит ВТОРОЙ удар на %d урона!", monster.Name, secondDamage))
	}
}