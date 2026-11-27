// Package level — тест уровня, который подстраивается под ученика (DECISIONS 2026-11-27): банк вопросов со
// сложностью в баллах IELTS, выбор следующего вопроса по ответам и оценка уровня с диапазоном.
//
// Модель ответа: вероятность верного ответа ученика уровня a на вопрос сложности d —
// g + (1−g) / (1 + e^(−slope·(a−d))), где g = 1/вариантов — угадать наугад. Сложность d — балл, на котором
// вопрос решают примерно наполовину сверх угадывания. Уровень — среднее апостериорного распределения на
// сетке баллов с широким априорным; диапазон — ± одно стандартное отклонение.
package level

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
)

// Item — вопрос по грамматике или словам.
type Item struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"` // grammar, vocab
	D       float64  `json:"d"`
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
}

// Passage — короткий текст Reading своего уровня и вопросы к нему.
type Passage struct {
	ID        string     `json:"id"`
	Level     float64    `json:"level"`
	Title     string     `json:"title"`
	Text      string     `json:"text"`
	Questions []Question `json:"questions"`
}

type Question struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
	D       float64  `json:"d"`
}

//go:embed bank.json
var bankJSON []byte

var (
	Core     []Item
	Passages []Passage
	byID     = map[string]Item{}
)

// CoreSteps — вопросов по грамматике и словам до текста Reading; ReadingSteps — вопросов к тексту.
const (
	CoreSteps    = 12
	ReadingSteps = 5
)

func init() {
	var b struct {
		Core     []Item    `json:"core"`
		Passages []Passage `json:"passages"`
	}
	if err := json.Unmarshal(bankJSON, &b); err != nil {
		panic("level: банк не читается: " + err.Error())
	}
	if err := check(b.Core, b.Passages); err != nil {
		panic("level: " + err.Error())
	}
	Core, Passages = b.Core, b.Passages
	for _, it := range Core {
		byID[it.ID] = it
	}
}

func check(core []Item, ps []Passage) error {
	seen := map[string]bool{}
	for _, it := range core {
		if it.ID == "" || seen[it.ID] {
			return fmt.Errorf("вопрос %q: пустой или повторный id", it.ID)
		}
		seen[it.ID] = true
		if len(it.Options) < 2 || it.Answer < 0 || it.Answer >= len(it.Options) || it.D < 1 || it.D > 9 {
			return fmt.Errorf("вопрос %s: варианты, ответ или сложность", it.ID)
		}
	}
	if len(core) < CoreSteps*2 {
		return fmt.Errorf("в банке %d вопросов — мало", len(core))
	}
	for _, p := range ps {
		if len(p.Questions) != ReadingSteps || p.Text == "" {
			return fmt.Errorf("текст %s: нужно %d вопросов", p.ID, ReadingSteps)
		}
		for _, q := range p.Questions {
			if len(q.Options) < 2 || q.Answer < 0 || q.Answer >= len(q.Options) || q.D < 1 || q.D > 9 {
				return fmt.Errorf("текст %s: вопрос %q", p.ID, q.Q)
			}
		}
	}
	if len(ps) == 0 {
		return fmt.Errorf("нет текстов Reading")
	}
	return nil
}

// ItemByID — вопрос банка; ok=false — такого нет (банк поменялся после попытки).
func ItemByID(id string) (Item, bool) {
	it, ok := byID[id]
	return it, ok
}

// PassageByID — текст банка по id.
func PassageByID(id string) (Passage, bool) {
	for _, p := range Passages {
		if p.ID == id {
			return p, true
		}
	}
	return Passage{}, false
}

// Response — ответ ученика: сложность вопроса, число вариантов, верно ли.
type Response struct {
	D       float64
	Options int
	OK      bool
}

const (
	slope     = 1.7 // насколько быстро растёт вероятность верного ответа с баллом
	priorMean = 5.0 // большинство готовящихся к IELTS — между 4 и 6.5
	priorSD   = 1.75
	gridLo    = 1.0
	gridHi    = 9.0
	gridStep  = 0.05
)

func prob(a, d float64, options int) float64 {
	g := 1 / float64(max(options, 2))
	return g + (1-g)/(1+math.Exp(-slope*(a-d)))
}

// Estimate — уровень по ответам: среднее и стандартное отклонение апостериорного распределения.
func Estimate(rs []Response) (mean, sd float64) {
	var sum, sum1, sum2 float64
	for a := gridLo; a <= gridHi+1e-9; a += gridStep {
		lp := -(a - priorMean) * (a - priorMean) / (2 * priorSD * priorSD)
		for _, r := range rs {
			p := prob(a, r.D, r.Options)
			if r.OK {
				lp += math.Log(p)
			} else {
				lp += math.Log(1 - p)
			}
		}
		w := math.Exp(lp)
		sum += w
		sum1 += w * a
		sum2 += w * a * a
	}
	mean = sum1 / sum
	return mean, math.Sqrt(math.Max(sum2/sum-mean*mean, 0))
}

// half — до ближайшей половины балла в пределах 1–9.
func half(x float64) float64 {
	return math.Min(math.Max(math.Round(x*2)/2, gridLo), gridHi)
}

// Result — уровень, который видит ученик: балл и диапазон «скорее всего от и до».
type Result struct {
	Band float64
	Low  float64
	High float64
}

// Score — итог по всем ответам теста.
func Score(rs []Response) Result {
	mean, sd := Estimate(rs)
	r := Result{Band: half(mean), Low: half(mean - sd), High: half(mean + sd)}
	r.Low = math.Min(r.Low, r.Band)
	r.High = math.Max(r.High, r.Band)
	return r
}

// CEFR — уровень по шкале CEFR для балла IELTS (таблица соответствия IELTS и CEFR).
func CEFR(band float64) string {
	switch {
	case band >= 8.5:
		return "C2"
	case band >= 7:
		return "C1"
	case band >= 5.5:
		return "B2"
	case band >= 4:
		return "B1"
	case band >= 3:
		return "A2"
	default:
		return "A1"
	}
}

// info — сколько вопрос сложности d говорит об ученике уровня a (информация Фишера модели с угадыванием).
func info(a float64, it Item) float64 {
	g := 1 / float64(len(it.Options))
	p := prob(a, it.D, len(it.Options))
	return slope * slope * (p - g) * (p - g) / ((1 - g) * (1 - g)) * (1 - p) / p
}

// Next — следующий вопрос: среди не заданных — один из трёх самых полезных при оценке est (чтобы у всех не
// был один и тот же порядок); если среди них есть другой вид, чем lastType, — он (грамматика и слова
// чередуются). ok=false — вопросы кончились.
func Next(used map[string]bool, lastType string, est float64, rnd *rand.Rand) (Item, bool) {
	var free []Item
	for _, it := range Core {
		if !used[it.ID] {
			free = append(free, it)
		}
	}
	if len(free) == 0 {
		return Item{}, false
	}
	sort.SliceStable(free, func(i, j int) bool { return info(est, free[i]) > info(est, free[j]) })
	top := free[:min(3, len(free))]
	var other []Item
	for _, it := range top {
		if it.Type != lastType {
			other = append(other, it)
		}
	}
	if len(other) > 0 {
		top = other
	}
	return top[rnd.IntN(len(top))], true
}

// PassageFor — текст Reading, ближайший по уровню к оценке est; при равенстве — более лёгкий.
func PassageFor(est float64) Passage {
	best := Passages[0]
	for _, p := range Passages[1:] {
		if math.Abs(p.Level-est) < math.Abs(best.Level-est) {
			best = p
		}
	}
	return best
}
