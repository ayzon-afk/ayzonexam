package level

import (
	"math"
	"math/rand/v2"
	"testing"
)

// run — весь тест для ученика настоящего уровня truth: 12 вопросов по ответам и текст под оценку.
func run(truth float64, rnd *rand.Rand) Result {
	used := map[string]bool{}
	var rs []Response
	last := ""
	for range CoreSteps {
		est, _ := Estimate(rs)
		it, ok := Next(used, last, est, rnd)
		if !ok {
			break
		}
		used[it.ID], last = true, it.Type
		rs = append(rs, Response{it.D, len(it.Options), rnd.Float64() < prob(truth, it.D, len(it.Options))})
	}
	est, _ := Estimate(rs)
	for _, q := range PassageFor(est).Questions {
		rs = append(rs, Response{q.D, len(q.Options), rnd.Float64() < prob(truth, q.D, len(q.Options))})
	}
	return Score(rs)
}

// Ученики уровней 3–8 по модели ответа: насколько близко тест называет их балл. Проверяет сам алгоритм —
// насколько верно размечена сложность вопросов, покажут ответы настоящих учеников.
func TestAccuracy(t *testing.T) {
	rnd := rand.New(rand.NewPCG(1, 2))
	for _, truth := range []float64{3, 4, 5, 5.5, 6, 6.5, 7, 8} {
		var errSum float64
		var within, covered int
		const n = 400
		for range n {
			r := run(truth, rnd)
			e := math.Abs(r.Band - truth)
			errSum += e
			if e <= 0.5 {
				within++
			}
			if r.Low <= truth && truth <= r.High {
				covered++
			}
		}
		mae := errSum / n
		t.Logf("уровень %.1f: средняя ошибка %.2f, в пределах 0.5 — %d%%, в диапазоне — %d%%", truth, mae, within*100/n, covered*100/n)
		if mae > 0.75 {
			t.Errorf("уровень %.1f: средняя ошибка %.2f больше 0.75", truth, mae)
		}
		if covered*100/n < 60 {
			t.Errorf("уровень %.1f: диапазон накрывает настоящий балл только в %d%%", truth, covered*100/n)
		}
	}
}

func TestScoreOrder(t *testing.T) {
	all := func(ok bool) []Response {
		var rs []Response
		for _, it := range Core {
			rs = append(rs, Response{it.D, len(it.Options), ok})
		}
		return rs
	}
	lo, hi := Score(all(false)), Score(all(true))
	if lo.Band > 2.5 || hi.Band < 8.5 {
		t.Fatalf("всё неверно — %.1f, всё верно — %.1f", lo.Band, hi.Band)
	}
	r := Score(nil)
	if r.Low > r.Band || r.Band > r.High || r.High-r.Low < 2 {
		t.Fatalf("без ответов диапазон должен быть широким: %+v", r)
	}
}

func TestNextAdapts(t *testing.T) {
	rnd := rand.New(rand.NewPCG(3, 4))
	easy, _ := Next(map[string]bool{}, "", 3.5, rnd)
	hard, _ := Next(map[string]bool{}, "", 8, rnd)
	if easy.D > 4.5 || hard.D < 7 {
		t.Fatalf("для 3.5 — вопрос %.1f, для 8 — %.1f", easy.D, hard.D)
	}
	used := map[string]bool{}
	for _, it := range Core {
		used[it.ID] = true
	}
	if _, ok := Next(used, "", 5, rnd); ok {
		t.Fatal("все вопросы заданы — следующего быть не должно")
	}
}

func TestCEFR(t *testing.T) {
	for band, want := range map[float64]string{9: "C2", 8.5: "C2", 7: "C1", 6.5: "B2", 5.5: "B2", 5: "B1", 4: "B1", 3.5: "A2", 2: "A1"} {
		if got := CEFR(band); got != want {
			t.Errorf("CEFR(%.1f) = %s, нужно %s", band, got, want)
		}
	}
}

func TestPassageFor(t *testing.T) {
	if p := PassageFor(2); p.Level != 4 {
		t.Fatalf("для 2.0 — текст %.1f", p.Level)
	}
	if p := PassageFor(6.2); p.Level != 6 {
		t.Fatalf("для 6.2 — текст %.1f", p.Level)
	}
	if p := PassageFor(9); p.Level != 8 {
		t.Fatalf("для 9.0 — текст %.1f", p.Level)
	}
}
