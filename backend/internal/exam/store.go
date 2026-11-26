package exam

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("не найдено")
	ErrFinished = errors.New("попытка уже сдана")
	ErrAnswers  = errors.New("ответы не по заданию")
)

// Grace — сколько после таймера ещё принимается сдача в рейтинг: медленный интернет и автосдача.
const Grace = 90 * time.Second

// MaxLeaves — с этого числа уходов из приложения попытка не идёт в рейтинг.
const MaxLeaves = 3

type Question struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
	Type    string   `json:"type"`
	Kind    string   `json:"kind,omitempty"`   // вид: KindChoice ("" — тоже), KindText, KindMulti
	Accept  []string `json:"accept,omitempty"` // KindText — допустимые ответы
	Set     []int    `json:"set,omitempty"`    // KindMulti — верные буквы группы
	Group   int      `json:"group,omitempty"`  // номер группы в Test.Groups с 1; 0 — без группы
}

type Test struct {
	ID         int64
	Kind       string
	Title      string
	FullSize   bool
	Minutes    int
	Passage    string
	Transcript string
	AudioID    *int64
	Questions  []Question
	Groups     []Group
	Source     string // тест канала (V10-T02-L) — повторная загрузка заменяет его
	Active     bool
}

// Summary — строка списка заданий: без текста и вопросов.
type Summary struct {
	ID       int64    `json:"id"`
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	Minutes  int      `json:"minutes"`
	Count    int      `json:"count"`
	FullSize bool     `json:"full_size"`
	LastBand *float64 `json:"last_band"`
}

type Attempt struct {
	ID           int64
	UserID       int64
	TestID       int64
	Kind         string
	Title        string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Answers      []Answer
	Correct      int
	Total        int
	Band         float64
	Leaves       int
	Ranked       bool
	MockID       *int64
	TournamentID *int64
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// visible — задание можно дать ученику: включено, а Listening — с аудио.
const visible = `t.active AND (t.kind <> 'listening' OR t.audio_id IS NOT NULL)`

// List — задания вида kind и последний балл ученика по каждому.
func (s *Store) List(ctx context.Context, kind string, userID int64) ([]Summary, error) {
	rows, err := s.pool.Query(ctx, `SELECT t.id, t.kind, t.title, t.minutes, jsonb_array_length(t.questions), t.full_size,
			(SELECT band FROM attempts a WHERE a.test_id = t.id AND a.user_id = $2 AND a.finished_at IS NOT NULL
			 ORDER BY a.finished_at DESC LIMIT 1)
		FROM tests t WHERE t.kind = $1 AND `+visible+` ORDER BY t.full_size, t.id`, kind, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Summary])
}

const testCols = `id, kind, title, full_size, minutes, passage, transcript, audio_id, questions, groups, source, active`

func scanTest(row pgx.Row) (Test, error) {
	var t Test
	var qs, gs []byte
	err := row.Scan(&t.ID, &t.Kind, &t.Title, &t.FullSize, &t.Minutes, &t.Passage, &t.Transcript, &t.AudioID, &qs, &gs, &t.Source, &t.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(gs, &t.Groups); err != nil {
		return t, err
	}
	return t, json.Unmarshal(qs, &t.Questions)
}

func (s *Store) Test(ctx context.Context, id int64) (Test, error) {
	return scanTest(s.pool.QueryRow(ctx, `SELECT `+testCols+` FROM tests WHERE id = $1`, id))
}

// Pick — задание вида kind для mock или турнира: полноразмерное, если есть, иначе любое; среди них —
// то, которое ученик решал реже всего (userID 0 — случайное).
func (s *Store) Pick(ctx context.Context, kind string, userID int64) (Test, error) {
	return scanTest(s.pool.QueryRow(ctx, `SELECT `+testCols+` FROM tests t WHERE t.kind = $1 AND `+visible+`
		ORDER BY t.full_size DESC,
			(SELECT count(*) FROM attempts a WHERE a.test_id = t.id AND a.user_id = $2),
			random()
		LIMIT 1`, kind, userID))
}

// Start — ученик начал задание: с этой минуты идёт таймер. mockID и tournamentID — в чём попытка.
func (s *Store) Start(ctx context.Context, userID, testID int64, mockID, tournamentID *int64, now time.Time) (int64, Test, error) {
	t, err := scanTest(s.pool.QueryRow(ctx, `SELECT `+testCols+` FROM tests t WHERE t.id = $1 AND `+visible, testID))
	if err != nil {
		return 0, t, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `INSERT INTO attempts (user_id, test_id, kind, started_at, mock_id, tournament_id)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, userID, t.ID, t.Kind, now, mockID, tournamentID).Scan(&id)
	return id, t, err
}

const attemptCols = `a.id, a.user_id, a.test_id, a.kind, t.title, a.started_at, a.finished_at, a.answers,
	coalesce(a.correct, 0), coalesce(a.total, 0), coalesce(a.band, 0), a.leaves, a.ranked, a.mock_id, a.tournament_id`

func scanAttempt(row pgx.Row) (Attempt, error) {
	var a Attempt
	var answers []byte
	err := row.Scan(&a.ID, &a.UserID, &a.TestID, &a.Kind, &a.Title, &a.StartedAt, &a.FinishedAt, &answers,
		&a.Correct, &a.Total, &a.Band, &a.Leaves, &a.Ranked, &a.MockID, &a.TournamentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	if err != nil || answers == nil {
		return a, err
	}
	return a, json.Unmarshal(answers, &a.Answers)
}

// Attempt — попытка ученика и её задание; чужая — ErrNotFound.
func (s *Store) Attempt(ctx context.Context, userID, id int64) (Attempt, Test, error) {
	a, err := scanAttempt(s.pool.QueryRow(ctx, `SELECT `+attemptCols+` FROM attempts a JOIN tests t ON t.id = a.test_id
		WHERE a.id = $1 AND a.user_id = $2`, id, userID))
	if err != nil {
		return a, Test{}, err
	}
	t, err := s.Test(ctx, a.TestID)
	return a, t, err
}

// Grade считает верные ответы. answers короче вопросов — недостающие без ответа.
func Grade(t Test, answers []Answer) (correct int, normalized []Answer, err error) {
	if len(answers) > len(t.Questions) {
		return 0, nil, ErrAnswers
	}
	normalized = make([]Answer, len(t.Questions))
	for i, q := range t.Questions {
		normalized[i] = NoAnswer
		if i >= len(answers) {
			continue
		}
		if normalized[i], err = clean(q, answers[i]); err != nil {
			return 0, nil, err
		}
	}
	for _, ok := range Marks(t, normalized) {
		if ok {
			correct++
		}
	}
	return correct, normalized, nil
}

// Submit сдаёт попытку: проверка по ключу, балл, решение «в рейтинг или нет». Вторая сдача — ErrFinished.
func (s *Store) Submit(ctx context.Context, userID, id int64, answers []Answer, leaves int, now time.Time) (Attempt, Test, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Attempt{}, Test{}, err
	}
	defer tx.Rollback(ctx)
	a, err := scanAttempt(tx.QueryRow(ctx, `SELECT `+attemptCols+` FROM attempts a JOIN tests t ON t.id = a.test_id
		WHERE a.id = $1 AND a.user_id = $2 FOR UPDATE OF a`, id, userID))
	if err != nil {
		return a, Test{}, err
	}
	if a.FinishedAt != nil {
		return a, Test{}, ErrFinished
	}
	t, err := scanTest(tx.QueryRow(ctx, `SELECT `+testCols+` FROM tests WHERE id = $1`, a.TestID))
	if err != nil {
		return a, t, err
	}
	correct, norm, err := Grade(t, answers)
	if err != nil {
		return a, t, err
	}
	a.Answers, a.Correct, a.Total = norm, correct, len(t.Questions)
	a.Band = Band(t.Kind, correct, a.Total)
	a.Leaves = max(leaves, 0)
	onTime := !now.After(a.StartedAt.Add(time.Duration(t.Minutes)*time.Minute + Grace))
	a.Ranked = t.Kind != Level && onTime && a.Leaves < MaxLeaves
	a.FinishedAt = &now
	js, _ := json.Marshal(norm)
	if _, err := tx.Exec(ctx, `UPDATE attempts SET finished_at = $2, answers = $3, correct = $4, total = $5, band = $6,
		leaves = $7, ranked = $8 WHERE id = $1`, a.ID, now, js, a.Correct, a.Total, a.Band, a.Leaves, a.Ranked); err != nil {
		return a, t, err
	}
	return a, t, tx.Commit(ctx)
}

// Finished — сданные попытки ученика, новые первыми (для истории).
func (s *Store) Finished(ctx context.Context, userID int64, limit int) ([]Attempt, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+attemptCols+` FROM attempts a JOIN tests t ON t.id = a.test_id
		WHERE a.user_id = $1 AND a.finished_at IS NOT NULL ORDER BY a.finished_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Attempt
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// TypeStats — сколько вопросов каждого типа и сколько из них верно (для слабых мест в отчёте).
type TypeStats struct {
	Kind    string
	Type    string
	Correct int
	Total   int
}

// Done — сданная попытка вместе с заданием.
type Done struct {
	A Attempt
	T Test
}

// Weak — точность по типам вопросов в попытках.
func Weak(pairs []Done) []TypeStats {
	idx := map[[2]string]int{}
	var out []TypeStats
	for _, p := range pairs {
		answers := make([]Answer, len(p.T.Questions))
		for i := range answers {
			answers[i] = NoAnswer
			if i < len(p.A.Answers) {
				answers[i] = p.A.Answers[i]
			}
		}
		marks := Marks(p.T, answers)
		for i, q := range p.T.Questions {
			k := [2]string{p.T.Kind, q.Type}
			n, ok := idx[k]
			if !ok {
				n = len(out)
				idx[k] = n
				out = append(out, TypeStats{Kind: p.T.Kind, Type: q.Type})
			}
			out[n].Total++
			if marks[i] {
				out[n].Correct++
			}
		}
	}
	return out
}
