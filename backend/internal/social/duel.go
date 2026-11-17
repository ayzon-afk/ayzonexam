package social

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Дуэль 1 на 1 (SPEC): 5 минут, подобрать синоним. Каждый играет свои 5 минут на одних и тех же словах;
// сервер проверяет каждый ответ и время — счёт нельзя прислать готовым.

const (
	DuelTime      = 5 * time.Minute
	DuelGrace     = 5 * time.Second
	DuelQuestions = 30
	DuelExpires   = 48 * time.Hour // столько ждёт вызов, пока друг его примет
)

var (
	ErrDuelBusy    = errors.New("в дуэли уже два игрока")
	ErrDuelExpired = errors.New("вызов истёк")
	ErrDuelOver    = errors.New("время вышло")
	ErrDuelOrder   = errors.New("ответ не на текущий вопрос")
)

// words — слово и его синоним. Синонимы не пересекаются по смыслу: чужой синоним среди вариантов не
// должен оказаться тоже верным.
var words = [][2]string{
	{"rapid", "quick"}, {"enormous", "huge"}, {"reluctant", "unwilling"}, {"obtain", "get"}, {"vital", "essential"},
	{"decline", "decrease"}, {"precise", "exact"}, {"abundant", "plentiful"}, {"assist", "help"}, {"attempt", "try"},
	{"commence", "begin"}, {"conceal", "hide"}, {"demonstrate", "show"}, {"eager", "keen"}, {"fragile", "delicate"},
	{"genuine", "real"}, {"hazardous", "dangerous"}, {"inevitable", "unavoidable"}, {"inquire", "ask"}, {"modify", "change"},
	{"obsolete", "outdated"}, {"purchase", "buy"}, {"reside", "live"}, {"sufficient", "enough"}, {"terminate", "end"},
	{"vacant", "empty"}, {"wealthy", "rich"}, {"ancient", "very old"}, {"beneficial", "helpful"}, {"brief", "short"},
	{"candid", "frank"}, {"cautious", "careful"}, {"comprehend", "understand"}, {"deliberate", "intentional"},
	{"durable", "long-lasting"}, {"eliminate", "remove"}, {"emerge", "appear"}, {"enhance", "improve"}, {"evident", "obvious"},
	{"expand", "grow"}, {"feasible", "possible"}, {"frequently", "often"}, {"hostile", "unfriendly"}, {"impartial", "neutral"},
	{"initial", "first"}, {"lucrative", "profitable"}, {"notion", "idea"}, {"obstacle", "barrier"}, {"occupation", "job"},
	{"option", "choice"}, {"persuade", "convince"}, {"predict", "forecast"}, {"prior", "earlier"}, {"prohibit", "forbid"},
	{"pursue", "chase"}, {"reliable", "dependable"}, {"remote", "distant"}, {"require", "need"}, {"resilient", "tough"},
	{"scarce", "rare"}, {"sole", "only"}, {"tedious", "boring"}, {"trivial", "unimportant"}, {"ultimate", "final"},
	{"vague", "unclear"}, {"vivid", "bright"}, {"widespread", "common"}, {"yield", "produce"}, {"adjacent", "neighbouring"},
	{"assess", "evaluate"}, {"compel", "force"}, {"contemporary", "modern"}, {"diverse", "varied"}, {"domestic", "household"},
	{"exhausted", "tired"}, {"flaw", "defect"}, {"gradual", "slow"}, {"hinder", "obstruct"}, {"implement", "carry out"},
	{"legitimate", "lawful"}, {"mandatory", "compulsory"}, {"mature", "grown-up"}, {"obscure", "unknown"},
	{"ponder", "think about"}, {"prominent", "famous"}, {"retain", "keep"}, {"rigid", "stiff"}, {"verify", "confirm"},
	{"withdraw", "pull out"}, {"innovative", "inventive"},
}

type Question struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
}

// newQuestions — n случайных слов, у каждого 4 варианта: синоним и три чужих синонима.
func newQuestions(n int) []Question {
	idx := rand.Perm(len(words))[:n]
	qs := make([]Question, 0, n)
	for _, i := range idx {
		opts := []string{words[i][1]}
		for _, j := range rand.Perm(len(words)) {
			if j != i && len(opts) < 4 {
				opts = append(opts, words[j][1])
			}
		}
		rand.Shuffle(len(opts), func(a, b int) { opts[a], opts[b] = opts[b], opts[a] })
		ans := 0
		for k, o := range opts {
			if o == words[i][1] {
				ans = k
			}
		}
		qs = append(qs, Question{Q: words[i][0], Options: opts, Answer: ans})
	}
	return qs
}

type Duel struct {
	ID           int64
	Code         string
	ChallengerID int64
	OpponentID   *int64
	Questions    []Question
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// CreateDuel — вызов: по ссылке (opponent nil — примет любой, кто откроет) или другу напрямую.
func (s *Store) CreateDuel(ctx context.Context, challenger int64, opponent *int64, now time.Time) (Duel, error) {
	d := Duel{ChallengerID: challenger, OpponentID: opponent, Questions: newQuestions(DuelQuestions), CreatedAt: now, ExpiresAt: now.Add(DuelExpires)}
	qs, _ := json.Marshal(d.Questions)
	for range 10 {
		d.Code = code(8)
		err := s.pool.QueryRow(ctx, `INSERT INTO duels (code, challenger_id, opponent_id, questions, created_at, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, d.Code, challenger, opponent, qs, now, d.ExpiresAt).Scan(&d.ID)
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			continue
		}
		return d, err
	}
	return d, errors.New("код дуэли не подобран")
}

func (s *Store) duel(ctx context.Context, q pgx.Tx, where string, arg any, lock bool) (Duel, error) {
	var d Duel
	var qs []byte
	sql := `SELECT id, code, challenger_id, opponent_id, questions, created_at, expires_at FROM duels WHERE ` + where
	if lock {
		sql += ` FOR UPDATE`
	}
	var row pgx.Row
	if q != nil {
		row = q.QueryRow(ctx, sql, arg)
	} else {
		row = s.pool.QueryRow(ctx, sql, arg)
	}
	err := row.Scan(&d.ID, &d.Code, &d.ChallengerID, &d.OpponentID, &qs, &d.CreatedAt, &d.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	return d, json.Unmarshal(qs, &d.Questions)
}

// DuelByCode — дуэль по коду ссылки.
func (s *Store) DuelByCode(ctx context.Context, c string) (Duel, error) {
	return s.duel(ctx, nil, `code = $1`, c, false)
}

// Player — игрок дуэли: публичное и счёт.
type Player struct {
	ID       int64      `json:"-"`
	Nickname string     `json:"nickname"`
	Avatar   string     `json:"avatar"`
	Score    int        `json:"score"`
	Answered int        `json:"answered"`
	Started  *time.Time `json:"started_at"`
	Finished bool       `json:"finished"`
}

// DuelView — дуэль глазами ученика.
type DuelView struct {
	ID         int64      `json:"id"`
	Code       string     `json:"code"`
	State      string     `json:"state"` // open — ждёт соперника, playing — кто-то не доиграл, done, expired
	Challenger Player     `json:"challenger"`
	Opponent   *Player    `json:"opponent"`
	Mine       bool       `json:"mine"` // ученик — участник
	ExpiresAt  time.Time  `json:"expires_at"`
	EndsAt     *time.Time `json:"ends_at"` // когда кончаются мои 5 минут, если я играю
}

func (s *Store) player(ctx context.Context, duelID, userID int64, now time.Time) (Player, error) {
	p := Player{ID: userID}
	var answers []byte
	var fin *time.Time
	err := s.pool.QueryRow(ctx, `SELECT u.nickname, u.photo_url, dp.started_at, dp.finished_at, dp.answers, coalesce(dp.score, 0)
		FROM users u LEFT JOIN duel_players dp ON dp.user_id = u.id AND dp.duel_id = $1 WHERE u.id = $2`, duelID, userID).
		Scan(&p.Nickname, &p.Avatar, &p.Started, &fin, &answers, &p.Score)
	if err != nil {
		return p, err
	}
	var a []int
	if answers != nil {
		json.Unmarshal(answers, &a)
	}
	p.Answered = len(a)
	p.Finished = fin != nil || (p.Started != nil && now.After(p.Started.Add(DuelTime+DuelGrace)))
	return p, nil
}

// View — дуэль и счёт обоих; me — кто смотрит.
func (s *Store) View(ctx context.Context, id, me int64, now time.Time) (DuelView, error) {
	d, err := s.duel(ctx, nil, `id = $1`, id, false)
	if err != nil {
		return DuelView{}, err
	}
	return s.view(ctx, d, me, now)
}

func (s *Store) view(ctx context.Context, d Duel, me int64, now time.Time) (DuelView, error) {
	v := DuelView{ID: d.ID, Code: d.Code, ExpiresAt: d.ExpiresAt}
	var err error
	if v.Challenger, err = s.player(ctx, d.ID, d.ChallengerID, now); err != nil {
		return v, err
	}
	if d.OpponentID != nil {
		p, err := s.player(ctx, d.ID, *d.OpponentID, now)
		if err != nil {
			return v, err
		}
		v.Opponent = &p
	}
	v.Mine = me == d.ChallengerID || (d.OpponentID != nil && me == *d.OpponentID)
	oppStarted := v.Opponent != nil && v.Opponent.Started != nil
	switch {
	case v.Challenger.Finished && v.Opponent != nil && v.Opponent.Finished:
		v.State = "done"
	case !oppStarted && now.After(d.ExpiresAt):
		v.State = "expired"
	case !oppStarted:
		v.State = "open"
	default:
		v.State = "playing"
	}
	for _, p := range []*Player{&v.Challenger, v.Opponent} {
		if p != nil && p.ID == me && p.Started != nil && !p.Finished {
			end := p.Started.Add(DuelTime)
			v.EndsAt = &end
		}
	}
	return v, nil
}

// Play — ученик начинает (или продолжает) свои 5 минут. Открытый вызов по ссылке принимает первый,
// кто его откроет, — и становится другом вызвавшего. Возвращает вопросы без ответов и сколько уже отвечено.
func (s *Store) Play(ctx context.Context, id, me int64, now time.Time) (Duel, int, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Duel{}, 0, false, err
	}
	defer tx.Rollback(ctx)
	d, err := s.duel(ctx, tx, `id = $1`, id, true)
	if err != nil {
		return d, 0, false, err
	}
	accepted := false
	switch {
	case me == d.ChallengerID:
	case d.OpponentID != nil && *d.OpponentID == me:
	case d.OpponentID == nil:
		if now.After(d.ExpiresAt) {
			return d, 0, false, ErrDuelExpired
		}
		if _, err := tx.Exec(ctx, `UPDATE duels SET opponent_id = $2 WHERE id = $1`, d.ID, me); err != nil {
			return d, 0, false, err
		}
		d.OpponentID = &me
		accepted = true
	default:
		return d, 0, false, ErrDuelBusy
	}
	if me != d.ChallengerID {
		var started bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM duel_players WHERE duel_id = $1 AND user_id = $2)`, d.ID, me).Scan(&started); err != nil {
			return d, 0, false, err
		}
		if !started && now.After(d.ExpiresAt) {
			return d, 0, false, ErrDuelExpired
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO duel_players (duel_id, user_id, started_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, d.ID, me, now); err != nil {
		return d, 0, false, err
	}
	if accepted {
		if _, err := tx.Exec(ctx, `INSERT INTO friends (user_id, friend_id) VALUES ($1, $2), ($2, $1) ON CONFLICT DO NOTHING`, d.ChallengerID, me); err != nil {
			return d, 0, false, err
		}
	}
	var answers []byte
	if err := tx.QueryRow(ctx, `SELECT answers FROM duel_players WHERE duel_id = $1 AND user_id = $2`, d.ID, me).Scan(&answers); err != nil {
		return d, 0, false, err
	}
	var a []int
	json.Unmarshal(answers, &a)
	return d, len(a), accepted, tx.Commit(ctx)
}

// AnswerResult — ответ на вопрос: верно ли, какой был верный, счёт; Finished — это был последний.
type AnswerResult struct {
	Correct  bool `json:"correct"`
	Answer   int  `json:"answer"`
	Score    int  `json:"score"`
	Finished bool `json:"finished"`
}

// Answer — ответ на вопрос i. Только по порядку и только в свои 5 минут.
func (s *Store) Answer(ctx context.Context, id, me int64, i, choice int, now time.Time) (AnswerResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AnswerResult{}, err
	}
	defer tx.Rollback(ctx)
	var started time.Time
	var fin *time.Time
	var answers []byte
	var score int
	err = tx.QueryRow(ctx, `SELECT started_at, finished_at, answers, score FROM duel_players WHERE duel_id = $1 AND user_id = $2 FOR UPDATE`,
		id, me).Scan(&started, &fin, &answers, &score)
	if errors.Is(err, pgx.ErrNoRows) {
		return AnswerResult{}, ErrNotFound
	}
	if err != nil {
		return AnswerResult{}, err
	}
	if fin != nil || now.After(started.Add(DuelTime+DuelGrace)) {
		return AnswerResult{}, ErrDuelOver
	}
	var a []int
	json.Unmarshal(answers, &a)
	d, err := s.duel(ctx, tx, `id = $1`, id, false)
	if err != nil {
		return AnswerResult{}, err
	}
	if i != len(a) || i >= len(d.Questions) {
		return AnswerResult{}, ErrDuelOrder
	}
	q := d.Questions[i]
	r := AnswerResult{Correct: choice == q.Answer, Answer: q.Answer}
	if r.Correct {
		score++
	}
	r.Score = score
	a = append(a, choice)
	r.Finished = len(a) == len(d.Questions)
	js, _ := json.Marshal(a)
	var finAt *time.Time
	if r.Finished {
		finAt = &now
	}
	if _, err := tx.Exec(ctx, `UPDATE duel_players SET answers = $3, score = $4, finished_at = $5 WHERE duel_id = $1 AND user_id = $2`,
		id, me, js, score, finAt); err != nil {
		return r, err
	}
	return r, tx.Commit(ctx)
}

// Finish — время ученика вышло (или он сдался). Повторный вызов ничего не меняет.
func (s *Store) Finish(ctx context.Context, id, me int64, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE duel_players SET finished_at = least($3, started_at + make_interval(secs => $4))
		WHERE duel_id = $1 AND user_id = $2 AND finished_at IS NULL`, id, me, now, DuelTime.Seconds())
	return err
}

// Duels — последние дуэли ученика, новые первыми.
func (s *Store) Duels(ctx context.Context, me int64, now time.Time, limit int) ([]DuelView, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM duels WHERE challenger_id = $1 OR opponent_id = $1
		ORDER BY created_at DESC LIMIT $2`, me, limit)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return nil, err
	}
	list := []DuelView{}
	for _, id := range ids {
		v, err := s.View(ctx, id, me, now)
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

// Questions без ответов — то, что уходит ученику.
func Public(qs []Question) []Question {
	out := make([]Question, len(qs))
	for i, q := range qs {
		out[i] = Question{Q: q.Q, Options: q.Options, Answer: -1}
	}
	return out
}
