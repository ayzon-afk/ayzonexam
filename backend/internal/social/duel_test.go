package social

import "testing"

func TestQuestions(t *testing.T) {
	syn := map[string]bool{}
	for _, w := range words {
		if syn[w[1]] {
			t.Fatalf("синоним %q повторяется — вариант ответа был бы двусмысленным", w[1])
		}
		syn[w[1]] = true
	}
	qs := newQuestions(DuelQuestions)
	seen := map[string]bool{}
	for _, q := range qs {
		if seen[q.Q] {
			t.Fatalf("слово %q дважды в одной дуэли", q.Q)
		}
		seen[q.Q] = true
		if len(q.Options) != 4 || q.Answer < 0 || q.Answer > 3 {
			t.Fatalf("вопрос %+v", q)
		}
		opts := map[string]bool{}
		for _, o := range q.Options {
			opts[o] = true
		}
		if len(opts) != 4 {
			t.Fatalf("варианты повторяются: %+v", q)
		}
	}
	for _, q := range Public(qs) {
		if q.Answer != -1 {
			t.Fatal("ключ ушёл бы ученику")
		}
	}
}
