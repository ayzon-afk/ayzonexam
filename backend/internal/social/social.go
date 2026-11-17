// Package social — друзья, приглашения с бонусами и дуэли 1 на 1 (этап 9).
package social

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/users"
)

var (
	ErrNotFound = errors.New("не найдено")
	ErrSelf     = errors.New("своя ссылка")
)

// NewUserWindow — сколько после регистрации ученик считается приглашённым по ссылке: бонусы — только
// за новых, а не за тех, кто давно в приложении и открыл чужую ссылку.
const NewUserWindow = 24 * time.Hour

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Who — кому написать ботом.
type Who struct {
	ID         int64
	TelegramID int64
	Lang       string
	Nickname   string
}

// code — 8 знаков без похожих (0/o, 1/l).
func code(n int) string {
	const abc = "abcdefghijkmnpqrstuvwxyz23456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = abc[int(b[i])%len(abc)]
	}
	return string(b)
}

// InviteCode — код ссылки «пригласить друга»; создаётся при первом обращении.
func (s *Store) InviteCode(ctx context.Context, u users.User) (string, error) {
	if u.InviteCode != nil {
		return *u.InviteCode, nil
	}
	for range 10 {
		c := code(8)
		var got string
		err := s.pool.QueryRow(ctx, `UPDATE users SET invite_code = coalesce(invite_code, $2) WHERE id = $1 RETURNING invite_code`, u.ID, c).Scan(&got)
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			continue
		}
		return got, err
	}
	return "", errors.New("код приглашения не подобран")
}

// Refer — ученик открыл ссылку друга: они становятся друзьями; новый ученик (не старше NewUserWindow,
// ещё ничьей ссылкой не пришедший) записывается приглашённым — за него пригласившему бонусы.
func (s *Store) Refer(ctx context.Context, u users.User, inviteCode string, now time.Time) (Who, bool, error) {
	var ref Who
	err := s.pool.QueryRow(ctx, `SELECT id, telegram_id, lang, nickname FROM users WHERE invite_code = $1 AND blocked_at IS NULL`,
		inviteCode).Scan(&ref.ID, &ref.TelegramID, &ref.Lang, &ref.Nickname)
	if errors.Is(err, pgx.ErrNoRows) {
		return ref, false, ErrNotFound
	}
	if err != nil {
		return ref, false, err
	}
	if ref.ID == u.ID {
		return ref, false, ErrSelf
	}
	if err := s.Befriend(ctx, u.ID, ref.ID); err != nil {
		return ref, false, err
	}
	if u.ReferredBy != nil || now.Sub(u.CreatedAt) > NewUserWindow {
		return ref, false, nil
	}
	tag, err := s.pool.Exec(ctx, `UPDATE users SET referred_by = $2 WHERE id = $1 AND referred_by IS NULL`, u.ID, ref.ID)
	return ref, err == nil && tag.RowsAffected() == 1, err
}

// Befriend — дружба в обе стороны; уже друзья — не ошибка.
func (s *Store) Befriend(ctx context.Context, a, b int64) error {
	if a == b {
		return nil
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO friends (user_id, friend_id) VALUES ($1, $2), ($2, $1) ON CONFLICT DO NOTHING`, a, b)
	return err
}

func (s *Store) Unfriend(ctx context.Context, a, b int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM friends WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`, a, b)
	return err
}

// RewardLevel — приглашённый впервые сдал тест уровня: пригласившему +1 бонусная проверка (SPEC).
// nil — бонус уже выдан или ученик не приглашён.
func (s *Store) RewardLevel(ctx context.Context, userID int64) (*Who, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var refID int64
	err = tx.QueryRow(ctx, `UPDATE users SET referral_rewarded = true
		WHERE id = $1 AND referred_by IS NOT NULL AND NOT referral_rewarded RETURNING referred_by`, userID).Scan(&refID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var w Who
	if err := tx.QueryRow(ctx, `UPDATE users SET bonus_checks = bonus_checks + 1 WHERE id = $1
		RETURNING id, telegram_id, lang, nickname`, refID).Scan(&w.ID, &w.TelegramID, &w.Lang, &w.Nickname); err != nil {
		return nil, err
	}
	return &w, tx.Commit(ctx)
}

// Friend — друг в списке: только публичное.
type Friend struct {
	ID       int64    `json:"id"`
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Band     *float64 `json:"band"`
}

// Friends — друзья ученика с их текущим баллом (как в рейтинге: последние таймерные части за 60 дней).
func (s *Store) Friends(ctx context.Context, userID int64, now time.Time) ([]Friend, error) {
	rows, err := s.pool.Query(ctx, `WITH last AS (
			SELECT DISTINCT ON (user_id, kind) user_id, kind, band FROM rated_parts
			WHERE at <= $2 AND at > $2 - interval '60 days' AND user_id IN (SELECT friend_id FROM friends WHERE user_id = $1)
			ORDER BY user_id, kind, at DESC),
		score AS (SELECT user_id, (floor(avg(band) * 2 + 0.5) / 2)::float8 AS band FROM last GROUP BY user_id)
		SELECT u.id, u.nickname, u.photo_url, score.band FROM friends f JOIN users u ON u.id = f.friend_id
		LEFT JOIN score ON score.user_id = u.id
		WHERE f.user_id = $1 AND u.blocked_at IS NULL ORDER BY score.band DESC NULLS LAST, u.nickname`, userID, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Friend])
}

// AreFriends — друзья ли a и b (вызвать на дуэль напрямую можно только друга).
func (s *Store) AreFriends(ctx context.Context, a, b int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM friends WHERE user_id = $1 AND friend_id = $2)`, a, b).Scan(&ok)
	return ok, err
}

// WhoByID — кому написать ботом.
func (s *Store) WhoByID(ctx context.Context, id int64) (Who, error) {
	var w Who
	err := s.pool.QueryRow(ctx, `SELECT id, telegram_id, lang, nickname FROM users WHERE id = $1`, id).Scan(&w.ID, &w.TelegramID, &w.Lang, &w.Nickname)
	return w, err
}
