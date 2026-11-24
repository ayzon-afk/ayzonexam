package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/bot"
	"ayzonexam/internal/speaking"
	"ayzonexam/internal/tgauth"
	"ayzonexam/internal/writing"
)

// Тесты этапов шага 1: задания, рейтинг и центры, турнир, mock, дуэли, приглашения, новости,
// поддержка, оплата, Speaking, бот. Каждый тест — свои Telegram id: база общая на пакет.

const adminTG = 777000

// clock — изменяемое время сервера.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// initDataAt — вход, подписанный в момент at (для тестов со сдвинутыми часами).
func initDataAt(tgID int64, at time.Time) string {
	user := `{"id":` + strconv.FormatInt(tgID, 10) + `,"first_name":"Ali","language_code":"ru"}`
	return tgauth.Sign(url.Values{"user": {user}, "auth_date": {strconv.FormatInt(at.Unix(), 10)}}, fakeBot)
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, _ := json.Marshal(v)
	return string(b)
}

// testIDs — id заданий по виду.
func testID(t *testing.T, h *Server, auth, kind string) int64 {
	t.Helper()
	r := do(t, h, "GET", "/api/tests?kind="+kind, auth, "")
	list, _ := r.body["tests"].([]any)
	if r.code != 200 || len(list) == 0 {
		t.Fatalf("задания %s: %d %s", kind, r.code, r.raw)
	}
	return int64(list[0].(map[string]any)["id"].(float64))
}

// key — ключ ответов задания прямо из базы.
func key(t *testing.T, testID int64) []int {
	t.Helper()
	var raw []byte
	if err := pool.QueryRow(t.Context(), `SELECT questions FROM tests WHERE id = $1`, testID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var qs []struct{ Answer int }
	json.Unmarshal(raw, &qs)
	out := make([]int, len(qs))
	for i, q := range qs {
		out[i] = q.Answer
	}
	return out
}

// solve начинает и сдаёт задание с correct верными ответами.
func solve(t *testing.T, h *Server, auth string, startPath, startBody string, correct int) resp {
	t.Helper()
	st := do(t, h, "POST", startPath, auth, startBody)
	if st.code != 200 {
		t.Fatalf("старт %s: %d %s", startPath, st.code, st.raw)
	}
	if strings.Contains(st.raw, `"answer"`) {
		t.Fatalf("ключ ушёл ученику до сдачи: %s", st.raw)
	}
	test := st.body["test"].(map[string]any)
	k := key(t, int64(test["id"].(float64)))
	answers := make([]int, len(k))
	for i := range k {
		answers[i] = k[i]
		if i >= correct {
			answers[i] = (k[i] + 1) % 3
		}
	}
	aid := int64(st.body["attempt_id"].(float64))
	return do(t, h, "POST", fmt.Sprintf("/api/attempts/%d/submit", aid), auth, jsonOf(t, map[string]any{"answers": answers, "leaves": 0}))
}

// addAudio — аудио Listening через админку (иначе Listening ученикам не виден).
func addAudio(t *testing.T, h *Server) {
	t.Helper()
	admin := initDataAt(adminTG, h.Now())
	var id int64
	pool.QueryRow(t.Context(), `SELECT id FROM tests WHERE kind = 'listening' ORDER BY id LIMIT 1`).Scan(&id)
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/admin/tests/%d/audio", id), bytes.NewReader(append([]byte("OggS"), make([]byte, 3000)...)))
	req.Header.Set("Authorization", "tma "+admin)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("аудио Listening: %d %s", rec.Code, rec.Body.String())
	}
}

func TestTestsFlow(t *testing.T) {
	h := newHandler(t, Deps{})
	auth, other := initData(9201, ""), initData(9202, "")
	if r := do(t, h, "GET", "/api/tests?kind=listening", auth, ""); r.code != 200 || !strings.Contains(r.raw, `"tests":[]`) && !strings.Contains(r.raw, `"kind":"listening"`) {
		t.Fatalf("Listening: %d %s", r.code, r.raw)
	}
	id := testID(t, h, auth, "reading")
	r := solve(t, h, auth, fmt.Sprintf("/api/tests/%d/start", id), "", 10)
	if r.code != 200 || r.body["band"] != 9.0 || r.body["ranked"] != true || r.body["correct"] != 10.0 {
		t.Fatalf("Reading все верно: %d %s", r.code, r.raw)
	}
	aid := int64(r.body["id"].(float64))
	if again := do(t, h, "POST", fmt.Sprintf("/api/attempts/%d/submit", aid), auth, `{"answers":[]}`); again.code != 409 {
		t.Fatalf("вторая сдача: %d %s", again.code, again.raw)
	}
	if g := do(t, h, "GET", fmt.Sprintf("/api/attempts/%d", aid), other, ""); g.code != 404 {
		t.Fatalf("чужая попытка видна: %d", g.code)
	}
	lvl := solve(t, h, auth, fmt.Sprintf("/api/tests/%d/start", testID(t, h, auth, "level")), "", 8)
	if lvl.code != 200 || lvl.body["ranked"] != false {
		t.Fatalf("тест уровня не идёт в рейтинг: %s", lvl.raw)
	}
	st := do(t, h, "POST", fmt.Sprintf("/api/tests/%d/start", id), auth, "")
	bad := do(t, h, "POST", fmt.Sprintf("/api/attempts/%d/submit", int64(st.body["attempt_id"].(float64))), auth, `{"answers":[9]}`)
	if bad.code != 400 || bad.body["error"] != "bad_answers" {
		t.Fatalf("ответ вне вариантов: %d %s", bad.code, bad.raw)
	}
}

func TestLateAttemptNotRanked(t *testing.T) {
	c := &clock{t: now}
	h := newHandler(t, Deps{Now: c.now})
	auth := initData(9203, "")
	id := testID(t, h, auth, "reading")
	st := do(t, h, "POST", fmt.Sprintf("/api/tests/%d/start", id), auth, "")
	c.add(25 * time.Minute) // таймер 20 минут + 90 с запаса
	r := do(t, h, "POST", fmt.Sprintf("/api/attempts/%d/submit", int64(st.body["attempt_id"].(float64))), auth, `{"answers":[0],"leaves":0}`)
	if r.code != 200 || r.body["ranked"] != false {
		t.Fatalf("опоздавшая сдача не в рейтинг: %s", r.raw)
	}
	st = do(t, h, "POST", fmt.Sprintf("/api/tests/%d/start", id), auth, "")
	r = do(t, h, "POST", fmt.Sprintf("/api/attempts/%d/submit", int64(st.body["attempt_id"].(float64))), auth, `{"answers":[0],"leaves":3}`)
	if r.body["ranked"] != false {
		t.Fatalf("три ухода из приложения — не в рейтинг: %s", r.raw)
	}
}

func TestRatingAndCenters(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	admin := initData(adminTG, "")
	if r := do(t, h, "POST", "/api/admin/centers", initData(9301, ""), `{"name":"X","city":"tashkent"}`); r.code != 404 {
		t.Fatalf("не админ создал центр: %d", r.code)
	}
	c := do(t, h, "POST", "/api/admin/centers", admin, `{"name":"Test Center 93","city":"tashkent","about":"a"}`)
	if c.code != 200 {
		t.Fatalf("центр: %d %s", c.code, c.raw)
	}
	cid := int64(c.body["id"].(float64))
	var code string
	pool.QueryRow(t.Context(), `SELECT invite_code FROM centers WHERE id = $1`, cid).Scan(&code)

	a, b := initData(9302, ""), initData(9303, "")
	for i, auth := range []string{a, b} {
		if r := do(t, h, "POST", "/api/centers/join", auth, `{"code":"`+code+`"}`); r.code != 200 || !strings.Contains(r.raw, `"status":"member"`) {
			t.Fatalf("вход по ссылке %d: %d %s", i, r.code, r.raw)
		}
		solve(t, h, auth, fmt.Sprintf("/api/tests/%d/start", testID(t, h, auth, "reading")), "", 10-i*3)
	}
	r := do(t, h, "GET", "/api/rating?scope=center", a, "")
	rows, _ := r.body["rows"].([]any)
	if r.code != 200 || len(rows) != 2 || r.body["total"] != 2.0 || r.body["mine"] == nil {
		t.Fatalf("рейтинг центра: %d %s", r.code, r.raw)
	}
	first := rows[0].(map[string]any)
	if first["band"] != 9.0 || first["me"] != true || first["center"] != "Test Center 93" {
		t.Fatalf("первое место: %v", first)
	}
	for _, leak := range []string{"Ali", "ali_tg", "9302", "telegram"} {
		if strings.Contains(r.raw, leak) {
			t.Fatalf("в рейтинге личное %q: %s", leak, r.raw)
		}
	}
	if city := do(t, h, "GET", "/api/rating?scope=city", b, ""); !strings.Contains(city.raw, `"me":true`) {
		t.Fatalf("город берётся у центра: %s", city.raw)
	}
	cr := do(t, h, "GET", "/api/rating/centers", a, "")
	if !strings.Contains(cr.raw, `"name":"Test Center 93"`) || !strings.Contains(cr.raw, `"scored":2`) {
		t.Fatalf("рейтинг центров: %s", cr.raw)
	}
	page := do(t, h, "GET", fmt.Sprintf("/api/centers/%d", cid), a, "")
	if page.code != 200 || page.body["status"] != "member" {
		t.Fatalf("страница центра: %s", page.raw)
	}

	// Сам выбрал центр — ждёт подтверждения; аккаунт центра подтверждает.
	pend := initData(9304, "")
	if r := do(t, h, "POST", fmt.Sprintf("/api/centers/%d/request", cid), pend, ""); !strings.Contains(r.raw, `"status":"pending"`) {
		t.Fatalf("просьба в центр: %s", r.raw)
	}
	if r := do(t, h, "GET", "/api/center/requests", a, ""); r.code != 403 {
		t.Fatalf("ученик не видит заявки: %d", r.code)
	}
	var aid int64
	pool.QueryRow(t.Context(), `SELECT id FROM users WHERE telegram_id = 9302`).Scan(&aid)
	do(t, h, "PATCH", fmt.Sprintf("/api/admin/users/%d", aid), admin, `{"role":"admin"}`)
	reqs := do(t, h, "GET", "/api/center/requests", a, "")
	list, _ := reqs.body["requests"].([]any)
	if len(list) != 1 || strings.Contains(reqs.raw, "Ali") {
		t.Fatalf("заявки центру: %s", reqs.raw)
	}
	uid := int64(list[0].(map[string]any)["user_id"].(float64))
	if r := do(t, h, "POST", fmt.Sprintf("/api/center/requests/%d", uid), a, `{"accept":true}`); r.code != 204 {
		t.Fatalf("подтверждение: %d %s", r.code, r.raw)
	}
	if me := do(t, h, "GET", "/api/me", pend, ""); !strings.Contains(me.raw, `"status":"member"`) {
		t.Fatalf("после подтверждения: %s", me.raw)
	}
}

func TestTournament(t *testing.T) {
	// 29.11.2026 — последнее воскресенье ноября; турнир 05:00–08:00 UTC.
	c := &clock{t: time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)}
	h := newHandler(t, Deps{Now: c.now, Admins: []int64{adminTG}})
	addAudio(t, h)
	auth := func() string { return initDataAt(9401, c.now()) }
	if r := do(t, h, "POST", "/api/tournament/start", auth(), `{"kind":"reading"}`); r.code != 409 {
		t.Fatalf("до начала: %d %s", r.code, r.raw)
	}
	if r := do(t, h, "POST", "/api/tournament/join", auth(), ""); r.code != 204 {
		t.Fatalf("запись: %d %s", r.code, r.raw)
	}
	g := do(t, h, "GET", "/api/tournament", auth(), "")
	cur := g.body["current"].(map[string]any)
	if cur["joined"] != true || cur["starts_at"] != "2026-11-29T05:00:00Z" || cur["players"] != 1.0 {
		t.Fatalf("турнир: %s", g.raw)
	}
	c.add(17*time.Hour + 30*time.Minute) // 29.11 05:30
	if r := do(t, h, "POST", "/api/tournament/start", initDataAt(9402, c.now()), `{"kind":"reading"}`); r.code != 403 {
		t.Fatalf("без записи: %d", r.code)
	}
	for _, k := range []string{"reading", "listening"} {
		if r := solve(t, h, auth(), "/api/tournament/start", `{"kind":"`+k+`"}`, 10); r.code != 200 || r.body["ranked"] != true {
			t.Fatalf("часть %s: %s", k, r.raw)
		}
	}
	if r := do(t, h, "POST", "/api/tournament/start", auth(), `{"kind":"reading"}`); r.code != 409 || r.body["error"] != "part_done" {
		t.Fatalf("часть дважды: %d %s", r.code, r.raw)
	}
	c.add(3 * time.Hour) // турнир кончился
	g = do(t, h, "GET", "/api/tournament", auth(), "")
	last := g.body["last"]
	if last == nil {
		// Сразу после окончания окно ещё текущего месяца: итоги — в current до 1-го числа нет, проверяем через last/current.
		last = g.body["current"]
	}
	if !strings.Contains(g.raw, `"place":1`) || !strings.Contains(g.raw, `"band":9`) {
		t.Fatalf("итоги турнира: %s", g.raw)
	}
	tours := do(t, h, "GET", "/api/admin/tournaments", initDataAt(adminTG, c.now()), "")
	tl := tours.body["tournaments"].([]any)
	tid := int64(tl[0].(map[string]any)["id"].(float64))
	var uid int64
	pool.QueryRow(t.Context(), `SELECT id FROM users WHERE telegram_id = 9401`).Scan(&uid)
	if r := do(t, h, "POST", fmt.Sprintf("/api/admin/tournaments/%d", tid), initDataAt(adminTG, c.now()), fmt.Sprintf(`{"champion":%d}`, uid)); r.code != 200 {
		t.Fatalf("чемпион: %d %s", r.code, r.raw)
	}
	g = do(t, h, "GET", "/api/tournament", auth(), "")
	if w, _ := g.body["winners"].([]any); len(w) == 0 || w[0].(map[string]any)["band"] != 9.0 {
		t.Fatalf("чемпион на главной: %s", g.raw)
	}
	_ = last
}

// queues — очереди Writing и Speaking на заглушке AI, как в cmd/server.
func queues(h *Server) (*writing.Queue, *speaking.Queue) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	wq := &writing.Queue{Store: h.Writing, AI: ai.Fake{}, Log: log, Now: h.Now,
		OnDone: func(ctx context.Context, id int64) { h.AfterCheck(ctx, "writing_checks", id) }}
	sq := &speaking.Queue{Store: h.Speaking(), AI: ai.Fake{}, Log: log, Now: h.Now,
		OnDone: func(ctx context.Context, id int64) { h.AfterCheck(ctx, "speaking_checks", id) }}
	return wq, sq
}

// webm — минимальные байты записи из браузера (EBML-заголовок).
var webm = append([]byte("\x1a\x45\xdf\xa3"), make([]byte, 30_000)...)

func TestMockFlow(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	addAudio(t, h)
	auth := initData(9501, "")
	m := do(t, h, "POST", "/api/mock", auth, "")
	if m.code != 200 || m.body["status"] != "active" {
		t.Fatalf("mock: %d %s", m.code, m.raw)
	}
	id := int64(m.body["id"].(float64))
	if again := do(t, h, "POST", "/api/mock", auth, ""); again.code != 409 {
		t.Fatalf("второй незаконченный mock: %d", again.code)
	}
	for _, k := range []string{"listening", "reading"} {
		if r := solve(t, h, auth, fmt.Sprintf("/api/mock/%d/part", id), `{"kind":"`+k+`"}`, 10); r.code != 200 || r.body["mock_id"] != float64(id) {
			t.Fatalf("часть %s: %s", k, r.raw)
		}
	}
	ws := do(t, h, "POST", fmt.Sprintf("/api/mock/%d/writing/start", id), auth, "")
	if ws.code != 200 || ws.body["task2"] == nil {
		t.Fatalf("Writing mock: %d %s", ws.code, ws.raw)
	}
	if r := do(t, h, "POST", fmt.Sprintf("/api/mock/%d/writing", id), auth, `{"task2":"short"}`); r.code != 400 || r.body["error"] != "too_short" {
		t.Fatalf("короткий ответ: %s", r.raw)
	}
	if r := do(t, h, "POST", fmt.Sprintf("/api/mock/%d/writing", id), auth, jsonOf(t, map[string]string{"task2": words(260)})); r.code != 200 {
		t.Fatalf("Writing отправлен: %d %s", r.code, r.raw)
	}
	wq, sq := queues(h)
	for {
		did, err := wq.Step(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !did {
			break
		}
	}
	sp := do(t, h, "POST", fmt.Sprintf("/api/mock/%d/speaking/start", id), auth, "")
	qs, _ := sp.body["questions"].([]any)
	if sp.code != 200 || len(qs) != 3 {
		t.Fatalf("Speaking mock: %d %s", sp.code, sp.raw)
	}
	files := []file{{"audio0", webm}, {"audio1", webm}, {"audio2", webm}}
	if r := form(t, h, fmt.Sprintf("/api/mock/%d/speaking", id), auth, map[string]string{"seconds": "300"}, files...); r.code != 200 {
		t.Fatalf("записи Speaking: %d %s", r.code, r.raw)
	}
	for {
		did, err := sq.Step(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !did {
			break
		}
	}
	rep := do(t, h, "GET", fmt.Sprintf("/api/mock/%d", id), auth, "")
	plan, _ := rep.body["plan"].([]any)
	if rep.body["status"] != "done" || rep.body["overall"] == nil || len(plan) != 4 {
		t.Fatalf("отчёт mock: %s", rep.raw)
	}
	// L 9.0, R 9.0, W 6.0 (заглушка: 6,6,6,5 → 5.75 → 6.0), S 6.0 (6,6,5,6) → 7.5
	if rep.body["overall"] != 7.5 {
		t.Fatalf("Overall: %v", rep.body["overall"])
	}
	stats := do(t, h, "GET", "/api/me/stats", auth, "")
	if stats.body["current"] != 7.5 || stats.body["parts"] != 4.0 {
		t.Fatalf("текущий балл из mock: %s", stats.raw)
	}
}

func TestMockNeedsPayment(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9502, "")
	do(t, h, "GET", "/api/me", auth, "")
	pool.Exec(t.Context(), `UPDATE users SET premium_until = '2026-01-01' WHERE telegram_id = 9502`)
	if r := do(t, h, "POST", "/api/mock", auth, ""); r.code != 402 {
		t.Fatalf("без Premium и покупки: %d %s", r.code, r.raw)
	}
	pool.Exec(t.Context(), `UPDATE users SET mock_credits = 1 WHERE telegram_id = 9502`)
	if r := do(t, h, "POST", "/api/mock", auth, ""); r.code != 200 {
		t.Fatalf("купленный mock: %d %s", r.code, r.raw)
	}
	var left int
	pool.QueryRow(t.Context(), `SELECT mock_credits FROM users WHERE telegram_id = 9502`).Scan(&left)
	if left != 0 {
		t.Fatalf("купленный mock не списан: %d", left)
	}
}

func TestDuelAndFriends(t *testing.T) {
	c := &clock{t: now}
	h := newHandler(t, Deps{Now: c.now})
	a, b, x := initData(9601, ""), initData(9602, ""), initData(9603, "")
	d := do(t, h, "POST", "/api/duels", a, `{}`)
	if d.code != 200 || d.body["state"] != "open" {
		t.Fatalf("вызов: %d %s", d.code, d.raw)
	}
	id := int64(d.body["id"].(float64))
	code := d.body["code"].(string)
	if v := do(t, h, "GET", "/api/duels/code/"+code, b, ""); v.code != 200 || v.body["mine"] != false {
		t.Fatalf("дуэль по ссылке: %s", v.raw)
	}
	var keys []int
	var raw []byte
	pool.QueryRow(t.Context(), `SELECT questions FROM duels WHERE id = $1`, id).Scan(&raw)
	var qs []struct{ Answer int }
	json.Unmarshal(raw, &qs)
	for _, q := range qs {
		keys = append(keys, q.Answer)
	}
	for _, auth := range []string{a, b} {
		p := do(t, h, "POST", fmt.Sprintf("/api/duels/%d/play", id), auth, "")
		if p.code != 200 || strings.Contains(p.raw, `"answer":0`) || strings.Contains(p.raw, `"answer":1`) {
			t.Fatalf("старт дуэли без ключа: %d %s", p.code, p.raw)
		}
	}
	if p := do(t, h, "POST", fmt.Sprintf("/api/duels/%d/play", id), x, ""); p.code != 409 {
		t.Fatalf("третий игрок: %d", p.code)
	}
	if r := do(t, h, "POST", fmt.Sprintf("/api/duels/%d/answer", id), a, `{"i":3,"choice":0}`); r.code != 409 || r.body["error"] != "order" {
		t.Fatalf("ответ не по порядку: %d %s", r.code, r.raw)
	}
	for i := range 5 {
		r := do(t, h, "POST", fmt.Sprintf("/api/duels/%d/answer", id), a, fmt.Sprintf(`{"i":%d,"choice":%d}`, i, keys[i]))
		if r.body["correct"] != true || r.body["score"] != float64(i+1) {
			t.Fatalf("ответ %d: %s", i, r.raw)
		}
	}
	do(t, h, "POST", fmt.Sprintf("/api/duels/%d/answer", id), b, fmt.Sprintf(`{"i":0,"choice":%d}`, keys[0]))
	c.add(6 * time.Minute)
	if r := do(t, h, "POST", fmt.Sprintf("/api/duels/%d/answer", id), a, `{"i":5,"choice":0}`); r.code != 409 || r.body["error"] != "over" {
		t.Fatalf("после 5 минут: %d %s", r.code, r.raw)
	}
	v := do(t, h, "GET", fmt.Sprintf("/api/duels/%d", id), a, "")
	if v.body["state"] != "done" || v.body["challenger"].(map[string]any)["score"] != 5.0 || v.body["opponent"].(map[string]any)["score"] != 1.0 {
		t.Fatalf("итог: %s", v.raw)
	}
	f := do(t, h, "GET", "/api/friends", a, "")
	if list, _ := f.body["friends"].([]any); len(list) != 1 {
		t.Fatalf("принявший вызов — друг: %s", f.raw)
	}
	var bid int64
	pool.QueryRow(t.Context(), `SELECT id FROM users WHERE telegram_id = 9602`).Scan(&bid)
	if r := do(t, h, "POST", "/api/duels", x, fmt.Sprintf(`{"friend_id":%d}`, bid)); r.code != 403 {
		t.Fatalf("вызов не другу напрямую: %d", r.code)
	}
}

func TestReferralBonus(t *testing.T) {
	h := newHandler(t, Deps{})
	a, b := initData(9701, ""), initData(9702, "")
	me := do(t, h, "GET", "/api/me", a, "")
	code := me.body["invite"].(string)
	if r := do(t, h, "POST", "/api/referral", a, `{"code":"`+code+`"}`); r.code != 400 {
		t.Fatalf("своя ссылка: %d", r.code)
	}
	if r := do(t, h, "POST", "/api/referral", b, `{"code":"`+code+`"}`); r.code != 200 || r.body["referred"] != true {
		t.Fatalf("приглашение: %d %s", r.code, r.raw)
	}
	solve(t, h, b, fmt.Sprintf("/api/tests/%d/start", testID(t, h, b, "level")), "", 5)
	solve(t, h, b, fmt.Sprintf("/api/tests/%d/start", testID(t, h, b, "level")), "", 5)
	var bonus int
	pool.QueryRow(t.Context(), `SELECT bonus_checks FROM users WHERE telegram_id = 9701`).Scan(&bonus)
	if bonus != 1 {
		t.Fatalf("бонус за тест уровня друга — один раз: %d", bonus)
	}
}

func TestNewsSupportAdmin(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	admin, user := initData(adminTG, ""), initData(9801, "")
	if r := do(t, h, "GET", "/api/admin/overview", user, ""); r.code != 404 {
		t.Fatalf("админка ученику: %d", r.code)
	}
	if r := do(t, h, "GET", "/api/admin/overview", admin, ""); r.code != 200 || r.body["users"] == nil {
		t.Fatalf("обзор: %d %s", r.code, r.raw)
	}
	p := do(t, h, "POST", "/api/admin/news", admin, `{"tag":"tip","title":"Совет","body":"Пишите 250+ слов"}`)
	if p.code != 200 {
		t.Fatalf("пост: %d %s", p.code, p.raw)
	}
	pid := int64(p.body["id"].(float64))
	if r := do(t, h, "GET", fmt.Sprintf("/api/news/%d", pid), user, ""); r.body["title"] != "Совет" {
		t.Fatalf("новость: %s", r.raw)
	}
	do(t, h, "PUT", fmt.Sprintf("/api/admin/news/%d", pid), admin, `{"tag":"tip","title":"Совет","body":"Пишите 250+ слов","hidden":true}`)
	if r := do(t, h, "GET", fmt.Sprintf("/api/news/%d", pid), user, ""); r.code != 404 {
		t.Fatalf("скрытый пост виден: %d", r.code)
	}
	s := form(t, h, "/api/support", user, map[string]string{"text": "Не открывается Listening"}, file{"photo", jpeg})
	if s.code != 200 {
		t.Fatalf("обращение: %d %s", s.code, s.raw)
	}
	tid := int64(s.body["id"].(float64))
	if r := do(t, h, "POST", fmt.Sprintf("/api/admin/tickets/%d", tid), admin, `{"reply":"Обновите приложение"}`); r.body["status"] != "answered" {
		t.Fatalf("ответ: %s", r.raw)
	}
	mine := do(t, h, "GET", "/api/support", user, "")
	if !strings.Contains(mine.raw, "Обновите приложение") || !strings.Contains(mine.raw, `"from_admin":true`) {
		t.Fatalf("ученик видит ответ: %s", mine.raw)
	}
	log := do(t, h, "GET", "/api/admin/log", admin, "")
	for _, a := range []string{"news.create", "news.update", "ticket.reply"} {
		if !strings.Contains(log.raw, a) {
			t.Fatalf("в журнале нет %s: %s", a, log.raw)
		}
	}
	// Закрытые файлы — только админке.
	var mid int64
	pool.QueryRow(t.Context(), `SELECT media_id FROM ticket_messages WHERE ticket_id = $1 AND media_id IS NOT NULL`, tid).Scan(&mid)
	if r := do(t, h, "GET", fmt.Sprintf("/media/%d", mid), user, ""); r.code != 404 {
		t.Fatalf("фото обращения открыто всем: %d", r.code)
	}
	if r := do(t, h, "GET", fmt.Sprintf("/api/admin/media/%d", mid), admin, ""); r.code != 200 {
		t.Fatalf("фото админу: %d", r.code)
	}
}

// payPost — «Я оплатил» на странице оплаты.
func payPost(t *testing.T, h *Server, path, plan string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := newMultipart(&buf, map[string]string{"plan": plan}, file{"screenshot", jpeg})
	req := httptest.NewRequest("POST", path, &buf)
	req.Header.Set("Content-Type", mw)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPayFlow(t *testing.T) {
	msg := &fakeBotAPI{}
	h := newHandler(t, Deps{Admins: []int64{adminTG}, PayCard: "8600 1234 5678 9012", PayCardName: "Owner", Msg: msg})
	auth := initData(9901, "")
	link := do(t, h, "GET", "/api/pay/link", auth, "")
	path := link.body["path"].(string)
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest("GET", path, nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "8600 1234 5678 9012") || !strings.Contains(page.Body.String(), "AY-") ||
		!strings.Contains(page.Body.String(), "70 000") {
		t.Fatalf("страница оплаты: %d %s", page.Code, page.Body.String())
	}
	bad := httptest.NewRecorder()
	h.ServeHTTP(bad, httptest.NewRequest("GET", path[:len(path)-3]+"xyz", nil))
	if bad.Code != 403 {
		t.Fatalf("подделанная ссылка: %d", bad.Code)
	}
	var before time.Time
	pool.QueryRow(t.Context(), `SELECT premium_until FROM users WHERE telegram_id = 9901`).Scan(&before)
	if rec := payPost(t, h, path, "month"); rec.Code != 200 {
		t.Fatalf("оплата: %d %s", rec.Code, rec.Body.String())
	}
	var after time.Time
	pool.QueryRow(t.Context(), `SELECT premium_until FROM users WHERE telegram_id = 9901`).Scan(&after)
	if after.Sub(before) < 29*24*time.Hour {
		t.Fatalf("Premium не открылся сразу: %v → %v", before, after)
	}
	if rec := payPost(t, h, path, "month"); rec.Code != 409 {
		t.Fatalf("вторая оплата на проверке: %d", rec.Code)
	}
	if len(msg.admins) != 1 || !strings.Contains(msg.admins[0], "70 000") {
		t.Fatalf("владельцу — скрин с суммой: %v", msg.admins)
	}
	admin := initData(adminTG, "")
	ps := do(t, h, "GET", "/api/admin/payments?status=pending", admin, "")
	pid := int64(ps.body["payments"].([]any)[0].(map[string]any)["id"].(float64))
	if r := do(t, h, "POST", fmt.Sprintf("/api/admin/payments/%d", pid), admin, `{"confirm":false,"reason":"notfound"}`); r.body["status"] != "rejected" {
		t.Fatalf("отказ: %s", r.raw)
	}
	var back time.Time
	var trust string
	pool.QueryRow(t.Context(), `SELECT premium_until, pay_trust FROM users WHERE telegram_id = 9901`).Scan(&back, &trust)
	if !back.Equal(before) || trust != "rejected" {
		t.Fatalf("после отказа Premium забран: %v (было %v), %s", back, before, trust)
	}
	if !strings.Contains(strings.Join(msg.users, "|"), "pay.rejected") {
		t.Fatalf("ученику — причина отказа: %v", msg.users)
	}
	// После отказа доступ — только после подтверждения.
	payPost(t, h, path, "quarter")
	pool.QueryRow(t.Context(), `SELECT premium_until FROM users WHERE telegram_id = 9901`).Scan(&after)
	if !after.Equal(before) {
		t.Fatalf("после отказа Premium не открывается сразу: %v", after)
	}
	ps = do(t, h, "GET", "/api/admin/payments?status=pending", admin, "")
	pid = int64(ps.body["payments"].([]any)[0].(map[string]any)["id"].(float64))
	if r := do(t, h, "POST", fmt.Sprintf("/api/admin/payments/%d", pid), admin, `{"confirm":true}`); r.body["status"] != "confirmed" {
		t.Fatalf("подтверждение: %s", r.raw)
	}
	pool.QueryRow(t.Context(), `SELECT premium_until FROM users WHERE telegram_id = 9901`).Scan(&after)
	if after.Sub(before) < 89*24*time.Hour {
		t.Fatalf("после подтверждения — 3 месяца: %v", after)
	}
}

func TestSpeakingPractice(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9951, "")
	g := do(t, h, "GET", "/api/speaking", auth, "")
	q := g.body["questions"].(map[string]any)["2"].(map[string]any)
	qid := strconv.FormatInt(int64(q["id"].(float64)), 10)
	pool.Exec(t.Context(), `UPDATE users SET premium_until = '2026-01-01' WHERE telegram_id = 9951`)
	if r := form(t, h, "/api/speaking", auth, map[string]string{"question_id": qid, "seconds": "90"}, file{"audio", webm}); r.code != 202 {
		t.Fatalf("ответ Speaking: %d %s", r.code, r.raw)
	}
	if r := form(t, h, "/api/speaking", auth, map[string]string{"question_id": qid}, file{"audio", webm}); r.code != 429 {
		t.Fatalf("без Premium — 1 в неделю: %d %s", r.code, r.raw)
	}
	if r := form(t, h, "/api/speaking", auth, map[string]string{"question_id": qid}, file{"audio", jpeg}); r.code != 400 {
		t.Fatalf("не аудио: %d", r.code)
	}
	_, sq := queues(h)
	if did, err := sq.Step(t.Context()); !did || err != nil {
		t.Fatalf("очередь Speaking: %v %v", did, err)
	}
	list := do(t, h, "GET", "/api/speaking", auth, "")
	hist := list.body["history"].([]any)[0].(map[string]any)
	c := do(t, h, "GET", fmt.Sprintf("/api/speaking/%d", int64(hist["id"].(float64))), auth, "")
	if c.body["status"] != "done" || c.body["band"] != 6.0 || c.body["transcript"] == "" {
		t.Fatalf("разбор Speaking: %s", c.raw)
	}
	var n int
	pool.QueryRow(t.Context(), `SELECT count(*) FROM speaking_audio`).Scan(&n)
	if n != 0 {
		t.Fatalf("записи после проверки удаляются: %d", n)
	}
}

// fakeBotAPI записывает, что API отправило бы ботом.
type fakeBotAPI struct {
	mu     sync.Mutex
	users  []string
	admins []string
	docs   []string
	next   int64
}

func (f *fakeBotAPI) ToUser(_ context.Context, tg int64, lang, key string, args map[string]string, open string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users = append(f.users, fmt.Sprintf("%d:%s:%s:%v", tg, key, open, args))
	return nil
}

func (f *fakeBotAPI) ToAdmins(_ context.Context, text string, _ *bot.Photo, _ [][]bot.Button) ([]bot.Sent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.admins = append(f.admins, text)
	f.next++
	return []bot.Sent{{ChatID: adminTG, MessageID: f.next}}, nil
}

func (f *fakeBotAPI) Send(context.Context, int64, string, [][]bot.Button) (bot.Sent, error) {
	return bot.Sent{}, nil
}
func (f *fakeBotAPI) Greet(context.Context, int64, string, string) error             { return nil }
func (f *fakeBotAPI) AnswerCallback(context.Context, string, string) error           { return nil }
func (f *fakeBotAPI) SetButtons(context.Context, int64, int64, [][]bot.Button) error { return nil }
func (f *fakeBotAPI) Download(context.Context, string, int64) ([]byte, error)        { return webm, nil }
func (f *fakeBotAPI) SendDocument(_ context.Context, chat int64, name string, data []byte, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs = append(f.docs, fmt.Sprintf("%d:%s:%s", chat, name, data))
	return nil
}

func update(t *testing.T, js string) bot.Update {
	t.Helper()
	var u bot.Update
	if err := json.Unmarshal([]byte(js), &u); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestBotFlow(t *testing.T) {
	msg := &fakeBotAPI{next: 5000} // номера сообщений не пересекаются с другими тестами: база общая
	h := newHandler(t, Deps{Admins: []int64{adminTG}, Msg: msg})
	ctx := t.Context()
	from := `"from":{"id":9961,"first_name":"Vali","language_code":"ru"},"chat":{"id":9961,"type":"private"}`
	if !h.HandleUpdate(ctx, update(t, `{"message":{"message_id":1,`+from+`,"text":"/help"}}`)) {
		t.Fatal("/help не обработан")
	}
	h.HandleUpdate(ctx, update(t, `{"message":{"message_id":2,`+from+`,"text":"Как сдать mock?"}}`))
	if len(msg.admins) != 1 || !strings.Contains(msg.admins[0], "Как сдать mock?") {
		t.Fatalf("обращение админам: %v", msg.admins)
	}
	// Админ отвечает reply на сообщение бота — ответ уходит ученику.
	admin := fmt.Sprintf(`"from":{"id":%d},"chat":{"id":%d,"type":"private"}`, adminTG, adminTG)
	h.HandleUpdate(ctx, update(t, `{"message":{"message_id":3,`+admin+`,"text":"Во вкладке Экзамены","reply_to_message":{"message_id":5001,"chat":{"id":`+strconv.Itoa(adminTG)+`}}}}`))
	if !strings.Contains(strings.Join(msg.users, "|"), "support.reply") {
		t.Fatalf("ответ ученику: %v", msg.users)
	}
	h.HandleUpdate(ctx, update(t, `{"message":{"message_id":4,`+admin+`,"text":"/post Итоги турнира #tournament\nПобедил aziz_8"}}`))
	var tag string
	pool.QueryRow(ctx, `SELECT tag FROM posts WHERE title = 'Итоги турнира'`).Scan(&tag)
	if tag != "tournament" {
		t.Fatalf("/post: метка %q", tag)
	}
	// Голосовое — проверка Speaking.
	h.HandleUpdate(ctx, update(t, `{"message":{"message_id":5,`+from+`,"voice":{"file_id":"v","duration":40}}}`))
	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM speaking_checks c JOIN users u ON u.id = c.user_id WHERE u.telegram_id = 9961 AND c.source = 'bot'`).Scan(&n)
	if n != 1 {
		t.Fatalf("голосовое не стало проверкой: %d", n)
	}
	// Чужие нажатия кнопок оплаты не работают.
	h.HandleUpdate(ctx, update(t, `{"callback_query":{"id":"q","from":{"id":9961},"data":"pay:ok:1"}}`))
	var confirmed int
	pool.QueryRow(ctx, `SELECT count(*) FROM payments WHERE status = 'confirmed' AND decided_by = 9961`).Scan(&confirmed)
	if confirmed != 0 {
		t.Fatal("ученик подтвердил оплату кнопкой")
	}
}

// newMultipart пишет форму в buf и возвращает Content-Type.
func newMultipart(buf *bytes.Buffer, fields map[string]string, files ...file) string {
	mw := multipart.NewWriter(buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	for _, f := range files {
		w, _ := mw.CreateFormFile(f.field, "shot.jpg")
		w.Write(f.data)
	}
	mw.Close()
	return mw.FormDataContentType()
}
