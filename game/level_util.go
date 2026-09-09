package game

import (
	"math/rand"
)

// =============================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// =============================================================================
// intInRange — возвращает случайное число в диапазоне [minVal, maxVal].
// Если maxVal < minVal, возвращает minVal (защита от отрицательного диапазона).
func intInRange(minVal, maxVal int) int {
	if maxVal < minVal {
		return minVal
	}
	return minVal + rand.Intn(maxVal-minVal+1)
}

// minInt — возвращает меньшее из двух чисел.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt — возвращает большее из двух чисел.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}