package api

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Mock как на экзамене (DECISIONS 2026-11-26): L → R → W подряд, таймер следующей части — от конца прошлой
// (+ минута на переход), балл и ключ частей — только в отчёте; mock до выкладки — по-старому.
func TestStrictMock(t *testing.T) {
	cur := now
	h := newHandler(t, Deps{Admins: []int64{adminTG}, Now: func() time.Time { return cur }})
	addAudio(t, h)
	auth := initData(9551, "")
	m := do(t, h, "POST", "/api/mock", auth, "")
	if m.code != 200 || m.body["strict"] != true {
		t.Fatalf("новый mock — строгий: %d %s", m.code, m.raw)
	}
	id := int64(m.body["id"].(float64))
	part := func(kind string) resp { return do(t, h, "POST", fmt.Sprintf("/api/mock/%d/part", id), auth, `{"kind":"`+kind+`"}`) }

	if r := part("reading"); r.code != 409 || r.body["error"] != "order" {
		t.Fatalf("Reading до Listening: %d %s", r.code, r.raw)
	}
	if r := do(t, h, "POST", fmt.Sprintf("/api/mock/%d/writing/start", id), auth, ""); r.code != 409 {
		t.Fatalf("Writing до Reading: %d %s", r.code, r.raw)
	}
	cur = now.Add(10 * time.Minute)
	l := solve(t, h, auth, fmt.Sprintf("/api/mock/%d/part", id), `{"kind":"listening"}`, 10)
	if l.code != 200 || l.body["hidden"] != true || strings.Contains(l.raw, `"band"`) || strings.Contains(l.raw, `"key"`) {
		t.Fatalf("балл Listening виден до конца mock: %s", l.raw)
	}
	if g := do(t, h, "GET", fmt.Sprintf("/api/attempts/%d", int64(l.body["id"].(float64))), auth, ""); g.code != 404 {
		t.Fatalf("попытка части видна до конца mock: %d", g.code)
	}
	if v := do(t, h, "GET", fmt.Sprintf("/api/mock/%d", id), auth, ""); !strings.Contains(v.raw, `"listening":{"state":"done","band":null`) {
		t.Fatalf("балл части в mock: %s", v.raw)
	}

	// Ученик ушёл на 30 минут: таймер Reading пошёл через минуту после конца Listening.
	listenedAt := cur
	cur = cur.Add(30 * time.Minute)
	r := part("reading")
	want := listenedAt.Add(time.Minute + 20*time.Minute) // задание Reading в тестах — 20 минут
	ends, _ := time.Parse(time.RFC3339, fmt.Sprint(r.body["ends_at"]))
	if r.code != 200 || !ends.Equal(want) {
		t.Fatalf("таймер Reading: %d %v, ждали %s", r.code, r.body["ends_at"], want)
	}

	// Mock до выкладки — по-старому: части в любом порядке, балл сразу.
	old := initData(9552, "")
	om := do(t, h, "POST", "/api/mock", old, "")
	oid := int64(om.body["id"].(float64))
	pool.Exec(t.Context(), `UPDATE mocks SET strict = false WHERE id = $1`, oid)
	or := solve(t, h, old, fmt.Sprintf("/api/mock/%d/part", oid), `{"kind":"reading"}`, 10)
	if or.code != 200 || or.body["band"] == nil {
		t.Fatalf("старый mock: %d %s", or.code, or.raw)
	}
}
