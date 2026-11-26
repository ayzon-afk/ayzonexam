package rating

import "slices"

// Regions — регионы Узбекистана: рейтинг своего региона и город центра (DECISIONS 2026-11-25).
// Порядок — как в списке интерфейса (frontend/src/lib/cities.ts); меняется одно — меняется и другое,
// и CHECK в базе — новой миграцией (0016).
var Regions = []string{"tashkent", "tashkent_region", "andijan", "bukhara", "fergana", "jizzakh", "kashkadarya",
	"khorezm", "namangan", "navoi", "samarkand", "surkhandarya", "syrdarya", "karakalpakstan"}

// ForeignCity — ученик живёт не в Узбекистане: рейтинга региона у него нет.
const ForeignCity = "other"

// ValidCity — город ученика: регион или другая страна.
func ValidCity(c string) bool { return c == ForeignCity || slices.Contains(Regions, c) }
