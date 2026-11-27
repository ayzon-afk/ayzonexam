package api

import (
	"fmt"
	"strings"
	"testing"

	"ayzonexam/internal/level"
)

// Тест уровня (DECISIONS 2026-11-27): вопросы по одному без ключа, продолжение после закрытия, текст Reading
// после 12 вопросов, итог с диапазоном и CEFR, ошибки с верными ответами, чужой тест не виден.
func TestLevelAdaptive(t *testing.T) {
	h := newHandler(t, Deps{})
	strong, weak := initData(9301, ""), initData(9302, "")

	// play — весь тест: right решает, отвечать ли верно.
	play := func(auth string, right func(n int) bool) int64 {
		t.Helper()
		st := do(t, h, "POST", "/api/level", auth, "")
		if st.code != 200 || st.body["item"] == nil {
			t.Fatalf("старт: %d %s", st.code, st.raw)
		}
		if strings.Contains(st.raw, `"answer"`) || strings.Contains(st.raw, `"d"`) {
			t.Fatalf("ключ или сложность ушли ученику: %s", st.raw)
		}
		id := int64(st.body["id"].(float64))
		if again := do(t, h, "POST", "/api/level", auth, ""); int64(again.body["id"].(float64)) != id {
			t.Fatalf("незаконченный тест не продолжился: %s", again.raw)
		}
		for n := 0; n < level.CoreSteps; n++ {
			item := st.body["item"].(map[string]any)
			it, _ := level.ItemByID(item["id"].(string))
			a := it.Answer
			if !right(n) {
				a = (a + 1) % len(it.Options)
			}
			st = do(t, h, "POST", fmt.Sprintf("/api/level/%d/answer", id), auth, fmt.Sprintf(`{"item":%q,"answer":%d}`, it.ID, a))
			if st.code != 200 {
				t.Fatalf("ответ %d: %d %s", n, st.code, st.raw)
			}
			// Повторная отправка того же ответа (медленный интернет) ничего не меняет.
			if n == 0 {
				dup := do(t, h, "POST", fmt.Sprintf("/api/level/%d/answer", id), auth, fmt.Sprintf(`{"item":%q,"answer":0}`, it.ID))
				if dup.body["step"] != 2.0 {
					t.Fatalf("повтор засчитан: %s", dup.raw)
				}
			}
		}
		if st.body["item"] != nil || st.body["passage"] == nil || st.body["step"] != float64(level.CoreSteps+1) {
			t.Fatalf("после %d вопросов — не текст: %s", level.CoreSteps, st.raw)
		}
		if early := do(t, h, "POST", fmt.Sprintf("/api/level/%d/answer", id), auth, `{"item":"c01","answer":0}`); early.code != 200 || early.body["passage"] == nil {
			t.Fatalf("ответ на этапе текста: %d %s", early.code, early.raw)
		}
		title := st.body["passage"].(map[string]any)["title"].(string)
		var p level.Passage
		for _, x := range level.Passages {
			if x.Title == title {
				p = x
			}
		}
		answers := make([]string, len(p.Questions))
		for i, q := range p.Questions {
			a := q.Answer
			if !right(level.CoreSteps + i) {
				a = (a + 1) % len(q.Options)
			}
			answers[i] = fmt.Sprint(a)
		}
		fin := do(t, h, "POST", fmt.Sprintf("/api/level/%d/finish", id), auth, `{"answers":[`+strings.Join(answers, ",")+`]}`)
		if fin.code != 200 {
			t.Fatalf("сдача: %d %s", fin.code, fin.raw)
		}
		if again := do(t, h, "POST", fmt.Sprintf("/api/level/%d/finish", id), auth, `{"answers":[]}`); again.code != 409 {
			t.Fatalf("вторая сдача: %d %s", again.code, again.raw)
		}
		return id
	}

	top := play(strong, func(int) bool { return true })
	res := do(t, h, "GET", fmt.Sprintf("/api/attempts/%d", top), strong, "")
	lv, _ := res.body["level"].(map[string]any)
	if res.code != 200 || lv == nil || res.body["band"].(float64) < 8 || res.body["ranked"] != false {
		t.Fatalf("всё верно — итог: %d %s", res.code, res.raw)
	}
	if lv["cefr"] != "C2" && lv["cefr"] != "C1" || len(lv["mistakes"].([]any)) != 0 || lv["core"].(map[string]any)["correct"] != float64(level.CoreSteps) {
		t.Fatalf("всё верно — разбор: %s", res.raw)
	}
	if other := do(t, h, "GET", fmt.Sprintf("/api/attempts/%d", top), weak, ""); other.code != 404 {
		t.Fatalf("чужой тест виден: %d", other.code)
	}
	if other := do(t, h, "POST", fmt.Sprintf("/api/level/%d/finish", top), weak, `{"answers":[]}`); other.code != 404 {
		t.Fatalf("чужой тест сдан: %d", other.code)
	}

	// Слабый ученик: верно только первые три — балл низкий, в разборе ошибки с верным ответом.
	low := play(weak, func(n int) bool { return n < 3 })
	res = do(t, h, "GET", fmt.Sprintf("/api/attempts/%d", low), weak, "")
	lv = res.body["level"].(map[string]any)
	band := res.body["band"].(float64)
	if band > 4.5 || lv["low"].(float64) > band || lv["high"].(float64) < band {
		t.Fatalf("слабый ученик — итог: %s", res.raw)
	}
	m := lv["mistakes"].([]any)
	if len(m) != level.CoreSteps+level.ReadingSteps-3 || m[0].(map[string]any)["given"] == m[0].(map[string]any)["answer"] {
		t.Fatalf("ошибки в разборе: %s", res.raw)
	}
	if hist := do(t, h, "GET", "/api/history", weak, ""); !strings.Contains(hist.raw, `"kind":"level"`) {
		t.Fatalf("тест уровня не в истории: %s", hist.raw)
	}
}
