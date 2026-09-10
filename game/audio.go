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
// ЛИНЕЙНАЯ ГРОМКОСТЬ
// =============================================================================
type linearVolume struct {
	streamer beep.Streamer
	volume   float64 // 0.0 (тишина) до 1.0 (полная громкость)
}

func (v *linearVolume) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = v.streamer.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= v.volume
		samples[i][1] *= v.volume
	}
	return n, ok
}

func (v *linearVolume) Err() error {
	return v.streamer.Err()
}

// =============================================================================
// УПРАВЛЕНИЕ МУЗЫКОЙ
// =============================================================================

// initMusic — инициализирует фоновую музыку с зацикливанием.
func (g *Game) initMusic() {
	// Останавливаем старую музыку (если играла)
	g.stopMusic()

	if !g.musicEnabled {
		g.logAndSync("MUSIC: Отключено")
		return
	}

	// Ищем файл в нескольких местах
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

	// Декодируем MP3
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		f.Close()
		g.logAndSync("MUSIC: Не удалось декодировать mp3: %v", err)
		g.musicEnabled = false
		return
	}

	// Инициализируем звуковое устройство
	err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	if err != nil {
		streamer.Close()
		f.Close()
		g.logAndSync("MUSIC: Ошибка инициализации speaker: %v", err)
		g.musicEnabled = false
		return
	}

	// 🆕 ИСПРАВЛЕНИЕ: Зацикливаем трек через beep.Loop(-1, streamer)
	// -1 означает бесконечный цикл
	loop := beep.Loop(-1, streamer)

	// Оборачиваем в линейную обёртку громкости
	g.musicVolume = &linearVolume{
		streamer: loop,
		volume:   g.musicLevel,
	}

	// Оборачиваем в beep.Ctrl для паузы/возобновления
	g.musicCtrl = &beep.Ctrl{
		Streamer: g.musicVolume,
		Paused:   false,
	}

	// Запускаем воспроизведение
	speaker.Play(g.musicCtrl)

	// Сохраняем ссылки для последующего закрытия
	g.musicStreamer = streamer
	g.musicFileH = f

	g.logAndSync("MUSIC: Запущена (rate=%d, volume=%.2f, looped)", format.SampleRate, g.musicLevel)
}

// stopMusic — полностью останавливает музыку и освобождает ресурсы.
func (g *Game) stopMusic() {
	speaker.Clear()

	if g.musicStreamer != nil {
		g.musicStreamer.Close()
		g.musicStreamer = nil
	}

	if g.musicFileH != nil {
		g.musicFileH.Close()
		g.musicFileH = nil
	}

	g.musicVolume = nil
	g.musicCtrl = nil

	speaker.Close()
	g.logAndSync("MUSIC: Остановлена")
}

// toggleMusic — включает/выключает музыку.
func (g *Game) toggleMusic() {
	if !g.musicEnabled || g.musicCtrl == nil {
		g.addMessage("Музыка недоступна.")
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

// changeMusicVolume — меняет громкость музыки.
func (g *Game) changeMusicVolume(delta float64) {
	if !g.musicEnabled || g.musicVolume == nil {
		g.addMessage("Музыка недоступна.")
		return
	}

	g.musicLevel += delta
	if g.musicLevel < 0 {
		g.musicLevel = 0
	}
	if g.musicLevel > 1.0 {
		g.musicLevel = 1.0
	}

	speaker.Lock()
	g.musicVolume.volume = g.musicLevel
	speaker.Unlock()

	percent := int(g.musicLevel * 100)
	g.addMessage(fmt.Sprintf("Громкость: %d%%", percent))
}