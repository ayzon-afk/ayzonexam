// Package exam — задания Reading, Listening и теста уровня, попытки с проверкой по ключу и полный mock.
package exam

import "math"

// Виды заданий.
const (
	Reading   = "reading"
	Listening = "listening"
	Level     = "level"
)

// Таблицы «верных из 40 → балл» — опубликованные примерные таблицы IELTS для Listening и Academic
// Reading. Сет другой длины пересчитывается на 40, поэтому балл короткого сета — примерный.
type step struct {
	min  int
	band float64
}

var listeningTable = []step{
	{39, 9}, {37, 8.5}, {35, 8}, {32, 7.5}, {30, 7}, {26, 6.5}, {23, 6}, {18, 5.5},
	{16, 5}, {13, 4.5}, {10, 4}, {8, 3.5}, {6, 3}, {4, 2.5}, {2, 2}, {1, 1},
}

var readingTable = []step{
	{39, 9}, {37, 8.5}, {35, 8}, {33, 7.5}, {30, 7}, {27, 6.5}, {23, 6}, {19, 5.5},
	{15, 5}, {13, 4.5}, {10, 4}, {8, 3.5}, {6, 3}, {4, 2.5}, {2, 2}, {1, 1},
}

// Band — балл за часть по числу верных ответов. Тест уровня считается по таблице Reading.
func Band(kind string, correct, total int) float64 {
	if total <= 0 || correct <= 0 {
		return 0
	}
	raw := int(math.Round(float64(min(correct, total)) * 40 / float64(total)))
	table := readingTable
	if kind == Listening {
		table = listeningTable
	}
	for _, s := range table {
		if raw >= s.min {
			return s.band
		}
	}
	return 0
}

// Overall — среднее баллов, округлённое как Overall Band IELTS: .25 — вверх до .5, .75 — вверх до целого,
// меньше — вниз (6.125 → 6.0, 6.25 → 6.5, 6.75 → 7.0). Пусто — 0.
func Overall(bands ...float64) float64 {
	if len(bands) == 0 {
		return 0
	}
	sum := 0.0
	for _, b := range bands {
		sum += b
	}
	return math.Floor(sum/float64(len(bands))*2+0.5) / 2
}
