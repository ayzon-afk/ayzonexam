package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/billing"
	"ayzonexam/internal/exam"
	"ayzonexam/internal/users"
	"ayzonexam/internal/writing"
)

// Полный mock по частям (этап 18). Listening и Reading считаются сразу, Writing и Speaking — очередями AI.

type partView struct {
	State      string     `json:"state"` // todo, running, checking, done
	Band       *float64   `json:"band"`
	AttemptID  *int64     `json:"attempt_id,omitempty"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	Violations int        `json:"violations"` // нарушения честного режима; с 3 — не в рейтинге
}

type mockCheck struct {
	ID     int64    `json:"id"`
	Task   string   `json:"task"`
	Status string   `json:"status"`
	Band   *float64 `json:"band"`
}

type weakView struct {
	Part string `json:"part"`
	Type string `json:"type"`
	Pct  int    `json:"pct"`
}

type mockView struct {
	ID         int64               `json:"id"`
	Status     string              `json:"status"`
	StartedAt  time.Time           `json:"started_at"`
	FinishedAt *time.Time          `json:"finished_at"`
	Overall    *float64            `json:"overall"`
	Parts      map[string]partView `json:"parts"`
	Writing    []mockCheck         `json:"writing_checks"`
	Speaking   *mockCheck          `json:"speaking_check"`
	Weak       []weakView          `json:"weak"`
	Plan       []string            `json:"plan"` // части по слабости: из них собирается план на 2 недели
	Prev       map[string]float64  `json:"prev"` // баллы прошлого mock — для стрелок роста
}

func (s *server) mockView(ctx context.Context, u users.User, m exam.Mock) (mockView, error) {
	v := mockView{ID: m.ID, Status: m.Status, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt, Overall: m.Overall,
		Parts: map[string]partView{}, Writing: []mockCheck{}, Weak: []weakView{}, Plan: []string{}, Prev: map[string]float64{}}
	for _, kind := range []string{exam.Listening, exam.Reading} {
		p := partView{State: "todo"}
		a, err := s.exam.PartAttempt(ctx, m.ID, kind)
		if err != nil {
			return v, err
		}
		if a != nil {
			p.AttemptID = &a.ID
			p.Violations = a.Leaves
			if a.FinishedAt != nil {
				p.State, p.Band = "done", &a.Band
			} else {
				p.State = "running"
				t, err := s.exam.Test(ctx, a.TestID)
				if err != nil {
					return v, err
				}
				end := a.StartedAt.Add(time.Duration(t.Minutes) * time.Minute)
				p.EndsAt = &end
			}
		}
		v.Parts[kind] = p
	}

	rows, err := s.DB.Query(ctx, `SELECT id, task, status, band::float8 FROM writing_checks WHERE mock_id = $1 ORDER BY id`, m.ID)
	if err != nil {
		return v, err
	}
	for rows.Next() {
		var c mockCheck
		if err := rows.Scan(&c.ID, &c.Task, &c.Status, &c.Band); err != nil {
			rows.Close()
			return v, err
		}
		v.Writing = append(v.Writing, c)
	}
	rows.Close()
	wp := partView{State: "todo", Band: m.Writing, Violations: m.WritingViolations}
	inQueue := 0 // проверки, которые не упали: упавшее задание можно отправить ещё раз
	for _, c := range v.Writing {
		if c.Status != "failed" {
			inQueue++
		}
	}
	switch {
	case m.Writing != nil:
		wp.State = "done"
	case inQueue > 0 && inQueue == len(v.Writing):
		wp.State = "checking"
	case m.WritingStartedAt != nil:
		wp.State = "running"
		end := m.WritingStartedAt.Add(exam.WritingMinutes * time.Minute)
		wp.EndsAt = &end
	}
	v.Parts["writing"] = wp

	var sc mockCheck
	err = s.DB.QueryRow(ctx, `SELECT id, status, band::float8 FROM speaking_checks WHERE mock_id = $1 ORDER BY id DESC LIMIT 1`, m.ID).
		Scan(&sc.ID, &sc.Status, &sc.Band)
	if err == nil {
		v.Speaking = &sc
	}
	sp := partView{State: "todo", Band: m.Speaking, Violations: m.SpeakingViolations}
	switch {
	case m.Speaking != nil:
		sp.State = "done"
	case v.Speaking != nil && v.Speaking.Status != "failed":
		sp.State = "checking"
	case m.SpeakingStartedAt != nil:
		sp.State = "running"
		end := m.SpeakingStartedAt.Add(exam.SpeakingMinutes * time.Minute)
		sp.EndsAt = &end
	}
	v.Parts["speaking"] = sp

	if m.Status == "done" {
		if err := s.fillReport(ctx, u, m, &v); err != nil {
			return v, err
		}
	}
	return v, nil
}

// fillReport — слабые места и порядок плана: самые слабые типы вопросов и критерии, части по баллу.
func (s *server) fillReport(ctx context.Context, u users.User, m exam.Mock, v *mockView) error {
	done, err := s.exam.MockAttempts(ctx, u.ID, m.ID)
	if err != nil {
		return err
	}
	for _, st := range exam.Weak(done) {
		if st.Total == 0 {
			continue
		}
		pct := st.Correct * 100 / st.Total
		if pct < 75 {
			v.Weak = append(v.Weak, weakView{Part: st.Kind, Type: st.Type, Pct: pct})
		}
	}
	// Самый слабый критерий Writing и Speaking.
	var wl, sl string
	err = s.DB.QueryRow(ctx, `SELECT
		coalesce((SELECT k FROM (SELECT 'ta' k, avg(ta) v FROM writing_checks WHERE mock_id = $1 AND status = 'done'
			UNION ALL SELECT 'cc', avg(cc) FROM writing_checks WHERE mock_id = $1 AND status = 'done'
			UNION ALL SELECT 'lr', avg(lr) FROM writing_checks WHERE mock_id = $1 AND status = 'done'
			UNION ALL SELECT 'gra', avg(gra) FROM writing_checks WHERE mock_id = $1 AND status = 'done') x WHERE v IS NOT NULL ORDER BY v LIMIT 1), ''),
		coalesce((SELECT k FROM (SELECT 'fc' k, fc v FROM speaking_checks WHERE mock_id = $1 AND status = 'done'
			UNION ALL SELECT 'lr', lr FROM speaking_checks WHERE mock_id = $1 AND status = 'done'
			UNION ALL SELECT 'gra', gra FROM speaking_checks WHERE mock_id = $1 AND status = 'done'
			UNION ALL SELECT 'pr', pr FROM speaking_checks WHERE mock_id = $1 AND status = 'done') y WHERE v IS NOT NULL ORDER BY v LIMIT 1), '')`,
		m.ID).Scan(&wl, &sl)
	if err != nil {
		return err
	}
	if wl != "" {
		v.Weak = append(v.Weak, weakView{Part: "writing", Type: wl})
	}
	if sl != "" {
		v.Weak = append(v.Weak, weakView{Part: "speaking", Type: sl})
	}
	type pb struct {
		part string
		band float64
	}
	parts := []pb{}
	for _, p := range []struct {
		name string
		b    *float64
	}{{"listening", m.Listening}, {"reading", m.Reading}, {"writing", m.Writing}, {"speaking", m.Speaking}} {
		if p.b != nil {
			parts = append(parts, pb{p.name, *p.b})
		}
	}
	slices.SortStableFunc(parts, func(a, b pb) int {
		switch {
		case a.band < b.band:
			return -1
		case a.band > b.band:
			return 1
		}
		return 0
	})
	for _, p := range parts {
		v.Plan = append(v.Plan, p.part)
	}
	// Прошлый законченный mock — для стрелок.
	var l, r, w, sp *float64
	err = s.DB.QueryRow(ctx, `SELECT listening_band::float8, reading_band::float8, writing_band::float8, speaking_band::float8
		FROM mocks WHERE user_id = $1 AND status = 'done' AND finished_at < $2 ORDER BY finished_at DESC LIMIT 1`, u.ID, m.FinishedAt).
		Scan(&l, &r, &w, &sp)
	if err == nil {
		for k, b := range map[string]*float64{"listening": l, "reading": r, "writing": w, "speaking": sp} {
			if b != nil {
				v.Prev[k] = *b
			}
		}
	}
	return nil
}

// mockQuota — сколько mock доступно в этом месяце: по Premium и купленные.
type mockQuota struct {
	Premium bool `json:"premium"`
	Used    int  `json:"used"`
	Limit   int  `json:"limit"`
	Credits int  `json:"credits"`
}

func (s *server) quotaMock(ctx context.Context, u users.User) (mockQuota, error) {
	now := s.Now()
	a := billing.Of(u, now)
	q := billing.MockQuota(a, now)
	used, err := s.billing.UsedMocks(ctx, u.ID, q.Since)
	return mockQuota{Premium: a.Premium, Used: used, Limit: q.Limit, Credits: u.MockCredits}, err
}

type mockSummary struct {
	ID         int64      `json:"id"`
	Status     string     `json:"status"`
	Overall    *float64   `json:"overall"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

func (s *server) getMocks(w http.ResponseWriter, r *http.Request, u users.User) {
	ctx := r.Context()
	list, err := s.exam.Mocks(ctx, u.ID, 30)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var active *mockView
	past := []mockSummary{}
	for _, m := range list {
		if m.Status == "active" {
			v, err := s.mockView(ctx, u, m)
			if err != nil {
				s.fail(w, r, err)
				return
			}
			active = &v
			continue
		}
		past = append(past, mockSummary{m.ID, m.Status, m.Overall, m.StartedAt, m.FinishedAt})
	}
	q, err := s.quotaMock(ctx, u)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"active": active, "past": past, "quota": q})
}

// createMock — новый mock: из двух в месяц по Premium, иначе — купленный; иначе 402.
func (s *server) createMock(w http.ResponseWriter, r *http.Request, u users.User) {
	ctx := r.Context()
	q, err := s.quotaMock(ctx, u)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	paid := ""
	switch {
	case q.Premium && q.Used < q.Limit:
		paid = "premium"
	case u.MockCredits > 0:
		paid = "credit"
	default:
		writeError(w, http.StatusPaymentRequired, "payment_required")
		return
	}
	m, err := s.exam.CreateMock(ctx, u.ID, paid, s.Now())
	if errors.Is(err, exam.ErrMockActive) {
		writeError(w, http.StatusConflict, "mock_active")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	v, err := s.mockView(ctx, u, m)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// loadMock — mock ученика из пути; для изменения — только незаконченный.
func (s *server) loadMock(w http.ResponseWriter, r *http.Request, u users.User, active bool) (exam.Mock, bool) {
	id, ok := pathID(w, r)
	if !ok {
		return exam.Mock{}, false
	}
	m, err := s.exam.Mock(r.Context(), u.ID, id)
	if errors.Is(err, exam.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return m, false
	}
	if err != nil {
		s.fail(w, r, err)
		return m, false
	}
	if active && m.Status != "active" {
		writeError(w, http.StatusConflict, "mock_done")
		return m, false
	}
	return m, true
}

func (s *server) getMock(w http.ResponseWriter, r *http.Request, u users.User) {
	m, ok := s.loadMock(w, r, u, false)
	if !ok {
		return
	}
	v, err := s.mockView(r.Context(), u, m)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// startMockPart — Listening или Reading mock: новая попытка или продолжение идущей.
func (s *server) startMockPart(w http.ResponseWriter, r *http.Request, u users.User) {
	m, ok := s.loadMock(w, r, u, true)
	if !ok {
		return
	}
	var b struct {
		Kind string `json:"kind"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	if b.Kind != exam.Listening && b.Kind != exam.Reading {
		writeError(w, http.StatusBadRequest, "bad_kind")
		return
	}
	ctx := r.Context()
	a, err := s.exam.PartAttempt(ctx, m.ID, b.Kind)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if a != nil && a.FinishedAt != nil {
		writeError(w, http.StatusConflict, "part_done")
		return
	}
	if a != nil {
		// Продолжение: тот же таймер, ответы интерфейс хранит у себя.
		t, err := s.exam.Test(ctx, a.TestID)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, startView{AttemptID: a.ID, EndsAt: a.StartedAt.Add(time.Duration(t.Minutes) * time.Minute), Test: toTestView(t)})
		return
	}
	t, err := s.exam.Pick(ctx, b.Kind, u.ID)
	if errors.Is(err, exam.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no_tests")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.start(w, r, u, t.ID, &m.ID, nil)
}

type mockPrompt struct {
	ID    int64  `json:"id"`
	Text  string `json:"text"`
	Image string `json:"image,omitempty"`
}

// startMockWriting — таймер Writing; темы Task 1 (если есть в банке) и Task 2.
func (s *server) startMockWriting(w http.ResponseWriter, r *http.Request, u users.User) {
	m, ok := s.loadMock(w, r, u, true)
	if !ok {
		return
	}
	ctx := r.Context()
	m, err := s.exam.StartWriting(ctx, m, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if m.Task2PromptID == nil {
		writeError(w, http.StatusNotFound, "no_tests")
		return
	}
	out := map[string]any{"ends_at": m.WritingStartedAt.Add(exam.WritingMinutes * time.Minute)}
	for key, id := range map[string]*int64{"task1": m.Task1PromptID, "task2": m.Task2PromptID} {
		if id == nil {
			out[key] = nil
			continue
		}
		var p mockPrompt
		var img *int64
		if err := s.DB.QueryRow(ctx, `SELECT id, text, image_id FROM writing_prompts WHERE id = $1`, *id).Scan(&p.ID, &p.Text, &img); err != nil {
			s.fail(w, r, err)
			return
		}
		if img != nil {
			p.Image = "/media/" + strconv.FormatInt(*img, 10)
		}
		out[key] = p
	}
	writeJSON(w, http.StatusOK, out)
}

// submitMockWriting — ответы Writing уходят в очередь проверки; лимит проверок не тратится.
// Задание, проверка которого не удалась, можно отправить ещё раз.
func (s *server) submitMockWriting(w http.ResponseWriter, r *http.Request, u users.User) {
	m, ok := s.loadMock(w, r, u, true)
	if !ok {
		return
	}
	var b struct {
		Task1      string `json:"task1"`
		Task2      string `json:"task2"`
		Violations int    `json:"violations"` // счёт честного режима в браузере
	}
	if !readJSON(w, r, &b, 64<<10) {
		return
	}
	if m.WritingStartedAt == nil || m.Task2PromptID == nil {
		writeError(w, http.StatusConflict, "not_started")
		return
	}
	ctx := r.Context()
	busy := map[string]bool{}
	rows, err := s.DB.Query(ctx, `SELECT task FROM writing_checks WHERE mock_id = $1 AND status <> 'failed'`, m.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	for rows.Next() {
		var t string
		rows.Scan(&t)
		busy[t] = true
	}
	rows.Close()
	type item struct {
		task, essay string
		prompt      int64
	}
	var items []item
	if m.Task1PromptID != nil && !busy[writing.Task1] {
		items = append(items, item{writing.Task1, strings.TrimSpace(b.Task1), *m.Task1PromptID})
	}
	if !busy[writing.Task2] {
		items = append(items, item{writing.Task2, strings.TrimSpace(b.Task2), *m.Task2PromptID})
	}
	if len(items) == 0 {
		writeError(w, http.StatusConflict, "part_done")
		return
	}
	// Нарушения не уменьшаются: повторная отправка упавшего задания не обнуляет счёт.
	if _, err := s.DB.Exec(ctx, `UPDATE mocks SET writing_violations = greatest(writing_violations, $2) WHERE id = $1`, m.ID, max(b.Violations, 0)); err != nil {
		s.fail(w, r, err)
		return
	}
	for _, it := range items {
		words := writing.CountWords(it.essay)
		if words < writing.MinWords {
			writeError(w, http.StatusBadRequest, "too_short")
			return
		}
		if utf8.RuneCountInString(it.essay) > maxEssayRunes {
			writeError(w, http.StatusBadRequest, "too_long")
			return
		}
	}
	for _, it := range items {
		var text string
		var imgID *int64
		if err := s.DB.QueryRow(ctx, `SELECT text, image_id FROM writing_prompts WHERE id = $1`, it.prompt).Scan(&text, &imgID); err != nil {
			s.fail(w, r, err)
			return
		}
		c := writing.NewCheck{UserID: u.ID, Task: it.task, PromptID: &it.prompt, Prompt: text, Essay: it.essay,
			Words: writing.CountWords(it.essay), Source: "text", Lang: u.Lang, MockID: &m.ID}
		if imgID != nil {
			var img ai.Image
			if err := s.DB.QueryRow(ctx, `SELECT mime, data FROM media WHERE id = $1`, *imgID).Scan(&img.MIME, &img.Data); err != nil {
				s.fail(w, r, err)
				return
			}
			c.Image = &img
		}
		if _, err := s.Writing.Create(ctx, c, 0, s.Now(), s.Now()); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	v, err := s.mockView(ctx, u, m)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// mockPartDone пересчитывает mock после сданной части; mock закрылся — бот сообщает, что отчёт готов.
func (s *server) mockPartDone(ctx context.Context, mockID int64) error {
	closed, err := s.exam.UpdateMock(ctx, mockID, s.Now())
	if err != nil || !closed {
		return err
	}
	var tg int64
	var lang string
	var overall float64
	if err := s.DB.QueryRow(ctx, `SELECT u.telegram_id, u.lang, m.overall::float8 FROM mocks m JOIN users u ON u.id = m.user_id
		WHERE m.id = $1`, mockID).Scan(&tg, &lang, &overall); err != nil {
		return err
	}
	s.say(ctx, tg, lang, "mock.done", map[string]string{"band": strconv.FormatFloat(overall, 'f', 1, 64)}, "mock-"+strconv.FormatInt(mockID, 10))
	return nil
}
