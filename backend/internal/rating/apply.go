package rating

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ayzonexam/internal/billing"
)

// Заявка «Добавить свой центр / Я учитель» (DECISIONS 2026-11-25): заявитель вошёл через Telegram и
// заполнил анкету; владелец одобряет кнопкой в боте — центр (или учитель) создаётся сам, заявитель
// становится его аккаунтом.

var (
	ErrApplyPending = errors.New("заявка уже на проверке")
	ErrApplyStaff   = errors.New("уже управляет центром")
	ErrApplyDecided = errors.New("заявка уже рассмотрена")
)

type Application struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	Students  int       `json:"students"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	// Для владельца и уведомлений — заявителю не отдаётся.
	Phone      string     `json:"-"`
	About      string     `json:"-"`
	Pilot      bool       `json:"-"`
	CenterID   *int64     `json:"-"`
	PaidUntil  *time.Time `json:"-"`
	TelegramID int64      `json:"-"`
	Lang       string     `json:"-"`
	Nickname   string     `json:"-"`
	Username   string     `json:"-"`
}

const applyCols = `a.id, a.kind, a.name, a.city, a.students, a.status, a.created_at, a.phone, a.about, a.pilot, a.center_id,
	c.paid_until, u.telegram_id, u.lang, u.nickname, u.username`

func (s *Store) application(ctx context.Context, where string, arg any) (Application, error) {
	var a Application
	err := s.pool.QueryRow(ctx, `SELECT `+applyCols+` FROM center_applications a JOIN users u ON u.id = a.user_id
		LEFT JOIN centers c ON c.id = a.center_id WHERE `+where+` ORDER BY a.id DESC LIMIT 1`, arg).
		Scan(&a.ID, &a.Kind, &a.Name, &a.City, &a.Students, &a.Status, &a.CreatedAt, &a.Phone, &a.About, &a.Pilot, &a.CenterID,
			&a.PaidUntil, &a.TelegramID, &a.Lang, &a.Nickname, &a.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

// MyApplication — последняя заявка человека; nil — не подавал.
func (s *Store) MyApplication(ctx context.Context, userID int64) (*Application, error) {
	a, err := s.application(ctx, `a.user_id = $1`, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return &a, err
}

// Apply — новая заявка. Кто уже управляет центром или ждёт ответа — нельзя.
func (s *Store) Apply(ctx context.Context, userID int64, a Application, now time.Time) (Application, error) {
	var staff, pending bool
	if err := s.pool.QueryRow(ctx, `SELECT
			EXISTS (SELECT 1 FROM users WHERE id = $1 AND center_role = 'admin' AND center_status = 'member'),
			EXISTS (SELECT 1 FROM center_applications WHERE user_id = $1 AND status = 'pending')`, userID).Scan(&staff, &pending); err != nil {
		return a, err
	}
	if staff {
		return a, ErrApplyStaff
	}
	if pending {
		return a, ErrApplyPending
	}
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO center_applications (user_id, kind, name, city, students, phone, about, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`, userID, a.Kind, a.Name, a.City, a.Students, a.Phone, a.About, now).Scan(&id)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return a, ErrApplyPending
	}
	if err != nil {
		return a, err
	}
	return s.application(ctx, `a.id = $1`, id)
}

// Approve — владелец одобрил: центр создаётся с регионом из анкеты, заявитель — его аккаунт (из прежнего
// центра уходит). pilot — бесплатный месяц: мест — сколько учеников в анкете, но не меньше минимума и не
// больше 100. Название занято — к нему добавляется номер заявки.
func (s *Store) Approve(ctx context.Context, id, adminTG int64, pilot bool, now time.Time) (Application, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Application{}, err
	}
	defer tx.Rollback(ctx)
	var userID int64
	var kind, name, city, about, status string
	var students int
	if err := tx.QueryRow(ctx, `SELECT user_id, kind, name, city, about, students, status FROM center_applications WHERE id = $1 FOR UPDATE`, id).
		Scan(&userID, &kind, &name, &city, &about, &students, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Application{}, ErrNotFound
		}
		return Application{}, err
	}
	if status != "pending" {
		return Application{}, ErrApplyDecided
	}
	var taken bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM centers WHERE name = $1)`, name).Scan(&taken); err != nil {
		return Application{}, err
	}
	if taken {
		name += " #" + strconv.FormatInt(id, 10)
	}
	seats, paid := 0, (*string)(nil)
	if pilot {
		seats = min(max(students, billing.MinSeats(kind)), 100)
		day := billing.CenterPaidUntil(nil, 1, now).Format(time.DateOnly)
		paid = &day
	}
	var centerID int64
	if err := tx.QueryRow(ctx, `INSERT INTO centers (name, kind, city, about, invite_code, seats, paid_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, name, kind, city, about, NewCode(), seats, paid).Scan(&centerID); err != nil {
		return Application{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET center_id = $2, center_status = 'member', center_role = 'admin',
			city = CASE WHEN city = '' THEN $3 ELSE city END, updated_at = now() WHERE id = $1`, userID, centerID, city); err != nil {
		return Application{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE center_applications SET status = 'approved', pilot = $2, center_id = $3, decided_by = $4,
			decided_at = $5 WHERE id = $1`, id, pilot, centerID, adminTG, now); err != nil {
		return Application{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Application{}, err
	}
	return s.application(ctx, `a.id = $1`, id)
}

// Reject — владелец отказал: заявитель может подать снова.
func (s *Store) Reject(ctx context.Context, id, adminTG int64, now time.Time) (Application, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE center_applications SET status = 'rejected', decided_by = $2, decided_at = $3
		WHERE id = $1 AND status = 'pending'`, id, adminTG, now)
	if err != nil {
		return Application{}, err
	}
	a, err := s.application(ctx, `a.id = $1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return a, ErrApplyDecided
	}
	return a, err
}
