package game

import (
	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// ОТРИСОВКА УРОВНЯ
// =============================================================================
// Render — отрисовывает уровень на экране.
//
// Отображение:
//   - Стены: '#' (серые если видимы, тускло-серые если исследованы)
//   - Пол: '.' (белый если видим, тускло-серый если исследован)
//   - Лестницы: '>' (вниз), '<' (вверх), '±' (обе на одной клетке)
//   - Предметы: их символы (только на видимых клетках)
//   - Монстры: их символы (только на видимых клетках)
//   - Сундуки: '&' (жёлтые если закрыты, тусклые если открыты)
//   - Алтари: '_' (белые)
//   - Торговцы: 'M' (голубые)
//
// Клетки, которые никогда не были видны (не исследованы), не отображаются.
//
// Параметры:
//   - screen: экран для отрисовки
//   - offsetX, offsetY: смещение на экране (обычно 1, 0 — отступ слева)
func (l *Level) Render(screen tcell.Screen, offsetX, offsetY int) {
	if l == nil || screen == nil {
		return
	}

	// Отрисовка клеток карты (стены, пол, лестницы)
	for y := 0; y < l.Height; y++ {
		if y >= len(l.Tiles) {
			continue
		}
		for x := 0; x < l.Width; x++ {
			if x >= len(l.Tiles[y]) {
				continue
			}
			tile := l.Tiles[y][x]

			// Неотрисованные клетки (никогда не виденные) пропускаем
			if !tile.Explored {
				continue
			}

			baseStyle := tcell.StyleDefault.Background(tcell.ColorBlack)

			if tile.Visible {
				// Клетка видима прямо сейчас — яркий цвет
				switch tile.Type {
				case TileWall:
					style := baseStyle.Foreground(tcell.ColorGray)
					screen.SetContent(x+offsetX, y+offsetY, '#', nil, style)
				case TileFloor:
					style := baseStyle.Foreground(tcell.ColorWhite)
					screen.SetContent(x+offsetX, y+offsetY, '.', nil, style)
				}
				// Лестницы на видимых клетках — жёлтые
				if ch, ok := l.stairRuneAt(x, y); ok {
					stairStyle := baseStyle.Foreground(tcell.ColorYellow)
					screen.SetContent(x+offsetX, y+offsetY, ch, nil, stairStyle)
				}
			} else {
				// Клетка исследована, но не видима сейчас — тусклый цвет
				darkStyle := baseStyle.Foreground(tcell.ColorDarkGray)
				if tile.Type == TileWall {
					screen.SetContent(x+offsetX, y+offsetY, '#', nil, darkStyle)
				} else {
					screen.SetContent(x+offsetX, y+offsetY, '.', nil, darkStyle)
				}
				// Лестницы на исследованных клетках — тоже тусклые
				if ch, ok := l.stairRuneAt(x, y); ok {
					screen.SetContent(x+offsetX, y+offsetY, ch, nil, darkStyle)
				}
			}
		}
	}

	// Отрисовка сундуков (только на видимых клетках)
	for _, chest := range l.Chests {
		if chest == nil {
			continue
		}
		if chest.Y >= 0 && chest.Y < l.Height && chest.X >= 0 && chest.X < l.Width {
			if chest.Y < len(l.Tiles) && chest.X < len(l.Tiles[chest.Y]) {
				if l.Tiles[chest.Y][chest.X].Visible {
					chest.Render(screen, offsetX, offsetY)
				}
			}
		}
	}

	// Отрисовка алтарей (только на видимых клетках)
	for _, altar := range l.Altars {
		if altar == nil {
			continue
		}
		if altar.Y >= 0 && altar.Y < l.Height && altar.X >= 0 && altar.X < l.Width {
			if altar.Y < len(l.Tiles) && altar.X < len(l.Tiles[altar.Y]) {
				if l.Tiles[altar.Y][altar.X].Visible {
					altar.Render(screen, offsetX, offsetY)
				}
			}
		}
	}

	// Отрисовка торговцев (только на видимых клетках)
	for _, merchant := range l.Merchants {
		if merchant == nil {
			continue
		}
		if merchant.Y >= 0 && merchant.Y < l.Height && merchant.X >= 0 && merchant.X < l.Width {
			if merchant.Y < len(l.Tiles) && merchant.X < len(l.Tiles[merchant.Y]) {
				if l.Tiles[merchant.Y][merchant.X].Visible {
					merchant.Render(screen, offsetX, offsetY)
				}
			}
		}
	}

	// Отрисовка предметов (только на видимых клетках)
	for _, item := range l.Items {
		if item == nil {
			continue
		}
		// Проверка границ и видимости клетки под предметом
		if item.Y >= 0 && item.Y < l.Height && item.X >= 0 && item.X < l.Width {
			if item.Y < len(l.Tiles) && item.X < len(l.Tiles[item.Y]) {
				if l.Tiles[item.Y][item.X].Visible {
					item.Render(screen, offsetX, offsetY)
				}
			}
		}
	}

	// Отрисовка монстров (только на видимых клетках)
	for _, monster := range l.Monsters {
		if monster == nil {
			continue
		}
		// Проверка границ и видимости клетки под монстром
		if monster.Y >= 0 && monster.Y < l.Height && monster.X >= 0 && monster.X < l.Width {
			if monster.Y < len(l.Tiles) && monster.X < len(l.Tiles[monster.Y]) {
				if l.Tiles[monster.Y][monster.X].Visible {
					monster.Render(screen, offsetX, offsetY)
				}
			}
		}
	}
}