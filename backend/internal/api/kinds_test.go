package api

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Тест канала (DECISIONS 2026-11-26): вписать, «выберите ДВА», буквы из общего списка — сохранение в админке,
// старт без ключа, сдача текстом, разбор с ключом и отметками.
func TestQuestionKinds(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	admin, auth := initData(adminTG, ""), initData(9251, "")
	body := `{"kind":"reading","title":"Volume 10 · Test 2 · Reading","minutes":20,"active":true,
		"parts":[{"title":"Passage 1","passage":"Siberian polecat","from":1,"to":3},{"title":"Passage 2","passage":"Text 2","from":4,"to":4}],
		"groups":[{"title":"Questions 2–3. Choose TWO letters","pick":2},{"title":"Questions 4","options":["price","date"]}],
		"questions":[
			{"q":"Move ____ players","kind":"text","accept":["cassette"],"type":"gap"},
			{"q":"","kind":"multi","options":["A","B","C"],"set":[0,2],"group":1,"type":"mcq"},
			{"q":"","kind":"multi","options":["A","B","C"],"set":[0,2],"group":1,"type":"mcq"},
			{"q":"Films","options":["A","B"],"answer":1,"group":2,"type":"matching"}]}`
	if r := do(t, h, "POST", "/api/admin/tests", admin, strings.Replace(body, `"pick":2`, `"pick":3`, 1)); r.code != 400 {
		t.Fatalf("«выберите N» не сходится — сохранилось: %d %s", r.code, r.raw)
	}
	if r := do(t, h, "POST", "/api/admin/tests", admin, strings.Replace(body, `"from":4,"to":4`, `"from":5,"to":5`, 1)); r.code != 400 {
		t.Fatalf("дыра в частях — сохранилось: %d %s", r.code, r.raw)
	}
	r := do(t, h, "POST", "/api/admin/tests", admin, body)
	if r.code != 200 {
		t.Fatalf("сохранение: %d %s", r.code, r.raw)
	}
	id := int64(r.body["id"].(float64))
	// t.Context() в Cleanup уже отменён; задание выключается, чтобы mock и турнир других тестов его не взяли.
	t.Cleanup(func() { pool.Exec(context.Background(), `UPDATE tests SET active = false WHERE id = $1`, id) })

	st := do(t, h, "POST", fmt.Sprintf("/api/tests/%d/start", id), auth, "")
	if st.code != 200 {
		t.Fatalf("старт: %d %s", st.code, st.raw)
	}
	for _, k := range []string{`"answer"`, `"accept"`, `"set"`, "cassette"} {
		if strings.Contains(st.raw, k) {
			t.Fatalf("ключ %s ушёл ученику до сдачи: %s", k, st.raw)
		}
	}
	if !strings.Contains(st.raw, `"parts":[{"title":"Passage 1","passage":"Siberian polecat","from":1,"to":3}`) {
		t.Fatalf("части не дошли: %s", st.raw)
	}
	if !strings.Contains(st.raw, `"kind":"text"`) || !strings.Contains(st.raw, `"pick":2`) || !strings.Contains(st.raw, `"options":["price","date"]`) {
		t.Fatalf("виды и группы не дошли: %s", st.raw)
	}
	aid := int64(st.body["attempt_id"].(float64))
	if bad := do(t, h, "POST", fmt.Sprintf("/api/attempts/%d/submit", aid), auth, `{"answers":[0]}`); bad.code != 400 {
		t.Fatalf("номер вместо текста: %d %s", bad.code, bad.raw)
	}
	res := do(t, h, "POST", fmt.Sprintf("/api/attempts/%d/submit", aid), auth, `{"answers":[" Cassette. ",2,2,1]}`)
	if res.code != 200 || res.body["correct"] != 3.0 {
		t.Fatalf("сдача: %d %s", res.code, res.raw)
	}
	if !strings.Contains(res.raw, `"answers":["Cassette.",2,2,1]`) || !strings.Contains(res.raw, `"marks":[true,true,false,true]`) ||
		!strings.Contains(res.raw, `"key":[["cassette"],[0,2],[0,2],1]`) {
		t.Fatalf("разбор: %s", res.raw)
	}
	if g := do(t, h, "GET", fmt.Sprintf("/api/attempts/%d", aid), auth, ""); g.code != 200 || !strings.Contains(g.raw, `"Cassette."`) {
		t.Fatalf("попытка из базы: %d %s", g.code, g.raw)
	}
}
