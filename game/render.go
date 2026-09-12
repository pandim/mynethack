package game

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

// =============================================================================
// ОТРИСОВКА
// =============================================================================
func (g *Game) drawString(x, y int, s string, style tcell.Style) {
	if g.screen == nil {
		return
	}
	if y < 0 || y >= screenHeight {
		return
	}
	for col := x; col < screenWidth; col++ {
		if col >= 0 {
			g.screen.SetContent(col, y, ' ', nil, style)
		}
	}
	col := x
	for _, ch := range s {
		if col >= 0 && col < screenWidth {
			g.screen.SetContent(col, y, ch, nil, style)
		}
		col++
	}
}

func (g *Game) drawCentered(y int, s string, style tcell.Style) {
	x := (screenWidth - stringWidth(s)) / 2
	g.drawString(x, y, s, style)
}

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
func (g *Game) render() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()

	// 🆕 МИГАНИЕ ЭКРАНА ПРИ ПОДБОРЕ АМУЛЕТА (3 секунды)
	if !g.amuletFlashUntil.IsZero() && time.Now().Before(g.amuletFlashUntil) {
		phase := (time.Now().UnixNano() / int64(250*time.Millisecond)) % 2
		if phase == 0 {
			flashStyle := tcell.StyleDefault.Background(tcell.ColorYellow)
			for y := 0; y < screenHeight; y++ {
				for x := 0; x < screenWidth; x++ {
					g.screen.SetContent(x, y, ' ', nil, flashStyle)
				}
			}
		}
	} else if !g.amuletFlashUntil.IsZero() {
		g.amuletFlashUntil = time.Time{}
	}

	if g.showInventory {
		g.renderInventory()
	} else {
		if g.level != nil && g.player != nil {
			g.level.UpdateFOV(g.player.X, g.player.Y)
			g.level.Render(g.screen, 1, 0)
			g.renderBossHealthBar()
			g.player.Render(g.screen, 1, 0)
		}
	}
	g.renderStatus()
	g.renderMessages()

	// 🆕 Проверка критического HP и сытости
	if g.player != nil {
		if g.player.HP <= 5 && !g.wasInCriticalHP {
			g.popupMessage = "Тебе надо срочно подлечиться, ты еле волочишь ноги!"
			g.wasInCriticalHP = true
		} else if g.player.HP > 5 {
			g.wasInCriticalHP = false
		}
		if g.player.Hunger >= 200 {
			g.wasTooFull = false
		}
	}

	// 🆕 Рисуем попап ТОЛЬКО если мы в основном экране игры и инвентарь закрыт
	if g.state == StatePlaying && !g.showInventory {
		g.renderPopup()
	}
	g.screen.Show()
}

// =============================================================================
// СТАРТОВОЕ МЕНЮ
// =============================================================================
func (g *Game) renderStartMenu() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()
	asciiTitle := []string{
		"                                                                                ",
		"███╗░░██╗███████╗████████╗██╗░░██╗░█████╗░░█████╗░██╗░░██╗░░░░░██████╗░░█████╗░ ",
		"████╗░██║██╔════╝╚══██╔══╝██║░░██║██╔══██╗██╔══██╗██║░██╔╝░░░░██╔════╝░██╔══██╗ ",
		"██╔██╗██║█████╗░░░░░██║░░░███████║███████║██║░░╚═╝█████═╝░███╗██║░░██╗░██║░░██║ ",
		"██║╚████║██╔══╝░░░░░██║░░░██╔══██║██╔══██║██║░░██╗██╔═██╗░╚══╝██║░░╚██╗██║░░██║ ",
		"██║░╚███║███████╗░░░██║░░░██║░░██║██║░░██║╚█████╔╝██║░╚██╗░░░░╚██████╔╝╚█████╔╝ ",
		"╚═╝░░╚══╝╚══════╝░░░╚═╝░░░╚═╝░░╚═╝╚═╝░░╚═╝░╚════╝░╚═╝░░╚═╝░░░░░╚═════╝░░╚════╝░ ",
	}
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	highlightStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	dimStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray).Background(tcell.ColorBlack)

	for i, line := range asciiTitle {
		g.drawString(0, i, line, titleStyle)
	}
	hasSave := false
	if _, err := os.Stat(saveFile); err == nil {
		hasSave = true
	}
	if hasSave {
		g.drawCentered(9, "Найдено сохранение игры!", highlightStyle)
	} else {
		g.drawCentered(9, "Добро пожаловать в подземелье!", style)
	}
	y := 11
	if hasSave {
		g.drawCentered(y, "[L] Загрузить игру", style)
		y++
	}
	g.drawCentered(y, "[N] Новая игра", style)
	y++
	g.drawCentered(y, "[M] Музыка вкл/выкл", style)
	g.drawCentered(screenHeight-2, "[ESC] Выход", dimStyle)
	g.screen.Show()
}

func (g *Game) renderQuitConfirm() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	textStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
	boxWidth := 42
	boxHeight := 7
	boxX := (screenWidth - boxWidth) / 2
	boxY := (screenHeight - boxHeight) / 2
	repeat := func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	}
	g.drawString(boxX, boxY, "╔"+repeat("═", boxWidth-2)+"╗", boxStyle)
	for i := 1; i < boxHeight-1; i++ {
		g.drawString(boxX, boxY+i, "║"+repeat(" ", boxWidth-2)+"║", boxStyle)
	}
	g.drawString(boxX, boxY+boxHeight-1, "╚"+repeat("═", boxWidth-2)+"╝", boxStyle)
	g.drawRawString(boxX+(boxWidth-stringWidth("Вы уверены, что хотите выйти?"))/2, boxY+2, "Вы уверены, что хотите выйти?", textStyle)
	g.drawRawString(boxX+(boxWidth-stringWidth("[Y] Да  [N] Нет"))/2, boxY+4, "[Y] Да  [N] Нет", style)
	g.screen.Show()
}

func (g *Game) renderDeathScreen() {
	if g.screen == nil || g.player == nil {
		return
	}
	g.screen.Clear()
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	boxWidth := 50
	boxHeight := 9
	boxX := (screenWidth - boxWidth) / 2
	boxY := (screenHeight - boxHeight) / 2
	repeat := func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	}
	g.drawString(boxX, boxY, "╔"+repeat("═", boxWidth-2)+"╗", boxStyle)
	for i := 1; i < boxHeight-1; i++ {
		g.drawString(boxX, boxY+i, "║"+repeat(" ", boxWidth-2)+"║", boxStyle)
	}
	g.drawString(boxX, boxY+boxHeight-1, "╚"+repeat("═", boxWidth-2)+"╝", boxStyle)
	
	g.drawRawString(boxX+(boxWidth-stringWidth("ВЫ ПОГИБЛИ!"))/2, boxY+2, "ВЫ ПОГИБЛИ!", titleStyle)
	
	reason := g.deathReason
	if reason == "" {
		reason = "Погиб в подземелье"
	}
	g.drawRawString(boxX+(boxWidth-stringWidth(reason))/2, boxY+3, reason, style)
	
	scoreMsg := fmt.Sprintf("Глубина: %d | Золото: %d | Уровень: %d", g.depth, g.player.Gold, g.player.Level)
	g.drawRawString(boxX+(boxWidth-stringWidth(scoreMsg))/2, boxY+5, scoreMsg, style)
	
	prompt := `Вы хотите выйти "Y" или начать игру заново "N"?`
	g.drawRawString(boxX+(boxWidth-stringWidth(prompt))/2, boxY+7, prompt, style)
	g.screen.Show()
}

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
	if g.player.EquippedWeapon != nil {
		g.drawString(1, y, fmt.Sprintf("Оружие: %s (ATK +%d)", g.player.EquippedWeapon.Name, g.player.EquippedWeapon.Value), equipStyle)
	} else {
		g.drawString(1, y, "Оружие: (нет)", equipStyle)
	}
	y++
	if g.player.EquippedArmor != nil {
		g.drawString(1, y, fmt.Sprintf("Броня:  %s (DEF +%d)", g.player.EquippedArmor.Name, g.player.EquippedArmor.Value), equipStyle)
	} else {
		g.drawString(1, y, "Броня:  (нет)", equipStyle)
	}
	y += 2
	g.drawString(1, y, "--- Предметы ---", style)
	y++
	if len(g.player.Inventory) == 0 {
		g.drawString(1, y, "Инвентарь пуст", style)
	} else {
		for i, item := range g.player.Inventory {
			if item == nil {
				continue
			}
			text := fmt.Sprintf("%d. %s", i+1, item.Name)
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

func (g *Game) renderStatus() {
	if g.screen == nil || g.player == nil {
		return
	}
	hpStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
	var hpPercent float64
	if g.player.MaxHP > 0 {
		hpPercent = float64(g.player.HP) / float64(g.player.MaxHP)
	}
	if hpPercent < 0.25 {
		hpStyle = hpStyle.Foreground(tcell.ColorRed)
	} else if hpPercent < 0.50 {
		hpStyle = hpStyle.Foreground(tcell.ColorFuchsia)
	}
	hungerStatus := "Сыт"
	hungerColor := tcell.ColorWhite
	if g.player.Hunger > 500 && g.player.Hunger <= 900 {
		hungerStatus = "Голоден"
		hungerColor = tcell.ColorFuchsia
	} else if g.player.Hunger > 900 {
		hungerStatus = "Умирает"
		hungerColor = tcell.ColorRed
	}
	g.drawString(1, screenHeight-1, fmt.Sprintf("HP:%d/%d", g.player.HP, g.player.MaxHP), hpStyle)
	g.drawString(12, screenHeight-1, fmt.Sprintf("Lv:%d", g.player.Level), tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(19, screenHeight-1, fmt.Sprintf("XP:%d/%d", g.player.XP, g.player.NextLevelXP()), tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(32, screenHeight-1, fmt.Sprintf("G:%d", g.player.Gold), tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(40, screenHeight-1, fmt.Sprintf("A:%d D:%d", g.player.AttackVal, g.player.Defense), tcell.StyleDefault.Foreground(tcell.ColorYellow))
	g.drawString(55, screenHeight-1, fmt.Sprintf("D:%d", g.depth), tcell.StyleDefault.Foreground(tcell.ColorYellow))
	musicStatus := "♪"
	musicColor := tcell.ColorAqua
	if !g.musicEnabled || (g.musicCtrl != nil && g.musicCtrl.Paused) {
		musicStatus = "X"
		musicColor = tcell.ColorDarkGray
	}
	g.drawString(62, screenHeight-1, fmt.Sprintf("[%s]", musicStatus), tcell.StyleDefault.Foreground(musicColor))
	g.drawString(66, screenHeight-1, fmt.Sprintf("[%s]", hungerStatus), tcell.StyleDefault.Foreground(hungerColor))
	if g.level != nil && g.level.IsStairsOverlap() && g.player.X == g.level.StairsDownX && g.player.Y == g.level.StairsDownY {
		g.drawString(73, screenHeight-1, "[±]", tcell.StyleDefault.Foreground(tcell.ColorYellow))
	}
}

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

func (g *Game) addMessage(msg string) {
	g.messages = append(g.messages, msg)
	if len(g.messages) > messageHeight {
		g.messages = g.messages[1:]
	}
}

func (g *Game) renderBossHealthBar() {
	if g.screen == nil || g.level == nil {
		return
	}
	boss := g.level.GetAliveBoss()
	if boss == nil {
		return
	}
	if boss.Y < 0 || boss.Y >= g.level.Height || boss.X < 0 || boss.X >= g.level.Width {
		return
	}
	if boss.Y >= len(g.level.Tiles) || boss.X >= len(g.level.Tiles[boss.Y]) {
		return
	}
	if !g.level.Tiles[boss.Y][boss.X].Visible {
		return
	}
	offsetX := 1
	offsetY := 0
	barY := boss.Y - 1 + offsetY
	barX := boss.X - 3 + offsetX
	if barY < 0 || barY >= screenHeight {
		return
	}
	barWidth := 7
	hpPercent := float64(boss.HP) / float64(boss.MaxHP)
	filledWidth := int(hpPercent * float64(barWidth))
	if filledWidth < 0 {
		filledWidth = 0
	}
	if filledWidth > barWidth {
		filledWidth = barWidth
	}
	color := tcell.ColorGreen
	if hpPercent < 0.25 {
		color = tcell.ColorRed
	} else if hpPercent < 0.50 {
		color = tcell.ColorYellow
	}
	style := tcell.StyleDefault.Foreground(color).Background(tcell.ColorBlack)
	for i := 0; i < barWidth; i++ {
		x := barX + i
		if x < 0 || x >= screenWidth {
			continue
		}
		ch := rune('-')
		if i < filledWidth {
			ch = rune('=')
		}
		g.screen.SetContent(x, barY, ch, nil, style)
	}
}

func (g *Game) renderHelpScreen() {
	if g.screen == nil {
		return
	}
	g.screen.Clear()
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	keyStyle := tcell.StyleDefault.Foreground(tcell.ColorAqua).Background(tcell.ColorBlack)
	if g.helpPage == helpPageControls {
		g.drawCentered(2, "=== УПРАВЛЕНИЕ ===", titleStyle)
		lines := []struct{ key, desc string }{
			{"WASD / Стрелки", "Движение"}, {"Q E Z C", "Диагонали"}, {"S / Пробел", "Ждать ход"},
			{"I", "Инвентарь"}, {"1-9", "Использовать"}, {"Shift+S", "Сохранить"},
			{">", "Лестница вниз"}, {"<", "Лестница вверх"}, {"T", "Торговля"},
			{"P", "Алтарь"}, {"O", "Сундук"}, {"M", "Музыка"}, {"+ / -", "Громкость"},
			{"?", "Помощь"}, {"ESC", "Выход"}, {"N", "Новая игра"},
		}
		y := 4
		for _, line := range lines {
			g.drawString(5, y, line.key, keyStyle)
			g.drawString(22, y, "- "+line.desc, style)
			y++
		}
	} else if g.helpPage == helpPageSymbols {
		g.drawCentered(2, "=== СИМВОЛЫ ===", titleStyle)
		symbols := []struct{ sym, desc string }{
			{"@", "Вы"}, {">", "Лестница вниз"}, {"<", "Лестница вверх"}, {"±", "Совмещённая лестница"},
			{"g o s r", "Монстры"}, {"! / [ $ %", "Предметы"}, {"~", "Свитки"}, {"k", "Ключ"},
			{"M", "Торговец"}, {"_", "Алтарь"}, {"&", "Сундук"}, {"G N D B K", "Боссы"},
		}
		y := 4
		for _, s := range symbols {
			g.drawString(5, y, s.sym, keyStyle)
			g.drawString(22, y, "- "+s.desc, style)
			y++
		}
	} else if g.helpPage == helpPageMechanics {
		g.drawCentered(2, "=== МЕХАНИКИ ===", titleStyle)
		mechanics := []struct{ title, desc string }{
			{"Свитки (~)", "Карта, Телепорт, Молния, Изгнание"},
			{"Реликвии (*)", "5 уникальных предметов. Продаются торговцу"},
			{"Золотые сундуки", "Требуют ключ. Ценный лут"},
			{"Алтари (_)", "Благословляют экипировку (+2) за золото"},
			{"Боссы", "Охраняют лестницу каждые 3 уровня"},
			{"Победа", "Убить Короля Бездны (15 ур.), забрать Амулет"},
			{"", "Вернуться на 1-й уровень с Амулетом!"},
		}
		y := 4
		for _, m := range mechanics {
			if m.title != "" {
				g.drawString(5, y, m.title, keyStyle)
				g.drawString(25, y, "- "+m.desc, style)
			} else {
				g.drawString(25, y, m.desc, style)
			}
			y++
		}
	}
	g.drawCentered(screenHeight-4, fmt.Sprintf("Страница %d/%d", g.helpPage+1, helpPageCount), titleStyle)
	g.drawCentered(screenHeight-3, "← → или A/D — перелистывание", style)
	g.drawCentered(screenHeight-2, "Любая другая клавиша — возврат в игру", style)
	g.screen.Show()
}

func (g *Game) renderPopup() {
	if g.popupMessage == "" {
		return
	}
	boxWidth := 60
	boxHeight := 5
	boxX := (screenWidth - boxWidth) / 2
	boxY := (screenHeight - boxHeight) / 2
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	textStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	repeat := func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	}
	g.drawString(boxX, boxY, "╔"+repeat("═", boxWidth-2)+"╗", boxStyle)
	for i := 1; i < boxHeight-1; i++ {
		g.drawString(boxX, boxY+i, "║"+repeat(" ", boxWidth-2)+"║", boxStyle)
	}
	g.drawString(boxX, boxY+boxHeight-1, "╚"+repeat("═", boxWidth-2)+"╝", boxStyle)
	g.drawRawString(boxX+(boxWidth-stringWidth(g.popupMessage))/2, boxY+2, g.popupMessage, textStyle)
}

// =============================================================================
// 🆕 ЭКРАН ПОБЕДЫ (С ПРОВЕРКОЙ НА ИДЕАЛЬНУЮ ПОБЕДУ)
// =============================================================================
func (g *Game) renderVictoryScreen() {
	if g.screen == nil || g.player == nil {
		return
	}
	g.screen.Clear()
	
	titleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	style := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	goldStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(tcell.ColorBlack)
	greenStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Background(tcell.ColorBlack)
	perfectStyle := tcell.StyleDefault.Foreground(tcell.ColorFuchsia).Background(tcell.ColorBlack) // 🆕 Для идеальной победы

	// 🆕 ПРОВЕРКА НА ИДЕАЛЬНУЮ ПОБЕДУ (Сбор всех 5 реликвий)
	relicCount := g.player.CountRelics()
	isPerfect := relicCount == 5

	if isPerfect {
		g.drawCentered(4, "★ ИДЕАЛЬНАЯ ПОБЕДА! ★", perfectStyle)
		g.drawCentered(6, "Вы вернулись с Амулетом и всеми 5 Реликвиями!", perfectStyle)
		g.drawCentered(8, "Вы — истинная легенда подземелья!", greenStyle)
	} else {
		g.drawCentered(4, "★ ПОБЕДА! ★", titleStyle)
		g.drawCentered(6, "Вы вернулись на поверхность с Амулетом Бездны!", style)
	}

	// 🆕 Добавляем счетчик реликвий в итоговую статистику
	statsMsg := fmt.Sprintf("Глубина: %d | Золото: %d | Уровень: %d | Реликвии: %d/5", g.depth, g.player.Gold, g.player.Level, relicCount)
	g.drawCentered(10, statsMsg, goldStyle)

	g.drawCentered(14, "Подземелье позади.", style)
	g.drawCentered(18, "Нажмите любую клавишу для выхода", style)
	g.screen.Show()
}