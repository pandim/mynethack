package game

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

// =============================================================================
// ⚠️  ГЛАВНЫЙ УРОК НАШЕЙ ЭПОПЕИ С ГРОМКОСТЬЮ:
//
// Изначально в этой игре была реализована кастомная обёртка `linearVolume`,
// которая работает ИДЕАЛЬНО для линейной регулировки громкости (0.0-1.0).
//
// Мы попытались заменить её на стандартный `effects.Volume` из библиотеки
// `beep`, но столкнулись с множеством проблем:
//
//   1. Поле `Base` в `effects.Volume` — это БАЗА логарифма в формуле
//      `gain = Base^Volume`. При `Base=0` и отрицательных `Volume`
//      получается бесконечное усиление (0^-1 = ∞) → переусиление и искажения!
//
//   2. Формула `gain = 2^Volume` требует конвертации линейной громкости
//      через `math.Log2()`, что усложняет код.
//
//   3. Попытки с `Base=2` работают, но требуют той же конвертации log2.
//
// ВЫВОД: `linearVolume` — простое, надёжное и интуитивное решение.
// Не нужно чинить то, что работает! Ниже приведена проверенная реализация.
// =============================================================================
// linearVolume — линейная обёртка громкости.
// Она реализует интерфейс `beep.Streamer` и умножает каждый сэмпл
// на коэффициент громкости напрямую.
//
// Преимущества:
//   - Интуитивная шкала: 0.0 = тишина, 1.0 = 100% громкости
//   - Никаких логарифмов и конвертаций
//   - Предсказуемый результат: 0.5 = ровно половина громкости
type linearVolume struct {
	streamer beep.Streamer // внутренний стример (наша зацикленная музыка)
	volume   float64       // 0.0 = тишина, 1.0 = 100% (без изменений)
}

// Stream — метод интерфейса `beep.Streamer`.
// Запрашивает сэмплы у внутреннего стримера и умножает каждый на громкость.
// Это происходит в реальном времени при воспроизведении звука.
func (v *linearVolume) Stream(samples [][2]float64) (int, bool) {
	n, ok := v.streamer.Stream(samples)
	for i := 0; i < n; i++ {
		samples[i][0] *= v.volume // левый канал
		samples[i][1] *= v.volume // правый канал
	}
	return n, ok
}

// Err — метод интерфейса `beep.Streamer`.
// Делегирует проверку ошибок внутреннему стримеру.
func (v *linearVolume) Err() error {
	return v.streamer.Err()
}

// =============================================================================
// МУЗЫКА
// =============================================================================
// initMusic — инициализирует фоновую музыку при запуске игры.
//
// Порядок действий:
//   1. Останавливаем старую музыку (если играла) — защита от утечки файловых дескрипторов
//   2. Ищем файл музыки в нескольких местах (текущая папка, родительская)
//   3. Декодируем MP3 через `mp3.Decode()`
//   4. Инициализируем звуковое устройство через `speaker.Init()`
//   5. Зацикливаем трек через `beep.Loop(-1, streamer)`
//   6. Оборачиваем в `linearVolume` для регулировки громкости
//   7. Оборачиваем в `beep.Ctrl` для паузы/возобновления
//   8. Запускаем воспроизведение через `speaker.Play()`
func (g *Game) initMusic() {
	// ⚠️ ВАЖНО: если музыка уже играла (например, после рестарта игры),
	// сначала полностью останавливаем и освобождаем ресурсы.
	// Это предотвращает утечку файловых дескрипторов.
	g.stopMusic()

	if !g.musicEnabled {
		g.logAndSync("MUSIC: Отключено")
		return
	}

	// Ищем файл в нескольких местах (игра может запускаться из разных папок)
	candidates := []string{
		musicFile,
		"./" + musicFile,
		"../" + musicFile,
	}
	var f *os.File
	var err error
	for _, path := range candidates {
		f, err = os.Open(path)
		if err == nil {
			g.logAndSync("MUSIC: Открыт файл %s", path)
			break
		}
	}
	if err != nil {
		g.logAndSync("MUSIC: Файл %s не найден: %v", musicFile, err)
		g.musicEnabled = false
		return
	}

	// Декодируем MP3. Важно: `mp3.Decode()` принимает `io.ReadCloser`
	// и возвращает `beep.StreamSeekCloser`, который мы должны закрыть.
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		g.logAndSync("MUSIC: Не удалось декодировать mp3: %v", err)
		g.musicEnabled = false
		return
	}

	// Инициализируем звуковое устройство с частотой из файла.
	// Размер буфера = 1/10 секунды (баланс между задержкой и нагрузкой на CPU).
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		streamer.Close()
		f.Close()
		g.logAndSync("MUSIC: Ошибка инициализации speaker: %v", err)
		g.musicEnabled = false
		return
	}

	// Зацикливаем трек: -1 = бесконечный цикл
	loop := beep.Loop(-1, streamer)

	// Оборачиваем в нашу линейную обёртку громкости.
	// Это ключевой момент: `linearVolume` даёт интуитивную шкалу 0.0-1.0.
	g.musicVolume = &linearVolume{
		streamer: loop,
		volume:   g.musicLevel,
	}

	// Оборачиваем в `beep.Ctrl` для возможности ставить на паузу.
	// `beep.Ctrl.Paused = true` останавливает поток сэмплов.
	g.musicCtrl = &beep.Ctrl{
		Streamer: g.musicVolume,
		Paused:   false,
	}

	// Запускаем воспроизведение
	speaker.Play(g.musicCtrl)

	// Сохраняем ссылки для последующего закрытия
	g.musicStreamer = streamer
	g.musicFileH = f
	g.logAndSync("MUSIC: Запущена (rate=%d, volume=%.2f)", format.SampleRate, g.musicLevel)
}

// stopMusic — полностью останавливает музыку и освобождает ресурсы.
// Вызывается при выходе из игры через `defer`, а также при повторной
// инициализации музыки (например, после рестарта игры из меню смерти).
func (g *Game) stopMusic() {
	speaker.Clear() // останавливает все активные стримеры
	if g.musicStreamer != nil {
		g.musicStreamer.Close()
		g.musicStreamer = nil
	}
	if g.musicFileH != nil {
		g.musicFileH.Close()
		g.musicFileH = nil
	}
	// Обнуляем контроллеры, чтобы toggleMusic/changeMusicVolume
	// корректно обрабатывали повторные вызовы и не падали
	g.musicVolume = nil
	g.musicCtrl = nil
	speaker.Close() // закрывает звуковое устройство
	g.logAndSync("MUSIC: Остановлена")
}

// toggleMusic — включает/выключает музыку (клавиша M).
//
// ⚠️ ВАЖНО: используем `speaker.Lock()` / `speaker.Unlock()` при изменении
// состояния ВО ВРЕМЯ воспроизведения. Это защищает от гонок данных (race
// conditions), которые могут вызвать треск, щелчки или даже панику.
func (g *Game) toggleMusic() {
	if !g.musicEnabled || g.musicCtrl == nil {
		g.addMessage("Музыка недоступна (нет файла music.mp3).")
		return
	}
	speaker.Lock()
	g.musicCtrl.Paused = !g.musicCtrl.Paused
	paused := g.musicCtrl.Paused
	speaker.Unlock()

	if paused {
		g.addMessage("Музыка выключена.")
	} else {
		g.addMessage("Музыка включена.")
	}
}

// changeMusicVolume — меняет громкость музыки линейно (клавиши + и -).
//
// ⚠️ ИСПОЛЬЗУЕМ `linearVolume`, а НЕ `effects.Volume`!
//
// Почему не `effects.Volume`? Мы прошли через это и выяснили:
//   - `effects.Volume.Base` — это база логарифма в формуле `gain = Base^Volume`
//   - При `Base=0` отрицательные `Volume` дают ∞ (переусиление!)
//   - При `Base=2` нужна конвертация через `math.Log2()`
//   - `linearVolume` проще, надёжнее и интуитивнее
//
// Также используем `speaker.Lock()` для потокобезопасности.
//
// Параметры:
//   - delta: изменение громкости (например, +0.1 = громче на 10%)
func (g *Game) changeMusicVolume(delta float64) {
	if !g.musicEnabled || g.musicVolume == nil {
		g.addMessage("Музыка недоступна.")
		return
	}

	// Изменяем линейный уровень громкости и ограничиваем диапазоном 0.0-1.0
	g.musicLevel += delta
	if g.musicLevel < 0 {
		g.musicLevel = 0
	}
	if g.musicLevel > 1.0 {
		g.musicLevel = 1.0
	}

	// ⚠️ ОБЯЗАТЕЛЬНО: блокируем динамик при изменении громкости во время
	// воспроизведения. Без этого возможны гонки данных и артефакты звука.
	speaker.Lock()
	g.musicVolume.volume = g.musicLevel
	speaker.Unlock()

	percent := int(g.musicLevel * 100)
	g.addMessage(fmt.Sprintf("Громкость: %d%%", percent))
}