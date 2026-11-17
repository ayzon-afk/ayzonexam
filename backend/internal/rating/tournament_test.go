package rating

import (
	"testing"
	"time"
)

// Те же примеры — в frontend/src/lib/tournament.test.ts: расписание одно.
func TestTourStart(t *testing.T) {
	cases := map[string]string{
		"2026-11-16T12:00:00Z": "2026-11-29T05:00:00Z", // впереди — последнее воскресенье ноября
		"2026-11-29T06:00:00Z": "2026-11-29T05:00:00Z", // идёт — он же
		"2026-11-29T08:00:00Z": "2026-12-27T05:00:00Z", // кончился — следующий месяц
		"2026-12-28T00:00:00Z": "2027-01-31T05:00:00Z", // через границу года
	}
	for in, want := range cases {
		now, _ := time.Parse(time.RFC3339, in)
		if got := TourStart(now).Format(time.RFC3339); got != want {
			t.Errorf("TourStart(%s) = %s, ждали %s", in, got, want)
		}
	}
	start, _ := time.Parse(time.RFC3339, "2026-11-29T05:00:00Z")
	if got := PrevTourStart(start).Format(time.RFC3339); got != "2026-10-25T05:00:00Z" {
		t.Errorf("прошлый турнир: %s", got)
	}
}

func TestNewCode(t *testing.T) {
	a, b := NewCode(), NewCode()
	if len(a) != 10 || a == b {
		t.Fatalf("коды приглашений: %q %q", a, b)
	}
}
