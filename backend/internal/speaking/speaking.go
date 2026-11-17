// Package speaking — проверки Speaking (этап 4): запись в приложении или голосовое боту, очередь AI.
// Устроено как Writing (writing.Queue): проверка — строка очереди, записи удаляются после проверки.
package speaking

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/ai"
)

var (
	ErrNotFound = errors.New("не найдено")
	ErrLimit    = errors.New("лимит Speaking исчерпан")
)

// Границы записи: одна запись — до 4 минут; в mock три записи.
const (
	MaxAudioBytes = 3 << 20
	MaxSeconds    = 4 * 60
)

type Question struct {
	ID   int64  `json:"id"`
	Part int    `json:"part"`
	Text string `json:"text"`
}

type Feedback struct {
	Summary  string            `json:"summary"`
	Criteria map[string]string `json:"criteria"`
	Mistakes []ai.Mistake      `json:"mistakes"`
	NextBand []string          `json:"next_band"`
	Better   string            `json:"better"`
}

type Check struct {
	ID         int64                 `json:"id"`
	MockID     *int64                `json:"mock_id"`
	Source     string                `json:"source"`
	Questions  []ai.SpeakingQuestion `json:"questions"`
	Seconds    int                   `json:"seconds"`
	Status     string                `json:"status"`
	Band       *float64              `json:"band"`
	Criteria   map[string]int        `json:"criteria"`
	Transcript string                `json:"transcript"`
	Feedback   *Feedback             `json:"feedback"`
	CreatedAt  time.Time             `json:"created_at"`
	FinishedAt *time.Time            `json:"finished_at"`
}

type NewCheck struct {
	UserID    int64
	MockID    *int64
	Source    string // app, bot
	Questions []ai.SpeakingQuestion
	Audio     []ai.Audio
	Seconds   int
	Lang      string
}

type Store struct {
	pool *pgxpool.Pool
	wake chan struct{}
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool, wake: make(chan struct{}, 1)} }

// Band — общий балл из четырёх критериев: среднее, вверх до 0.5, как Writing (writing.Band).
func Band(fc, lr, gra, pr int) float64 {
	return math.Ceil(float64(fc+lr+gra+pr)/2) / 2
}

// Random — по одному случайному вопросу каждой части; exclude — какие не повторять.
func (s *Store) Random(ctx context.Context, part int, n int) ([]Question, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, part, text FROM speaking_questions WHERE active AND part = $1 ORDER BY random() LIMIT $2`, part, n)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Question])
}

func (s *Store) Question(ctx context.Context, id int64) (Question, error) {
	var q Question
	err := s.pool.QueryRow(ctx, `SELECT id, part, text FROM speaking_questions WHERE id = $1 AND active`, id).Scan(&q.ID, &q.Part, &q.Text)
	if errors.Is(err, pgx.ErrNoRows) {
		return q, ErrNotFound
	}
	return q, err
}

// Used — сколько проверок Speaking (не из mock, не неудавшихся) с since.
func (s *Store) Used(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM speaking_checks
		WHERE user_id = $1 AND created_at >= $2 AND status <> 'failed' AND mock_id IS NULL`, userID, since).Scan(&n)
	return n, err
}

// Create ставит проверку в очередь при свободном лимите (limit с since); сверх лимита — бонусная
// проверка, если есть. Speaking из mock в лимит не идёт.
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
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM speaking_checks
			WHERE user_id = $1 AND created_at >= $2 AND status <> 'failed' AND mock_id IS NULL`, c.UserID, since).Scan(&used); err != nil {
			return 0, err
		}
		if used >= limit {
			tag, err := tx.Exec(ctx, `UPDATE users SET bonus_checks = bonus_checks - 1 WHERE id = $1 AND bonus_checks > 0`, c.UserID)
			if err != nil {
				return 0, err
			}
			if tag.RowsAffected() == 0 {
				return 0, ErrLimit
			}
		}
	}
	qs, _ := json.Marshal(c.Questions)
	var id int64
	if err := tx.QueryRow(ctx, `INSERT INTO speaking_checks (user_id, mock_id, source, questions, seconds, lang, run_after, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7) RETURNING id`, c.UserID, c.MockID, c.Source, qs, c.Seconds, c.Lang, now).Scan(&id); err != nil {
		return 0, err
	}
	for i, a := range c.Audio {
		if _, err := tx.Exec(ctx, `INSERT INTO speaking_audio (check_id, n, mime, data) VALUES ($1, $2, $3, $4)`, id, i, a.MIME, a.Data); err != nil {
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

const checkCols = `id, mock_id, source, questions, seconds, status, band::float8, fc, lr, gra, pr, transcript, feedback, created_at, finished_at`

func scanCheck(row pgx.Row) (Check, error) {
	var c Check
	var qs, fb []byte
	var fc, lr, gra, pr *int
	err := row.Scan(&c.ID, &c.MockID, &c.Source, &qs, &c.Seconds, &c.Status, &c.Band, &fc, &lr, &gra, &pr, &c.Transcript, &fb, &c.CreatedAt, &c.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	json.Unmarshal(qs, &c.Questions)
	if fc != nil && lr != nil && gra != nil && pr != nil {
		c.Criteria = map[string]int{"fc": *fc, "lr": *lr, "gra": *gra, "pr": *pr}
	}
	if fb != nil {
		c.Feedback = &Feedback{}
		if err := json.Unmarshal(fb, c.Feedback); err != nil {
			return c, err
		}
	}
	return c, nil
}

// Get — проверка ученика; чужая — ErrNotFound.
func (s *Store) Get(ctx context.Context, userID, id int64) (Check, error) {
	return scanCheck(s.pool.QueryRow(ctx, `SELECT `+checkCols+` FROM speaking_checks WHERE id = $1 AND user_id = $2`, id, userID))
}

// List — последние проверки ученика без разбора.
func (s *Store) List(ctx context.Context, userID int64, limit int) ([]Check, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+checkCols+` FROM speaking_checks WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Check{}
	for rows.Next() {
		c, err := scanCheck(rows)
		if err != nil {
			return nil, err
		}
		c.Feedback, c.Transcript = nil, ""
		list = append(list, c)
	}
	return list, rows.Err()
}

// Notifier сообщает ученику итог: band nil — оценить не удалось. OnDone — проверка закрыта (mock).
type Notifier interface {
	SpeakingChecked(ctx context.Context, telegramID int64, lang string, checkID int64, band *float64) error
}

const (
	maxAttempts = 3
	lease       = 3 * time.Minute
	callTimeout = 150 * time.Second
	pollEvery   = 5 * time.Second
)

type Queue struct {
	Store   *Store
	AI      ai.Checker
	Notify  Notifier
	OnDone  func(ctx context.Context, checkID int64)
	Log     *slog.Logger
	Workers int
	Now     func() time.Time
}

func (q *Queue) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range max(q.Workers, 1) {
		wg.Go(func() {
			for ctx.Err() == nil {
				did, err := q.Step(ctx)
				if err != nil && ctx.Err() == nil {
					q.Log.Error("очередь Speaking", "err", err)
				}
				if did && err == nil {
					continue
				}
				select {
				case <-ctx.Done():
				case <-q.Store.wake:
				case <-time.After(pollEvery):
				}
			}
		})
	}
	wg.Wait()
}

type job struct {
	id, telegramID int64
	lang           string
	questions      []ai.SpeakingQuestion
	attempts       int
}

// Step берёт одну проверку и доводит её до результата, повтора или отказа. false — заданий нет.
func (q *Queue) Step(ctx context.Context) (bool, error) {
	now := q.Now()
	var j job
	var qs []byte
	err := q.Store.pool.QueryRow(ctx, `UPDATE speaking_checks c
		SET status = 'processing', attempts = c.attempts + 1, locked_until = $2
		FROM users u
		WHERE u.id = c.user_id AND c.id = (
			SELECT id FROM speaking_checks
			WHERE (status = 'queued' AND run_after <= $1) OR (status = 'processing' AND locked_until < $1)
			ORDER BY run_after LIMIT 1
			FOR UPDATE SKIP LOCKED)
		RETURNING c.id, u.telegram_id, c.lang, c.questions, c.attempts`, now, now.Add(lease)).Scan(&j.id, &j.telegramID, &j.lang, &qs, &j.attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	json.Unmarshal(qs, &j.questions)
	if j.attempts > maxAttempts {
		return true, q.fail(ctx, j, "попытки исчерпаны")
	}
	rows, err := q.Store.pool.Query(ctx, `SELECT mime, data FROM speaking_audio WHERE check_id = $1 ORDER BY n`, j.id)
	if err != nil {
		return true, err
	}
	var audio []ai.Audio
	for rows.Next() {
		var a ai.Audio
		if err := rows.Scan(&a.MIME, &a.Data); err != nil {
			rows.Close()
			return true, err
		}
		audio = append(audio, a)
	}
	rows.Close()
	if len(audio) == 0 {
		return true, q.fail(ctx, j, "нет записей")
	}
	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	res, err := q.AI.GradeSpeaking(callCtx, ai.SpeakingRequest{Questions: j.questions, Audio: audio, Lang: j.lang})
	cancel()
	if ctx.Err() != nil {
		_, err := q.Store.pool.Exec(context.WithoutCancel(ctx), `UPDATE speaking_checks
			SET status = 'queued', attempts = attempts - 1, locked_until = NULL
			WHERE id = $1 AND status = 'processing' AND attempts = $2`, j.id, j.attempts)
		return true, err
	}
	if errors.Is(err, ai.ErrNoSpeech) {
		return true, q.fail(ctx, j, "в записи нет речи")
	}
	if err != nil {
		q.Log.Warn("проверка Speaking не удалась", "check", j.id, "attempt", j.attempts, "err", err)
		if j.attempts >= maxAttempts {
			return true, q.fail(ctx, j, err.Error())
		}
		delay := 30 * time.Second
		if j.attempts > 1 {
			delay = 2 * time.Minute
		}
		_, err := q.Store.pool.Exec(ctx, `UPDATE speaking_checks
			SET status = 'queued', run_after = $3, locked_until = NULL, last_error = $4
			WHERE id = $1 AND status = 'processing' AND attempts = $2`, j.id, j.attempts, now.Add(delay), clip(err.Error()))
		return true, err
	}
	return true, q.finish(ctx, j, res)
}

func (q *Queue) finish(ctx context.Context, j job, r ai.SpeakingResult) error {
	band := Band(r.FC.Score, r.LR.Score, r.GRA.Score, r.PR.Score)
	fb, err := json.Marshal(Feedback{
		Summary:  r.Summary,
		Criteria: map[string]string{"fc": r.FC.Comment, "lr": r.LR.Comment, "gra": r.GRA.Comment, "pr": r.PR.Comment},
		Mistakes: r.Mistakes, NextBand: r.NextBand, Better: r.Better,
	})
	if err != nil {
		return err
	}
	done, err := q.close(ctx, j, `UPDATE speaking_checks SET status = 'done', band = $3, fc = $4, lr = $5, gra = $6, pr = $7,
			transcript = $8, feedback = $9, model = $10, locked_until = NULL, last_error = '', finished_at = $11
		WHERE id = $1 AND status = 'processing' AND attempts = $2`,
		band, r.FC.Score, r.LR.Score, r.GRA.Score, r.PR.Score, r.Transcript, fb, q.AI.Model(), q.Now())
	if err != nil || !done {
		return err
	}
	q.notify(ctx, j, &band)
	return nil
}

func (q *Queue) fail(ctx context.Context, j job, reason string) error {
	done, err := q.close(ctx, j, `UPDATE speaking_checks SET status = 'failed', locked_until = NULL, last_error = $3, finished_at = $4
		WHERE id = $1 AND status = 'processing' AND attempts = $2`, clip(reason), q.Now())
	if err != nil || !done {
		return err
	}
	q.notify(ctx, j, nil)
	return nil
}

// close записывает итог и удаляет записи. false — задание уже забрал другой воркер.
func (q *Queue) close(ctx context.Context, j job, sql string, args ...any) (bool, error) {
	tx, err := q.Store.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, sql, append([]any{j.id, j.attempts}, args...)...)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM speaking_audio WHERE check_id = $1`, j.id); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (q *Queue) notify(ctx context.Context, j job, band *float64) {
	if q.OnDone != nil {
		q.OnDone(ctx, j.id)
	}
	if q.Notify == nil {
		return
	}
	if err := q.Notify.SpeakingChecked(ctx, j.telegramID, j.lang, j.id, band); err != nil {
		q.Log.Warn("уведомление о Speaking не отправлено", "check", j.id, "err", err)
	}
}

func clip(s string) string {
	if len(s) > 500 {
		return strings.ToValidUTF8(s[:500], "")
	}
	return s
}
