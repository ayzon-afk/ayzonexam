// Package ai — проверка работ искусственным интеллектом: распознавание рукописи и оценка Writing.
// Checker — интерфейс, чтобы тесты и локальная разработка шли без платного API (Fake).
package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Image struct {
	MIME string
	Data []byte
}

type WritingRequest struct {
	Task   string // task1 — Academic, отчёт по графику; task2 — эссе
	Prompt string // текст задания; для Task 1 может быть пустым — задание на фото
	Essay  string
	Words  int
	Lang   string // язык пояснений: uz, ru, en
	Image  *Image // фото задания Task 1
}

type Criterion struct {
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

type Mistake struct {
	Quote     string `json:"quote"`     // как написал ученик
	Fix       string `json:"fix"`       // как правильно
	Why       string `json:"why"`       // почему — на языке ученика
	Criterion string `json:"criterion"` // TA, CC, LR или GRA
}

// WritingResult — оценка по четырём критериям IELTS. TA — Task Achievement (Task 1) или Task Response (Task 2).
type WritingResult struct {
	TA        Criterion `json:"ta"`
	CC        Criterion `json:"cc"`
	LR        Criterion `json:"lr"`
	GRA       Criterion `json:"gra"`
	Summary   string    `json:"summary"`
	Mistakes  []Mistake `json:"mistakes"`
	Corrected string    `json:"corrected"`
	NextBand  []string  `json:"next_band"`
}

type Checker interface {
	// Recognize переписывает рукописный текст с фото как есть, не исправляя ошибок.
	Recognize(ctx context.Context, pages []Image) (string, error)
	GradeWriting(ctx context.Context, req WritingRequest) (WritingResult, error)
	Model() string
}

// ErrNoText — на фото не нашлось рукописного текста.
var ErrNoText = errors.New("на фото нет текста")

// Границы ответа: AI иногда пишет слишком много — лишнее обрезается, а не ломает разбор.
const (
	maxMistakes  = 15
	maxTips      = 5
	maxComment   = 1200
	maxShort     = 500
	maxCorrected = 8000
)

// normalize проверяет баллы и обрезает слишком длинное. Балл вне 0–9 — ответ негоден, проверка повторится.
func (r *WritingResult) normalize() error {
	for _, c := range []*Criterion{&r.TA, &r.CC, &r.LR, &r.GRA} {
		if c.Score < 0 || c.Score > 9 {
			return fmt.Errorf("балл критерия %d вне 0–9", c.Score)
		}
		c.Comment = clip(c.Comment, maxComment)
	}
	if r.TA.Comment == "" || r.Summary == "" {
		return errors.New("в ответе нет пояснений")
	}
	r.Summary = clip(r.Summary, maxComment)
	r.Corrected = clip(r.Corrected, maxCorrected)

	mistakes := r.Mistakes[:0]
	for _, m := range r.Mistakes {
		m.Criterion = strings.ToUpper(strings.TrimSpace(m.Criterion))
		switch m.Criterion {
		case "TA", "CC", "LR", "GRA":
		default:
			m.Criterion = "GRA"
		}
		if m.Quote = clip(m.Quote, maxShort); m.Quote == "" {
			continue
		}
		m.Fix, m.Why = clip(m.Fix, maxShort), clip(m.Why, maxShort)
		mistakes = append(mistakes, m)
		if len(mistakes) == maxMistakes {
			break
		}
	}
	r.Mistakes = mistakes

	tips := r.NextBand[:0]
	for _, t := range r.NextBand {
		if t = clip(t, maxShort); t != "" && len(tips) < maxTips {
			tips = append(tips, t)
		}
	}
	r.NextBand = tips
	return nil
}

func clip(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}
