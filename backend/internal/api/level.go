package api

import (
	"errors"
	"net/http"

	"ayzonexam/internal/exam"
	"ayzonexam/internal/level"
	"ayzonexam/internal/users"
)

// Тест уровня, который подстраивается под ученика (DECISIONS 2026-11-27): сервер выдаёт по одному вопросу и
// по ответам выбирает следующий; ключ в приложение не уходит, пока тест не сдан.

type levelItemView struct {
	ID      string   `json:"id"`
	Q       string   `json:"q"`
	Options []string `json:"options"`
}

type levelPassageView struct {
	Title     string         `json:"title"`
	Text      string         `json:"text"`
	Questions []questionView `json:"questions"`
}

// levelState — что сейчас на экране: вопрос (item) или текст Reading (passage). step — номер вопроса с 1.
type levelState struct {
	ID      int64             `json:"id"`
	Step    int               `json:"step"`
	Steps   int               `json:"steps"`
	Item    *levelItemView    `json:"item"`
	Passage *levelPassageView `json:"passage"`
}

func toLevelState(r level.Run) levelState {
	v := levelState{ID: r.AttemptID, Step: len(r.Items) + 1, Steps: level.CoreSteps + level.ReadingSteps}
	if it, ok := level.ItemByID(r.Current); ok {
		v.Item = &levelItemView{it.ID, it.Q, it.Options}
	} else if p, ok := level.PassageByID(r.Passage); ok {
		pv := levelPassageView{Title: p.Title, Text: p.Text}
		for _, q := range p.Questions {
			pv.Questions = append(pv.Questions, questionView{Q: q.Q, Options: q.Options})
		}
		v.Passage = &pv
	}
	return v
}

// levelReview — вопрос теста в разборе: ответ ученика и верный.
type levelReview struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
	Given   int      `json:"given"`
}

type levelPartView struct {
	Correct int `json:"correct"`
	Total   int `json:"total"`
}

// levelResultView — итог: балл, диапазон «скорее всего», CEFR, части и ошибки с верными ответами.
type levelResultView struct {
	Low      float64       `json:"low"`
	High     float64       `json:"high"`
	CEFR     string        `json:"cefr"`
	Core     levelPartView `json:"core"`
	Reading  levelPartView `json:"reading"`
	Passage  string        `json:"passage"`
	Mistakes []levelReview `json:"mistakes"`
}

func toLevelResult(r level.Run) levelResultView {
	v := levelResultView{Low: r.Low, High: r.High, CEFR: level.CEFR(r.Band), Mistakes: []levelReview{}}
	for _, st := range r.Items {
		it, ok := level.ItemByID(st.ID)
		if !ok {
			continue
		}
		v.Core.Total++
		if st.OK {
			v.Core.Correct++
		} else {
			v.Mistakes = append(v.Mistakes, levelReview{it.Q, it.Options, it.Answer, st.A})
		}
	}
	if p, ok := level.PassageByID(r.Passage); ok {
		v.Passage = p.Title
		for i, q := range p.Questions {
			given := -1
			if i < len(r.Reading) {
				given = r.Reading[i]
			}
			v.Reading.Total++
			if given == q.Answer {
				v.Reading.Correct++
			} else {
				v.Mistakes = append(v.Mistakes, levelReview{q.Q, q.Options, q.Answer, given})
			}
		}
	}
	return v
}

func levelError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, level.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
	case errors.Is(err, level.ErrFinished):
		writeError(w, http.StatusConflict, "finished")
	case errors.Is(err, level.ErrStage):
		writeError(w, http.StatusConflict, "stage")
	default:
		return false
	}
	return true
}

func (s *server) startLevel(w http.ResponseWriter, r *http.Request, u users.User) {
	run, err := s.level.Start(r.Context(), u.ID, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toLevelState(run))
}

func (s *server) answerLevel(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		Item   string `json:"item"`
		Answer int    `json:"answer"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	run, err := s.level.Answer(r.Context(), u.ID, id, b.Item, b.Answer)
	if levelError(w, err) {
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toLevelState(run))
}

func (s *server) finishLevel(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		Answers []int `json:"answers"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	run, err := s.level.Finish(r.Context(), u.ID, id, b.Answers, s.Now())
	if levelError(w, err) {
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.afterAttempt(r, u, exam.Attempt{ID: run.AttemptID, UserID: u.ID, Kind: exam.Level, Band: run.Band})
	writeJSON(w, http.StatusOK, map[string]int64{"id": run.AttemptID})
}

// addLevel — к попытке теста уровня — его итог; у попыток старого теста (без хода) итога нет.
func (s *server) addLevel(r *http.Request, u users.User, a exam.Attempt, v *resultView) error {
	run, err := s.level.Get(r.Context(), u.ID, a.ID)
	if errors.Is(err, level.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	lv := toLevelResult(run)
	v.Level = &lv
	v.Key, v.Marks, v.Answers = []any{}, []bool{}, []exam.Answer{}
	v.Test.Questions = []questionView{}
	return nil
}
