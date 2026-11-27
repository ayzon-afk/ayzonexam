package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"ayzonexam/internal/exam"
	"ayzonexam/internal/users"
)

// Reading, Listening и тест уровня (этап 3): ученик получает задание без ключа, ключ — после сдачи.

// questionView — вопрос без ключа: Answer, Accept и Set ученик видит только после сдачи (resultView.Key).
type questionView struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Kind    string   `json:"kind,omitempty"`
	Group   int      `json:"group,omitempty"`
}

type testView struct {
	ID        int64          `json:"id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Minutes   int            `json:"minutes"`
	Passage   string         `json:"passage,omitempty"`
	Audio     string         `json:"audio,omitempty"`
	Questions []questionView `json:"questions"`
	Groups    []groupView    `json:"groups,omitempty"`
	Parts     []exam.Part    `json:"parts,omitempty"`
}

// groupView — общий блок вопросов; картинка — открытый файл /media/{id}.
type groupView struct {
	Title   string   `json:"title"`
	Text    string   `json:"text,omitempty"`
	Options []string `json:"options,omitempty"`
	Image   string   `json:"image,omitempty"`
	Pick    int      `json:"pick,omitempty"`
}

func toTestView(t exam.Test) testView {
	v := testView{ID: t.ID, Kind: t.Kind, Title: t.Title, Minutes: t.Minutes, Passage: t.Passage, Parts: t.Parts}
	if t.AudioID != nil {
		v.Audio = "/media/" + strconv.FormatInt(*t.AudioID, 10)
	}
	for _, q := range t.Questions {
		v.Questions = append(v.Questions, questionView{q.Q, q.Options, q.Kind, q.Group})
	}
	for _, g := range t.Groups {
		gv := groupView{Title: g.Title, Text: g.Text, Options: g.Options, Pick: g.Pick}
		if g.Image != nil {
			gv.Image = "/media/" + strconv.FormatInt(*g.Image, 10)
		}
		v.Groups = append(v.Groups, gv)
	}
	return v
}

// resultView — сданная попытка: ответы ученика, ключ и (для Listening) сценарий.
type resultView struct {
	ID         int64            `json:"id"`
	Kind       string           `json:"kind"`
	Title      string           `json:"title"`
	Correct    int              `json:"correct"`
	Total      int              `json:"total"`
	Band       float64          `json:"band"`
	Ranked     bool             `json:"ranked"`
	Answers    []exam.Answer    `json:"answers"`
	Key        []any            `json:"key"` // по вопросу: номер варианта, верные буквы (multi) или допустимые ответы (text)
	Marks      []bool           `json:"marks"`
	Transcript string           `json:"transcript,omitempty"`
	FinishedAt time.Time        `json:"finished_at"`
	Test       testView         `json:"test"`
	MockID     *int64           `json:"mock_id"`
	Level      *levelResultView `json:"level,omitempty"` // тест уровня: диапазон, CEFR, ошибки
}

func toResult(a exam.Attempt, t exam.Test) resultView {
	v := resultView{ID: a.ID, Kind: a.Kind, Title: t.Title, Correct: a.Correct, Total: a.Total, Band: a.Band,
		Ranked: a.Ranked, Answers: a.Answers, Transcript: t.Transcript, Test: toTestView(t), MockID: a.MockID}
	if a.FinishedAt != nil {
		v.FinishedAt = *a.FinishedAt
	}
	for _, q := range t.Questions {
		switch q.Kind {
		case exam.KindText:
			v.Key = append(v.Key, q.Accept)
		case exam.KindMulti:
			v.Key = append(v.Key, q.Set)
		default:
			v.Key = append(v.Key, q.Answer)
		}
	}
	answers := make([]exam.Answer, len(t.Questions))
	for i := range answers {
		answers[i] = exam.NoAnswer
		if i < len(a.Answers) {
			answers[i] = a.Answers[i]
		}
	}
	v.Marks = exam.Marks(t, answers)
	return v
}

func validKind(k string) bool { return k == exam.Reading || k == exam.Listening || k == exam.Level }

func (s *server) listTests(w http.ResponseWriter, r *http.Request, u users.User) {
	kind := r.URL.Query().Get("kind")
	if !validKind(kind) {
		writeError(w, http.StatusBadRequest, "bad_kind")
		return
	}
	list, err := s.exam.List(r.Context(), kind, u.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if list == nil {
		list = []exam.Summary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tests": list})
}

type startView struct {
	AttemptID int64     `json:"attempt_id"`
	EndsAt    time.Time `json:"ends_at"`
	Test      testView  `json:"test"`
}

func (s *server) startTest(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.start(w, r, u, id, nil, nil, s.Now())
}

// start начинает попытку — общий путь практики, mock и турнира. at — когда пошёл таймер: в строгом mock —
// от конца прошлой части (exam.NextStart), иначе сейчас.
func (s *server) start(w http.ResponseWriter, r *http.Request, u users.User, testID int64, mockID, tournamentID *int64, at time.Time) {
	aid, t, err := s.exam.Start(r.Context(), u.ID, testID, mockID, tournamentID, at)
	if errors.Is(err, exam.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, startView{AttemptID: aid, EndsAt: at.Add(time.Duration(t.Minutes) * time.Minute), Test: toTestView(t)})
}

type submitBody struct {
	Answers []exam.Answer `json:"answers"`
	Leaves  int           `json:"leaves"`
}

func (s *server) submitAttempt(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b submitBody
	if !readJSON(w, r, &b, 32<<10) {
		return
	}
	a, t, err := s.exam.Submit(r.Context(), u.ID, id, b.Answers, b.Leaves, s.Now())
	switch {
	case errors.Is(err, exam.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
		return
	case errors.Is(err, exam.ErrFinished):
		writeError(w, http.StatusConflict, "finished")
		return
	case errors.Is(err, exam.ErrAnswers):
		writeError(w, http.StatusBadRequest, "bad_answers")
		return
	case err != nil:
		s.fail(w, r, err)
		return
	}
	s.afterAttempt(r, u, a)
	if s.hiddenPart(r.Context(), u, a) {
		writeJSON(w, http.StatusOK, map[string]any{"id": a.ID, "mock_id": a.MockID, "hidden": true})
		return
	}
	writeJSON(w, http.StatusOK, toResult(a, t))
}

func (s *server) getAttempt(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	a, t, err := s.exam.Attempt(r.Context(), u.ID, id)
	if errors.Is(err, exam.ErrNotFound) || (err == nil && a.FinishedAt == nil) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if s.hiddenPart(r.Context(), u, a) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	v := toResult(a, t)
	if a.Kind == exam.Level {
		if err := s.addLevel(r, u, a, &v); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, v)
}

// hiddenPart — попытка — часть строгого mock, который ещё идёт: как на экзамене, балл и ключ — в отчёте.
func (s *server) hiddenPart(ctx context.Context, u users.User, a exam.Attempt) bool {
	if a.MockID == nil {
		return false
	}
	m, err := s.exam.Mock(ctx, u.ID, *a.MockID)
	return err == nil && m.Strict && m.Status != "done"
}

// media отдаёт открытый файл (аудио Listening). Файлы не меняются — кэш навсегда; Range для перемотки.
func (s *server) media(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.serveMedia(w, r, id, true, "public, max-age=31536000, immutable") {
		http.NotFound(w, r)
	}
}

// pathID читает {id} из пути; неверный — 404.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "not_found")
		return 0, false
	}
	return id, true
}

// readJSON читает тело JSON не больше limit байт; лишние поля — ошибка.
func readJSON(w http.ResponseWriter, r *http.Request, v any, limit int64) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json")
		return false
	}
	return true
}
