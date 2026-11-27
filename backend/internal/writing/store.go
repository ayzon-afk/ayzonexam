package writing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/ai"
)

var (
	ErrNotFound   = errors.New("не найдено")
	ErrDailyLimit = errors.New("лимит проверок исчерпан")
)

// Статусы проверки.
const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

type Prompt struct {
	ID   int64  `json:"id"`
	Task string `json:"task"`
	Text string `json:"text"`
}

// Feedback — разбор, который видит ученик. Хранится в writing_checks.feedback.
type Feedback struct {
	Summary   string       `json:"summary"`
	Criteria  Comments     `json:"criteria"`
	Mistakes  []ai.Mistake `json:"mistakes"`
	Corrected string       `json:"corrected"`
	NextBand  []string     `json:"next_band"`
}

type Comments struct {
	TA  string `json:"ta"`
	CC  string `json:"cc"`
	LR  string `json:"lr"`
	GRA string `json:"gra"`
}

type Check struct {
	ID         int64
	UserID     int64
	Task       string
	PromptID   *int64
	Prompt     string
	Essay      string // пусто в списке истории
	Words      int
	Source     string
	Lang       string
	Status     string
	Band       *float64
	TA, CC     *int
	LR, GRA    *int
	Feedback   *Feedback
	CreatedAt  time.Time
	FinishedAt *time.Time
	MockID     *int64
	Chart      json.RawMessage // график Task 1 из банка (writing.Chart); nil — нет
}

type NewCheck struct {
	UserID   int64
	Task     string
	PromptID *int64
	Prompt   string
	Essay    string
	Words    int
	Source   string
	Lang     string
	Image    *ai.Image // фото задания Task 1
	MockID   *int64    // Writing полного mock: в лимит не идёт, он оплачен самим mock
	Chart    []byte    // график Task 1 данными (Chart) — вместо фото
}

type Store struct {
	pool *pgxpool.Pool
	wake chan struct{}
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, wake: make(chan struct{}, 1)}
}

// CreateScored — проверка с готовым баллом без AI: в mock пустой ответ (0) и ответ до 20 слов (Band 1) —
// как в критериях IELTS; summary — пояснение ученику.
func (s *Store) CreateScored(ctx context.Context, c NewCheck, band int, summary string, now time.Time) (int64, error) {
	fb, _ := json.Marshal(Feedback{Summary: summary, Criteria: Comments{TA: summary, CC: summary, LR: summary, GRA: summary},
		Mistakes: []ai.Mistake{}, NextBand: []string{}})
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO writing_checks
		(user_id, task, prompt_id, prompt, essay, words, source, lang, run_after, created_at, mock_id, chart,
		 status, band, ta, cc, lr, gra, feedback, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $10, $11, 'done', $12::int, $12::int, $12::int, $12::int, $12::int, $13, $9) RETURNING id`,
		c.UserID, c.Task, c.PromptID, c.Prompt, c.Essay, c.Words, c.Source, c.Lang, now, c.MockID, c.Chart, band, fb).Scan(&id)
	return id, err
}

// Prompts — активные темы банка.
func (s *Store) Prompts(ctx context.Context, task string) ([]Prompt, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, task, text FROM writing_prompts WHERE task = $1 AND active ORDER BY id`, task)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Prompt])
}

func (s *Store) Prompt(ctx context.Context, id int64) (Prompt, error) {
	var p Prompt
	err := s.pool.QueryRow(ctx, `SELECT id, task, text FROM writing_prompts WHERE id = $1 AND active`, id).Scan(&p.ID, &p.Task, &p.Text)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// Create ставит проверку в очередь, если у ученика не исчерпан лимит: limit проверок с since (полночь по
// Ташкенту или неделя назад — по тарифу, billing). Сверх лимита тратится бонусная проверка, если есть.
// Неудавшиеся проверки и Writing из mock в лимит не идут. Строка ученика блокируется на время
// транзакции: две одновременные отправки не обойдут лимит.
func (s *Store) Create(ctx context.Context, c NewCheck, limit int, since, now time.Time) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, c.UserID); err != nil {
		return 0, err
	}
	if c.MockID == nil {
		var used int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM writing_checks
			WHERE user_id = $1 AND created_at >= $2 AND status <> 'failed' AND mock_id IS NULL`, c.UserID, since).Scan(&used); err != nil {
			return 0, err
		}
		if used >= limit {
			tag, err := tx.Exec(ctx, `UPDATE users SET bonus_checks = bonus_checks - 1 WHERE id = $1 AND bonus_checks > 0`, c.UserID)
			if err != nil {
				return 0, err
			}
			if tag.RowsAffected() == 0 {
				return 0, ErrDailyLimit
			}
		}
	}
	var id int64
	if err := tx.QueryRow(ctx, `INSERT INTO writing_checks
		(user_id, task, prompt_id, prompt, essay, words, source, lang, run_after, created_at, mock_id, chart)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9, $10, $11) RETURNING id`,
		c.UserID, c.Task, c.PromptID, c.Prompt, c.Essay, c.Words, c.Source, c.Lang, now, c.MockID, c.Chart).Scan(&id); err != nil {
		return 0, err
	}
	if c.Image != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO writing_images (check_id, mime, data) VALUES ($1, $2, $3)`,
			id, c.Image.MIME, c.Image.Data); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return id, nil
}

// Used — сколько проверок ученик потратил с since (для экрана «осталось N»); Writing из mock не в счёт.
func (s *Store) Used(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM writing_checks
		WHERE user_id = $1 AND created_at >= $2 AND status <> 'failed' AND mock_id IS NULL`, userID, since).Scan(&n)
	return n, err
}

const checkCols = `id, user_id, task, prompt_id, prompt, words, source, lang, status, band, ta, cc, lr, gra, created_at, finished_at, mock_id`

func scanCheck(row pgx.Row, extra ...any) (Check, error) {
	var c Check
	err := row.Scan(append([]any{&c.ID, &c.UserID, &c.Task, &c.PromptID, &c.Prompt, &c.Words, &c.Source, &c.Lang,
		&c.Status, &c.Band, &c.TA, &c.CC, &c.LR, &c.GRA, &c.CreatedAt, &c.FinishedAt, &c.MockID}, extra...)...)
	return c, err
}

// Get — проверка ученика; чужая — ErrNotFound, как и несуществующая.
func (s *Store) Get(ctx context.Context, userID, id int64) (Check, error) {
	var essay string
	var feedback []byte
	var chart []byte
	c, err := scanCheck(s.pool.QueryRow(ctx, `SELECT `+checkCols+`, essay, feedback, chart FROM writing_checks
		WHERE id = $1 AND user_id = $2`, id, userID), &essay, &feedback, &chart)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	c.Essay, c.Chart = essay, chart
	if feedback != nil {
		c.Feedback = &Feedback{}
		if err := json.Unmarshal(feedback, c.Feedback); err != nil {
			return c, err
		}
	}
	return c, nil
}

// List — последние проверки ученика без текста эссе и разбора.
func (s *Store) List(ctx context.Context, userID int64, limit int) ([]Check, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+checkCols+` FROM writing_checks
		WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Check
	for rows.Next() {
		c, err := scanCheck(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// TakeUsage записывает платный вызов AI (kind), если с since их было меньше limit. false — лимит исчерпан.
func (s *Store) TakeUsage(ctx context.Context, userID int64, kind string, limit int, since, now time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `INSERT INTO ai_usage (user_id, kind, created_at)
		SELECT $1, $2, $3 WHERE (SELECT count(*) FROM ai_usage WHERE user_id = $1 AND kind = $2 AND created_at >= $4) < $5`,
		userID, kind, now, since, limit)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ReturnUsage отменяет последнюю запись kind — вызов AI не удался, ученик не должен за него платить лимитом.
func (s *Store) ReturnUsage(ctx context.Context, userID int64, kind string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM ai_usage WHERE ctid = (SELECT ctid FROM ai_usage
		WHERE user_id = $1 AND kind = $2 ORDER BY created_at DESC LIMIT 1)`, userID, kind)
	return err
}
