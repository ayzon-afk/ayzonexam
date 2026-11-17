// Package support — обращения в поддержку (этап 17): из приложения и из бота (/help), ответы — из
// админки или ответом на сообщение бота в Telegram.
package support

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("не найдено")
	ErrEmpty    = errors.New("пустое сообщение")
)

const MaxText = 4000

type Message struct {
	ID        int64     `json:"id"`
	FromAdmin bool      `json:"from_admin"`
	Text      string    `json:"text"`
	Photo     *int64    `json:"photo"`
	CreatedAt time.Time `json:"created_at"`
}

type Ticket struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Status    string    `json:"status"` // open, answered, closed
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
	// Для админки: кто пишет, его тариф и центр (SPEC, «Поддержка»).
	Nickname   string `json:"nickname"`
	TelegramID int64  `json:"telegram_id"`
	Lang       string `json:"lang"`
	Center     string `json:"center"`
	Premium    bool   `json:"premium"`
	Last       string `json:"last"`
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Write — сообщение ученика: в открытое обращение (не закрытое) или в новое. Возвращает id обращения.
func (s *Store) Write(ctx context.Context, userID int64, text string, photo *int64) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" && photo == nil {
		return 0, ErrEmpty
	}
	if utf8.RuneCountInString(text) > MaxText {
		text = string([]rune(text)[:MaxText])
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return 0, err
	}
	var id int64
	err = tx.QueryRow(ctx, `SELECT id FROM tickets WHERE user_id = $1 AND status <> 'closed' ORDER BY id DESC LIMIT 1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO tickets (user_id) VALUES ($1) RETURNING id`, userID).Scan(&id)
	}
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ticket_messages (ticket_id, text, media_id) VALUES ($1, $2, $3)`, id, text, photo); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE tickets SET status = 'open', updated_at = now() WHERE id = $1`, id); err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}

// Reply — ответ админа. Возвращает, кому и на каком языке его переслать ботом.
func (s *Store) Reply(ctx context.Context, ticketID, adminTG int64, text string) (tgID int64, lang string, err error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, "", ErrEmpty
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `UPDATE tickets t SET status = 'answered', updated_at = now() FROM users u
		WHERE t.id = $1 AND u.id = t.user_id RETURNING u.telegram_id, u.lang`, ticketID).Scan(&tgID, &lang)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	if err != nil {
		return 0, "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ticket_messages (ticket_id, admin_tg, text) VALUES ($1, $2, $3)`, ticketID, adminTG, text); err != nil {
		return 0, "", err
	}
	return tgID, lang, tx.Commit(ctx)
}

func (s *Store) SetStatus(ctx context.Context, id int64, status string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE tickets SET status = $2, updated_at = now() WHERE id = $1`, id, status)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) messages(ctx context.Context, ticketID int64) ([]Message, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, admin_tg IS NOT NULL, text, media_id, created_at FROM ticket_messages
		WHERE ticket_id = $1 ORDER BY id`, ticketID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Message])
}

const ticketCols = `t.id, t.user_id, t.status, t.created_at, t.updated_at, u.nickname, u.telegram_id, u.lang, coalesce(c.name, ''),
	(coalesce(u.premium_until > now(), false) OR coalesce(c.paid_until >= current_date, false)),
	coalesce((SELECT text FROM ticket_messages m WHERE m.ticket_id = t.id ORDER BY m.id DESC LIMIT 1), '')`

const ticketFrom = ` FROM tickets t JOIN users u ON u.id = t.user_id
	LEFT JOIN centers c ON c.id = u.center_id AND u.center_status = 'member'`

func scanTicket(row pgx.Row) (Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.UserID, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.Nickname, &t.TelegramID, &t.Lang, &t.Center, &t.Premium, &t.Last)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

// Get — обращение с перепиской; userID 0 — любое (админка), иначе только своё.
func (s *Store) Get(ctx context.Context, id, userID int64) (Ticket, error) {
	t, err := scanTicket(s.pool.QueryRow(ctx, `SELECT `+ticketCols+ticketFrom+` WHERE t.id = $1 AND ($2 = 0 OR t.user_id = $2)`, id, userID))
	if err != nil {
		return t, err
	}
	t.Messages, err = s.messages(ctx, id)
	return t, err
}

// List — обращения: ученика (userID ≠ 0) или все со статусом status (” — любые) для админки.
func (s *Store) List(ctx context.Context, userID int64, status string, limit int) ([]Ticket, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+ticketCols+ticketFrom+`
		WHERE ($1 = 0 OR t.user_id = $1) AND ($2 = '' OR t.status = $2) ORDER BY t.updated_at DESC LIMIT $3`, userID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// SaveRef — сообщение бота у админа (chat, message) относится к обращению или оплате.
func (s *Store) SaveRef(ctx context.Context, chatID, messageID int64, kind string, refID int64) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO bot_refs (chat_id, message_id, kind, ref_id) VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`, chatID, messageID, kind, refID)
	return err
}

// Ref — к чему относится сообщение бота, на которое админ ответил.
func (s *Store) Ref(ctx context.Context, chatID, messageID int64) (kind string, refID int64, err error) {
	err = s.pool.QueryRow(ctx, `SELECT kind, ref_id FROM bot_refs WHERE chat_id = $1 AND message_id = $2`, chatID, messageID).Scan(&kind, &refID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, ErrNotFound
	}
	return kind, refID, err
}
