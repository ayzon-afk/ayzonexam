package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"ayzonexam/internal/users"
)

// Учёт реального использования для жюри хакатона (DECISIONS 2026-11-24): сессии, честные активные минуты,
// сданные тесты, повторные заходы. Владелец и помощники (ADMIN_IDS) не считаются — это проверки, а не ученики.

const (
	beatMax = 30 // секунд засчитывает один пульс: приложение шлёт его раз в 30 с, пока открыто и активно
	beatGap = 45 // пауза длиннее — телефон спал или пульсы терялись: пульс ничего не засчитывает
)

var sidRe = regexp.MustCompile(`^[A-Za-z0-9_-]{8,40}$`)

// beat — пульс открытого и активного приложения. start — первый пульс после открытия, возврата из фона или
// простоя: время до него не засчитывается.
func (s *server) beat(w http.ResponseWriter, r *http.Request, u users.User) {
	var b struct {
		SID   string `json:"sid"`
		Start bool   `json:"start"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	if !sidRe.MatchString(b.SID) {
		writeError(w, http.StatusBadRequest, "bad_sid")
		return
	}
	_, err := s.DB.Exec(r.Context(), `INSERT INTO app_sessions (user_id, sid, started_at, last_beat) VALUES ($1, $2, $3, $3)
		ON CONFLICT (user_id, sid) DO UPDATE SET
			active_seconds = app_sessions.active_seconds + CASE
				WHEN $4 OR $3 - app_sessions.last_beat > make_interval(secs => $5) OR $3 < app_sessions.last_beat THEN 0
				ELSE least(floor(extract(epoch FROM $3 - app_sessions.last_beat)), $6)::int END,
			last_beat = greatest(app_sessions.last_beat, $3),
			beats = app_sessions.beats + 1`,
		u.ID, b.SID, s.Now(), b.Start, float64(beatGap), beatMax)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// usageEvents — всё, что ученики сделали, по дням Ташкента: открытия с активными секундами, сданные тесты,
// готовые проверки Writing и Speaking (и голосовые боту — без открытия приложения). $1 — ADMIN_IDS.
const usageEvents = `WITH real AS (SELECT id, nickname, created_at FROM users WHERE telegram_id <> ALL($1::bigint[])),
ev AS (
	SELECT user_id, (started_at AT TIME ZONE 'Asia/Tashkent')::date AS day,
		1 AS sessions, active_seconds AS secs, 0 AS tests, 0 AS writing, 0 AS speaking FROM app_sessions
	UNION ALL SELECT user_id, (finished_at AT TIME ZONE 'Asia/Tashkent')::date, 0, 0, 1, 0, 0
		FROM attempts WHERE finished_at IS NOT NULL
	UNION ALL SELECT user_id, (created_at AT TIME ZONE 'Asia/Tashkent')::date, 0, 0, 0, 1, 0
		FROM writing_checks WHERE status = 'done'
	UNION ALL SELECT user_id, (created_at AT TIME ZONE 'Asia/Tashkent')::date, 0, 0, 0, 0, 1
		FROM speaking_checks WHERE status = 'done'
),
rev AS (SELECT ev.* FROM ev JOIN real ON real.id = ev.user_id)
`

func (s *server) adminIDs() []int64 {
	if s.Admins == nil {
		return []int64{} // NULL в <> ALL отсеял бы всех
	}
	return s.Admins
}

type usageDay struct {
	Day      string  `json:"day"`
	Users    int     `json:"users"`
	New      int     `json:"new"`
	Sessions int     `json:"sessions"`
	Minutes  float64 `json:"minutes"`
	Tests    int     `json:"tests"`
	Writing  int     `json:"writing"`
	Speaking int     `json:"speaking"`
}

func (s *server) adminUsage(w http.ResponseWriter, r *http.Request, _ users.User) {
	ctx := r.Context()
	var o struct {
		Users     int        `json:"users"`     // зарегистрировались
		Active    int        `json:"active"`    // сделали хоть что-то
		Sessions  int        `json:"sessions"`  // открытий приложения
		Minutes   float64    `json:"minutes"`   // честных активных минут
		Tests     int        `json:"tests"`     // сданных Reading, Listening, тестов уровня
		Writing   int        `json:"writing"`   // готовых проверок Writing
		Speaking  int        `json:"speaking"`  // готовых проверок Speaking
		Mocks     int        `json:"mocks"`     // завершённых mock
		Returning int        `json:"returning"` // приходили в 2+ разных дня
		Repeat    int        `json:"repeat"`    // открывали приложение 2+ раза
		Days      []usageDay `json:"days"`
	}
	err := s.DB.QueryRow(ctx, usageEvents+`SELECT
		(SELECT count(*) FROM real),
		(SELECT count(DISTINCT user_id) FROM rev),
		(SELECT coalesce(sum(sessions), 0) FROM rev),
		(SELECT round(coalesce(sum(secs), 0) / 60.0, 1)::float8 FROM rev),
		(SELECT coalesce(sum(tests), 0) FROM rev),
		(SELECT coalesce(sum(writing), 0) FROM rev),
		(SELECT coalesce(sum(speaking), 0) FROM rev),
		(SELECT count(*) FROM mocks m JOIN real ON real.id = m.user_id WHERE m.status = 'done'),
		(SELECT count(*) FROM (SELECT user_id FROM rev GROUP BY user_id HAVING count(DISTINCT day) >= 2) x),
		(SELECT count(*) FROM (SELECT user_id FROM rev GROUP BY user_id HAVING sum(sessions) >= 2) x)`, s.adminIDs()).
		Scan(&o.Users, &o.Active, &o.Sessions, &o.Minutes, &o.Tests, &o.Writing, &o.Speaking, &o.Mocks, &o.Returning, &o.Repeat)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rows, err := s.DB.Query(ctx, usageEvents+`SELECT d.day::text,
		count(DISTINCT rev.user_id)::int,
		(SELECT count(*) FROM real WHERE (real.created_at AT TIME ZONE 'Asia/Tashkent')::date = d.day)::int,
		coalesce(sum(rev.sessions), 0)::int, round(coalesce(sum(rev.secs), 0) / 60.0, 1)::float8,
		coalesce(sum(rev.tests), 0)::int, coalesce(sum(rev.writing), 0)::int, coalesce(sum(rev.speaking), 0)::int
		FROM (SELECT DISTINCT day FROM rev UNION SELECT (created_at AT TIME ZONE 'Asia/Tashkent')::date FROM real) d
		LEFT JOIN rev ON rev.day = d.day
		GROUP BY d.day ORDER BY d.day DESC LIMIT 60`, s.adminIDs())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer rows.Close()
	o.Days = []usageDay{}
	for rows.Next() {
		var d usageDay
		if err := rows.Scan(&d.Day, &d.Users, &d.New, &d.Sessions, &d.Minutes, &d.Tests, &d.Writing, &d.Speaking); err != nil {
			s.fail(w, r, err)
			return
		}
		o.Days = append(o.Days, d)
	}
	if err := rows.Err(); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// usageCSV — строка на ученика и день. Без имён и Telegram id: файл показывают жюри; ник и так публичный.
func (s *server) usageCSV(ctx context.Context) ([]byte, error) {
	rows, err := s.DB.Query(ctx, usageEvents+`SELECT rev.day::text, rev.user_id, real.nickname,
		sum(rev.sessions)::int, round(sum(rev.secs) / 60.0, 1)::float8,
		sum(rev.tests)::int, sum(rev.writing)::int, sum(rev.speaking)::int
		FROM rev JOIN real ON real.id = rev.user_id
		GROUP BY rev.day, rev.user_id, real.nickname ORDER BY rev.day, rev.user_id`, s.adminIDs())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM: Excel открывает UTF-8 только с ним
	cw := csv.NewWriter(&buf)
	cw.Write([]string{"day", "user_id", "nickname", "sessions", "active_minutes", "tests", "writing", "speaking"})
	for rows.Next() {
		var day, nick string
		var id int64
		var sessions, tests, writing, speaking int
		var minutes float64
		if err := rows.Scan(&day, &id, &nick, &sessions, &minutes, &tests, &writing, &speaking); err != nil {
			return nil, err
		}
		cw.Write([]string{day, strconv.FormatInt(id, 10), nick, strconv.Itoa(sessions),
			strconv.FormatFloat(minutes, 'f', 1, 64), strconv.Itoa(tests), strconv.Itoa(writing), strconv.Itoa(speaking)})
	}
	cw.Flush()
	return buf.Bytes(), rows.Err()
}

func (s *server) usageFile() string {
	return fmt.Sprintf("ayzon-usage-%s.csv", s.Now().In(tashkent).Format(time.DateOnly))
}

// adminUsageCSV — файл в браузер (вне Telegram скачивание работает).
func (s *server) adminUsageCSV(w http.ResponseWriter, r *http.Request, u users.User) {
	data, err := s.usageCSV(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.audit(r.Context(), u.TelegramID, "usage.csv", "", nil)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+s.usageFile()+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Write(data)
}

// adminUsageSend — файл в чат с ботом: внутри Telegram на телефоне скачать из приложения нельзя.
func (s *server) adminUsageSend(w http.ResponseWriter, r *http.Request, u users.User) {
	if s.Msg == nil {
		writeError(w, http.StatusServiceUnavailable, "no_bot")
		return
	}
	data, err := s.usageCSV(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Msg.SendDocument(r.Context(), u.TelegramID, s.usageFile(), data, "AYZON EXAM — использование по дням"); err != nil {
		s.fail(w, r, err)
		return
	}
	s.audit(r.Context(), u.TelegramID, "usage.csv", "telegram", nil)
	w.WriteHeader(http.StatusNoContent)
}
