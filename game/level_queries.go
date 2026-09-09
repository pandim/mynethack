package game

// =============================================================================
// ПРОВЕРКИ И ЗАПРОСЫ
// =============================================================================
// hasMonsterAt — проверяет, есть ли монстр в клетке (x, y).
// Используется при спавне монстров и предметов, а также при проверке смерти.
func (l *Level) hasMonsterAt(x, y int) bool {
	if l == nil {
		return false
	}
	for _, m := range l.Monsters {
		if m != nil && m.X == x && m.Y == y {
			return true
		}
	}
	return false
}

// hasItemAt — проверяет, есть ли предмет в клетке (x, y).
// Используется при спавне монстров и предметов.
func (l *Level) hasItemAt(x, y int) bool {
	if l == nil {
		return false
	}
	for _, item := range l.Items {
		if item != nil && item.X == x && item.Y == y {
			return true
		}
	}
	return false
}

// hasChestAt — проверяет, есть ли сундук в клетке (x, y).
// Используется при спавне сундуков, чтобы не ставить их друг на друга.
func (l *Level) hasChestAt(x, y int) bool {
	if l == nil {
		return false
	}
	for _, c := range l.Chests {
		if c != nil && c.X == x && c.Y == y {
			return true
		}
	}
	return false
}

// CanMoveTo — проверяет, можно ли переместиться в клетку (x, y).
// Клетка проходима если она в пределах карты и является полом.
// Монстры и предметы НЕ блокируют движение (блокировка проверяется отдельно).
func (l *Level) CanMoveTo(x, y int) bool {
	if l == nil {
		return false
	}
	// Проверка границ карты
	if x < 0 || x >= l.Width || y < 0 || y >= l.Height {
		return false
	}
	// Дополнительная проверка на случай повреждения данных
	if y >= len(l.Tiles) || x >= len(l.Tiles[y]) {
		return false
	}
	return l.Tiles[y][x].Type == TileFloor
}

// GetItemAt — возвращает предмет в клетке (x, y) или nil.
// Используется при подборе предметов.
func (l *Level) GetItemAt(x, y int) *Item {
	if l == nil {
		return nil
	}
	for _, item := range l.Items {
		if item != nil && item.X == x && item.Y == y {
			return item
		}
	}
	return nil
}

// GetMonsterAt — возвращает монстра в клетке (x, y) или nil.
// Используется при атаке монстра.
func (l *Level) GetMonsterAt(x, y int) *Monster {
	if l == nil {
		return nil
	}
	for _, m := range l.Monsters {
		if m != nil && m.X == x && m.Y == y {
			return m
		}
	}
	return nil
}

// GetChestAt — возвращает сундук в клетке (x, y) или nil.
// Используется при открытии сундуков.
func (l *Level) GetChestAt(x, y int) *Chest {
	if l == nil {
		return nil
	}
	for _, c := range l.Chests {
		if c != nil && c.X == x && c.Y == y {
			return c
		}
	}
	return nil
}

// GetMerchantAt — возвращает торговца в клетке (x, y) или nil.
// Используется при взаимодействии с торговцем.
func (l *Level) GetMerchantAt(x, y int) *Merchant {
	if l == nil {
		return nil
	}
	for _, m := range l.Merchants {
		if m != nil && m.X == x && m.Y == y {
			return m
		}
	}
	return nil
}

// GetAltarAt — возвращает алтарь в клетке (x, y) или nil.
// Используется при взаимодействии с алтарём.
func (l *Level) GetAltarAt(x, y int) *Altar {
	if l == nil {
		return nil
	}
	for _, a := range l.Altars {
		if a != nil && a.X == x && a.Y == y {
			return a
		}
	}
	return nil
}

// =============================================================================
// УДАЛЕНИЕ ОБЪЕКТОВ
// =============================================================================
// RemoveItem — удаляет предмет из уровня.
// Вызывается после подбора предмета.
func (l *Level) RemoveItem(item *Item) {
	if l == nil || item == nil {
		return
	}
	for i, it := range l.Items {
		if it == item {
			l.Items = append(l.Items[:i], l.Items[i+1:]...)
			return
		}
	}
}

// RemoveMonster — удаляет монстра из уровня.
// Вызывается после смерти монстра.
func (l *Level) RemoveMonster(monster *Monster) {
	if l == nil || monster == nil {
		return
	}
	for i, m := range l.Monsters {
		if m == monster {
			l.Monsters = append(l.Monsters[:i], l.Monsters[i+1:]...)
			return
		}
	}
}