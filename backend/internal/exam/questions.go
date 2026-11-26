package exam

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

// Виды вопросов (DECISIONS 2026-11-26): тесты канала «IELTS Volume» — не только выбор варианта.
const (
	KindChoice = "choice" // выбор варианта; "" — тоже выбор (задания до миграции 0022)
	KindText   = "text"   // вписать слово или число: верно, если совпало с одним из Accept
	KindMulti  = "multi"  // «выберите N букв»: вопросы группы делят Set, порядок не важен
)

// MaxText — самый длинный вписанный ответ (IELTS — до трёх слов и числа).
const MaxText = 60

// Group — общий блок нескольких вопросов: инструкция, текст-контекст (заметки, таблица), общий список
// вариантов с текстами (у вопросов — только буквы), картинка (план, карта), сколько выбрать для multi.
type Group struct {
	Title   string   `json:"title"`
	Text    string   `json:"text,omitempty"`
	Options []string `json:"options,omitempty"`
	Image   *int64   `json:"image,omitempty"`
	Pick    int      `json:"pick,omitempty"`
}

// Answer — ответ на вопрос. В JSON — число (номер варианта, -1 — без ответа) или строка (вписанное);
// старые попытки ([-1, 2, …]) читаются как есть.
type Answer struct {
	Choice int
	Text   string
	IsText bool
}

// NoAnswer — вопрос без ответа.
var NoAnswer = Answer{Choice: -1}

func (a Answer) MarshalJSON() ([]byte, error) {
	if a.IsText {
		return json.Marshal(a.Text)
	}
	return json.Marshal(a.Choice)
}

func (a *Answer) UnmarshalJSON(b []byte) error {
	*a = NoAnswer
	switch {
	case string(b) == "null":
		return nil
	case len(b) > 0 && b[0] == '"':
		a.IsText = true
		return json.Unmarshal(b, &a.Text)
	default:
		return json.Unmarshal(b, &a.Choice)
	}
}

// Answers — ответы из чисел (старые места и тесты).
func Answers(choices ...int) []Answer {
	out := make([]Answer, len(choices))
	for i, c := range choices {
		out[i] = Answer{Choice: c}
	}
	return out
}

func kindOf(q Question) string {
	if q.Kind == "" {
		return KindChoice
	}
	return q.Kind
}

// normalize — сравнение вписанного: регистр, пробелы, кавычки и точка по краям, тире и апострофы не важны.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("’", "'", "‘", "'", "–", "-", "—", "-").Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.Trim(s, ` .,;:!?"'“”«»`)
}

// clean проверяет ответ на вопрос q и приводит его к хранимому виду.
func clean(q Question, a Answer) (Answer, error) {
	if kindOf(q) == KindText {
		if !a.IsText {
			if a.Choice != -1 {
				return a, ErrAnswers
			}
			return NoAnswer, nil
		}
		t := strings.Join(strings.Fields(a.Text), " ")
		if t == "" {
			return NoAnswer, nil
		}
		if utf8.RuneCountInString(t) > MaxText {
			return a, ErrAnswers
		}
		return Answer{Choice: -1, Text: t, IsText: true}, nil
	}
	if a.IsText || a.Choice < -1 || a.Choice >= len(q.Options) {
		return a, ErrAnswers
	}
	return a, nil
}

// Marks — верен ли каждый ответ. answers — той же длины, что вопросы (после clean).
func Marks(t Test, answers []Answer) []bool {
	marks := make([]bool, len(t.Questions))
	used := map[int]map[int]bool{} // группа multi → уже засчитанные буквы
	for i, q := range t.Questions {
		a := answers[i]
		switch kindOf(q) {
		case KindText:
			if a.IsText {
				got := normalize(a.Text)
				for _, ok := range q.Accept {
					if got == normalize(ok) {
						marks[i] = true
						break
					}
				}
			}
		case KindMulti:
			if used[q.Group] == nil {
				used[q.Group] = map[int]bool{}
			}
			if a.Choice >= 0 && !used[q.Group][a.Choice] && inSet(q.Set, a.Choice) {
				used[q.Group][a.Choice] = true
				marks[i] = true
			}
		default:
			marks[i] = a.Choice >= 0 && a.Choice == q.Answer
		}
	}
	return marks
}

func inSet(set []int, v int) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// Validate — задание собрано правильно: виды вопросов, ключи, группы. Проверяет сохранение в админке и загрузку.
func Validate(qs []Question, groups []Group) error {
	if len(qs) == 0 || len(qs) > 60 {
		return errors.New("вопросов — от 1 до 60")
	}
	if len(groups) > 30 {
		return errors.New("групп — не больше 30")
	}
	for _, g := range groups {
		if strings.TrimSpace(g.Title) == "" || utf8.RuneCountInString(g.Title) > 500 || utf8.RuneCountInString(g.Text) > 4000 ||
			len(g.Options) > 15 || g.Pick < 0 || g.Pick > 5 {
			return errors.New("группа: заголовок, текст до 4000, вариантов до 15")
		}
		for _, o := range g.Options {
			if strings.TrimSpace(o) == "" || utf8.RuneCountInString(o) > 300 {
				return errors.New("группа: пустой или длинный вариант")
			}
		}
	}
	multi := map[int]int{}
	for n, q := range qs {
		bad := func(why string) error { return errors.New("вопрос " + itoa(n+1) + ": " + why) }
		if q.Group < 0 || q.Group > len(groups) {
			return bad("нет такой группы")
		}
		if utf8.RuneCountInString(q.Q) > 1000 {
			return bad("длинный текст")
		}
		for _, o := range q.Options {
			if strings.TrimSpace(o) == "" || utf8.RuneCountInString(o) > 300 {
				return bad("пустой или длинный вариант")
			}
		}
		switch kindOf(q) {
		case KindChoice:
			if strings.TrimSpace(q.Q) == "" || len(q.Options) < 2 || len(q.Options) > 15 || q.Answer < 0 || q.Answer >= len(q.Options) {
				return bad("выбор: текст, 2–15 вариантов, ответ среди них")
			}
		case KindText:
			if strings.TrimSpace(q.Q) == "" || len(q.Options) > 0 || len(q.Accept) == 0 || len(q.Accept) > 12 {
				return bad("вписать: текст и 1–12 допустимых ответов")
			}
			for _, ok := range q.Accept {
				if normalize(ok) == "" || utf8.RuneCountInString(ok) > MaxText {
					return bad("вписать: пустой или длинный допустимый ответ")
				}
			}
		case KindMulti:
			if q.Group == 0 || len(q.Options) < 2 || len(q.Options) > 15 || len(q.Set) == 0 {
				return bad("выберите N: группа, 2–15 вариантов, верные буквы")
			}
			for _, s := range q.Set {
				if s < 0 || s >= len(q.Options) {
					return bad("выберите N: верная буква вне вариантов")
				}
			}
			multi[q.Group]++
		default:
			return bad("неизвестный вид")
		}
	}
	for g, n := range multi {
		if groups[g-1].Pick != n {
			return errors.New("группа " + itoa(g) + ": «выберите N» не совпадает с числом вопросов")
		}
	}
	return nil
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
