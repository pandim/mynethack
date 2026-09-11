package game

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

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
	
	// 🆕 УСТАНАВЛИВАЕМ ПРИЧИНУ СМЕРТИ
	if g.deathReason == "" {
		if g.player.Hunger >= 1000 {
			g.deathReason = "Умер от голода"
		} else {
			g.deathReason = "Погиб в подземелье"
		}
	}
	
	g.player.HP = 0
	g.logAndSync("GAME_OVER: Игрок погиб. Причина: %s", g.deathReason)
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
			
			// 🆕 ПРОВЕРКА ЛОВУШЕК
			for _, trap := range g.level.Traps {
				if trap != nil && !trap.Triggered && trap.X == g.player.X && trap.Y == g.player.Y {
					trap.Triggered = true
					g.player.HP -= 3
					g.addMessage("Щёлк! Вы наступили на ловушку! -3 HP. Телепортация...")
					newX, newY := g.level.FindFreeSpot()
					g.player.X = newX
					g.player.Y = newY
					if g.checkPlayerDeath() {
						return
					}
					break
				}
			}
			
			if item := g.level.GetItemAt(g.player.X, g.player.Y); item != nil {
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
			boss.HP += 3
			if boss.HP > boss.MaxHP {
				boss.HP = boss.MaxHP
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
		g.addMessage(fmt.Sprintf("Вы подобрали %s (%d золота)", item.Name, item.Value))
		g.level.RemoveItem(item)
		return
	}
	if item.Type == ItemTypeAmulet {
		g.player.HasAmulet = true
		g.amuletFlashUntil = time.Now().Add(3 * time.Second) // 🆕 МИГАНИЕ 3 СЕКУНДЫ
		g.addMessage("Вы подобрали АМУЛЕТ БЕЗДНЫ!")
		g.addMessage("Монстры стали агрессивнее! Вернитесь на уровень 1!")
		g.level.RemoveItem(item)
		return
	}
	g.addToInventoryWithStack(item)
	g.level.RemoveItem(item)
	g.addMessage(fmt.Sprintf("Вы подобрали %s", item.Name))
}

// =============================================================================
// СКЛОНЕНИЕ ИМЁН, БОЕВЫЕ КЛИЧИ И ФРАЗЫ ПРИ СМЕРТИ
// =============================================================================
func getGenitiveName(name string) string {
	genitiveMap := map[string]string{
		"Гоблин": "гоблина", "Орк": "орка", "Скелет": "скелета", "Крыса": "крысы",
		"Ловушка": "ловушки", "Вождь Гоблинов": "вождя гоблинов", "Некромант": "некроманта",
		"Древний Дракон": "древнего дракона", "Повелитель Бездны": "повелителя бездны", "Король Бездны": "короля бездны",
	}
	if gen, ok := genitiveMap[name]; ok {
		return gen
	}
	return strings.ToLower(name)
}

func getBattleCry(name string) string {
	cries := map[string][]string{
		"Гоблин": {"Резать! Кусать!", "Смерть длинноногому!"},
		"Орк": {"Сокрушу твои кости!", "Умри, ничтожество!"},
		"Скелет": {"Плоть гниёт, а кости вечны...", "Твоё тепло скоро угаснет."},
		"Крыса": {"Грызть! Рвать! Жрать!", "Нас много, а ты один!"},
		"Вождь Гоблинов": {"Разорвать его на части!"},
		"Некромант": {"Твоя душа станет моей марионеткой!"},
		"Древний Дракон": {"Сгори в моём пламени!"},
		"Повелитель Бездны": {"Бездна голодна..."},
		"Король Бездны": {"Я — конец всего сущего!"},
	}
	if phrases, ok := cries[name]; ok && len(phrases) > 0 {
		return phrases[rand.Intn(len(phrases))]
	}
	return ""
}

func getDeathPhrase(name string) string {
	phrases := map[string][]string{
		"Гоблин": {"Нет! Моё золото!", "Мама!"},
		"Орк": {"Слава Оркам!", "Грррр..."},
		"Скелет": {"Кости... крошатся...", "Во прах..."},
		"Крыса": {"Писк..."},
		"Ловушка": {"Щёлк... и тишина."},
		"Вождь Гоблинов": {"Племя... не простит тебя!"},
		"Некромант": {"Смерть... это лишь начало..."},
		"Древний Дракон": {"Мой огонь... погаснет..."},
		"Повелитель Бездны": {"Бездна... ждёт тебя..."},
		"Король Бездны": {"Ты не победил..."},
	}
	if p, ok := phrases[name]; ok && len(p) > 0 {
		return p[rand.Intn(len(p))]
	}
	return ""
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

	cry := getBattleCry(monster.Name)
	if cry != "" {
		g.addMessage(fmt.Sprintf("%s кричит: \"%s\" и атакует на %d урона!", monster.Name, cry, damage))
	} else {
		g.addMessage(fmt.Sprintf("Вы атаковали %s на %d урона!", monster.Name, damage))
	}

	if monster.HP <= 0 {
		g.level.RemoveMonster(monster)
		g.player.Gold += monster.GoldValue
		leveledUp := g.player.GainXP(monster.XPValue)
		deathVerb := "умер"
		if monster.Name == "Ловушка" || monster.Name == "Крыса" {
			deathVerb = "умерла"
		}
		deathPhrase := getDeathPhrase(monster.Name)

		if monster.Name == FinalBossName {
			amulet := NewItem(monster.X, monster.Y, "Амулет Бездны", ItemTypeAmulet, 0, '&', tcell.ColorYellow)
			g.level.Items = append(g.level.Items, amulet)
			if deathPhrase != "" {
				g.addMessage(fmt.Sprintf("⚔ КОРОЛЬ БЕЗДНЫ ПОВЕРЖЕН со словами: \"%s\"!", deathPhrase))
			} else {
				g.addMessage("⚔ КОРОЛЬ БЕЗДНЫ ПОВЕРЖЕН!")
			}
			g.addMessage("Амулет Бездны появился на его месте! Подберите его!")
		} else if monster.IsBoss {
			if deathPhrase != "" {
				g.addMessage(fmt.Sprintf("⚔ %s ПОВЕРЖЕН со словами: \"%s\"! Путь к лестнице открыт!", monster.Name, deathPhrase))
			} else {
				g.addMessage(fmt.Sprintf("⚔ %s ПОВЕРЖЕН! Путь к лестнице открыт!", monster.Name))
			}
		} else if leveledUp {
			if deathPhrase != "" {
				g.addMessage(fmt.Sprintf("%s %s со словами: \"%s\"! +%d золота, +%d опыта. Уровень повышен до %d!", monster.Name, deathVerb, deathPhrase, monster.GoldValue, monster.XPValue, g.player.Level))
			} else {
				g.addMessage(fmt.Sprintf("%s %s! +%d золота, +%d опыта. Уровень повышен до %d!", monster.Name, deathVerb, monster.GoldValue, monster.XPValue, g.player.Level))
			}
		} else {
			if deathPhrase != "" {
				g.addMessage(fmt.Sprintf("%s %s со словами: \"%s\"! +%d золота, +%d опыта.", monster.Name, deathVerb, deathPhrase, monster.GoldValue, monster.XPValue))
			} else {
				g.addMessage(fmt.Sprintf("%s %s! +%d золота, +%d опыта.", monster.Name, deathVerb, monster.GoldValue, monster.XPValue))
			}
		}

		lootRoll := rand.Intn(100)
		monsterGenitive := getGenitiveName(monster.Name)
		if lootRoll < 5 {
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
			scroll := NewScroll(0, 0, st.sType, st.name, '~', st.color)
			g.addToInventoryWithStack(scroll)
			g.addMessage(fmt.Sprintf("С %s выпал %s!", monsterGenitive, st.name))
		} else if lootRoll < 20 {
			key := NewItem(0, 0, "Ключ", ItemTypeKey, 0, 'k', tcell.ColorAqua)
			g.addToInventoryWithStack(key)
			g.addMessage(fmt.Sprintf("С %s выпал Ключ!", monsterGenitive))
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
	
	//oldHP := g.player.HP
	g.player.HP -= actualDamage
	
	// 🆕 ФИКСИРУЕМ ПРИЧИНУ СМЕРТИ ОТ МОНСТРА
	if g.player.HP <= 0 {
		g.deathReason = fmt.Sprintf("Убит: %s", monster.Name)
	}
	
	g.addMessage(fmt.Sprintf("%s атакует вас на %d урона!", monster.Name, actualDamage))

	if monster.IsBoss && monster.BossAbility == BossAbilityDoubleAttack && g.player.HP > 0 {
		secondDamage := monsterDamage - g.player.Defense
		if secondDamage < 1 {
			secondDamage = 1
		}
		g.player.HP -= secondDamage
		g.addMessage(fmt.Sprintf("%s наносит ВТОРОЙ удар на %d урона!", monster.Name, secondDamage))
	}
}