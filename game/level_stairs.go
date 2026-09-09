package game

import (
	"math/rand"
)

// =============================================================================
// РАЗМЕЩЕНИЕ ЛЕСТНИЦ
// =============================================================================
// placeStairs — размещает лестницы на уровне.
//
// Лестница вниз есть всегда (спуск глубже).
// Лестница вверх есть только если глубина > 1 (на поверхности нет лестницы вверх).
//
// Лестницы размещаются на случайных клетках пола. Если свободных клеток нет —
// используется аварийный вариант (лестницы совпадают на одной клетке).
func (l *Level) placeStairs(depth int) {
	if l == nil {
		return
	}
	l.StairsDown = true    // лестница вниз есть всегда
	l.StairsUp = depth > 1 // лестница вверх — только если не на поверхности
	l.StairsDownX = -1
	l.StairsDownY = -1
	l.StairsUpX = -1
	l.StairsUpY = -1

	// Собираем все клетки пола
	floors := l.floorTiles()

	// Если клеток пола нет (аварийная ситуация) — создаём одну в центре
	if len(floors) == 0 {
		cx, cy := l.Width/2, l.Height/2
		if cx >= 0 && cx < l.Width && cy >= 0 && cy < l.Height {
			if cy < len(l.Tiles) && cx < len(l.Tiles[cy]) {
				l.Tiles[cy][cx].Type = TileFloor
				floors = append(floors, point{X: cx, Y: cy})
			}
		}
	}

	if len(floors) == 0 {
		return // совсем некуда ставить лестницы
	}

	// Лестница вниз — на случайной клетке пола
	down := floors[rand.Intn(len(floors))]
	l.StairsDownX = down.X
	l.StairsDownY = down.Y

	// Лестница вверх (если нужна) — на другой случайной клетке
	if l.StairsUp {
		// Собираем все клетки пола, кроме той, где стоит лестница вниз
		upFloors := make([]point, 0, len(floors))
		for _, p := range floors {
			if p != down {
				upFloors = append(upFloors, p)
			}
		}
		// Если других клеток нет — создаём клетку рядом с лестницей вниз
		if len(upFloors) == 0 {
			if nx, ny, ok := l.createAdditionalFloorNear(down.X, down.Y); ok {
				upFloors = append(upFloors, point{X: nx, Y: ny})
			}
		}
		if len(upFloors) > 0 {
			up := upFloors[rand.Intn(len(upFloors))]
			l.StairsUpX = up.X
			l.StairsUpY = up.Y
		} else {
			// Аварийный вариант: лестницы совпадают на одной клетке
			// (отображается символ '±')
			l.StairsUpX = down.X
			l.StairsUpY = down.Y
		}
	}
}

// floorTiles — возвращает список всех клеток пола на уровне.
// Используется при размещении лестниц.
func (l *Level) floorTiles() []point {
	floors := make([]point, 0)
	if l == nil {
		return floors
	}
	for y := 0; y < l.Height; y++ {
		if y >= len(l.Tiles) {
			continue
		}
		for x := 0; x < l.Width; x++ {
			if x >= len(l.Tiles[y]) {
				continue
			}
			if l.Tiles[y][x].Type == TileFloor {
				floors = append(floors, point{X: x, Y: y})
			}
		}
	}
	return floors
}

// createAdditionalFloorNear — пытается создать клетку пола рядом с (x, y).
// Сначала проверяет 8 соседних клеток, потом любую стену на карте.
// Возвращает координаты созданной клетки и true, если удалось.
func (l *Level) createAdditionalFloorNear(x, y int) (int, int, bool) {
	if l == nil {
		return 0, 0, false
	}
	// 8 направлений вокруг клетки (включая диагонали)
	dirs := []point{
		{X: 1, Y: 0}, {X: -1, Y: 0}, {X: 0, Y: 1}, {X: 0, Y: -1},
		{X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1},
	}
	for _, d := range dirs {
		nx := x + d.X
		ny := y + d.Y
		if nx >= 0 && nx < l.Width && ny >= 0 && ny < l.Height {
			if ny < len(l.Tiles) && nx < len(l.Tiles[ny]) {
				if l.Tiles[ny][nx].Type == TileWall {
					l.Tiles[ny][nx].Type = TileFloor
					return nx, ny, true
				}
			}
		}
	}
	// Аварийный вариант: превращаем любую стену на карте в пол
	for yy := 0; yy < l.Height; yy++ {
		if yy >= len(l.Tiles) {
			continue
		}
		for xx := 0; xx < l.Width; xx++ {
			if xx >= len(l.Tiles[yy]) {
				continue
			}
			if l.Tiles[yy][xx].Type == TileWall {
				l.Tiles[yy][xx].Type = TileFloor
				return xx, yy, true
			}
		}
	}
	return 0, 0, false // не удалось создать клетку
}

// isStairsAt — проверяет, есть ли лестница в клетке (x, y).
// Используется при спавне монстров/предметов, чтобы не ставить их на лестницы.
func (l *Level) isStairsAt(x, y int) bool {
	if l == nil {
		return false
	}
	if l.StairsDown && x == l.StairsDownX && y == l.StairsDownY {
		return true
	}
	if l.StairsUp && x == l.StairsUpX && y == l.StairsUpY {
		return true
	}
	return false
}

// stairRuneAt — возвращает символ лестницы в клетке (x, y).
//
// Символы:
//   - '>' — лестница вниз (спуск)
//   - '<' — лестница вверх (подъём)
//   - '±' — обе лестницы на одной клетке (аварийная ситуация)
//
// Второй возвращаемый параметр — есть ли лестница в этой клетке.
func (l *Level) stairRuneAt(x, y int) (rune, bool) {
	if l == nil {
		return 0, false
	}
	isUp := l.StairsUp && x == l.StairsUpX && y == l.StairsUpY
	isDown := l.StairsDown && x == l.StairsDownX && y == l.StairsDownY

	// Совпадение координат обеих лестниц — показываем специальный символ
	if isUp && isDown {
		return '±', true
	}
	if isUp {
		return '<', true
	}
	if isDown {
		return '>', true
	}
	return 0, false
}

// IsStairsOverlap — возвращает true, если обе лестницы находятся на одной клетке.
// Используется в renderStatus для отображения индикатора "[±]".
func (l *Level) IsStairsOverlap() bool {
	if l == nil {
		return false
	}
	return l.StairsUp && l.StairsDown &&
		l.StairsUpX == l.StairsDownX &&
		l.StairsUpY == l.StairsDownY
}

// =============================================================================
// ПОИСК СВОБОДНОГО МЕСТА
// =============================================================================
// FindFreeSpot — находит свободную клетку для размещения игрока.
// Это обёртка над FindFreeSpotExcluding без исключений.
func (l *Level) FindFreeSpot() (int, int) {
	return l.FindFreeSpotExcluding(-1, -1)
}

// FindFreeSpotExcluding — находит свободную клетку для размещения игрока,
// исключая клетку (excludeX, excludeY).
//
// Используется при переходе между уровнями: игрок появляется у лестницы,
// но если лестница на одной клетке с другой лестницей — исключаем одну из них.
//
// Алгоритм:
//   1. 1000 случайных попыток найти проходимую клетку без монстров/предметов/лестниц
//   2. Если не удалось — полный перебор всех клеток
//   3. Если и это не помогло — центр первой комнаты
//   4. Аварийный вариант — центр карты (превращаем в пол при необходимости)
func (l *Level) FindFreeSpotExcluding(excludeX, excludeY int) (int, int) {
	if l == nil {
		return 0, 0
	}

	// Этап 1: случайные попытки (быстро для больших карт)
	if l.Width >= 3 && l.Height >= 3 {
		attempts := 0
		for attempts < 1000 {
			x := 1 + rand.Intn(l.Width-2)
			y := 1 + rand.Intn(l.Height-2)
			if x == excludeX && y == excludeY {
				attempts++
				continue
			}
			if l.CanMoveTo(x, y) &&
				!l.hasMonsterAt(x, y) &&
				!l.hasItemAt(x, y) &&
				!l.isStairsAt(x, y) {
				return x, y
			}
			attempts++
		}

		// Этап 2: полный перебор (гарантированно найдёт, если есть свободная клетка)
		for y := 1; y < l.Height-1; y++ {
			for x := 1; x < l.Width-1; x++ {
				if x == excludeX && y == excludeY {
					continue
				}
				if l.CanMoveTo(x, y) &&
					!l.hasMonsterAt(x, y) &&
					!l.hasItemAt(x, y) &&
					!l.isStairsAt(x, y) {
					return x, y
				}
			}
		}
	}

	// Этап 3: центр первой комнаты (если комнаты есть)
	if len(l.Rooms) > 0 {
		r := l.Rooms[0]
		x := r.X + r.W/2
		y := r.Y + r.H/2
		if l.CanMoveTo(x, y) && !l.isStairsAt(x, y) {
			return x, y
		}
	}

	// Этап 4: аварийный вариант — центр карты
	cx, cy := l.Width/2, l.Height/2
	if cx >= 0 && cx < l.Width && cy >= 0 && cy < l.Height {
		if cy < len(l.Tiles) && cx < len(l.Tiles[cy]) {
			// Если центр — стена, превращаем его в пол
			if l.Tiles[cy][cx].Type != TileFloor {
				l.Tiles[cy][cx].Type = TileFloor
			}
		}
		return cx, cy
	}

	return 0, 0 // совсем аварийный вариант
}