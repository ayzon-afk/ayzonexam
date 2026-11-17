package rating

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// «Турнир месяца» (SPEC): последнее воскресенье месяца, 10:00–13:00 по Ташкенту, Reading и Listening
// с таймером. Турнир создаётся сам при первой записи — владельцу ничего не нужно заводить заранее.

// TourLength — сколько длится окно турнира.
const TourLength = 3 * time.Hour

var (
	ErrNotLive    = errors.New("турнир сейчас не идёт")
	ErrNotJoined  = errors.New("ученик не записан на турнир")
	ErrPartDone   = errors.New("часть турнира уже сдана")
	ErrTourClosed = errors.New("запись закрыта")
)

// TourStart — начало турнира, окно которого ещё не кончилось к now: последнее воскресенье месяца,
// 05:00 UTC (10:00 по Ташкенту). Те же правила — в frontend/src/lib/tournament.ts.
func TourStart(now time.Time) time.Time {
	now = now.UTC()
	for add := 0; ; add++ {
		last := time.Date(now.Year(), now.Month()+time.Month(add)+1, 0, 5, 0, 0, 0, time.UTC)
		last = last.AddDate(0, 0, -int(last.Weekday()))
		if now.Before(last.Add(TourLength)) {
			return last
		}
	}
}

// PrevTourStart — турнир месяцем раньше турнира start.
func PrevTourStart(start time.Time) time.Time {
	last := time.Date(start.Year(), start.Month(), 0, 5, 0, 0, 0, time.UTC)
	return last.AddDate(0, 0, -int(last.Weekday()))
}

type Tour struct {
	ID        int64
	StartsAt  time.Time
	EndsAt    time.Time
	Reading   *int64
	Listening *int64
}

// EnsureTour — турнир с началом start, создаётся при первом обращении.
func (s *Store) EnsureTour(ctx context.Context, start time.Time) (Tour, error) {
	t := Tour{StartsAt: start, EndsAt: start.Add(TourLength)}
	err := s.pool.QueryRow(ctx, `INSERT INTO tournaments (starts_at, ends_at) VALUES ($1, $2)
		ON CONFLICT (starts_at) DO UPDATE SET starts_at = EXCLUDED.starts_at
		RETURNING id, ends_at, reading_test_id, listening_test_id`, start, t.EndsAt).Scan(&t.ID, &t.EndsAt, &t.Reading, &t.Listening)
	return t, err
}

func (s *Store) Join(ctx context.Context, userID int64, now time.Time) error {
	t, err := s.EnsureTour(ctx, TourStart(now))
	if err != nil {
		return err
	}
	if !now.Before(t.EndsAt) {
		return ErrTourClosed
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO tournament_entries (tournament_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, t.ID, userID)
	return err
}

// PartTest — какое задание дать ученику в части kind идущего турнира. Задание закрепляется за турниром
// при первом старте: у всех участников одно и то же. pick выбирает задание, если оно ещё не закреплено.
func (s *Store) PartTest(ctx context.Context, userID int64, kind string, now time.Time, pick func() (int64, error)) (tourID, testID int64, err error) {
	start := TourStart(now)
	if now.Before(start) {
		return 0, 0, ErrNotLive
	}
	t, err := s.EnsureTour(ctx, start)
	if err != nil {
		return 0, 0, err
	}
	var joined, done bool
	if err := s.pool.QueryRow(ctx, `SELECT
			EXISTS (SELECT 1 FROM tournament_entries WHERE tournament_id = $1 AND user_id = $2 AND NOT disqualified),
			EXISTS (SELECT 1 FROM attempts WHERE tournament_id = $1 AND user_id = $2 AND kind = $3)`,
		t.ID, userID, kind).Scan(&joined, &done); err != nil {
		return 0, 0, err
	}
	if !joined {
		return 0, 0, ErrNotJoined
	}
	if done {
		return 0, 0, ErrPartDone
	}
	col := "reading_test_id"
	cur := t.Reading
	if kind == "listening" {
		col, cur = "listening_test_id", t.Listening
	}
	if cur != nil {
		return t.ID, *cur, nil
	}
	id, err := pick()
	if err != nil {
		return 0, 0, err
	}
	// Двое стартуют разом — закрепится одно задание, второй получит его же.
	err = s.pool.QueryRow(ctx, `UPDATE tournaments SET `+col+` = coalesce(`+col+`, $2) WHERE id = $1 RETURNING `+col, t.ID, id).Scan(&testID)
	return t.ID, testID, err
}

// TourState — турнир глазами ученика.
type TourState struct {
	ID       int64              `json:"id"`
	StartsAt time.Time          `json:"starts_at"`
	EndsAt   time.Time          `json:"ends_at"`
	Players  int                `json:"players"`
	Joined   bool               `json:"joined"`
	Parts    map[string]float64 `json:"parts"` // сданные части: kind → балл
	Band     *float64           `json:"band"`  // итог, когда сданы обе
	Place    *int               `json:"place"` // место в стране после окончания
	Finished bool               `json:"finished"`
	Live     bool               `json:"live"`
	Results  []Row              `json:"results"` // после окончания — первые 10
}

// State — турнир, окно которого идёт или впереди (current), и прошедший, если есть.
func (s *Store) State(ctx context.Context, userID int64, start time.Time, now time.Time) (TourState, error) {
	st := TourState{StartsAt: start, EndsAt: start.Add(TourLength), Parts: map[string]float64{}}
	err := s.pool.QueryRow(ctx, `SELECT t.id, t.ends_at,
			(SELECT count(*) FROM tournament_entries e WHERE e.tournament_id = t.id),
			EXISTS (SELECT 1 FROM tournament_entries e WHERE e.tournament_id = t.id AND e.user_id = $2)
		FROM tournaments t WHERE t.starts_at = $1`, start, userID).Scan(&st.ID, &st.EndsAt, &st.Players, &st.Joined)
	if errors.Is(err, pgx.ErrNoRows) {
		st.Live = !now.Before(st.StartsAt) && now.Before(st.EndsAt)
		return st, nil
	}
	if err != nil {
		return st, err
	}
	st.Live = !now.Before(st.StartsAt) && now.Before(st.EndsAt)
	st.Finished = !now.Before(st.EndsAt)
	rows, err := s.pool.Query(ctx, `SELECT kind, band::float8 FROM attempts
		WHERE tournament_id = $1 AND user_id = $2 AND finished_at IS NOT NULL`, st.ID, userID)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var k string
		var b float64
		if err := rows.Scan(&k, &b); err != nil {
			rows.Close()
			return st, err
		}
		st.Parts[k] = b
	}
	rows.Close()
	if st.Finished {
		res, err := s.Results(ctx, st.ID, userID, 10)
		if err != nil {
			return st, err
		}
		st.Results = res.Rows
		if res.Mine != nil {
			st.Place, st.Band = &res.Mine.Place, &res.Mine.Band
		}
	}
	return st, rows.Err()
}

// Results — итог турнира по стране: у кого сданы обе части в рейтинг (вовремя, без частых уходов),
// балл — Overall из двух; при равенстве выше тот, кто сдал раньше.
func (s *Store) Results(ctx context.Context, tourID, me int64, top int) (Board, error) {
	rows, err := s.pool.Query(ctx, `WITH parts AS (
			SELECT a.user_id, avg(a.band) AS avg, count(*) AS n, max(a.finished_at) AS done
			FROM attempts a JOIN tournament_entries e ON e.tournament_id = a.tournament_id AND e.user_id = a.user_id
			WHERE a.tournament_id = $1 AND a.ranked AND NOT e.disqualified GROUP BY a.user_id),
		r AS (SELECT u.id, u.nickname, u.photo_url, coalesce(c.name, '') AS center, floor(p.avg * 2 + 0.5) / 2 AS band,
				rank() OVER (ORDER BY floor(p.avg * 2 + 0.5) / 2 DESC, p.done) AS place, count(*) OVER () AS total
			FROM parts p JOIN users u ON u.id = p.user_id
			LEFT JOIN centers c ON c.id = u.center_id AND u.center_status = 'member'
			WHERE p.n = 2 AND u.blocked_at IS NULL)
		SELECT id, nickname, photo_url, center, band::float8, place, total FROM r
		WHERE place <= $3 OR id = $2 ORDER BY place`, tourID, me, top)
	if err != nil {
		return Board{}, err
	}
	defer rows.Close()
	b := Board{Rows: []Row{}}
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.UserID, &r.Nickname, &r.Avatar, &r.Center, &r.Band, &r.Place, &b.Total); err != nil {
			return b, err
		}
		r.Me = r.UserID == me
		if r.Me {
			mine := r
			b.Mine = &mine
		}
		if r.Place <= top {
			b.Rows = append(b.Rows, r)
		}
	}
	return b, rows.Err()
}

// Winner — чемпион прошедшего турнира: стоит на главной до следующего.
type Winner struct {
	Month    time.Time `json:"month"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Center   string    `json:"center"`
	CenterID *int64    `json:"center_id"`
	Band     float64   `json:"band"`
}

// Winners — чемпионы прошедших турниров, свежий первым.
func (s *Store) Winners(ctx context.Context, limit int) ([]Winner, error) {
	rows, err := s.pool.Query(ctx, `SELECT t.starts_at, u.nickname, u.photo_url, coalesce(c.name, ''), c.id, t.champion_band::float8
		FROM tournaments t JOIN users u ON u.id = t.champion_id
		LEFT JOIN centers c ON c.id = u.center_id AND u.center_status = 'member'
		WHERE t.champion_band IS NOT NULL ORDER BY t.starts_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Winner])
}

// PastTours — турниры, которые начались (для админки: итоги и подтверждение тройки).
func (s *Store) PastTours(ctx context.Context, now time.Time, limit int) ([]Tour, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, starts_at, ends_at, reading_test_id, listening_test_id FROM tournaments
		WHERE starts_at <= $1 ORDER BY starts_at DESC LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Tour])
}

// SetChampion — владелец подтвердил тройку и назначил чемпиона (балл — из итогов турнира).
// Ученик должен быть в итогах; userID 0 — снять чемпиона.
func (s *Store) SetChampion(ctx context.Context, tourID, userID int64) error {
	if userID == 0 {
		_, err := s.pool.Exec(ctx, `UPDATE tournaments SET champion_id = NULL, champion_band = NULL WHERE id = $1`, tourID)
		return err
	}
	res, err := s.Results(ctx, tourID, userID, 0)
	if err != nil {
		return err
	}
	if res.Mine == nil {
		return ErrNotFound
	}
	_, err = s.pool.Exec(ctx, `UPDATE tournaments SET champion_id = $2, champion_band = $3 WHERE id = $1`, tourID, userID, res.Mine.Band)
	return err
}

// Disqualify — тройка не подтвердила результат Speaking: место переходит следующему.
func (s *Store) Disqualify(ctx context.Context, tourID, userID int64, off bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE tournament_entries SET disqualified = $3 WHERE tournament_id = $1 AND user_id = $2`, tourID, userID, off)
	return err
}
