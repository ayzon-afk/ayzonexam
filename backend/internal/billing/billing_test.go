package billing

import (
	"testing"
	"time"

	"ayzonexam/internal/users"
)

var now = time.Date(2026, 11, 15, 7, 0, 0, 0, time.UTC) // полдень по Ташкенту

func ptr[T any](v T) *T { return &v }

func TestAccess(t *testing.T) {
	created := now.Add(-24 * time.Hour)
	trial := users.User{CreatedAt: created, PremiumUntil: ptr(created.Add(72 * time.Hour))}
	if a := Of(trial, now); !a.Premium || a.Source != SourceTrial {
		t.Fatalf("пробные 3 дня: %+v", a)
	}
	if a := Of(trial, created.Add(73*time.Hour)); a.Premium {
		t.Fatalf("пробные кончились: %+v", a)
	}
	paid := users.User{CreatedAt: created, PremiumUntil: ptr(now.Add(30 * 24 * time.Hour))}
	if a := Of(paid, now); a.Source != SourcePaid {
		t.Fatalf("оплаченный: %+v", a)
	}
	day := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	center := users.User{CreatedAt: created, CenterPaid: &day}
	a := Of(center, now)
	if !a.Premium || a.Source != SourceCenter || a.Until.In(Tashkent).Format(time.DateTime) != "2026-11-16 00:00:00" {
		t.Fatalf("центр — до конца оплаченного дня по Ташкенту: %+v %v", a, a.Until)
	}
	if a := Of(users.User{CreatedAt: created}, now); a.Premium {
		t.Fatal("без всего — бесплатно")
	}
}

func TestQuotasAndPrices(t *testing.T) {
	free, prem := Access{}, Access{Premium: true}
	if q := WritingQuota(free, 3, now); q.Limit != 2 || q.Period != "week" {
		t.Fatalf("Writing бесплатно: %+v", q)
	}
	if q := WritingQuota(prem, 3, now); q.Limit != 3 || q.Period != "day" || q.Since.In(Tashkent).Hour() != 0 {
		t.Fatalf("Writing Premium: %+v", q)
	}
	if q := SpeakingQuota(free, now); q.Limit != 1 {
		t.Fatalf("Speaking бесплатно: %+v", q)
	}
	if q := MockQuota(prem, now); q.Limit != 2 || q.Since.In(Tashkent).Day() != 1 {
		t.Fatalf("mock Premium: %+v", q)
	}
	if a, d, _, _ := Price("month", true); a != 70_000 || d != 30 {
		t.Fatalf("скидка первого месяца: %d %d", a, d)
	}
	if a, _, m, _ := Price("mock", true); a != 50_000 || m != 1 {
		t.Fatalf("mock без скидки: %d %d", a, m)
	}
	if _, _, _, ok := Price("year", false); ok {
		t.Fatal("неизвестный тариф")
	}
	u := users.User{CreatedAt: now.Add(-6 * 24 * time.Hour)}
	if DiscountUntil(u, false, now) == nil || DiscountUntil(u, true, now) != nil {
		t.Fatal("скидка — первые 7 дней и только на первый месяц")
	}
}

func TestCenterPrice(t *testing.T) {
	for _, c := range []struct{ seats, months, want int }{{20, 1, 400_000}, {99, 1, 1_980_000}, {100, 1, 1_500_000}, {25, 12, 6_000_000}} {
		if got, ok := CenterPrice("center", c.seats, c.months); !ok || got != c.want {
			t.Fatalf("%d мест × %d мес.: %d, %v", c.seats, c.months, got, ok)
		}
	}
	for _, c := range [][2]int{{19, 1}, {20, 2}, {20, 0}, {CenterMaxSeats + 1, 1}} {
		if _, ok := CenterPrice("center", c[0], c[1]); ok {
			t.Fatalf("так нельзя: %v", c)
		}
	}
	// Учитель без центра — от 10 мест (SPEC).
	if got, ok := CenterPrice("teacher", 10, 1); !ok || got != 200_000 {
		t.Fatalf("учитель, 10 мест: %d %v", got, ok)
	}
	if _, ok := CenterPrice("teacher", 9, 1); ok {
		t.Fatal("учитель, 9 мест — нельзя")
	}
}

func TestCenterPaidUntil(t *testing.T) {
	now := time.Date(2026, 11, 25, 20, 0, 0, 0, time.UTC) // 26.11 по Ташкенту
	day := func(s string) *time.Time { d, _ := time.Parse(time.DateOnly, s); return &d }
	for _, c := range []struct {
		paid   *time.Time
		months int
		want   string
	}{
		{nil, 1, "2026-12-26"},                // не платил — от сегодня
		{day("2026-11-01"), 3, "2027-02-26"},  // срок кончился — от сегодня
		{day("2026-12-10"), 1, "2027-01-10"},  // ещё оплачено — от конца срока
		{day("2026-11-26"), 12, "2027-11-26"}, // кончается сегодня — от сегодня
	} {
		if got := CenterPaidUntil(c.paid, c.months, now).Format(time.DateOnly); got != c.want {
			t.Fatalf("%v + %d мес.: %s, ждали %s", c.paid, c.months, got, c.want)
		}
	}
}
