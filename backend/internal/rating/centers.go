package rating

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Аккаунт центра (SPEC, «Учителя и учебные центры»): ученик входит по ссылке-приглашению сразу или
// просится сам — тогда центр подтверждает.

type CenterInfo struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	City     string   `json:"city"`
	About    string   `json:"about"`
	Students int      `json:"students"`
	Avg      *float64 `json:"avg"`
	Place    *int     `json:"place"`
	Top      []Row    `json:"top"`
}

// List — активные центры для выбора учеником.
func (s *Store) List(ctx context.Context) ([]CenterInfo, error) {
	rows, err := s.pool.Query(ctx, `SELECT c.id, c.name, c.city, c.about,
			(SELECT count(*) FROM users m WHERE m.center_id = c.id AND m.center_status = 'member')
		FROM centers c WHERE c.active ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []CenterInfo{}
	for rows.Next() {
		var c CenterInfo
		if err := rows.Scan(&c.ID, &c.Name, &c.City, &c.About, &c.Students); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// Center — страница центра: место среди центров и лучшие ученики по текущему баллу.
func (s *Store) Center(ctx context.Context, id int64, now time.Time) (CenterInfo, error) {
	var c CenterInfo
	err := s.pool.QueryRow(ctx, `SELECT c.id, c.name, c.city, c.about,
			(SELECT count(*) FROM users m WHERE m.center_id = c.id AND m.center_status = 'member')
		FROM centers c WHERE c.id = $1 AND c.active`, id).Scan(&c.ID, &c.Name, &c.City, &c.About, &c.Students)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	all, err := s.Centers(ctx, now)
	if err != nil {
		return c, err
	}
	for _, r := range all {
		if r.ID == id {
			avg, place := r.Avg, r.Place
			c.Avg, c.Place = &avg, &place
		}
	}
	b, err := s.Students(ctx, Center, 0, &id, now)
	if err != nil {
		return c, err
	}
	c.Top = b.Rows[:min(len(b.Rows), 5)]
	return c, nil
}

// JoinByCode — вход по ссылке-приглашению: сразу member, без подтверждения. Город берётся у центра,
// если ученик свой не указал.
func (s *Store) JoinByCode(ctx context.Context, userID int64, code string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `UPDATE users u SET center_id = c.id, center_status = 'member',
			city = CASE WHEN u.city = '' THEN c.city ELSE u.city END, updated_at = now()
		FROM centers c WHERE u.id = $1 AND c.invite_code = $2 AND c.active RETURNING c.id`, userID, code).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

// Request — ученик сам выбрал центр: ждёт подтверждения. Прежний центр (и роль в нём) снимается.
func (s *Store) Request(ctx context.Context, userID, centerID int64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET center_id = $2, center_status = 'pending', center_role = NULL, updated_at = now()
		WHERE id = $1 AND EXISTS (SELECT 1 FROM centers WHERE id = $2 AND active)
			AND NOT (center_id IS NOT DISTINCT FROM $2 AND center_status = 'member')`, userID, centerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Leave(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET center_id = NULL, center_status = NULL, center_role = NULL, updated_at = now()
		WHERE id = $1`, userID)
	return err
}

// Pending — ученик, который просится в центр. Центру видны только ник и аватар (DECISIONS 2026-11-15).
type Pending struct {
	UserID   int64     `json:"user_id"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Since    time.Time `json:"since"`
}

func (s *Store) Requests(ctx context.Context, centerID int64) ([]Pending, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, nickname, photo_url, updated_at FROM users
		WHERE center_id = $1 AND center_status = 'pending' ORDER BY updated_at`, centerID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Pending])
}

// Decide — центр принял или отклонил просьбу ученика.
func (s *Store) Decide(ctx context.Context, centerID, userID int64, accept bool) error {
	sql := `UPDATE users u SET center_status = 'member', city = CASE WHEN u.city = '' THEN c.city ELSE u.city END, updated_at = now()
		FROM centers c WHERE c.id = u.center_id AND u.id = $2 AND u.center_id = $1 AND u.center_status = 'pending'`
	if !accept {
		sql = `UPDATE users SET center_id = NULL, center_status = NULL, updated_at = now()
			WHERE id = $2 AND center_id = $1 AND center_status = 'pending'`
	}
	tag, err := s.pool.Exec(ctx, sql, centerID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// NewCode — код ссылки-приглашения: 10 знаков без похожих (0/o, 1/l), подобрать нельзя.
func NewCode() string {
	const abc = "abcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, 10)
	rand.Read(b)
	for i := range b {
		b[i] = abc[int(b[i])%len(abc)]
	}
	return string(b)
}
