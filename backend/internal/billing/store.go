package billing

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("не найдено")
	ErrPending  = errors.New("уже есть оплата на проверке")
	ErrDecided  = errors.New("оплата уже проверена")
	ErrPlan     = errors.New("неизвестный тариф")
)

type Payment struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Code         string     `json:"code"`
	Plan         string     `json:"plan"`
	Amount       int        `json:"amount"`
	Status       string     `json:"status"`
	Screenshot   int64      `json:"screenshot"`
	GrantedDays  int        `json:"granted_days"`
	GrantedMocks int        `json:"granted_mocks"`
	Reason       string     `json:"reason"`
	CreatedAt    time.Time  `json:"created_at"`
	DecidedAt    *time.Time `json:"decided_at"`
	// Для админки и уведомлений.
	Nickname   string `json:"nickname"`
	TelegramID int64  `json:"telegram_id"`
	Lang       string `json:"lang"`
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// PayCode — код платежа ученика для комментария к переводу (AY-4821): один на ученика, владелец по нему
// находит перевод в выписке. Создаётся при первом обращении.
func (s *Store) PayCode(ctx context.Context, userID int64) (string, error) {
	var code *string
	if err := s.pool.QueryRow(ctx, `SELECT pay_code FROM users WHERE id = $1`, userID).Scan(&code); err != nil {
		return "", err
	}
	if code != nil {
		return *code, nil
	}
	for range 20 {
		n, _ := rand.Int(rand.Reader, big.NewInt(900_000))
		c := fmt.Sprintf("AY-%d", 100_000+n.Int64())
		tag, err := s.pool.Exec(ctx, `UPDATE users SET pay_code = $2 WHERE id = $1 AND pay_code IS NULL`, userID, c)
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			continue
		}
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 {
			return s.PayCode(ctx, userID)
		}
		return c, nil
	}
	return "", errors.New("код платежа не подобран")
}

// PaidBefore — была ли у ученика подтверждённая оплата (скидка — только на первый месяц).
func (s *Store) PaidBefore(ctx context.Context, userID int64) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM payments WHERE user_id = $1 AND status = 'confirmed')`, userID).Scan(&ok)
	return ok, err
}

const payCols = `p.id, p.user_id, p.code, p.plan, p.amount, p.status, p.screenshot, p.granted_days, p.granted_mocks, p.reason,
	p.created_at, p.decided_at, u.nickname, u.telegram_id, u.lang`

func scanPayment(row pgx.Row) (Payment, error) {
	var p Payment
	err := row.Scan(&p.ID, &p.UserID, &p.Code, &p.Plan, &p.Amount, &p.Status, &p.Screenshot, &p.GrantedDays, &p.GrantedMocks,
		&p.Reason, &p.CreatedAt, &p.DecidedAt, &p.Nickname, &p.TelegramID, &p.Lang)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// Pending — непроверенная оплата ученика, если есть.
func (s *Store) Pending(ctx context.Context, userID int64) (*Payment, error) {
	p, err := scanPayment(s.pool.QueryRow(ctx, `SELECT `+payCols+` FROM payments p JOIN users u ON u.id = p.user_id
		WHERE p.user_id = $1 AND p.status = 'pending'`, userID))
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return &p, err
}

func (s *Store) Get(ctx context.Context, id int64) (Payment, error) {
	return scanPayment(s.pool.QueryRow(ctx, `SELECT `+payCols+` FROM payments p JOIN users u ON u.id = p.user_id WHERE p.id = $1`, id))
}

// List — оплаты для админки, новые первыми; status ” — все.
func (s *Store) List(ctx context.Context, status string, limit int) ([]Payment, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+payCols+` FROM payments p JOIN users u ON u.id = p.user_id
		WHERE $1 = '' OR p.status = $1 ORDER BY p.created_at DESC LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Payment{}
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// Submit — ученик перевёл деньги и приложил скрин. Premium (или mock) открывается сразу «на проверке»,
// кроме тех, кому уже отказывали: им — только после подтверждения (SPEC, «Защита от обмана»).
func (s *Store) Submit(ctx context.Context, userID int64, plan, mime string, screenshot []byte, now time.Time) (Payment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer tx.Rollback(ctx)
	var createdAt time.Time
	var trust string
	var code *string
	if err := tx.QueryRow(ctx, `SELECT created_at, pay_trust, pay_code FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&createdAt, &trust, &code); err != nil {
		return Payment{}, err
	}
	if code == nil {
		return Payment{}, errors.New("нет кода платежа")
	}
	var pending, paid bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM payments WHERE user_id = $1 AND status = 'pending'),
		EXISTS (SELECT 1 FROM payments WHERE user_id = $1 AND status = 'confirmed')`, userID).Scan(&pending, &paid); err != nil {
		return Payment{}, err
	}
	if pending {
		return Payment{}, ErrPending
	}
	discount := !paid && now.Before(createdAt.Add(DiscountDays*24*time.Hour))
	amount, days, mocks, ok := Price(plan, discount)
	if !ok {
		return Payment{}, ErrPlan
	}
	if trust == "rejected" {
		days, mocks = 0, 0
	}
	var mediaID, id int64
	if err := tx.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, true) RETURNING id`, mime, screenshot).Scan(&mediaID); err != nil {
		return Payment{}, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO payments (user_id, code, plan, amount, screenshot, granted_days, granted_mocks, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`, userID, *code, plan, amount, mediaID, days, mocks, now).Scan(&id); err != nil {
		return Payment{}, err
	}
	if err := grant(ctx, tx, userID, days, mocks, now); err != nil {
		return Payment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}
	return s.Get(ctx, id)
}

// grant продлевает Premium на days (от сейчас или от конца текущего) и добавляет mocks разовых mock.
func grant(ctx context.Context, tx pgx.Tx, userID int64, days, mocks int, now time.Time) error {
	if days == 0 && mocks == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `UPDATE users SET
			premium_until = CASE WHEN $2 > 0 THEN greatest(coalesce(premium_until, $4), $4) + make_interval(days => $2) ELSE premium_until END,
			mock_credits = mock_credits + $3, updated_at = now()
		WHERE id = $1`, userID, days, mocks, now)
	return err
}

// Decided — итог проверки: кого уведомить и, если пригласивший получил бонус, — его.
type Decided struct {
	Payment
	ReferrerTG   int64 // 0 — бонуса нет
	ReferrerLang string
}

// Confirm — владелец нашёл перевод. Тем, кому доступ не открывали сразу, он открывается сейчас.
// Первая оплата приглашённого другом — пригласившему +7 дней Premium.
func (s *Store) Confirm(ctx context.Context, id, adminTG int64, now time.Time) (Decided, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Decided{}, err
	}
	defer tx.Rollback(ctx)
	var userID int64
	var plan, status string
	var days, mocks int
	if err := tx.QueryRow(ctx, `SELECT user_id, plan, status, granted_days, granted_mocks FROM payments WHERE id = $1 FOR UPDATE`,
		id).Scan(&userID, &plan, &status, &days, &mocks); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Decided{}, ErrNotFound
		}
		return Decided{}, err
	}
	if status != "pending" {
		return Decided{}, ErrDecided
	}
	var firstPaid bool
	if err := tx.QueryRow(ctx, `SELECT NOT EXISTS (SELECT 1 FROM payments WHERE user_id = $1 AND status = 'confirmed')`, userID).Scan(&firstPaid); err != nil {
		return Decided{}, err
	}
	if days == 0 && mocks == 0 {
		// Доступ не открывали сразу (ученику уже отказывали) — открыть сейчас.
		_, d, m, _ := Price(plan, false)
		if err := grant(ctx, tx, userID, d, m, now); err != nil {
			return Decided{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET granted_days = $2, granted_mocks = $3 WHERE id = $1`, id, d, m); err != nil {
			return Decided{}, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'confirmed', decided_by = $2, decided_at = $3 WHERE id = $1`, id, adminTG, now); err != nil {
		return Decided{}, err
	}
	var out Decided
	if firstPaid && plan != "mock" {
		var refID *int64
		if err := tx.QueryRow(ctx, `SELECT referred_by FROM users WHERE id = $1`, userID).Scan(&refID); err != nil {
			return Decided{}, err
		}
		if refID != nil {
			if err := grant(ctx, tx, *refID, ReferralDays, 0, now); err != nil {
				return Decided{}, err
			}
			if err := tx.QueryRow(ctx, `SELECT telegram_id, lang FROM users WHERE id = $1`, *refID).Scan(&out.ReferrerTG, &out.ReferrerLang); err != nil {
				return Decided{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Decided{}, err
	}
	out.Payment, err = s.Get(ctx, id)
	return out, err
}

// Reject — перевод не найден: открытый «на проверке» доступ забирается, дальше — только после
// подтверждения.
func (s *Store) Reject(ctx context.Context, id, adminTG int64, reason string, now time.Time) (Payment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payment{}, err
	}
	defer tx.Rollback(ctx)
	var userID int64
	var status string
	var days, mocks int
	if err := tx.QueryRow(ctx, `SELECT user_id, status, granted_days, granted_mocks FROM payments WHERE id = $1 FOR UPDATE`, id).Scan(&userID, &status, &days, &mocks); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, ErrNotFound
		}
		return Payment{}, err
	}
	if status != "pending" {
		return Payment{}, ErrDecided
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET
			premium_until = CASE WHEN $2 > 0 THEN premium_until - make_interval(days => $2) ELSE premium_until END,
			mock_credits = greatest(mock_credits - $3, 0), pay_trust = 'rejected', updated_at = now()
		WHERE id = $1`, userID, days, mocks); err != nil {
		return Payment{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'rejected', reason = $2, decided_by = $3, decided_at = $4 WHERE id = $1`,
		id, reason, adminTG, now); err != nil {
		return Payment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Payment{}, err
	}
	return s.Get(ctx, id)
}

// SetPremium — владелец вручную выдал, продлил (days > 0) или забрал (days < 0 — до сейчас) Premium.
func (s *Store) SetPremium(ctx context.Context, userID int64, days int, now time.Time) error {
	if days < 0 {
		_, err := s.pool.Exec(ctx, `UPDATE users SET premium_until = least(premium_until, $2), updated_at = now() WHERE id = $1`, userID, now)
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := grant(ctx, tx, userID, days, 0, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UsedMocks — сколько полных mock ученик начал с since за Premium (разовые покупки не считаются).
func (s *Store) UsedMocks(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM mocks WHERE user_id = $1 AND started_at >= $2 AND paid_with = 'premium'`, userID, since).Scan(&n)
	return n, err
}

// CenterPayment — владелец принял оплату центра: тариф продлевается до paidUntil на seats учеников.
func (s *Store) CenterPayment(ctx context.Context, centerID int64, amount, seats int, paidUntil time.Time, receipt *int64, note string, adminTG int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	day := paidUntil.Format(time.DateOnly)
	if _, err := tx.Exec(ctx, `INSERT INTO center_payments (center_id, amount, seats, paid_until, receipt, note, admin_tg)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, centerID, amount, seats, day, receipt, note, adminTG); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE centers SET seats = $2, paid_until = $3 WHERE id = $1`, centerID, seats, day)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}
