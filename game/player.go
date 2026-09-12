package game

import (
	"log"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Player struct {
	X, Y      int
	HP        int
	MaxHP     int
	Level     int
	XP        int
	Gold      int
	AttackVal int
	Defense   int
	Hunger    int
	Inventory []*Item

	EquippedWeapon *Item
	EquippedArmor  *Item

	HasAmulet bool
	logger    *log.Logger
}

func NewPlayer(x, y int) *Player {
	return &Player{
		X: x, Y: y, HP: 30, MaxHP: 30, Level: 1, XP: 0, Gold: 0,
		AttackVal: 5, Defense: 2, Hunger: 0,
		Inventory: make([]*Item, 0), HasAmulet: false,
	}
}

func (p *Player) SetLogger(logger *log.Logger) {
	if p == nil { return }
	p.logger = logger
}

func (p *Player) Move(dx, dy int) {
	if p == nil { return }
	p.X += dx
	p.Y += dy
	p.Hunger += 1 // Расход сытости 2 за ход
	
	if p.logger != nil {
		p.logger.Printf("MOVE: Игрок переместился на (%d, %d). Голод: %d", p.X, p.Y, p.Hunger)
	}
	if p.Hunger > 1000 {
		p.HP -= 1
		p.Hunger = 1000
		if p.logger != nil {
			p.logger.Printf("STARVATION: Игрок умирает от голода! HP: %d", p.HP)
		}
	}
}

func (p *Player) Attack() int {
	if p == nil { return 0 }
	return p.AttackVal
}

func (p *Player) Heal(amount int) {
	if p == nil { return }
	oldHP := p.HP
	p.HP += amount
	if p.HP > p.MaxHP { p.HP = p.MaxHP }
	if p.logger != nil {
		p.logger.Printf("HEAL: Восстановлено %d HP. Было: %d, Стало: %d", amount, oldHP, p.HP)
	}
}

func (p *Player) EquipWeapon(item *Item) {
	if p == nil || item == nil { return }
	if p.EquippedWeapon == nil {
		p.EquippedWeapon = &Item{
			Name: item.Name, Type: item.Type, Value: item.Value,
			Symbol: item.Symbol, Color: item.Color, Count: 1,
		}
		p.AttackVal += item.Value
		if p.logger != nil {
			p.logger.Printf("EQUIP_WEAPON: %s экипировано. ATK: %d", item.Name, p.AttackVal)
		}
		return
	}
	oldValue := p.EquippedWeapon.Value
	p.EquippedWeapon.Value += 1
	p.AttackVal += 1
	if p.logger != nil {
		p.logger.Printf("UPGRADE_WEAPON: %s улучшено. ATK: %d -> %d", p.EquippedWeapon.Name, oldValue, p.EquippedWeapon.Value)
	}
}

func (p *Player) EquipArmor(item *Item) {
	if p == nil || item == nil { return }
	if p.EquippedArmor == nil {
		p.EquippedArmor = &Item{
			Name: item.Name, Type: item.Type, Value: item.Value,
			Symbol: item.Symbol, Color: item.Color, Count: 1,
		}
		p.Defense += item.Value
		if p.logger != nil {
			p.logger.Printf("EQUIP_ARMOR: %s экипирована. DEF: %d", item.Name, p.Defense)
		}
		return
	}
	oldValue := p.EquippedArmor.Value
	p.EquippedArmor.Value += 1
	p.Defense += 1
	if p.logger != nil {
		p.logger.Printf("UPGRADE_ARMOR: %s улучшена. DEF: %d -> %d", p.EquippedArmor.Name, oldValue, p.EquippedArmor.Value)
	}
}

func (p *Player) GainXP(amount int) bool {
	if p == nil || amount <= 0 { return false }
	p.XP += amount
	threshold := p.Level * 20
	if p.XP >= threshold {
		p.XP -= threshold
		p.Level++
		p.MaxHP += 5
		p.HP = p.MaxHP
		p.AttackVal += 1
		p.Defense += 1
		if p.logger != nil {
			p.logger.Printf("LEVEL_UP: Игрок достиг уровня %d. MaxHP=%d ATK=%d DEF=%d", p.Level, p.MaxHP, p.AttackVal, p.Defense)
		}
		return true
	}
	return false
}

func (p *Player) NextLevelXP() int {
	if p == nil { return 0 }
	return p.Level * 20
}

func (p *Player) CountRelics() int {
	if p == nil { return 0 }
	seen := make(map[int]bool)
	for _, item := range p.Inventory {
		if item != nil && item.Type == ItemTypeRelic && item.RelicID >= 0 {
			seen[item.RelicID] = true
		}
	}
	return len(seen)
}

func (p *Player) isCritical() bool {
	if p == nil { return false }
	if p.MaxHP > 0 {
		hpPercent := float64(p.HP) / float64(p.MaxHP)
		if hpPercent < 0.20 { return true }
	}
	if p.Hunger > 800 { return true }
	return false
}

func (p *Player) Render(screen tcell.Screen, offsetX, offsetY int) {
	if p == nil || screen == nil { return }
	fg := tcell.ColorGreen
	if p.HasAmulet {
		phase := (time.Now().UnixNano() / int64(400*time.Millisecond)) % 2
		if phase == 0 { fg = tcell.ColorYellow } else { fg = tcell.ColorWhite }
	} else if p.isCritical() {
		phase := (time.Now().UnixNano() / int64(500*time.Millisecond)) % 2
		if phase == 0 { fg = tcell.ColorRed } else { fg = tcell.ColorGreen }
	}
	style := tcell.StyleDefault.Foreground(fg).Background(tcell.ColorBlack)
	screen.SetContent(p.X+offsetX, p.Y+offsetY, '@', nil, style)
}