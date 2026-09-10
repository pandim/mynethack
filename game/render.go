package game

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// ОТРИСОВКА
// =============================================================================

// drawString — рисует строку в заданной позиции экрана.
//
// ⚠️ ВАЖНО: перед записью символов очищает всю строку от позиции x до конца
// экрана пробелами. Это предотвращает появление "хвостов" — остатков предыдущих
// более длинных строк, которые были на том же месте.
//
// Пример проблемы без очистки:
//   Было: "Вы атаковали гоблина на 50 урона!"
//   Стало: "Подобрано меч"
//   Результат без очистки: "Подобрано мечна 50 урона!"  ← хвост от старой строки!
//
// Параметры:
//   - x, y: позиция на экране
//   - s: строка для отрисовки
//   - style: стиль (цвет текста и фона)
func (g *Game) drawString(x, y int, s string, style tcell.Style) {
	if g.screen == nil {
		return
	}

	// Проверка вертикальной границы (чтобы не делать лишнюю работу)
	if y < 0 || y >= screenHeight {
		return
	}

	// ШАГ 1: Очищаем всю строку от x до конца экрана пробелами.
	// Используем тот же стиль (важен фон — обычно чёрный),
	// чтобы не осталось артефактов от предыдущих строк.
	for col := x; col < screenWidth; col++ {
		if col >= 0 {
			g.screen.SetContent(col, y, ' ', nil, style)
		}
	}

	// ШАГ 2: Рисуем саму строку поверх очищенной области
	col := x
	for _, ch := range s {
		if col >= 0 && col < screenWidth {
			g.screen.SetContent(col, y, ch, nil, style)
		}
		col++
	}
}

// drawCentered — рисует строку по центру экрана
func (g *Game) drawCentered(y int, s string, style tcell.Style) {
	x := (screenWidth - stringWidth(s)) / 2
	g.drawString(x, y, s, style)
}
// drawRawString — рисует строку БЕЗ очистки строки экрана.
// Используется для рисования текста внутри рамок,
// чтобы не стирать правую границу рамки.
//
// В отличие от drawString, эта функция НЕ очищает строку до конца экрана.
// Она просто записывает символы начиная с позиции x.
func (g *Game) drawRawString(x, y int, s string, style tcell.Style) {
	if g.screen == nil {
		return
	}
	if y < 0 || y >= screenHeight {
		return
	}
	col := x
	for _, ch := range s {
		if col >= 0 && col < screenWidth {
			g.screen.SetContent(col, y, ch, nil, style)
		}
		col++
	}
}
// render — основная функция отрисовки игрового экрана
//
// После отрисовки уровня вызывается renderBossHealthBar,
// которая рисует полоску здоровья босса над ним на карте.
func (g *Game) render() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()

	if g.showInventory {
		g.renderInventory()
	} else {
		if g.level != nil && g.player != nil {
			// Обновляем поле зрения (туман войны) и рисуем уровень
			g.level.UpdateFOV(g.player.X, g.player.Y)
			g.level.Render(g.screen, 1, 0)

			// Полоска здоровья босса (рисуется поверх карты)
			g.renderBossHealthBar()

			g.player.Render(g.screen, 1, 0)
		}
	}

	g.renderStatus()
	g.renderMessages()
	g.screen.Show()
}

// =============================================================================
// 🆕 СТАРТОВОЕ МЕНЮ С ASCII-АРТ ЗАГОЛОВКОМ
// =============================================================================
//
// renderStartMenu — отрисовка стартового меню
//
// Показывается ВСЕГДА при запуске игры (не только при наличии сохранения).
// Сверху отображается ASCII-арт заголовок "NETHACK-GO" (6 строк по 80 символов).
//
// Если сохранение есть — показывается опция [L] Загрузить игру.
// Если сохранения нет — эта опция скрывается.
func (g *Game) renderStartMenu() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()

	// 🆕 ASCII-арт заголовок "NETHACK-GO"
	// 6 строк, каждая ровно 80 символов (занимает всю ширину экрана)
	asciiTitle := []string{
		"███╗░░██╗███████╗████████╗██╗░░██╗░█████╗░░█████╗░██╗░░██╗░░░░░██████╗░░█████╗░",
		"████╗░██║██╔════╝╚══██╔══╝██║░░██║██╔══██╗██╔══██╗██║░██╔╝░░░░██╔════╝░██╔══██╗",
		"██╔██╗██║█████╗░░░░░██║░░░███████║███████║██║░░╚═╝█████═╝░███╗██║░░██╗░██║░░██║",
		"██║╚████║██╔══╝░░░░░██║░░░██╔══██║██╔══██║██║░░██╗██╔═██╗░╚══╝██║░░╚██╗██║░░██║",
		"██║░╚███║███████╗░░░██║░░░██║░░██║██║░░██║╚█████╔╝██║░╚██╗░░░░╚██████╔╝╚█████╔╝",
		"╚═╝░░╚══╝╚══════╝░░░╚═╝░░░╚═╝░░╚═╝╚═╝░░╚═╝░╚════╝░╚═╝░░╚═╝░░░░░╚═════╝░░╚════╝░",
	}

	// Стили
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	highlightStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	dimStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray).Background(tcell.ColorBlack)

	// Рисуем ASCII-арт заголовок (строки 0-5, начиная с y=0)
	for i, line := range asciiTitle {
		g.drawString(0, i, line, titleStyle)
	}

	// Проверяем наличие сохранения
	hasSave := false
	if _, err := os.Stat(saveFile); err == nil {
		hasSave = true
	}

	// Строка 7: статус сохранения
	if hasSave {
		g.drawCentered(7, "Найдено сохранение игры!", highlightStyle)
	} else {
		g.drawCentered(7, "Добро пожаловать в подземелье!", style)
	}

	// Строки 9+: опции меню
	y := 9
	if hasSave {
		g.drawCentered(y, "[L] Загрузить игру", style)
		y++
	}
	g.drawCentered(y, "[N] Новая игра", style)
	y++
	g.drawCentered(y, "[M] Музыка вкл/выкл", style)
	y += 2

	// Строка выхода — ближе к низу экрана
	g.drawCentered(screenHeight-2, "[ESC] Выход", dimStyle)

	g.screen.Show()
}

// renderQuitConfirm — отрисовка диалога подтверждения выхода
//
// 🆕 ОКНО ИЗ ДВОЙНЫХ ПОЛОСОК ASCII:
// Рамка рисуется символами ╔═╗, ║, ╚═╝ для красивого оформления.
//
// ⚠️ ВАЖНО: Рамка рисуется через drawString (полные строки),
// а текст внутри — через drawRawString (без очистки),
// чтобы не стирать правую границу рамки.
func (g *Game) renderQuitConfirm() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()

	// Стили
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	textStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)

	// Размеры окна
	boxWidth := 42
	boxHeight := 7
	boxX := (screenWidth - boxWidth) / 2
	boxY := (screenHeight - boxHeight) / 2

	// Вспомогательная функция для повторения строки
	repeat := func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	}

	// Верхняя граница: ╔═══...═══╗
	// Рисуем всю строку целиком через drawString
	g.drawString(boxX, boxY, "╔"+repeat("═", boxWidth-2)+"╗", boxStyle)

	// Средние строки: ║     ...     ║
	// Рисуем всю строку целиком через drawString
	for i := 1; i < boxHeight-1; i++ {
		g.drawString(boxX, boxY+i, "║"+repeat(" ", boxWidth-2)+"║", boxStyle)
	}

	// Нижняя граница: ╚═══...═══╝
	// Рисуем всю строку целиком через drawString
	g.drawString(boxX, boxY+boxHeight-1, "╚"+repeat("═", boxWidth-2)+"╝", boxStyle)

	// Текст внутри рамки — рисуем через drawRawString (БЕЗ очистки),
	// чтобы не стереть правую границу рамки
	text1 := "Вы уверены, что хотите выйти?"
	text1X := boxX + (boxWidth-stringWidth(text1))/2
	g.drawRawString(text1X, boxY+2, text1, textStyle)

	text2 := "[Y] Да  [N] Нет"
	text2X := boxX + (boxWidth-stringWidth(text2))/2
	g.drawRawString(text2X, boxY+4, text2, style)

	g.screen.Show()
}

// renderDeathScreen — отрисовка экрана смерти
//
// 🆕 ОКНО ИЗ ДВОЙНЫХ ПОЛОСОК ASCII:
// Рамка рисуется символами ╔═╗, ║, ╚═╝ для красивого оформления.
//
// ⚠️ ВАЖНО: Рамка рисуется через drawString (полные строки),
// а текст внутри — через drawRawString (без очистки),
// чтобы не стирать правую границу рамки.
func (g *Game) renderDeathScreen() {
	if g.screen == nil || g.player == nil {
		return
	}
	g.screen.Clear()

	// Стили
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)

	// Размеры окна
	boxWidth := 50
	boxHeight := 9
	boxX := (screenWidth - boxWidth) / 2
	boxY := (screenHeight - boxHeight) / 2

	// Вспомогательная функция для повторения строки
	repeat := func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	}

	// Верхняя граница: ╔═══...═══╗
	g.drawString(boxX, boxY, "╔"+repeat("═", boxWidth-2)+"╗", boxStyle)

	// Средние строки: ║     ...     ║
	for i := 1; i < boxHeight-1; i++ {
		g.drawString(boxX, boxY+i, "║"+repeat(" ", boxWidth-2)+"║", boxStyle)
	}

	// Нижняя граница: ╚═══...═══╝
	g.drawString(boxX, boxY+boxHeight-1, "╚"+repeat("═", boxWidth-2)+"╝", boxStyle)

	// Текст внутри рамки — рисуем через drawRawString (БЕЗ очистки)
	// Заголовок
	title := "ВЫ ПОГИБЛИ!"
	titleX := boxX + (boxWidth-stringWidth(title))/2
	g.drawRawString(titleX, boxY+2, title, titleStyle)

	// Статистика
	scoreMsg := fmt.Sprintf("Глубина: %d | Золото: %d | Уровень: %d",
		g.depth, g.player.Gold, g.player.Level)
	scoreX := boxX + (boxWidth-stringWidth(scoreMsg))/2
	g.drawRawString(scoreX, boxY+4, scoreMsg, style)

	// Подсказка
	prompt := `Вы хотите выйти "Y" или начать игру заново "N"?`
	promptX := boxX + (boxWidth-stringWidth(prompt))/2
	g.drawRawString(promptX, boxY+6, prompt, style)

	g.screen.Show()
}

// renderInventory — отрисовка инвентаря с экипировкой
func (g *Game) renderInventory() {
	if g.screen == nil || g.player == nil {
		return
	}
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	equipStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)

	g.drawString(1, 2, "=== ИНВЕНТАРЬ === (ESC - закрыть)", style)

	y := 4
	g.drawString(1, y, "--- Экипировка ---", equipStyle)
	y++

	// Показываем экипированное оружие
	if g.player.EquippedWeapon != nil {
		g.drawString(1, y,
			fmt.Sprintf("Оружие: %s (ATK +%d)", g.player.EquippedWeapon.Name, g.player.EquippedWeapon.Value),
			equipStyle)
	} else {
		g.drawString(1, y, "Оружие: (нет)", equipStyle)
	}
	y++

	// Показываем экипированную броню
	if g.player.EquippedArmor != nil {
		g.drawString(1, y,
			fmt.Sprintf("Броня:  %s (DEF +%d)", g.player.EquippedArmor.Name, g.player.EquippedArmor.Value),
			equipStyle)
	} else {
		g.drawString(1, y, "Броня:  (нет)", equipStyle)
	}
	y += 2

	g.drawString(1, y, "--- Предметы ---", style)
	y++

	// Показываем предметы в инвентаре
	if len(g.player.Inventory) == 0 {
		g.drawString(1, y, "Инвентарь пуст", style)
	} else {
		for i, item := range g.player.Inventory {
			if item == nil {
				continue
			}
			text := fmt.Sprintf("%d. %s", i+1, item.Name)
			// Показываем количество предметов в стопке
			if item.Count > 1 {
				text += fmt.Sprintf(" x%d", item.Count)
			}
			g.drawString(1, y, text, style)
			y++
			if y >= screenHeight-5 {
				break
			}
		}
	}

	g.drawString(1, screenHeight-3, "Нажмите цифру для использования", style)
}

// renderStatus — отрисовка статус-бара внизу экрана
func (g *Game) renderStatus() {
	if g.screen == nil || g.player == nil {
		return
	}

	// Цвет HP зависит от процента здоровья
	hpStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
	var hpPercent float64
	if g.player.MaxHP > 0 {
		hpPercent = float64(g.player.HP) / float64(g.player.MaxHP)
	}
	if hpPercent < 0.25 {
		hpStyle = hpStyle.Foreground(tcell.ColorRed) // критическое здоровье
	} else if hpPercent < 0.50 {
		hpStyle = hpStyle.Foreground(tcell.ColorFuchsia) // низкое здоровье
	}

	// Статус голода
	hungerStatus := "Сыт"
	hungerColor := tcell.ColorWhite
	if g.player.Hunger > 500 && g.player.Hunger <= 900 {
		hungerStatus = "Голоден"
		hungerColor = tcell.ColorFuchsia
	} else if g.player.Hunger > 900 {
		hungerStatus = "Умирает"
		hungerColor = tcell.ColorRed
	}

	// Рисуем все показатели статуса
	g.drawString(1, screenHeight-1,
		fmt.Sprintf("HP:%d/%d", g.player.HP, g.player.MaxHP), hpStyle)
	g.drawString(12, screenHeight-1,
		fmt.Sprintf("Lv:%d", g.player.Level),
		tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(19, screenHeight-1,
		fmt.Sprintf("XP:%d/%d", g.player.XP, g.player.NextLevelXP()),
		tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(32, screenHeight-1,
		fmt.Sprintf("G:%d", g.player.Gold),
		tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(40, screenHeight-1,
		fmt.Sprintf("A:%d D:%d", g.player.AttackVal, g.player.Defense),
		tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(55, screenHeight-1,
		fmt.Sprintf("D:%d", g.depth),
		tcell.StyleDefault.Foreground(tcell.ColorYellow))

	// Иконка музыки: ♪ если играет, X если выключена
	musicStatus := "♪"
	musicColor := tcell.ColorAqua
	if !g.musicEnabled || (g.musicCtrl != nil && g.musicCtrl.Paused) {
		musicStatus = "X"
		musicColor = tcell.ColorDarkGray
	}
	g.drawString(62, screenHeight-1,
		fmt.Sprintf("[%s]", musicStatus),
		tcell.StyleDefault.Foreground(musicColor))

	// Индикатор голода
	hungerText := fmt.Sprintf("[%s]", hungerStatus)
	hungerStyle := tcell.StyleDefault.Foreground(hungerColor)
	g.drawString(66, screenHeight-1, hungerText, hungerStyle)

	// Индикатор совпадения лестниц (редкая ситуация, когда лестницы на одной клетке)
	if g.level != nil && g.level.IsStairsOverlap() &&
		g.player.X == g.level.StairsDownX &&
		g.player.Y == g.level.StairsDownY {
		g.drawString(73, screenHeight-1, "[±]",
			tcell.StyleDefault.Foreground(tcell.ColorYellow))
	}
}

// renderMessages — отрисовка последних сообщений
func (g *Game) renderMessages() {
	if g.screen == nil {
		return
	}
	startY := screenHeight - messageHeight - 1
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite)

	for i, msg := range g.messages {
		if i >= messageHeight {
			break
		}
		g.drawString(1, startY+i, msg, style)
	}
}

// addMessage — добавляет сообщение в очередь (старые удаляются)
func (g *Game) addMessage(msg string) {
	g.messages = append(g.messages, msg)
	if len(g.messages) > messageHeight {
		g.messages = g.messages[1:]
	}
}

// =============================================================================
// ПОЛОСКА ЗДОРОВЬЯ БОССА
// =============================================================================
//
// renderBossHealthBar — рисует полоску здоровья босса над ним на карте.
//
// Полоска здоровья отображается только если:
//   - На уровне есть живой босс
//   - Босс находится в видимой клетке (в радиусе зрения игрока)
//
// Полоска рисуется над боссом (в строке boss.Y - 1) и имеет ширину 7 символов.
// Заполненная часть отображается символом '=', пустая — символом '-'.
// Цвет полоски зависит от процента здоровья:
//   - Зелёный: > 50%
//   - Жёлтый: 25-50%
//   - Красный: < 25%
//
// Если босс находится у верхнего края карты (boss.Y == 0), полоска не рисуется.
func (g *Game) renderBossHealthBar() {
	if g.screen == nil || g.level == nil {
		return
	}

	// Получаем живого босса на уровне
	boss := g.level.GetAliveBoss()
	if boss == nil {
		return
	}

	// Проверяем, что босс в пределах карты
	if boss.Y < 0 || boss.Y >= g.level.Height || boss.X < 0 || boss.X >= g.level.Width {
		return
	}
	if boss.Y >= len(g.level.Tiles) || boss.X >= len(g.level.Tiles[boss.Y]) {
		return
	}

	// Полоска здоровья отображается только если босс виден
	// (находится в радиусе зрения игрока)
	if !g.level.Tiles[boss.Y][boss.X].Visible {
		return
	}

	// Позиция полоски здоровья: над боссом
	// Используем тот же offset, что и в level.Render (offsetX=1, offsetY=0)
	offsetX := 1
	offsetY := 0
	barY := boss.Y - 1 + offsetY
	barX := boss.X - 3 + offsetX // центрируем полоску над боссом (ширина 7)

	// Если босс у верхнего края карты, полоска не рисуется
	if barY < 0 || barY >= screenHeight {
		return
	}

	// Ширина полоски: 7 символов
	barWidth := 7

	// Процент здоровья босса
	hpPercent := float64(boss.HP) / float64(boss.MaxHP)
	filledWidth := int(hpPercent * float64(barWidth))
	if filledWidth < 0 {
		filledWidth = 0
	}
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	// Цвет полоски в зависимости от процента здоровья
	color := tcell.ColorGreen // > 50% — зелёный
	if hpPercent < 0.25 {
		color = tcell.ColorRed // < 25% — красный
	} else if hpPercent < 0.50 {
		color = tcell.ColorYellow // 25-50% — жёлтый
	}

	// Рисуем полоску здоровья
	style := tcell.StyleDefault.
		Foreground(color).
		Background(tcell.ColorBlack)

	for i := 0; i < barWidth; i++ {
		x := barX + i
		if x < 0 || x >= screenWidth {
			continue
		}
		ch := rune('-') // пустая часть полоски
		if i < filledWidth {
			ch = rune('=') // заполненная часть полоски
		}
		g.screen.SetContent(x, barY, ch, nil, style)
	}
}

// renderHelpScreen — отрисовка экрана помощи (клавиша ?).
//
// ⚠️ Экран разбит на 2 страницы, так как вся информация не помещается
// на один экран (высота 24 строки):
//   - Страница 1 (helpPageControls): управление — список клавиш
//   - Страница 2 (helpPageSymbols):  символы — легенда карты
//
// Перелистывание: стрелки ← → или клавиши A/D.
// Любая другая клавиша — возврат в игру.
func (g *Game) renderHelpScreen() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()

	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	keyStyle := tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(tcell.ColorBlack)

	if g.helpPage == helpPageControls {
		// =====================================================================
		// СТРАНИЦА 1: УПРАВЛЕНИЕ
		// =====================================================================
		g.drawCentered(2, "=== УПРАВЛЕНИЕ ===", titleStyle)

		// Список клавиш управления
		lines := []struct {
			key  string
			desc string
		}{
			{"WASD / Стрелки", "Движение (4 направления)"},
			{"Q E Z C", "Движение по диагонали"},
			{"S / Пробел", "Ждать ход"},
			{"I", "Открыть инвентарь"},
			{"1-9", "Использовать предмет"},
			{"Shift+S", "Сохранить игру"},
			{">", "Спуститься по лестнице"},
			{"<", "Подняться по лестнице"},
			{"T", "Торговля (на торговце)"},
			{"B", "Благословение (на алтаре)"},
			{"O", "Открыть сундук"},
			{"M", "Музыка вкл/выкл"},
			{"+ / -", "Громкость музыки"},
			{"?", "Экран помощи (этот)"},
			{"ESC / Ctrl+C", "Выход из игры"},
			{"N", "Новая игра (в игре)"},
		}

		y := 4
		for _, line := range lines {
			g.drawString(5, y, line.key, keyStyle)
			g.drawString(22, y, "- "+line.desc, style)
			y++
		}
	} else {
		// =====================================================================
		// СТРАНИЦА 2: СИМВОЛЫ
		// =====================================================================
		g.drawCentered(2, "=== СИМВОЛЫ ===", titleStyle)

		// Легенда символов
		symbols := []struct {
			sym  string
			desc string
		}{
			{"@", "Вы (игрок)"},
			{">", "Лестница вниз"},
			{"<", "Лестница вверх"},
			{"±", "Лестница вверх И вниз (совмещённая)"},
			{"g o s r", "Монстры (гоблин, орк, скелет, крыса)"},
			{"! / [ $ %", "Предметы (зелье, меч, щит, золото, еда)"},
			{"M", "Торговец (T — торговля)"},
			{"_", "Алтарь (B — благословение)"},
			{"&", "Сундук (O — открыть)"},
			{"G N D B", "Боссы (охраняют лестницу, каждые 6 уровней)"},
		}

		y := 4
		for _, s := range symbols {
			g.drawString(5, y, s.sym, keyStyle)
			g.drawString(22, y, "- "+s.desc, style)
			y++
		}
	}

	// =========================================================================
	// ИНДИКАТОР СТРАНИЦЫ И ПОДСКАЗКИ (общие для обеих страниц)
	// =========================================================================

	// Индикатор текущей страницы
	pageIndicator := fmt.Sprintf("Страница %d/%d", g.helpPage+1, helpPageCount)
	g.drawCentered(screenHeight-4, pageIndicator, titleStyle)

	// Подсказки по навигации
	g.drawCentered(screenHeight-3, "← → или A/D — перелистывание", style)
	g.drawCentered(screenHeight-2, "Любая другая клавиша — возврат в игру", style)

	g.screen.Show()
}
// =============================================================================
// 🆕 ЭТАП 3: ЭКРАН ПОБЕДЫ
// =============================================================================
//
// renderVictoryScreen — отрисовка экрана победы.
// Показывается, когда игрок возвращается на уровень 1 с Амулетом Бездны.
//
// Проверка победы происходит в input.go → handleMovement → case '<'.
// Состояние StateVictory определено в game.go.
// Обработка ввода: input.go → handleVictoryInput (любая клавиша — выход).
//
// Экран показывает:
//   - Поздравление с победой
//   - Статистику: глубина, золото, уровень персонажа
//   - Подсказку: любая клавиша — выход из игры
func (g *Game) renderVictoryScreen() {
	if g.screen == nil || g.player == nil {
		return
	}
	g.screen.Clear()

	// Стили для разных элементов экрана
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	goldStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	greenStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)

	// Заголовок
	g.drawCentered(4, "★ ПОБЕДА! ★", titleStyle)
	g.drawCentered(6, "Вы вернулись на поверхность с Амулетом Бездны!", style)

	// Статистика игры
	scoreMsg := fmt.Sprintf("Глубина: %d | Золото: %d | Уровень: %d",
		g.depth, g.player.Gold, g.player.Level)
	g.drawCentered(9, scoreMsg, goldStyle)

	// Поздравление
	g.drawCentered(12, "Подземелье позади. Вы — легенда!", greenStyle)
	
	// Подсказка
	g.drawCentered(18, "Нажмите любую клавишу для выхода", style)

	g.screen.Show()
}