// Package rating — текущий балл ученика, рейтинги (Ташкент, центр, друзья, страна, центры), центры и
// «Турнир месяца». Только реальные баллы таймерных попыток, не активность (SPEC).
package rating

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Window — за сколько дней берутся попытки: старый балл не держит место в рейтинге вечно.
const Window = 60 * 24 * time.Hour

// Week — с какой даты считаются стрелки роста.
const Week = 7 * 24 * time.Hour

// Top — сколько строк рейтинга отдаётся; своё место приходит отдельно, даже если оно ниже.
const Top = 50

// Области рейтинга учеников.
const (
	City    = "city" // свой регион (Regions)
	Center  = "center"
	Friends = "friends"
	Country = "country"
)

var ErrNotFound = errors.New("не найдено")

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// scoresSQL — CTE score(user_id, band) на момент $1: последняя таймерная попытка по каждой части за
// Window, среднее по частям, округление как Overall Band (exam.Overall). Подставляется в запросы.
const scoresSQL = `last_%[1]s AS (
		SELECT DISTINCT ON (user_id, kind) user_id, kind, band FROM rated_parts
		WHERE at <= %[2]s AND at > %[2]s - interval '60 days'
		ORDER BY user_id, kind, at DESC),
	%[1]s AS (SELECT user_id, floor(avg(band) * 2 + 0.5) / 2 AS band, count(*) AS parts FROM last_%[1]s GROUP BY user_id)`

// Current — текущий балл ученика (nil — нет таймерных попыток за Window) и сколько частей в нём.
func (s *Store) Current(ctx context.Context, userID int64, now time.Time) (*float64, int, error) {
	var band *float64
	var parts int
	err := s.pool.QueryRow(ctx, `WITH `+sprintf(scoresSQL, "cur", "$2")+`
		SELECT band, parts FROM cur WHERE user_id = $1`, userID, now).Scan(&band, &parts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, nil
	}
	return band, parts, err
}

// Row — строка рейтинга: только публичное (ник, аватар, центр, балл). Delta — на сколько мест
// поднялся за неделю (0 — не изменилось или неделю назад не было в рейтинге).
type Row struct {
	UserID   int64   `json:"-"`
	Place    int     `json:"place"`
	Nickname string  `json:"nickname"`
	Avatar   string  `json:"avatar"`
	Center   string  `json:"center"`
	Band     float64 `json:"band"`
	Delta    int     `json:"delta"`
	Me       bool    `json:"me"`
}

type Board struct {
	Rows  []Row `json:"rows"`
	Mine  *Row  `json:"mine"`
	Total int   `json:"total"`
}

// Students — рейтинг учеников в области scope для ученика me (его центр и друзья — для своих областей).
func (s *Store) Students(ctx context.Context, scope string, me int64, centerID *int64, now time.Time) (Board, error) {
	// $1 — сейчас, $2 — ученик, $3 — сколько строк; $4 — центр, только для своей области:
	// PostgreSQL не принимает параметр, который не встречается в запросе.
	args := []any{now, me, Top}
	var filter string
	switch scope {
	case Country:
		filter = `true`
	case City:
		// Регион ученика: свой, а не указан — регион его центра. Нет региона (не указан, другая
		// страна) — рейтинг пустой: сравнение с NULL никого не берёт.
		filter = `coalesce(nullif(u.city, ''), c.city) = (SELECT nullif(nullif(coalesce(nullif(m.city, ''), mc.city), ''), 'other')
			FROM users m LEFT JOIN centers mc ON mc.id = m.center_id AND m.center_status = 'member' WHERE m.id = $2)`
	case Center:
		if centerID == nil {
			return Board{Rows: []Row{}}, nil
		}
		filter = `u.center_id = $4 AND u.center_status = 'member'`
		args = append(args, *centerID)
	case Friends:
		filter = `(u.id = $2 OR u.id IN (SELECT friend_id FROM friends WHERE user_id = $2))`
	default:
		return Board{}, errors.New("неизвестная область рейтинга")
	}
	rows, err := s.pool.Query(ctx, `WITH `+sprintf(scoresSQL, "cur", "$1")+`, `+sprintf(scoresSQL, "old", "($1::timestamptz - interval '7 days')")+`,
		pool AS (
			SELECT u.id, u.nickname, u.photo_url, coalesce(c.name, '') AS center
			FROM users u LEFT JOIN centers c ON c.id = u.center_id AND u.center_status = 'member'
			WHERE u.blocked_at IS NULL AND `+filter+`),
		now_r AS (SELECT p.*, cur.band, rank() OVER (ORDER BY cur.band DESC) AS place FROM pool p JOIN cur ON cur.user_id = p.id),
		old_r AS (SELECT p.id, rank() OVER (ORDER BY old.band DESC) AS place FROM pool p JOIN old ON old.user_id = p.id),
		full_r AS (SELECT n.id, n.nickname, n.photo_url, n.center, n.band, n.place, o.place AS old_place, count(*) OVER () AS total
			FROM now_r n LEFT JOIN old_r o ON o.id = n.id)
		SELECT id, nickname, photo_url, center, band::float8, place, old_place, total FROM full_r
		WHERE place <= $3 OR id = $2 ORDER BY place, id`, args...)
	if err != nil {
		return Board{}, err
	}
	defer rows.Close()
	b := Board{Rows: []Row{}}
	for rows.Next() {
		var r Row
		var oldPlace *int
		var total int
		if err := rows.Scan(&r.UserID, &r.Nickname, &r.Avatar, &r.Center, &r.Band, &r.Place, &oldPlace, &total); err != nil {
			return b, err
		}
		b.Total = total
		if oldPlace != nil {
			r.Delta = *oldPlace - r.Place
		}
		r.Me = r.UserID == me
		if r.Me {
			mine := r
			b.Mine = &mine
		}
		if r.Place <= Top {
			b.Rows = append(b.Rows, r)
		}
	}
	return b, rows.Err()
}

// CenterRow — центр в рейтинге центров: средний текущий балл его учеников.
type CenterRow struct {
	ID       int64   `json:"id"`
	Place    int     `json:"place"`
	Name     string  `json:"name"`
	City     string  `json:"city"`
	Avg      float64 `json:"avg"`
	Scored   int     `json:"scored"`   // учеников с баллом
	Students int     `json:"students"` // всех учеников центра
	Delta    int     `json:"delta"`
}

const centerAvgSQL = `%[1]s_c AS (
		SELECT u.center_id AS id, round(avg(%[1]s.band), 2)::float8 AS avg, count(*) AS scored
		FROM users u JOIN %[1]s ON %[1]s.user_id = u.id
		WHERE u.center_status = 'member' AND u.blocked_at IS NULL GROUP BY u.center_id)`

// Centers — рейтинг центров. Центр без учеников с баллом в рейтинг не входит; учитель без центра — тоже:
// здесь соревнуются центры.
func (s *Store) Centers(ctx context.Context, now time.Time) ([]CenterRow, error) {
	rows, err := s.pool.Query(ctx, `WITH `+sprintf(scoresSQL, "cur", "$1")+`, `+sprintf(scoresSQL, "old", "($1::timestamptz - interval '7 days')")+`,
		`+sprintf(centerAvgSQL, "cur")+`, `+sprintf(centerAvgSQL, "old")+`,
		now_r AS (SELECT c.id, c.name, c.city, a.avg, a.scored, rank() OVER (ORDER BY a.avg DESC) AS place
			FROM centers c JOIN cur_c a ON a.id = c.id WHERE c.active AND c.kind = 'center'),
		old_r AS (SELECT c.id, rank() OVER (ORDER BY a.avg DESC) AS place FROM centers c JOIN old_c a ON a.id = c.id
			WHERE c.active AND c.kind = 'center')
		SELECT n.id, n.place, n.name, n.city, n.avg, n.scored,
			(SELECT count(*) FROM users m WHERE m.center_id = n.id AND m.center_status = 'member'), o.place
		FROM now_r n LEFT JOIN old_r o ON o.id = n.id ORDER BY n.place, n.id`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []CenterRow{}
	for rows.Next() {
		var c CenterRow
		var old *int
		if err := rows.Scan(&c.ID, &c.Place, &c.Name, &c.City, &c.Avg, &c.Scored, &c.Students, &old); err != nil {
			return nil, err
		}
		if old != nil {
			c.Delta = *old - c.Place
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// Progress — баллы таймерных частей ученика по времени (для графика в профиле), старые первыми.
func (s *Store) Progress(ctx context.Context, userID int64, limit int) ([]float64, error) {
	rows, err := s.pool.Query(ctx, `SELECT band::float8 FROM (
		SELECT band, at FROM rated_parts WHERE user_id = $1 ORDER BY at DESC LIMIT $2) x ORDER BY at`, userID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[float64])
}

func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }
