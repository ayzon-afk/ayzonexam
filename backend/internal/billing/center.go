package billing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
)

// Тариф центра и учителя без центра (SPEC, «Деньги»): за ученика в месяц, от MinSeats; от CenterBigSeats —
// дешевле. Те же правила — в интерфейсе (screens/cabinet/cabinet.ts, centerPrice): меняется одно — меняется
// и другое.
const (
	CenterSeat      = 20_000
	CenterSeatBig   = 15_000
	CenterBigSeats  = 100
	CenterMinSeats  = 20
	TeacherMinSeats = 10
	CenterMaxSeats  = 5000
)

// MinSeats — меньше скольких мест не продлить; kind — centers.kind (center, teacher).
func MinSeats(kind string) int {
	if kind == "teacher" {
		return TeacherMinSeats
	}
	return CenterMinSeats
}

// CenterMonths — на сколько месяцев центр может продлить тариф сам.
var CenterMonths = []int{1, 3, 6, 12}

// CenterPrice — сумма за seats мест на months месяцев для вида kind; false — так продлить нельзя.
func CenterPrice(kind string, seats, months int) (int, bool) {
	if seats < MinSeats(kind) || seats > CenterMaxSeats || !slices.Contains(CenterMonths, months) {
		return 0, false
	}
	per := CenterSeat
	if seats >= CenterBigSeats {
		per = CenterSeatBig
	}
	return seats * per * months, true
}

// CenterPayCode — код для комментария к переводу центра: по нему владелец находит платёж в выписке.
func CenterPayCode(centerID int64) string { return fmt.Sprintf("AYC-%d", centerID) }

// CenterPaidUntil — новый срок: от конца оплаченного (если ещё не кончился) или от сегодня по Ташкенту.
func CenterPaidUntil(paidUntil *time.Time, months int, now time.Time) time.Time {
	today := DayStart(now)
	from := today
	if paidUntil != nil {
		p := time.Date(paidUntil.Year(), paidUntil.Month(), paidUntil.Day(), 0, 0, 0, 0, Tashkent)
		if !p.Before(today) {
			from = p
		}
	}
	return from.AddDate(0, months, 0)
}

type CenterOrder struct {
	ID        int64      `json:"id"`
	CenterID  int64      `json:"-"`
	Seats     int        `json:"seats"`
	Months    int        `json:"months"`
	Amount    int        `json:"amount"`
	Status    string     `json:"status"`
	Reason    string     `json:"reason"`
	CreatedAt time.Time  `json:"created_at"`
	PaidUntil *time.Time `json:"-"` // после подтверждения — новый срок центра
	// Для уведомлений: кто отправил и центр.
	Center     string `json:"-"`
	TelegramID int64  `json:"-"`
	Lang       string `json:"-"`
	Nickname   string `json:"-"`
}

const orderCols = `o.id, o.center_id, o.seats, o.months, o.amount, o.status, o.reason, o.created_at, c.paid_until,
	c.name, u.telegram_id, u.lang, u.nickname`

func scanOrder(row pgx.Row) (CenterOrder, error) {
	var o CenterOrder
	err := row.Scan(&o.ID, &o.CenterID, &o.Seats, &o.Months, &o.Amount, &o.Status, &o.Reason, &o.CreatedAt, &o.PaidUntil,
		&o.Center, &o.TelegramID, &o.Lang, &o.Nickname)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

func (s *Store) order(ctx context.Context, where string, arg any) (CenterOrder, error) {
	return scanOrder(s.pool.QueryRow(ctx, `SELECT `+orderCols+` FROM center_orders o
		JOIN centers c ON c.id = o.center_id JOIN users u ON u.id = o.user_id WHERE `+where, arg))
}

// PendingCenterOrder — непроверенная заявка центра, если есть.
func (s *Store) PendingCenterOrder(ctx context.Context, centerID int64) (*CenterOrder, error) {
	o, err := s.order(ctx, `o.center_id = $1 AND o.status = 'pending'`, centerID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return &o, err
}

// SubmitCenterOrder — аккаунт центра перевёл деньги и приложил скрин; тариф продлится после подтверждения.
func (s *Store) SubmitCenterOrder(ctx context.Context, centerID, userID int64, seats, months int, mime string, screenshot []byte, now time.Time) (CenterOrder, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CenterOrder{}, err
	}
	defer tx.Rollback(ctx)
	// Блокировка центра: две заявки одновременно не пройдут проверку «нет непроверенной».
	var kind string
	if err := tx.QueryRow(ctx, `SELECT kind FROM centers WHERE id = $1 FOR UPDATE`, centerID).Scan(&kind); err != nil {
		return CenterOrder{}, err
	}
	amount, ok := CenterPrice(kind, seats, months)
	if !ok {
		return CenterOrder{}, ErrPlan
	}
	var pending bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM center_orders WHERE center_id = $1 AND status = 'pending')`, centerID).Scan(&pending); err != nil {
		return CenterOrder{}, err
	}
	if pending {
		return CenterOrder{}, ErrPending
	}
	var mediaID, id int64
	if err := tx.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, true) RETURNING id`, mime, screenshot).Scan(&mediaID); err != nil {
		return CenterOrder{}, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO center_orders (center_id, user_id, seats, months, amount, screenshot, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, centerID, userID, seats, months, amount, mediaID, now).Scan(&id); err != nil {
		return CenterOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CenterOrder{}, err
	}
	return s.order(ctx, `o.id = $1`, id)
}

// ConfirmCenterOrder — владелец нашёл перевод: оплата записывается в center_payments (скрин — чек),
// места — как в заявке, срок — CenterPaidUntil.
func (s *Store) ConfirmCenterOrder(ctx context.Context, id, adminTG int64, now time.Time) (CenterOrder, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CenterOrder{}, err
	}
	defer tx.Rollback(ctx)
	var centerID, screenshot int64
	var seats, months, amount int
	var status string
	if err := tx.QueryRow(ctx, `SELECT center_id, seats, months, amount, screenshot, status FROM center_orders WHERE id = $1 FOR UPDATE`, id).
		Scan(&centerID, &seats, &months, &amount, &screenshot, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CenterOrder{}, ErrNotFound
		}
		return CenterOrder{}, err
	}
	if status != "pending" {
		return CenterOrder{}, ErrDecided
	}
	var paid *time.Time
	if err := tx.QueryRow(ctx, `SELECT paid_until FROM centers WHERE id = $1 FOR UPDATE`, centerID).Scan(&paid); err != nil {
		return CenterOrder{}, err
	}
	day := CenterPaidUntil(paid, months, now).Format(time.DateOnly)
	if _, err := tx.Exec(ctx, `INSERT INTO center_payments (center_id, amount, seats, paid_until, receipt, note, admin_tg)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, centerID, amount, seats, day, screenshot, fmt.Sprintf("заявка #%d из кабинета: %d мес.", id, months), adminTG); err != nil {
		return CenterOrder{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE centers SET seats = $2, paid_until = $3 WHERE id = $1`, centerID, seats, day); err != nil {
		return CenterOrder{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE center_orders SET status = 'confirmed', decided_by = $2, decided_at = $3 WHERE id = $1`, id, adminTG, now); err != nil {
		return CenterOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CenterOrder{}, err
	}
	return s.order(ctx, `o.id = $1`, id)
}

// RejectCenterOrder — перевод не найден: тариф не меняется, центр может отправить заявку снова.
func (s *Store) RejectCenterOrder(ctx context.Context, id, adminTG int64, reason string, now time.Time) (CenterOrder, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE center_orders SET status = 'rejected', reason = $2, decided_by = $3, decided_at = $4
		WHERE id = $1 AND status = 'pending'`, id, reason, adminTG, now)
	if err != nil {
		return CenterOrder{}, err
	}
	o, err := s.order(ctx, `o.id = $1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return o, ErrDecided
	}
	return o, err
}
