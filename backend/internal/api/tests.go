package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"ayzonexam/internal/exam"
	"ayzonexam/internal/users"
)

// Reading, Listening и тест уровня (этап 3): ученик получает задание без ключа, ключ — после сдачи.

type questionView struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
}

type testView struct {
	ID        int64          `json:"id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Minutes   int            `json:"minutes"`
	Passage   string         `json:"passage,omitempty"`
	Audio     string         `json:"audio,omitempty"`
	Questions []questionView `json:"questions"`
}

func toTestView(t exam.Test) testView {
	v := testView{ID: t.ID, Kind: t.Kind, Title: t.Title, Minutes: t.Minutes, Passage: t.Passage}
	if t.AudioID != nil {
		v.Audio = "/media/" + strconv.FormatInt(*t.AudioID, 10)
	}
	for _, q := range t.Questions {
		v.Questions = append(v.Questions, questionView{q.Q, q.Options})
	}
	return v
}

// resultView — сданная попытка: ответы ученика, ключ и (для Listening) сценарий.
type resultView struct {
	ID         int64     `json:"id"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	Correct    int       `json:"correct"`
	Total      int       `json:"total"`
	Band       float64   `json:"band"`
	Ranked     bool      `json:"ranked"`
	Answers    []int     `json:"answers"`
	Key        []int     `json:"key"`
	Transcript string    `json:"transcript,omitempty"`
	FinishedAt time.Time `json:"finished_at"`
	Test       testView  `json:"test"`
	MockID     *int64    `json:"mock_id"`
}

func toResult(a exam.Attempt, t exam.Test) resultView {
	v := resultView{ID: a.ID, Kind: a.Kind, Title: t.Title, Correct: a.Correct, Total: a.Total, Band: a.Band,
		Ranked: a.Ranked, Answers: a.Answers, Transcript: t.Transcript, Test: toTestView(t), MockID: a.MockID}
	if a.FinishedAt != nil {
		v.FinishedAt = *a.FinishedAt
	}
	for _, q := range t.Questions {
		v.Key = append(v.Key, q.Answer)
	}
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
	s.start(w, r, u, id, nil, nil)
}

// start начинает попытку — общий путь практики, mock и турнира.
func (s *server) start(w http.ResponseWriter, r *http.Request, u users.User, testID int64, mockID, tournamentID *int64) {
	now := s.Now()
	aid, t, err := s.exam.Start(r.Context(), u.ID, testID, mockID, tournamentID, now)
	if errors.Is(err, exam.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, startView{AttemptID: aid, EndsAt: now.Add(time.Duration(t.Minutes) * time.Minute), Test: toTestView(t)})
}

type submitBody struct {
	Answers []int `json:"answers"`
	Leaves  int   `json:"leaves"`
}

func (s *server) submitAttempt(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b submitBody
	if !readJSON(w, r, &b, 16<<10) {
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
	writeJSON(w, http.StatusOK, toResult(a, t))
}

// media отдаёт открытый файл (аудио Listening). Файлы не меняются — кэш навсегда; Range для перемотки.
func (s *server) media(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var mime string
	var data []byte
	err = s.DB.QueryRow(r.Context(), `SELECT mime, data FROM media WHERE id = $1 AND NOT private`, id).Scan(&mime, &data)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
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
