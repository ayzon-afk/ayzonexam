// Package users хранит учеников.
package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/tgauth"
)

type User struct {
	ID          int64
	TelegramID  int64
	FirstName   string
	LastName    string
	Username    string
	PhotoURL    string
	Nickname    string
	Lang        string
	GoalBand    *float64
	ExamDate    *time.Time
	OnboardedAt *time.Time
	CreatedAt   time.Time

	City            string // '' — не указан, tashkent, other
	CenterID        *int64
	CenterStatus    *string // pending, member
	CenterRole      *string // admin, teacher — аккаунт центра и учитель
	VerifiedTeacher bool
	BlockedAt       *time.Time

	InviteCode   *string
	ReferredBy   *int64
	BonusChecks  int
	BotState     string
	PremiumUntil *time.Time
	PayTrust     string     // new; rejected — после отказа Premium только после подтверждения
	MockCredits  int        // купленные разовые mock
	CenterPaid   *time.Time // последний оплаченный день центра ученика (дата, если он member)
	SeenAt       *time.Time
}

const cols = `id, telegram_id, first_name, last_name, username, photo_url, nickname, lang, goal_band, exam_date, onboarded_at,
	created_at, city, center_id, center_status, center_role, verified_teacher, blocked_at,
	invite_code, referred_by, bonus_checks, bot_state, premium_until, pay_trust, mock_credits,
	(SELECT c.paid_until FROM centers c WHERE c.id = users.center_id AND users.center_status = 'member' AND c.active), seen_at`

func scan(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.LastName, &u.Username, &u.PhotoURL,
		&u.Nickname, &u.Lang, &u.GoalBand, &u.ExamDate, &u.OnboardedAt,
		&u.CreatedAt, &u.City, &u.CenterID, &u.CenterStatus, &u.CenterRole, &u.VerifiedTeacher, &u.BlockedAt,
		&u.InviteCode, &u.ReferredBy, &u.BonusChecks, &u.BotState, &u.PremiumUntil, &u.PayTrust, &u.MockCredits, &u.CenterPaid, &u.SeenAt)
	return u, err
}

// ByTelegram — ученик по Telegram id (бот); pgx.ErrNoRows — такого нет.
func (s *Store) ByTelegram(ctx context.Context, tgID int64) (User, error) {
	return scan(s.pool.QueryRow(ctx, `SELECT `+cols+` FROM users WHERE telegram_id = $1`, tgID))
}

// Seen отмечает, что ученик открыл приложение; чаще раза в час не пишет.
func (s *Store) Seen(ctx context.Context, u User, now time.Time) error {
	if u.SeenAt != nil && now.Sub(*u.SeenAt) < time.Hour {
		return nil
	}
	_, err := s.pool.Exec(ctx, `UPDATE users SET seen_at = $2 WHERE id = $1`, u.ID, now)
	return err
}

// SetBotState — что бот ждёт от ученика следующим сообщением (” — ничего).
func (s *Store) SetBotState(ctx context.Context, id int64, state string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET bot_state = $2 WHERE id = $1`, id, state)
	return err
}

// ErrNickTaken — ник уже занят другим учеником.
var ErrNickTaken = errors.New("ник занят")

// Member — ученик состоит в центре (не просто просится).
func (u User) Member() bool {
	return u.CenterID != nil && u.CenterStatus != nil && *u.CenterStatus == "member"
}

// Get — ученик по id.
func (s *Store) Get(ctx context.Context, id int64) (User, error) {
	return scan(s.pool.QueryRow(ctx, `SELECT `+cols+` FROM users WHERE id = $1`, id))
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Ensure возвращает ученика по Telegram-аккаунту, создавая его при первом входе.
// Имя, @username и фото обновляются, если человек сменил их в Telegram.
func (s *Store) Ensure(ctx context.Context, tg tgauth.User) (User, error) {
	u, err := scan(s.pool.QueryRow(ctx, `SELECT `+cols+` FROM users WHERE telegram_id = $1`, tg.ID))
	if err == nil {
		if u.FirstName == tg.FirstName && u.LastName == tg.LastName && u.Username == tg.Username && u.PhotoURL == tg.PhotoURL {
			return u, nil
		}
		return scan(s.pool.QueryRow(ctx, `UPDATE users
			SET first_name = $2, last_name = $3, username = $4, photo_url = $5, updated_at = now()
			WHERE id = $1 RETURNING `+cols, u.ID, tg.FirstName, tg.LastName, tg.Username, tg.PhotoURL))
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}

	// Ник по умолчанию — user<id>: не раскрывает ни имя, ни Telegram id. Новому — 3 дня Premium (SPEC).
	u, err = scan(s.pool.QueryRow(ctx, `INSERT INTO users (id, telegram_id, first_name, last_name, username, photo_url, lang, nickname, premium_until)
		SELECT n, $1, $2, $3, $4, $5, $6, 'user' || n, now() + interval '3 days' FROM nextval('users_id_seq') AS n
		ON CONFLICT (telegram_id) DO NOTHING
		RETURNING `+cols, tg.ID, tg.FirstName, tg.LastName, tg.Username, tg.PhotoURL, LangFromTelegram(tg.LanguageCode)))
	if errors.Is(err, pgx.ErrNoRows) {
		// Два первых запроса пришли одновременно — второй находит созданного первым.
		return scan(s.pool.QueryRow(ctx, `SELECT `+cols+` FROM users WHERE telegram_id = $1`, tg.ID))
	}
	return u, err
}

// LangFromTelegram — язык интерфейса по языку Telegram; по умолчанию узбекский (SPEC).
func LangFromTelegram(code string) string {
	switch code {
	case "ru", "en":
		return code
	}
	return "uz"
}

// Patch — что меняется в профиле; nil — не трогать.
type Patch struct {
	Lang        *string
	GoalBand    *float64
	SetExamDate bool       // true — записать ExamDate, даже если он nil («пока не знаю»)
	ExamDate    *time.Time // только дата, время не важно
	Onboarded   bool
	Nickname    *string
	City        *string
}

func (s *Store) Update(ctx context.Context, id int64, p Patch) (User, error) {
	var examDate any
	if p.ExamDate != nil {
		examDate = p.ExamDate.Format(time.DateOnly)
	}
	u, err := scan(s.pool.QueryRow(ctx, `UPDATE users SET
			lang = COALESCE($2, lang),
			goal_band = COALESCE($3::numeric, goal_band),
			exam_date = CASE WHEN $4 THEN $5::date ELSE exam_date END,
			onboarded_at = CASE WHEN $6 AND onboarded_at IS NULL THEN now() ELSE onboarded_at END,
			nickname = COALESCE($7, nickname),
			city = COALESCE($8, city),
			updated_at = now()
		WHERE id = $1 RETURNING `+cols,
		id, p.Lang, p.GoalBand, p.SetExamDate, examDate, p.Onboarded, p.Nickname, p.City))
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return u, ErrNickTaken
	}
	return u, err
}
