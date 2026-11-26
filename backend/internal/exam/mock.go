package exam

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Полный mock по частям (этап 18): Listening и Reading — обычные попытки с mock_id, Writing — две
// проверки Writing с mock_id, Speaking — проверка Speaking с mock_id. Когда у всех четырёх частей есть
// балл, mock закрывается с Overall Band.

// Время частей mock, минут.
const (
	WritingMinutes  = 60
	SpeakingMinutes = 20 // 11–14 минут разговора и запас на запись и отправку
)

var (
	ErrMockActive = errors.New("уже есть незаконченный mock")
	ErrPartTaken  = errors.New("часть уже сдана")
)

type Mock struct {
	ID                int64
	UserID            int64
	Status            string
	PaidWith          string
	Listening         *float64
	Reading           *float64
	Writing           *float64
	Speaking          *float64
	Overall           *float64
	WritingStartedAt  *time.Time
	Task1PromptID     *int64
	Task2PromptID     *int64
	WritingAt         *time.Time
	SpeakingStartedAt *time.Time
	SpeakingQuestions []byte // jsonb: вопросы Speaking mock
	SpeakingAt        *time.Time
	StartedAt         time.Time
	FinishedAt        *time.Time
	// Нарушения честного режима (камера, уход, полный экран, вставка): с MaxLeaves часть не в рейтинге.
	WritingViolations  int
	SpeakingViolations int
}

const mockCols = `id, user_id, status, paid_with, listening_band::float8, reading_band::float8, writing_band::float8,
	speaking_band::float8, overall::float8, writing_started_at, task1_prompt_id, task2_prompt_id, writing_at,
	speaking_started_at, speaking_questions, speaking_at, started_at, finished_at, writing_violations, speaking_violations`

func scanMock(row pgx.Row) (Mock, error) {
	var m Mock
	err := row.Scan(&m.ID, &m.UserID, &m.Status, &m.PaidWith, &m.Listening, &m.Reading, &m.Writing, &m.Speaking, &m.Overall,
		&m.WritingStartedAt, &m.Task1PromptID, &m.Task2PromptID, &m.WritingAt, &m.SpeakingStartedAt, &m.SpeakingQuestions, &m.SpeakingAt,
		&m.StartedAt, &m.FinishedAt, &m.WritingViolations, &m.SpeakingViolations)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

// ActiveMock — незаконченный mock ученика; ErrNotFound — нет.
func (s *Store) ActiveMock(ctx context.Context, userID int64) (Mock, error) {
	return scanMock(s.pool.QueryRow(ctx, `SELECT `+mockCols+` FROM mocks WHERE user_id = $1 AND status = 'active'`, userID))
}

// Mock — mock ученика; чужой — ErrNotFound.
func (s *Store) Mock(ctx context.Context, userID, id int64) (Mock, error) {
	return scanMock(s.pool.QueryRow(ctx, `SELECT `+mockCols+` FROM mocks WHERE id = $1 AND user_id = $2`, id, userID))
}

// Mocks — mock ученика, новые первыми.
func (s *Store) Mocks(ctx context.Context, userID int64, limit int) ([]Mock, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+mockCols+` FROM mocks WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Mock
	for rows.Next() {
		m, err := scanMock(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// CreateMock — новый mock. paidWith: premium, credit (тогда списывается купленный mock), trial.
// Оплату решает вызывающий (billing); здесь — только запись и списание купленного.
func (s *Store) CreateMock(ctx context.Context, userID int64, paidWith string, now time.Time) (Mock, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Mock{}, err
	}
	defer tx.Rollback(ctx)
	if paidWith == "credit" {
		tag, err := tx.Exec(ctx, `UPDATE users SET mock_credits = mock_credits - 1 WHERE id = $1 AND mock_credits > 0`, userID)
		if err != nil {
			return Mock{}, err
		}
		if tag.RowsAffected() == 0 {
			return Mock{}, errors.New("нет купленного mock")
		}
	}
	m, err := scanMock(tx.QueryRow(ctx, `INSERT INTO mocks (user_id, paid_with, started_at) VALUES ($1, $2, $3) RETURNING `+mockCols,
		userID, paidWith, now))
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return m, ErrMockActive
	}
	if err != nil {
		return m, err
	}
	return m, tx.Commit(ctx)
}

// PartAttempt — попытка части Listening или Reading этого mock: сданная или идущая; nil — не начата.
func (s *Store) PartAttempt(ctx context.Context, mockID int64, kind string) (*Attempt, error) {
	a, err := scanAttempt(s.pool.QueryRow(ctx, `SELECT `+attemptCols+` FROM attempts a JOIN tests t ON t.id = a.test_id
		WHERE a.mock_id = $1 AND a.kind = $2 ORDER BY a.started_at DESC LIMIT 1`, mockID, kind))
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return &a, err
}

// StartWriting — таймер Writing пошёл; темы закрепляются за mock: Task 2 — из банка, Task 1 — из банка
// тем с графиком (нет таких — mock без Task 1). Повторный вызов отдаёт уже закреплённые.
func (s *Store) StartWriting(ctx context.Context, m Mock, now time.Time) (Mock, error) {
	if m.WritingStartedAt != nil {
		return m, nil
	}
	return scanMock(s.pool.QueryRow(ctx, `UPDATE mocks SET writing_started_at = $2,
			task1_prompt_id = (SELECT id FROM writing_prompts WHERE task = 'task1' AND active AND image_id IS NOT NULL ORDER BY random() LIMIT 1),
			task2_prompt_id = (SELECT id FROM writing_prompts WHERE task = 'task2' AND active ORDER BY random() LIMIT 1)
		WHERE id = $1 AND writing_started_at IS NULL RETURNING `+mockCols, m.ID, now))
}

// StartSpeaking — таймер Speaking пошёл; вопросы (по одному на часть) закрепляются за mock.
func (s *Store) StartSpeaking(ctx context.Context, m Mock, now time.Time) (Mock, error) {
	if m.SpeakingStartedAt != nil {
		return m, nil
	}
	return scanMock(s.pool.QueryRow(ctx, `UPDATE mocks SET speaking_started_at = $2, speaking_questions = (
			SELECT jsonb_agg(jsonb_build_object('part', part, 'text', text) ORDER BY part) FROM (
				SELECT DISTINCT ON (part) part, text FROM speaking_questions WHERE active ORDER BY part, random()) q)
		WHERE id = $1 RETURNING `+mockCols, m.ID, now))
}

// WritingBand — балл части Writing из баллов заданий: Task 2 весит вдвое больше Task 1, как на экзамене;
// без Task 1 — балл Task 2. Округление до 0.5 — как Overall.
func WritingBand(task1 *float64, task2 float64) float64 {
	if task1 == nil {
		return task2
	}
	return math.Floor((*task1+2*task2)/3*2+0.5) / 2
}

// UpdateMock пересчитывает баллы частей по сданным попыткам и проверкам и, когда есть все четыре,
// закрывает mock с Overall Band. true — mock только что закрылся.
func (s *Store) UpdateMock(ctx context.Context, mockID int64, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	m, err := scanMock(tx.QueryRow(ctx, `SELECT `+mockCols+` FROM mocks WHERE id = $1 FOR UPDATE`, mockID))
	if err != nil || m.Status == "done" {
		return false, err
	}
	for _, kind := range []string{Listening, Reading} {
		var band *float64
		err := tx.QueryRow(ctx, `SELECT band::float8 FROM attempts WHERE mock_id = $1 AND kind = $2 AND finished_at IS NOT NULL
			ORDER BY finished_at LIMIT 1`, mockID, kind).Scan(&band)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
		if kind == Listening {
			m.Listening = band
		} else {
			m.Reading = band
		}
	}
	ranked := false
	if m.Writing == nil && m.Task2PromptID != nil {
		var t1, t2 *float64
		var last, first *time.Time
		var pending int
		err := tx.QueryRow(ctx, `SELECT
				(SELECT band::float8 FROM writing_checks WHERE mock_id = $1 AND task = 'task1' AND status = 'done' ORDER BY id DESC LIMIT 1),
				(SELECT band::float8 FROM writing_checks WHERE mock_id = $1 AND task = 'task2' AND status = 'done' ORDER BY id DESC LIMIT 1),
				(SELECT count(*) FROM writing_checks WHERE mock_id = $1 AND status IN ('queued', 'processing')),
				(SELECT min(created_at) FROM writing_checks WHERE mock_id = $1),
				(SELECT max(finished_at) FROM writing_checks WHERE mock_id = $1 AND status = 'done')`, mockID).
			Scan(&t1, &t2, &pending, &first, &last)
		if err != nil {
			return false, err
		}
		needT1 := m.Task1PromptID != nil
		if t2 != nil && (!needT1 || t1 != nil) && pending == 0 {
			b := WritingBand(t1, *t2)
			m.Writing, m.WritingAt = &b, last
			ranked = m.WritingStartedAt != nil && first != nil &&
				!first.After(m.WritingStartedAt.Add(WritingMinutes*time.Minute+Grace)) && m.WritingViolations < MaxLeaves
			if _, err := tx.Exec(ctx, `UPDATE mocks SET writing_band = $2, writing_at = $3, writing_ranked = $4 WHERE id = $1`,
				mockID, b, last, ranked); err != nil {
				return false, err
			}
		}
	}
	if m.Speaking == nil {
		var band *float64
		var at, created *time.Time
		err := tx.QueryRow(ctx, `SELECT band::float8, finished_at, created_at FROM speaking_checks
			WHERE mock_id = $1 AND status = 'done' ORDER BY id DESC LIMIT 1`, mockID).Scan(&band, &at, &created)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
		if band != nil {
			m.Speaking = band
			sr := m.SpeakingStartedAt != nil && created != nil && !created.After(m.SpeakingStartedAt.Add(SpeakingMinutes*time.Minute+Grace)) &&
				m.SpeakingViolations < MaxLeaves
			if _, err := tx.Exec(ctx, `UPDATE mocks SET speaking_band = $2, speaking_at = $3, speaking_ranked = $4 WHERE id = $1`,
				mockID, *band, at, sr); err != nil {
				return false, err
			}
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE mocks SET listening_band = $2, reading_band = $3 WHERE id = $1`, mockID, m.Listening, m.Reading); err != nil {
		return false, err
	}
	closed := false
	if m.Listening != nil && m.Reading != nil && m.Writing != nil && m.Speaking != nil {
		o := Overall(*m.Listening, *m.Reading, *m.Writing, *m.Speaking)
		if _, err := tx.Exec(ctx, `UPDATE mocks SET overall = $2, status = 'done', finished_at = $3 WHERE id = $1`, mockID, o, now); err != nil {
			return false, err
		}
		closed = true
	}
	return closed, tx.Commit(ctx)
}

// MockOfCheck — к какому mock относится проверка Writing или Speaking (очереди сообщают, что она готова).
func (s *Store) MockOfCheck(ctx context.Context, table string, checkID int64) (*int64, error) {
	var id *int64
	err := s.pool.QueryRow(ctx, `SELECT mock_id FROM `+pgx.Identifier{table}.Sanitize()+` WHERE id = $1`, checkID).Scan(&id)
	return id, err
}

// MockAttempts — сданные Listening и Reading mock вместе с заданиями (для слабых мест в отчёте).
func (s *Store) MockAttempts(ctx context.Context, userID, mockID int64) ([]Done, error) {
	rows, err := s.pool.Query(ctx, `SELECT a.id FROM attempts a WHERE a.mock_id = $1 AND a.user_id = $2 AND a.finished_at IS NOT NULL`, mockID, userID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}
	var out []Done
	for _, id := range ids {
		a, t, err := s.Attempt(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, Done{a, t})
	}
	return out, nil
}
