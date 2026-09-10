package game

import (
	"fmt"
	"os"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

// linearVolume — обёртка для beep.Streamer, реализующая линейную громкость.
// Стандартный effects.Volume в beep использует логарифмическую шкалу,
// что неудобно для пользовательского управления (например, ползунка или шагов по 10%).
// Эта обёртка просто умножает каждый сэмпл на коэффициент volume (0.0 - 1.0).
type linearVolume struct {
	streamer beep.Streamer
	volume   float64 // 0.0 (тишина) до 1.0 (полная громкость)
}

// Stream реализует интерфейс beep.Streamer.
func (v *linearVolume) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = v.streamer.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= v.volume
		samples[i][1] *= v.volume
	}
	return n, ok
}

// Err реализует интерфейс beep.Streamer.
func (v *linearVolume) Err() error {
	return v.streamer.Err()
}

// =============================================================================
// УПРАВЛЕНИЕ МУЗЫКОЙ
// =============================================================================

// initMusic — инициализирует фоновую музыку.
// Вызывается один раз при старте игры.
func (g *Game) initMusic() {
	// Открываем файл музыки
	file, err := os.Open(musicFile)
	if err != nil {
		g.musicEnabled = false
		if g.logger != nil {
			g.logger.Printf("MUSIC: Не удалось открыть файл %s: %v", musicFile, err)
		}
		return
	}
	g.musicFileH = file

	// Декодируем MP3
	streamer, format, err := mp3.Decode(g.musicFileH)
	if err != nil {
		g.musicEnabled = false
		if g.musicFileH != nil {
			g.musicFileH.Close()
		}
		if g.logger != nil {
			g.logger.Printf("MUSIC: Не удалось декодировать %s: %v", musicFile, err)
		}
		return
	}
	g.musicStreamer = streamer

	// Инициализируем аудиоустройство (speaker)
	// Буферизация: 1/10 секунды
	if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
		g.musicEnabled = false
		if g.musicFileH != nil {
			g.musicFileH.Close()
		}
		if streamer != nil {
			streamer.Close()
		}
		// Гарантированное освобождение ресурсов аудиоустройства при ошибке инициализации
		speaker.Close()
		if g.logger != nil {
			g.logger.Printf("MUSIC: Не удалось инициализировать speaker: %v", err)
		}
		return
	}

	// Создаём контроллер для паузы/возобновления
	g.musicCtrl = &beep.Ctrl{Streamer: g.musicStreamer}
	
	// Создаём обёртку линейной громкости (по умолчанию 50%)
	g.musicLevel = 0.5
	g.musicVolume = &linearVolume{
		streamer: g.musicCtrl,
		volume:   g.musicLevel,
	}

	// Запускаем воспроизведение в бесконечном цикле
	speaker.Play(beep.Seq(g.musicVolume, beep.Callback(func() {
		// Когда трек заканчивается, перематываем его в начало
		if seeker, ok := g.musicStreamer.(beep.StreamSeeker); ok {
			seeker.Seek(0)
		}
		speaker.Play(beep.Seq(g.musicVolume, beep.Callback(func() {}))) // Рекурсивный перезапуск не нужен, просто вернемся
	})))

	g.musicEnabled = true
	if g.logger != nil {
		g.logger.Println("MUSIC: Музыка успешно инициализирована и запущена")
	}
}

// toggleMusic — включает или выключает музыку.
func (g *Game) toggleMusic() {
	if !g.musicEnabled || g.musicCtrl == nil {
		return
	}

	if g.musicCtrl.Paused {
		g.musicCtrl.Paused = false
		g.addMessage("Музыка включена")
		if g.logger != nil {
			g.logger.Println("MUSIC: Музыка возобновлена")
		}
	} else {
		g.musicCtrl.Paused = true
		g.addMessage("Музыка выключена")
		if g.logger != nil {
			g.logger.Println("MUSIC: Музыка приостановлена")
		}
	}
}

// changeMusicVolume — изменяет громкость музыки на заданную дельту (например, +0.1 или -0.1).
// Значение ограничивается диапазоном [0.0, 1.0].
func (g *Game) changeMusicVolume(delta float64) {
	if !g.musicEnabled || g.musicVolume == nil {
		return
	}

	g.musicLevel += delta
	if g.musicLevel < 0.0 {
		g.musicLevel = 0.0
	} else if g.musicLevel > 1.0 {
		g.musicLevel = 1.0
	}

	g.musicVolume.volume = g.musicLevel

	// Форматируем в проценты для сообщения
	percent := int(g.musicLevel * 100)
	g.addMessage(fmt.Sprintf("Громкость музыки: %d%%", percent))
	if g.logger != nil {
		g.logger.Printf("MUSIC: Громкость изменена на %d%%", percent)
	}
}

// stopMusic — корректно закрывает все ресурсы, связанные с музыкой (ИСПРАВЛЕНО ИМЯ)
// Должна вызываться при выходе из игры.
func (g *Game) stopMusic() {
	if g.musicStreamer != nil {
		g.musicStreamer.Close()
	}
	if g.musicFileH != nil {
		g.musicFileH.Close()
	}
	speaker.Close()
	if g.logger != nil {
		g.logger.Println("MUSIC: Ресурсы музыки освобождены")
	}
}