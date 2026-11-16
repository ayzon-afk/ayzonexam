package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/testpg"
	"ayzonexam/internal/tgauth"
	"ayzonexam/internal/users"
	"ayzonexam/internal/writing"
)

// fakeBot — выдуманный бот для тестов.
const fakeBot = "777:TEST"

var (
	pool *pgxpool.Pool
	// Полдень по Ташкенту, 15.11.2026.
	now = time.Date(2026, 11, 15, 7, 0, 0, 0, time.UTC)
)

func TestMain(m *testing.M) {
	os.Exit(testpg.Run(54371, func(p *pgxpool.Pool) int {
		pool = p
		return m.Run()
	}))
}

func newHandler(t *testing.T, d Deps) http.Handler {
	t.Helper()
	d.Users = users.NewStore(pool)
	d.Writing = writing.NewStore(pool)
	if d.AI == nil {
		d.AI = ai.Fake{}
	}
	if d.WritingDailyLimit == 0 {
		d.WritingDailyLimit = 3
	}
	d.BotToken = fakeBot
	d.Now = func() time.Time { return now }
	return New(d)
}

func initData(tgID int64, extra string) string {
	user := `{"id":` + strconv.FormatInt(tgID, 10) + `,"first_name":"Ali","username":"ali_tg","language_code":"ru"` + extra + `}`
	return tgauth.Sign(url.Values{"user": {user}, "auth_date": {strconv.FormatInt(now.Unix(), 10)}}, fakeBot)
}

type resp struct {
	code int
	body map[string]any
	raw  string
}

func do(t *testing.T, h http.Handler, method, path, auth, body string) resp {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if auth != "" {
		req.Header.Set("Authorization", "tma "+auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := resp{code: rec.Code, raw: rec.Body.String()}
	json.Unmarshal(rec.Body.Bytes(), &r.body)
	return r
}

func TestHealth(t *testing.T) {
	h := newHandler(t, Deps{})
	if r := do(t, h, "GET", "/api/health", "", ""); r.code != 200 {
		t.Fatalf("health: %d %s", r.code, r.raw)
	}
}

func TestAuthRequired(t *testing.T) {
	h := newHandler(t, Deps{})
	expired := tgauth.Sign(url.Values{"user": {`{"id":5}`}, "auth_date": {strconv.FormatInt(now.Add(-48*time.Hour).Unix(), 10)}}, fakeBot)
	cases := []struct{ name, auth, want string }{
		{"без заголовка", "", "unauthorized"},
		{"подделка", "user=%7B%22id%22%3A5%7D&auth_date=1&hash=00", "unauthorized"},
		{"чужой бот", tgauth.Sign(url.Values{"user": {`{"id":5}`}, "auth_date": {strconv.FormatInt(now.Unix(), 10)}}, "1:other"), "unauthorized"},
		{"устарел", expired, "expired"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := do(t, h, "GET", "/api/me", c.auth, "")
			if r.code != 401 || r.body["error"] != c.want {
				t.Fatalf("ждали 401 %s, получили %d %s", c.want, r.code, r.raw)
			}
		})
	}
}

func TestFirstLoginCreatesUser(t *testing.T) {
	h := newHandler(t, Deps{})
	r := do(t, h, "GET", "/api/me", initData(9001, ""), "")
	if r.code != 200 {
		t.Fatalf("%d %s", r.code, r.raw)
	}
	nick, _ := r.body["nickname"].(string)
	if !strings.HasPrefix(nick, "user") || strings.Contains(nick, "9001") {
		t.Fatalf("ник должен быть user<id> и не раскрывать Telegram id: %q", nick)
	}
	if r.body["lang"] != "ru" || r.body["onboarded"] != false || r.body["goal_band"] != nil || r.body["first_name"] != "Ali" {
		t.Fatalf("неверный новый ученик: %s", r.raw)
	}
	if strings.Contains(r.raw, "ali_tg") || strings.Contains(r.raw, "telegram_id") {
		t.Fatalf("@username и Telegram id не отдаются: %s", r.raw)
	}

	again := do(t, h, "GET", "/api/me", initData(9001, ""), "")
	if again.body["nickname"] != nick {
		t.Fatalf("второй вход создал другого ученика: %s → %s", nick, again.raw)
	}
	var n int
	pool.QueryRow(t.Context(), "SELECT count(*) FROM users WHERE telegram_id = 9001").Scan(&n)
	if n != 1 {
		t.Fatalf("учеников с одним Telegram id: %d", n)
	}
}

func TestTelegramProfileRefreshed(t *testing.T) {
	h := newHandler(t, Deps{})
	do(t, h, "GET", "/api/me", initData(9002, ""), "")
	// Узбекская кириллица и эмодзи: база обязана быть в UTF-8.
	user := `{"id":9002,"first_name":"Ўктам Ғайрат Қҳ 🙂","username":"new_name"}`
	r := do(t, h, "GET", "/api/me", tgauth.Sign(url.Values{"user": {user}, "auth_date": {strconv.FormatInt(now.Unix(), 10)}}, fakeBot), "")
	if r.body["first_name"] != "Ўктам Ғайрат Қҳ 🙂" {
		t.Fatalf("имя из Telegram не обновилось: %s", r.raw)
	}
	var username string
	pool.QueryRow(t.Context(), "SELECT username FROM users WHERE telegram_id = 9002").Scan(&username)
	if username != "new_name" {
		t.Fatalf("@username не обновился: %q", username)
	}
}

func TestOnboarding(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9003, "")

	if r := do(t, h, "PATCH", "/api/me", auth, `{"onboarded":true}`); r.code != 400 || r.body["error"] != "goal_required" {
		t.Fatalf("онбординг без цели: %d %s", r.code, r.raw)
	}
	r := do(t, h, "PATCH", "/api/me", auth, `{"goal_band":7.5,"exam_date":"2026-12-20","onboarded":true}`)
	if r.code != 200 || r.body["goal_band"] != 7.5 || r.body["exam_date"] != "2026-12-20" || r.body["onboarded"] != true {
		t.Fatalf("онбординг: %d %s", r.code, r.raw)
	}

	// «Пока не знаю» стирает дату, цель остаётся.
	r = do(t, h, "PATCH", "/api/me", auth, `{"exam_date":null}`)
	if r.code != 200 || r.body["exam_date"] != nil || r.body["goal_band"] != 7.5 {
		t.Fatalf("сброс даты: %d %s", r.code, r.raw)
	}
	// Поле не пришло — не трогается.
	r = do(t, h, "PATCH", "/api/me", auth, `{"lang":"en"}`)
	if r.code != 200 || r.body["lang"] != "en" || r.body["goal_band"] != 7.5 || r.body["onboarded"] != true {
		t.Fatalf("смена языка: %d %s", r.code, r.raw)
	}
}

func TestPatchValidation(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9004, "")
	cases := []struct{ name, body, want string }{
		{"цель не по шагу 0.5", `{"goal_band":7.3}`, "bad_goal_band"},
		{"цель выше 9", `{"goal_band":9.5}`, "bad_goal_band"},
		{"цель ниже 4", `{"goal_band":3.5}`, "bad_goal_band"},
		{"язык", `{"lang":"de"}`, "bad_lang"},
		{"дата вчера", `{"exam_date":"2026-11-14"}`, "bad_exam_date"},
		{"дата через 3 года", `{"exam_date":"2029-11-15"}`, "bad_exam_date"},
		{"дата не дата", `{"exam_date":"завтра"}`, "bad_exam_date"},
		{"дата числом", `{"exam_date":20261220}`, "bad_exam_date"},
		{"лишнее поле", `{"nickname":"admin"}`, "bad_json"},
		{"не JSON", `{`, "bad_json"},
		{"слишком большое тело", `{"lang":"` + strings.Repeat("a", 5000) + `"}`, "bad_json"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := do(t, h, "PATCH", "/api/me", auth, c.body)
			if r.code != 400 || r.body["error"] != c.want {
				t.Fatalf("ждали 400 %s, получили %d %s", c.want, r.code, r.raw)
			}
		})
	}
	// Сегодня по Ташкенту — можно.
	if r := do(t, h, "PATCH", "/api/me", auth, `{"exam_date":"2026-11-15"}`); r.code != 200 {
		t.Fatalf("сегодняшняя дата: %d %s", r.code, r.raw)
	}
}

func TestUsersIsolated(t *testing.T) {
	h := newHandler(t, Deps{})
	a, b := initData(9005, ""), initData(9006, "")
	do(t, h, "PATCH", "/api/me", a, `{"goal_band":8}`)
	r := do(t, h, "GET", "/api/me", b, "")
	if r.body["goal_band"] != nil {
		t.Fatalf("цель одного ученика видна другому: %s", r.raw)
	}
}

func TestDevInitDataOnlyWhenEnabled(t *testing.T) {
	if r := do(t, newHandler(t, Deps{}), "GET", "/api/dev/init-data", "", ""); r.code != 404 {
		t.Fatalf("без DEV_AUTH маршрута быть не должно: %d", r.code)
	}
	h := newHandler(t, Deps{DevAuth: true})
	r := do(t, h, "GET", "/api/dev/init-data?id=555&lang=en", "", "")
	init, _ := r.body["init_data"].(string)
	me := do(t, h, "GET", "/api/me", init, "")
	if me.code != 200 || me.body["lang"] != "en" {
		t.Fatalf("тестовый вход не работает: %d %s", me.code, me.raw)
	}
}

func TestSPA(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>app"), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "main-abc.js"), []byte("js"), 0o644)
	h := newHandler(t, Deps{WebDir: dir})

	// «/../..» сервер Go сначала перенаправляет на очищенный путь; файлы вне папки недоступны.
	if r := do(t, h, "GET", "/../../etc/passwd", "", ""); r.code != 307 {
		t.Fatalf("неочищенный путь: %d", r.code)
	}
	for _, p := range []string{"/", "/profile", "/etc/passwd"} {
		r := do(t, h, "GET", p, "", "")
		if r.code != 200 || !strings.Contains(r.raw, "app") {
			t.Fatalf("%s: %d %q", p, r.code, r.raw)
		}
	}
	req := httptest.NewRequest("GET", "/assets/main-abc.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("assets: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	if r := do(t, h, "GET", "/api/nope", "", ""); r.code != 404 || r.body["error"] != "not_found" {
		t.Fatalf("неизвестный /api/ не должен отдавать index.html: %d %s", r.code, r.raw)
	}
}
