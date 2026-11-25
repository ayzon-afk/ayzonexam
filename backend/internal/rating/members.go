package rating

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Кабинет центра: ученики центра с результатами. Центру видны только ник, аватар и результаты —
// имя, @username и Telegram id не уходят (DECISIONS 2026-11-15).

// Member — ученик центра в кабинете. Band — текущий балл по правилу рейтинга (nil — нет таймерных
// попыток за Window), Parts — последний балл по каждой части за Window. Month — заданий за 30 дней.
type Member struct {
	ID       int64              `json:"id"`
	Nickname string             `json:"nickname"`
	Avatar   string             `json:"avatar"`
	Role     *string            `json:"role"`
	Band     *float64           `json:"band"`
	Parts    map[string]float64 `json:"parts"`
	Month    int                `json:"month"`
	LastAt   *time.Time         `json:"last_at"`
}

// Members — все ученики центра (вместе с аккаунтом центра и учителями), лучшие по баллу — первыми.
func (s *Store) Members(ctx context.Context, centerID int64, now time.Time) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `WITH `+sprintf(scoresSQL, "cur", "$2")+`,
		pool AS (SELECT id, nickname, photo_url, center_role FROM users
			WHERE center_id = $1 AND center_status = 'member' AND blocked_at IS NULL),
		done AS (
			SELECT user_id, finished_at AS at FROM attempts WHERE finished_at IS NOT NULL AND user_id IN (SELECT id FROM pool)
			UNION ALL
			SELECT user_id, created_at FROM writing_checks WHERE user_id IN (SELECT id FROM pool)
			UNION ALL
			SELECT user_id, created_at FROM speaking_checks WHERE user_id IN (SELECT id FROM pool)),
		act AS (SELECT user_id, count(*) FILTER (WHERE at > $2::timestamptz - interval '30 days' AND at <= $2) AS month,
			max(at) AS last_at FROM done GROUP BY user_id)
		SELECT p.id, p.nickname, p.photo_url, p.center_role, cur.band::float8,
			(SELECT coalesce(json_object_agg(l.kind, l.band::float8), '{}') FROM last_cur l WHERE l.user_id = p.id),
			coalesce(act.month, 0), act.last_at
		FROM pool p LEFT JOIN cur ON cur.user_id = p.id LEFT JOIN act ON act.user_id = p.id
		ORDER BY cur.band DESC NULLS LAST, act.last_at DESC NULLS LAST, p.nickname, p.id`, centerID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Nickname, &m.Avatar, &m.Role, &m.Band, &m.Parts, &m.Month, &m.LastAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ErrForbidden — действие над учеником, которое аккаунту центра не позволено: над собой или над другим
// аккаунтом центра (их назначает только владелец).
var ErrForbidden = errors.New("нельзя")

// target — роль ученика в центре; ErrNotFound — он не ученик этого центра.
func (s *Store) target(ctx context.Context, centerID, actor, userID int64) (*string, error) {
	if actor == userID {
		return nil, ErrForbidden
	}
	var role *string
	err := s.pool.QueryRow(ctx, `SELECT center_role FROM users WHERE id = $1 AND center_id = $2 AND center_status = 'member'`,
		userID, centerID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if role != nil && *role == "admin" {
		return role, ErrForbidden
	}
	return role, nil
}

// Remove — аккаунт центра убрал ученика (или учителя) из центра.
func (s *Store) Remove(ctx context.Context, centerID, actor, userID int64) error {
	if _, err := s.target(ctx, centerID, actor, userID); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `UPDATE users SET center_id = NULL, center_status = NULL, center_role = NULL, updated_at = now()
		WHERE id = $1 AND center_id = $2 AND center_role IS DISTINCT FROM 'admin'`, userID, centerID)
	return err
}

// SetTeacher — аккаунт центра сделал ученика учителем центра или снял эту роль.
func (s *Store) SetTeacher(ctx context.Context, centerID, actor, userID int64, teacher bool) error {
	if _, err := s.target(ctx, centerID, actor, userID); err != nil {
		return err
	}
	var role any
	if teacher {
		role = "teacher"
	}
	_, err := s.pool.Exec(ctx, `UPDATE users SET center_role = $3, updated_at = now() WHERE id = $1 AND center_id = $2 AND center_role IS DISTINCT FROM 'admin'`, userID, centerID, role)
	return err
}
