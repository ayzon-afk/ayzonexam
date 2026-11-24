package ai

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrQuota — запросы к Gemini на эти сутки кончились (GEMINI_DAILY_LIMIT). Запрос не отправлялся;
// проверки ждут QuotaReset.
var ErrQuota = errors.New("gemini: дневной лимит запросов исчерпан")

// Quota считает запросы к Gemini: не больше Limit за сутки Gemini на модель — как дневной лимит
// бесплатного ключа (20 в день). Счёт в базе, а не в памяти: перезапуск сервера его не обнуляет.
type Quota struct {
	Pool  *pgxpool.Pool
	Limit int
	Now   func() time.Time
}

// Take занимает один запрос к model. false — на эти сутки запросы кончились.
func (q *Quota) Take(ctx context.Context, model string) (bool, error) {
	day := quotaDayStart(q.Now()).Format(time.DateOnly)
	tag, err := q.Pool.Exec(ctx, `INSERT INTO gemini_usage AS g (day, model, calls) VALUES ($1::date, $2, 1)
		ON CONFLICT (day, model) DO UPDATE SET calls = g.calls + 1 WHERE g.calls < $3`, day, model, q.Limit)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// QuotaReset — когда Gemini обнулит дневные лимиты: в полночь по тихоокеанскому времени
// (12:00 летом и 13:00 зимой по Ташкенту).
func QuotaReset(now time.Time) time.Time {
	y, m, d := quotaDayStart(now).Date()
	return pacificMidnight(y, m, d+1)
}

// Тихоокеанское время без tzdata: в образе сервера её нет, а правило США простое.
var (
	pst = time.FixedZone("PST", -8*3600)
	pdt = time.FixedZone("PDT", -7*3600)
)

// quotaDayStart — начало суток Gemini, в которые попадает t.
func quotaDayStart(t time.Time) time.Time {
	y, m, d := t.In(pst).Date()
	// Летом полночь следующей по PST даты наступает на час раньше — t может уже быть в ней.
	if next := pacificMidnight(y, m, d+1); !next.After(t) {
		return next
	}
	return pacificMidnight(y, m, d)
}

// pacificMidnight — начало дня по тихоокеанскому времени. Летнее время — со второго воскресенья марта до
// первого воскресенья ноября; переход в 2:00, поэтому полночь обоих дней перехода — ещё по прежнему.
func pacificMidnight(y int, m time.Month, d int) time.Time {
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	loc := pst
	if day.After(nthSunday(day.Year(), time.March, 2)) && !day.After(nthSunday(day.Year(), time.November, 1)) {
		loc = pdt
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
}

func nthSunday(y int, m time.Month, n int) time.Time {
	first := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 0, (7-int(first.Weekday()))%7+7*(n-1))
}
