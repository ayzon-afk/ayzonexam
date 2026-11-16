// Package writing — проверки Writing: правила, хранилище и очередь AI-проверок.
package writing

import (
	"math"
	"strings"
	"unicode"
)

const (
	Task1 = "task1" // Academic: отчёт по графику
	Task2 = "task2" // эссе
)

// Границы длины ответа. Минимум ниже экзаменационного (150/250), чтобы короткий текст получил разбор
// с потерей баллов, а не отказ; максимум защищает от вставки чужих текстов и расходов.
const (
	MinWords      = 50
	MaxWordsTask1 = 400
	MaxWordsTask2 = 600
)

// CountWords считает слова так же, как интерфейс (frontend/src/lib/writing.ts): куски между пробелами,
// в которых есть буква или цифра. «—» и «-» отдельно — не слова.
func CountWords(s string) int {
	n := 0
	for _, w := range strings.Fields(s) {
		if strings.IndexFunc(w, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
			n++
		}
	}
	return n
}

// Band — общий балл за задание из четырёх критериев: среднее, округлённое вверх до 0.5
// (6.25 → 6.5, 6.75 → 7.0), как округляется Overall Band IELTS. DECISIONS 2026-11-16.
func Band(ta, cc, lr, gra int) float64 {
	return math.Ceil(float64(ta+cc+lr+gra)/2) / 2
}
