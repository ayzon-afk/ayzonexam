// Package billing — тарифы, лимиты AI-проверок и оплата Premium переводом на карту (этап 6).
// Цены и правила — SPEC, «Деньги»; почему так — DECISIONS 2026-11-16.
package billing

import (
	"time"

	"ayzonexam/internal/users"
)

// Ташкент без перехода на летнее время.
var Tashkent = time.FixedZone("UZT", 5*3600)

// Цены в сумах.
const (
	PriceMonth         = 100_000
	PriceMonthDiscount = 70_000 // первый месяц, если перейти в первые 7 дней после регистрации
	PriceQuarter       = 250_000
	PriceMock          = 50_000
)

const (
	DiscountDays    = 7
	ReferralDays    = 7 // друг купил Premium — пригласившему +7 дней
	MonthDays       = 30
	QuarterDays     = 90
	MocksPerMonth   = 2 // полных mock в месяц с Premium
	FreeWritingWeek = 2
	FreeSpeakWeek   = 1
	PremiumDaily    = 3 // Writing и Speaking в день с Premium
)

// Источник Premium.
const (
	SourceNone   = ""
	SourceTrial  = "trial"  // первые 3 дня
	SourcePaid   = "paid"   // оплата или бонус за друга
	SourceCenter = "center" // центр ученика оплатил тариф
)

// Access — есть ли Premium сейчас и откуда.
type Access struct {
	Premium bool
	Until   *time.Time
	Source  string
}

// Of — доступ ученика на момент now. Premium центра — до конца оплаченного дня по Ташкенту.
func Of(u users.User, now time.Time) Access {
	var a Access
	if u.CenterPaid != nil {
		d := u.CenterPaid
		end := time.Date(d.Year(), d.Month(), d.Day()+1, 0, 0, 0, 0, Tashkent)
		if now.Before(end) {
			a = Access{Premium: true, Until: &end, Source: SourceCenter}
		}
	}
	if u.PremiumUntil != nil && now.Before(*u.PremiumUntil) && (a.Until == nil || u.PremiumUntil.After(*a.Until)) {
		src := SourcePaid
		if u.PremiumUntil.Sub(u.CreatedAt) <= 3*24*time.Hour+time.Minute {
			src = SourceTrial
		}
		until := *u.PremiumUntil
		a = Access{Premium: true, Until: &until, Source: src}
	}
	return a
}

// DiscountUntil — до какого момента действует скидка первого месяца; nil — уже нет.
// paidBefore — у ученика уже была подтверждённая оплата (скидка только на первый месяц).
func DiscountUntil(u users.User, paidBefore bool, now time.Time) *time.Time {
	end := u.CreatedAt.Add(DiscountDays * 24 * time.Hour)
	if paidBefore || !now.Before(end) {
		return nil
	}
	return &end
}

// Quota — лимит проверок: сколько можно за период и с какого момента он считается.
type Quota struct {
	Limit  int       `json:"limit"`
	Period string    `json:"period"` // day, week
	Since  time.Time `json:"-"`
}

// DayStart — полночь сегодняшнего дня по Ташкенту.
func DayStart(now time.Time) time.Time {
	n := now.In(Tashkent)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, Tashkent)
}

// MonthStart — первое число месяца по Ташкенту.
func MonthStart(now time.Time) time.Time {
	n := now.In(Tashkent)
	return time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, Tashkent)
}

// WritingQuota — Premium: premiumDaily в день (настройка сервера), бесплатно: 2 за 7 дней.
func WritingQuota(a Access, premiumDaily int, now time.Time) Quota {
	if a.Premium {
		return Quota{Limit: premiumDaily, Period: "day", Since: DayStart(now)}
	}
	return Quota{Limit: FreeWritingWeek, Period: "week", Since: now.Add(-7 * 24 * time.Hour)}
}

// SpeakingQuota — Premium: 3 в день, бесплатно: 1 за 7 дней.
func SpeakingQuota(a Access, now time.Time) Quota {
	if a.Premium {
		return Quota{Limit: PremiumDaily, Period: "day", Since: DayStart(now)}
	}
	return Quota{Limit: FreeSpeakWeek, Period: "week", Since: now.Add(-7 * 24 * time.Hour)}
}

// MockQuota — с Premium 2 полных mock в календарный месяц; без него — только купленные.
func MockQuota(a Access, now time.Time) Quota {
	if a.Premium {
		return Quota{Limit: MocksPerMonth, Period: "month", Since: MonthStart(now)}
	}
	return Quota{Limit: 0, Period: "month", Since: MonthStart(now)}
}

// Price — сумма и сколько дней Premium или mock даёт тариф.
func Price(plan string, discount bool) (amount, days, mocks int, ok bool) {
	switch plan {
	case "month":
		if discount {
			return PriceMonthDiscount, MonthDays, 0, true
		}
		return PriceMonth, MonthDays, 0, true
	case "quarter":
		return PriceQuarter, QuarterDays, 0, true
	case "mock":
		return PriceMock, 0, 1, true
	}
	return 0, 0, 0, false
}
