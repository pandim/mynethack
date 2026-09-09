package game

import (
	"math/rand"
)

// =============================================================================
// ГЕНЕРАЦИЯ ПОДЗЕМЕЛЬЯ
// =============================================================================
// generateDungeon — генерирует подземелье из комнат и коридоров.
//
// Алгоритм:
//   1. Пытаемся разместить до 10 непересекающихся комнат
//   2. Каждую новую комнату соединяем коридором с предыдущей
//   3. Если ни одна комната не поместилась — создаём одну большую
//
// Если карта слишком мала (< 8 клеток), создаём одну комнату почти на всю карту.
func (l *Level) generateDungeon() {
	if l == nil {
		return
	}

	// Для очень маленьких карт — одна большая комната
	if l.Width < 8 || l.Height < 8 {
		if l.Width > 2 && l.Height > 2 {
			r := Room{X: 1, Y: 1, W: l.Width - 2, H: l.Height - 2}
			if r.W > 0 && r.H > 0 {
				l.createRoom(r)
				l.Rooms = append(l.Rooms, r)
			}
		}
		return
	}

	// Пытаемся разместить до 10 комнат
	numRooms := 10
	for i := 0; i < numRooms; i++ {
		// Случайные размеры комнаты (ограничены размером карты)
		w := intInRange(4, minInt(13, l.Width-4))
		h := intInRange(4, minInt(11, l.Height-4))
		// Случайная позиция (не вплотную к краям карты)
		x := intInRange(1, l.Width-w-1)
		y := intInRange(1, l.Height-h-1)
		newRoom := Room{x, y, w, h}

		// Проверяем, не пересекается ли с уже существующими комнатами
		overlap := false
		for _, other := range l.Rooms {
			if newRoom.intersects(other) {
				overlap = true
				break
			}
		}

		// Если не пересекается — создаём комнату и коридор к предыдущей
		if !overlap {
			l.createRoom(newRoom)
			if len(l.Rooms) > 0 {
				prevRoom := l.Rooms[len(l.Rooms)-1]
				l.createCorridor(prevRoom, newRoom)
			}
			l.Rooms = append(l.Rooms, newRoom)
		}
	}

	// Аварийный вариант: если ни одна комната не поместилась
	if len(l.Rooms) == 0 {
		r := Room{X: 1, Y: 1, W: l.Width - 2, H: l.Height - 2}
		if r.W > 0 && r.H > 0 {
			l.createRoom(r)
			l.Rooms = append(l.Rooms, r)
		}
	}
}

// intersects — проверяет, пересекаются ли две комнаты.
// Используется при генерации, чтобы комнаты не накладывались друг на друга.
func (r Room) intersects(other Room) bool {
	return !(r.X+r.W < other.X ||
		other.X+other.W < r.X ||
		r.Y+r.H < other.Y ||
		other.Y+other.H < r.Y)
}

// createRoom — "вырезает" комнату в карте: превращает клетки внутри
// прямоугольника из стен в пол.
func (l *Level) createRoom(r Room) {
	if l == nil {
		return
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if y >= 0 && y < l.Height && x >= 0 && x < l.Width {
				l.Tiles[y][x].Type = TileFloor
			}
		}
	}
}

// createCorridor — создаёт Г-образный коридор между центрами двух комнат.
// Случайно выбирает, сначала горизонтальный или вертикальный сегмент.
func (l *Level) createCorridor(r1, r2 Room) {
	if l == nil {
		return
	}
	// Центры комнат
	x1 := r1.X + r1.W/2
	y1 := r1.Y + r1.H/2
	x2 := r2.X + r2.W/2
	y2 := r2.Y + r2.H/2

	// Случайно выбираем порядок сегментов (Г-образный коридор)
	if rand.Intn(2) == 0 {
		l.createHCorridor(x1, x2, y1)
		l.createVCorridor(y1, y2, x2)
	} else {
		l.createVCorridor(y1, y2, x1)
		l.createHCorridor(x1, x2, y2)
	}
}

// createHCorridor — создаёт горизонтальный коридор от x1 до x2 на строке y.
func (l *Level) createHCorridor(x1, x2, y int) {
	if l == nil {
		return
	}
	for x := minInt(x1, x2); x <= maxInt(x1, x2); x++ {
		if y >= 0 && y < l.Height && x >= 0 && x < l.Width {
			l.Tiles[y][x].Type = TileFloor
		}
	}
}

// createVCorridor — создаёт вертикальный коридор от y1 до y2 на столбце x.
func (l *Level) createVCorridor(y1, y2, x int) {
	if l == nil {
		return
	}
	for y := minInt(y1, y2); y <= maxInt(y1, y2); y++ {
		if y >= 0 && y < l.Height && x >= 0 && x < l.Width {
			l.Tiles[y][x].Type = TileFloor
		}
	}
}