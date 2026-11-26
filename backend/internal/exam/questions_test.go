package exam

import (
	"encoding/json"
	"strings"
	"testing"
)

// Тест канала: выбор, вписать, «выберите ДВА» в группе 1, буквы из общего списка группы 2.
func channelTest() Test {
	return Test{
		Groups: []Group{{Title: "Questions 3–4. Choose TWO letters", Pick: 2}, {Title: "Questions 5", Options: []string{"price", "date"}}},
		Questions: []Question{
			{Q: "TRUE?", Options: []string{"TRUE", "FALSE", "NOT GIVEN"}, Answer: 2},
			{Q: "Move ____ players", Kind: KindText, Accept: []string{"8:55", "8.55 a.m.", "354 Forest"}},
			{Q: "", Kind: KindMulti, Options: []string{"A", "B", "C", "D"}, Set: []int{0, 2}, Group: 1},
			{Q: "", Kind: KindMulti, Options: []string{"A", "B", "C", "D"}, Set: []int{0, 2}, Group: 1},
			{Q: "Films", Options: []string{"A", "B"}, Answer: 1, Group: 2},
		},
	}
}

func text(s string) Answer { return Answer{Choice: -1, Text: s, IsText: true} }

func TestGradeKinds(t *testing.T) {
	tt := channelTest()
	cases := []struct {
		name    string
		answers []Answer
		want    int
	}{
		{"всё верно, буквы в другом порядке", []Answer{{Choice: 2}, text("354 forest"), {Choice: 2}, {Choice: 0}, {Choice: 1}}, 5},
		{"регистр, пробелы, точка и кавычки", []Answer{NoAnswer, text("  “8.55  A.M.” ."), NoAnswer, NoAnswer, NoAnswer}, 1},
		{"одна буква дважды — один балл", []Answer{NoAnswer, text("forest"), {Choice: 0}, {Choice: 0}, NoAnswer}, 1},
		{"буква не из верных", []Answer{NoAnswer, NoAnswer, {Choice: 1}, {Choice: 2}, NoAnswer}, 1},
		{"пустое вписанное — без ответа", []Answer{NoAnswer, text("   ")}, 0},
	}
	for _, c := range cases {
		got, norm, err := Grade(tt, c.answers)
		if err != nil || got != c.want {
			t.Errorf("%s: %d верных (ждали %d), %v", c.name, got, c.want, err)
		}
		if err == nil && len(norm) != len(tt.Questions) {
			t.Errorf("%s: ответов %d", c.name, len(norm))
		}
	}
	_, norm, _ := Grade(tt, []Answer{NoAnswer, text("   ")})
	if norm[1] != NoAnswer {
		t.Errorf("пустое вписанное хранится как %+v", norm[1])
	}
	bad := map[string][]Answer{
		"текст на выбор":      {text("TRUE")},
		"номер на вписать":    {NoAnswer, {Choice: 0}},
		"длинный текст":       {NoAnswer, text(strings.Repeat("a", MaxText+1))},
		"буква вне вариантов": {NoAnswer, NoAnswer, {Choice: 4}},
	}
	for name, a := range bad {
		if _, _, err := Grade(tt, a); err != ErrAnswers {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAnswerJSON(t *testing.T) {
	var old []Answer
	if err := json.Unmarshal([]byte(`[-1, 2, null, "Forest"]`), &old); err != nil {
		t.Fatal(err)
	}
	want := []Answer{NoAnswer, {Choice: 2}, NoAnswer, text("Forest")}
	for i := range want {
		if old[i] != want[i] {
			t.Errorf("ответ %d: %+v", i, old[i])
		}
	}
	b, _ := json.Marshal(want)
	if string(b) != `[-1,2,-1,"Forest"]` {
		t.Errorf("в JSON: %s", b)
	}
}

func TestValidate(t *testing.T) {
	tt := channelTest()
	if err := Validate(tt.Questions, tt.Groups, nil); err != nil {
		t.Fatalf("верный тест: %v", err)
	}
	broken := func(change func(qs []Question, gs []Group) ([]Question, []Group)) error {
		qs := append([]Question(nil), tt.Questions...)
		gs := append([]Group(nil), tt.Groups...)
		qs, gs = change(qs, gs)
		return Validate(qs, gs, nil)
	}
	cases := map[string]func([]Question, []Group) ([]Question, []Group){
		"вписать без ответов":    func(q []Question, g []Group) ([]Question, []Group) { q[1].Accept = nil; return q, g },
		"вписать с вариантами":   func(q []Question, g []Group) ([]Question, []Group) { q[1].Options = []string{"a", "b"}; return q, g },
		"нет такой группы":       func(q []Question, g []Group) ([]Question, []Group) { q[4].Group = 3; return q, g },
		"выберите N не сходится": func(q []Question, g []Group) ([]Question, []Group) { g[0].Pick = 3; return q, g },
		"верная буква вне":       func(q []Question, g []Group) ([]Question, []Group) { q[2].Set = []int{9}; return q, g },
		"неизвестный вид":        func(q []Question, g []Group) ([]Question, []Group) { q[0].Kind = "drag"; return q, g },
		"ответ вне вариантов":    func(q []Question, g []Group) ([]Question, []Group) { q[0].Answer = 3; return q, g },
	}
	for name, change := range cases {
		if broken(change) == nil {
			t.Errorf("%s: прошло проверку", name)
		}
	}
}

func TestValidateParts(t *testing.T) {
	tt := channelTest() // 5 вопросов
	ok := []Part{{Title: "Part 1", Intro: "Listen and answer questions 1–2.", From: 1, To: 2}, {Title: "Part 2", From: 3, To: 5}}
	if err := Validate(tt.Questions, tt.Groups, ok); err != nil {
		t.Fatalf("верные части: %v", err)
	}
	bad := map[string][]Part{
		"дыра":            {{Title: "P1", From: 1, To: 2}, {Title: "P2", From: 4, To: 5}},
		"пересечение":     {{Title: "P1", From: 1, To: 3}, {Title: "P2", From: 3, To: 5}},
		"не с первого":    {{Title: "P1", From: 2, To: 5}},
		"не все вопросы":  {{Title: "P1", From: 1, To: 4}},
		"без заголовка":   {{Title: " ", From: 1, To: 5}},
		"больше четырёх":  {{Title: "1", From: 1, To: 1}, {Title: "2", From: 2, To: 2}, {Title: "3", From: 3, To: 3}, {Title: "4", From: 4, To: 4}, {Title: "5", From: 5, To: 5}},
		"конец до начала": {{Title: "P1", From: 1, To: 0}},
	}
	for name, ps := range bad {
		if Validate(tt.Questions, tt.Groups, ps) == nil {
			t.Errorf("%s: прошло проверку", name)
		}
	}
}
