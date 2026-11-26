package exam

import "testing"

func TestBand(t *testing.T) {
	cases := []struct {
		kind           string
		correct, total int
		want           float64
	}{
		{Listening, 40, 40, 9}, {Listening, 30, 40, 7}, {Listening, 23, 40, 6}, {Listening, 0, 40, 0},
		{Reading, 30, 40, 7}, {Reading, 27, 40, 6.5}, {Reading, 23, 40, 6},
		{Reading, 10, 10, 9}, {Reading, 5, 10, 5.5}, // 5/10 → 20/40
		{Level, 15, 15, 9}, {Reading, 3, 0, 0},
	}
	for _, c := range cases {
		if got := Band(c.kind, c.correct, c.total); got != c.want {
			t.Errorf("Band(%s, %d/%d) = %.1f, ждали %.1f", c.kind, c.correct, c.total, got, c.want)
		}
	}
}

func TestOverall(t *testing.T) {
	cases := []struct {
		bands []float64
		want  float64
	}{
		{[]float64{6.5, 6.5, 5.0, 7.0}, 6.5}, // 6.25 → 6.5
		{[]float64{6.5, 6.5, 6.5, 7.5}, 7.0}, // 6.75 → 7.0
		{[]float64{6.0, 6.0, 6.0, 6.5}, 6.0}, // 6.125 → 6.0
		{[]float64{9, 9}, 9},
		{nil, 0},
	}
	for _, c := range cases {
		if got := Overall(c.bands...); got != c.want {
			t.Errorf("Overall(%v) = %.2f, ждали %.1f", c.bands, got, c.want)
		}
	}
}

func TestWritingBand(t *testing.T) {
	t1 := 6.0
	if got := WritingBand(&t1, 7); got != 6.5 { // (6 + 14) / 3 = 6.67 → 6.5
		t.Errorf("Task 2 вдвое тяжелее: %.1f", got)
	}
	if got := WritingBand(nil, 6.5); got != 6.5 {
		t.Errorf("без Task 1 — балл Task 2: %.1f", got)
	}
}

func TestGrade(t *testing.T) {
	test := Test{Questions: []Question{{Options: []string{"a", "b"}, Answer: 1}, {Options: []string{"a", "b"}, Answer: 0}}}
	c, norm, err := Grade(test, Answers(1))
	if err != nil || c != 1 || len(norm) != 2 || norm[1] != NoAnswer {
		t.Fatalf("недостающие ответы — без ответа: %d %v %v", c, norm, err)
	}
	if _, _, err := Grade(test, Answers(0, 0, 0)); err != ErrAnswers {
		t.Fatal("лишние ответы — ошибка")
	}
	if _, _, err := Grade(test, Answers(2)); err != ErrAnswers {
		t.Fatal("ответ вне вариантов — ошибка")
	}
}
