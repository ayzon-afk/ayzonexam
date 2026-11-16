package ai

import (
	"context"
	"fmt"
)

// Fake — заглушка вместо Gemini: локально без ключа и в тестах. Балл зависит только от длины текста.
type Fake struct{}

func (Fake) Model() string { return "fake" }

func (Fake) Recognize(_ context.Context, pages []Image) (string, error) {
	if len(pages) == 0 {
		return "", ErrNoText
	}
	return fmt.Sprintf("This text was recognised from %d photo(s) by the test checker.\n\n"+
		"Nowadays many people beleive that technology make our lifes easier. In this essay I will discuss both sides.", len(pages)), nil
}

func (Fake) GradeWriting(_ context.Context, req WritingRequest) (WritingResult, error) {
	min := 250
	if req.Task == "task1" {
		min = 150
	}
	ta := 6
	if req.Words < min {
		ta = 5
	}
	r := WritingResult{
		TA:        Criterion{ta, fmt.Sprintf("Test checker: %d words (minimum %d).", req.Words, min)},
		CC:        Criterion{6, "Test checker: paragraphs are logical."},
		LR:        Criterion{6, "Test checker: adequate range of vocabulary."},
		GRA:       Criterion{5, "Test checker: frequent grammar errors."},
		Summary:   "This is a test result — a real check needs GEMINI_API_KEY.",
		Mistakes:  []Mistake{{Quote: "make our lifes", Fix: "makes our lives", Why: "Subject–verb agreement and plural of life.", Criterion: "GRA"}},
		Corrected: req.Essay,
		NextBand:  []string{"Write a clear overview or position in the introduction.", "Use linking words between paragraphs."},
	}
	return r, r.normalize()
}
