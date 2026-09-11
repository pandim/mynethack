package game

import (
	"math/rand/v2"

	"github.com/gdamore/tcell/v2"
)

const (
	FinalBossDepth = 15 // 🆕 Финальный босс на 15 уровне
)

func (l *Level) spawnBoss(depth int) {
	if l == nil { return }
	if depth == FinalBossDepth {
		l.spawnFinalBoss()
		return
	}
	// 🆕 Боссы каждые 3 уровня (3, 6, 9, 12)
	if depth%3 != 0 {
		return
	}

	bossTypes := []struct {
		name string; hp, attack, gold, xp int; symbol rune; color tcell.Color; ability BossAbility
	}{
		{"Вождь Гоблинов", 100, 10, 100, 50, 'G', tcell.ColorGreen, BossAbilityNone},
		{"Некромант", 150, 15, 200, 100, 'N', tcell.ColorFuchsia, BossAbilityRegen},
		{"Древний Дракон", 200, 20, 300, 150, 'D', tcell.ColorRed, BossAbilityDoubleAttack},
		{"Повелитель Бездны", 250, 25, 500, 200, 'B', tcell.ColorWhite, BossAbilitySummon},
	}

	bossTier := depth / 3
	bossIndex := (bossTier - 1) % len(bossTypes)
	cycle := (bossTier - 1) / len(bossTypes)
	multiplier := 1 + cycle

	bt := bossTypes[bossIndex]
	hp := bt.hp * multiplier
	attack := bt.attack * multiplier
	gold := bt.gold * multiplier
	xp := bt.xp * multiplier

	x, y := l.FindFreeSpotNearStairsDown()
	boss := NewBoss(x, y, bt.name, hp, attack, gold, xp, bt.symbol, bt.color, bt.ability)
	boss.SetLogger(l.logger)
	l.Monsters = append(l.Monsters, boss)
}

func (l *Level) spawnFinalBoss() {
	if l == nil { return }
	x, y := l.FindFreeSpotNearStairsDown()
	boss := NewBoss(x, y, "Король Бездны", 500, 50, 1000, 500, 'K', tcell.ColorRed, BossAbilitySummon)
	boss.SetLogger(l.logger)
	l.Monsters = append(l.Monsters, boss)
}

func (l *Level) FindFreeSpotNearStairsDown() (int, int) {
	if l == nil { return 0, 0 }
	sx, sy := l.StairsDownX, l.StairsDownY
	if sx < 0 || sy < 0 { return l.FindFreeSpot() }
	for radius := 1; radius <= 2; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if abs(dx) < radius && abs(dy) < radius { continue }
				nx, ny := sx+dx, sy+dy
				if l.CanMoveTo(nx, ny) && !l.hasMonsterAt(nx, ny) && !l.hasItemAt(nx, ny) && !l.isStairsAt(nx, ny) {
					return nx, ny
				}
			}
		}
	}
	return l.FindFreeSpot()
}

func (l *Level) FindFreeSpotNear(x, y int) (int, int) {
	if l == nil { return -1, -1 }
	dirs := []point{
		{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1},
		{X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1},
	}
	rand.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
	for _, d := range dirs {
		nx, ny := x+d.X, y+d.Y
		if l.CanMoveTo(nx, ny) && !l.hasMonsterAt(nx, ny) && !l.hasItemAt(nx, ny) && !l.isStairsAt(nx, ny) {
			return nx, ny
		}
	}
	return -1, -1
}

func (l *Level) HasAliveBoss() bool {
	if l == nil { return false }
	for _, m := range l.Monsters {
		if m != nil && m.IsBoss && m.HP > 0 { return true }
	}
	return false
}

func (l *Level) GetBossName() string {
	if l == nil { return "" }
	for _, m := range l.Monsters {
		if m != nil && m.IsBoss && m.HP > 0 { return m.Name }
	}
	return ""
}

func (l *Level) GetAliveBoss() *Monster {
	if l == nil { return nil }
	for _, m := range l.Monsters {
		if m != nil && m.IsBoss && m.HP > 0 { return m }
	}
	return nil
}