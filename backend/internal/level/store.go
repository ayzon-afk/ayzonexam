package level

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("тест уровня не найден")
	ErrFinished = errors.New("тест уровня уже сдан")
	ErrStage    = errors.New("не тот этап теста")
)

// Resume — сколько незаконченный тест ждёт ученика: вернулся в течение этого времени — продолжает.
const Resume = 3 * time.Hour

// Step — заданный вопрос и ответ ученика (-1 — без ответа).
type Step struct {
	ID string `json:"id"`
	A  int    `json:"a"`
	OK bool   `json:"ok"`
}

// Run — ход теста уровня.
type Run struct {
	AttemptID  int64
	UserID     int64
	StartedAt  time.Time
	FinishedAt *time.Time
	Items      []Step
	Current    string
	Passage    string
	Reading    []int
	Band       float64
	Low, High  float64
}

// Responses — ответы для оценки: вопросы банка и текст Reading.
func (r Run) Responses() []Response {
	var rs []Response
	for _, s := range r.Items {
		if it, ok := ItemByID(s.ID); ok {
			rs = append(rs, Response{it.D, len(it.Options), s.OK})
		}
	}
	if p, ok := PassageByID(r.Passage); ok {
		for i, q := range p.Questions {
			rs = append(rs, Response{q.D, len(q.Options), i < len(r.Reading) && r.Reading[i] == q.Answer})
		}
	}
	return rs
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// rnd — свой генератор на вызов: запросы идут параллельно, а *rand.Rand не для общего доступа.
func rnd() *rand.Rand { return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) }

const runCols = `r.attempt_id, r.user_id, a.started_at, a.finished_at, r.items, r.current, r.passage, r.reading,
	coalesce(a.band, 0)::float8, coalesce(r.low, 0)::float8, coalesce(r.high, 0)::float8`

func scanRun(row pgx.Row) (Run, error) {
	var r Run
	var items, reading []byte
	err := row.Scan(&r.AttemptID, &r.UserID, &r.StartedAt, &r.FinishedAt, &items, &r.Current, &r.Passage, &reading,
		&r.Band, &r.Low, &r.High)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	if reading != nil {
		if err := json.Unmarshal(reading, &r.Reading); err != nil {
			return r, err
		}
	}
	return r, json.Unmarshal(items, &r.Items)
}

// Get — тест уровня ученика по попытке; чужой — ErrNotFound.
func (s *Store) Get(ctx context.Context, userID, attemptID int64) (Run, error) {
	return scanRun(s.pool.QueryRow(ctx, `SELECT `+runCols+` FROM level_runs r JOIN attempts a ON a.id = r.attempt_id
		WHERE r.attempt_id = $1 AND r.user_id = $2`, attemptID, userID))
}

// Start — незаконченный тест ученика за последние Resume или новый с первым вопросом.
func (s *Store) Start(ctx context.Context, userID int64, now time.Time) (Run, error) {
	r, err := scanRun(s.pool.QueryRow(ctx, `SELECT `+runCols+` FROM level_runs r JOIN attempts a ON a.id = r.attempt_id
		WHERE r.user_id = $1 AND a.finished_at IS NULL AND a.started_at > $2 ORDER BY a.started_at DESC LIMIT 1`,
		userID, now.Add(-Resume)))
	if !errors.Is(err, ErrNotFound) {
		return r, err
	}
	est, _ := Estimate(nil)
	first, _ := Next(map[string]bool{}, "", est, rnd())
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return r, err
	}
	defer tx.Rollback(ctx)
	r = Run{UserID: userID, StartedAt: now, Items: []Step{}, Current: first.ID}
	err = tx.QueryRow(ctx, `INSERT INTO attempts (user_id, test_id, kind, started_at)
		SELECT $1, id, 'level', $2 FROM tests WHERE source = 'level-adaptive' RETURNING id`, userID, now).Scan(&r.AttemptID)
	if err != nil {
		return r, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO level_runs (attempt_id, user_id, current) VALUES ($1, $2, $3)`,
		r.AttemptID, userID, r.Current); err != nil {
		return r, err
	}
	return r, tx.Commit(ctx)
}

// lock — ход теста под блокировкой, незаконченный.
func lock(ctx context.Context, tx pgx.Tx, userID, attemptID int64) (Run, error) {
	r, err := scanRun(tx.QueryRow(ctx, `SELECT `+runCols+` FROM level_runs r JOIN attempts a ON a.id = r.attempt_id
		WHERE r.attempt_id = $1 AND r.user_id = $2 FOR UPDATE OF r`, attemptID, userID))
	if err == nil && r.FinishedAt != nil {
		return r, ErrFinished
	}
	return r, err
}

// Answer — ответ на вопрос itemID (номер варианта, -1 — пропуск). Ответ на вопрос, которого уже нет на экране
// (повторная отправка), ничего не меняет. После CoreSteps вопросов — текст Reading под оценку.
func (s *Store) Answer(ctx context.Context, userID, attemptID int64, itemID string, answer int) (Run, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	defer tx.Rollback(ctx)
	r, err := lock(ctx, tx, userID, attemptID)
	if err != nil || itemID != r.Current || r.Current == "" {
		return r, err
	}
	it, ok := ItemByID(r.Current)
	if !ok {
		return r, ErrStage
	}
	if answer < -1 || answer >= len(it.Options) {
		answer = -1
	}
	r.Items = append(r.Items, Step{ID: it.ID, A: answer, OK: answer == it.Answer})
	est, _ := Estimate(r.Responses())
	used := map[string]bool{}
	for _, st := range r.Items {
		used[st.ID] = true
	}
	r.Current = ""
	if len(r.Items) < CoreSteps {
		if next, ok := Next(used, it.Type, est, rnd()); ok {
			r.Current = next.ID
		}
	}
	if r.Current == "" {
		r.Passage = PassageFor(est).ID
	}
	js, _ := json.Marshal(r.Items)
	if _, err := tx.Exec(ctx, `UPDATE level_runs SET items = $2, current = $3, passage = $4 WHERE attempt_id = $1`,
		r.AttemptID, js, r.Current, r.Passage); err != nil {
		return r, err
	}
	return r, tx.Commit(ctx)
}

// Finish — ответы к тексту Reading: итог теста. Попытка закрывается с баллом; в рейтинг тест уровня не идёт.
func (s *Store) Finish(ctx context.Context, userID, attemptID int64, answers []int, now time.Time) (Run, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Run{}, err
	}
	defer tx.Rollback(ctx)
	r, err := lock(ctx, tx, userID, attemptID)
	if err != nil {
		return r, err
	}
	p, ok := PassageByID(r.Passage)
	if r.Current != "" || !ok || len(answers) > len(p.Questions) {
		return r, ErrStage
	}
	r.Reading = make([]int, len(p.Questions))
	correct := 0
	for i, q := range p.Questions {
		r.Reading[i] = -1
		if i < len(answers) && answers[i] >= 0 && answers[i] < len(q.Options) {
			r.Reading[i] = answers[i]
		}
		if r.Reading[i] == q.Answer {
			correct++
		}
	}
	for _, st := range r.Items {
		if st.OK {
			correct++
		}
	}
	res := Score(r.Responses())
	r.Band, r.Low, r.High, r.FinishedAt = res.Band, res.Low, res.High, &now
	js, _ := json.Marshal(r.Reading)
	if _, err := tx.Exec(ctx, `UPDATE level_runs SET reading = $2, low = $3, high = $4 WHERE attempt_id = $1`,
		r.AttemptID, js, r.Low, r.High); err != nil {
		return r, err
	}
	if _, err := tx.Exec(ctx, `UPDATE attempts SET finished_at = $2, correct = $3, total = $4, band = $5, ranked = false
		WHERE id = $1`, r.AttemptID, now, correct, len(r.Items)+len(p.Questions), r.Band); err != nil {
		return r, err
	}
	return r, tx.Commit(ctx)
}
