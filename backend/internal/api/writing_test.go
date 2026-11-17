package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/writing"
)

// jpeg — минимальные байты, которые http.DetectContentType признаёт за JPEG.
var jpeg = append([]byte("\xff\xd8\xff\xe0"), make([]byte, 100)...)

type file struct {
	field string
	data  []byte
}

// form отправляет multipart-форму.
func form(t *testing.T, h http.Handler, path, auth string, fields map[string]string, files ...file) resp {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	for i, f := range files {
		w, _ := mw.CreateFormFile(f.field, "photo"+strconv.Itoa(i)+".jpg")
		w.Write(f.data)
	}
	mw.Close()
	req := httptest.NewRequest("POST", path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "tma "+auth)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	r := resp{code: rec.Code, raw: rec.Body.String()}
	json.Unmarshal(rec.Body.Bytes(), &r.body)
	return r
}

func words(n int) string {
	return strings.TrimSpace(strings.Repeat("word ", n))
}

const ownPrompt = "Some people think cities are better places to live. Discuss."

func task2(essay string) map[string]string {
	return map[string]string{"task": "task2", "source": "text", "prompt": ownPrompt, "essay": essay}
}

func TestWritingRoutesNeedAuth(t *testing.T) {
	h := newHandler(t, Deps{})
	for _, p := range []string{"GET /api/writing/prompts?task=task2", "GET /api/writing/checks", "POST /api/writing/checks", "POST /api/writing/recognize", "GET /api/writing/checks/1"} {
		method, path, _ := strings.Cut(p, " ")
		if r := do(t, h, method, path, "", ""); r.code != 401 {
			t.Errorf("%s без входа: %d", p, r.code)
		}
	}
}

func TestWritingPrompts(t *testing.T) {
	h := newHandler(t, Deps{})
	r := do(t, h, "GET", "/api/writing/prompts?task=task2", initData(9101, ""), "")
	if list, _ := r.body["prompts"].([]any); r.code != 200 || len(list) != 12 {
		t.Fatalf("темы Task 2: %d %s", r.code, r.raw)
	}
	if r := do(t, h, "GET", "/api/writing/prompts?task=task1", initData(9101, ""), ""); r.code != 200 || r.raw != "{\"prompts\":[]}\n" {
		t.Fatalf("темы Task 1 пока пустые, но массивом: %s", r.raw)
	}
	if r := do(t, h, "GET", "/api/writing/prompts?task=task3", initData(9101, ""), ""); r.code != 400 {
		t.Fatalf("неизвестная задача: %d", r.code)
	}
}

func TestWritingFlow(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9102, "")
	do(t, h, "PATCH", "/api/me", auth, `{"lang":"en"}`)

	r := do(t, h, "GET", "/api/writing/prompts?task=task2", auth, "")
	first := r.body["prompts"].([]any)[0].(map[string]any)
	promptID := strconv.Itoa(int(first["id"].(float64)))

	essay := words(260)
	r = form(t, h, "/api/writing/checks", auth, map[string]string{"task": "task2", "source": "photo", "prompt_id": promptID, "essay": essay})
	if r.code != 202 {
		t.Fatalf("отправка: %d %s", r.code, r.raw)
	}
	id := strconv.Itoa(int(r.body["id"].(float64)))

	r = do(t, h, "GET", "/api/writing/checks/"+id, auth, "")
	if r.code != 200 || r.body["status"] != "queued" || r.body["prompt"] != first["text"] || r.body["essay"] != essay || r.body["criteria"] != nil {
		t.Fatalf("проверка в очереди: %s", r.raw)
	}

	// Очередь с заглушкой доводит проверку до результата.
	q := &writing.Queue{Store: writing.NewStore(pool), AI: ai.Fake{}, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Workers: 1, Now: func() time.Time { return now }}
	for {
		did, err := q.Step(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !did {
			break
		}
	}
	r = do(t, h, "GET", "/api/writing/checks/"+id, auth, "")
	crit, _ := r.body["criteria"].(map[string]any)
	fb, _ := r.body["feedback"].(map[string]any)
	if r.body["status"] != "done" || r.body["band"] != 6.0 || crit["ta"] != 6.0 || fb["summary"] == "" || r.body["source"] != "photo" {
		t.Fatalf("готовая проверка: %s", r.raw)
	}

	r = do(t, h, "GET", "/api/writing/checks", auth, "")
	checks, _ := r.body["checks"].([]any)
	if r.code != 200 || len(checks) != 1 || r.body["used"] != 1.0 || r.body["limit"] != 3.0 || r.body["period"] != "day" {
		t.Fatalf("история: %s", r.raw)
	}
	if strings.Contains(r.raw, "word word") || strings.Contains(r.raw, "feedback") {
		t.Fatalf("история не отдаёт текст эссе и разбор: %s", r.raw)
	}

	// Чужой ученик не видит ни проверку, ни историю.
	other := initData(9103, "")
	if r := do(t, h, "GET", "/api/writing/checks/"+id, other, ""); r.code != 404 {
		t.Fatalf("чужая проверка: %d %s", r.code, r.raw)
	}
	if r := do(t, h, "GET", "/api/writing/checks", other, ""); strings.Contains(r.raw, `"id"`) {
		t.Fatalf("чужая история: %s", r.raw)
	}
	if r := do(t, h, "GET", "/api/writing/checks/abc", auth, ""); r.code != 404 {
		t.Fatalf("id не число: %d", r.code)
	}
}

func TestWritingTask1(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9104, "")
	fields := map[string]string{"task": "task1", "source": "text", "essay": words(170)}
	if r := form(t, h, "/api/writing/checks", auth, fields); r.code != 400 || r.body["error"] != "image_required" {
		t.Fatalf("Task 1 без фото задания: %d %s", r.code, r.raw)
	}
	r := form(t, h, "/api/writing/checks", auth, fields, file{"image", jpeg})
	if r.code != 202 {
		t.Fatalf("Task 1 с фото: %d %s", r.code, r.raw)
	}
	var mime string
	pool.QueryRow(t.Context(), `SELECT mime FROM writing_images WHERE check_id = $1`, int64(r.body["id"].(float64))).Scan(&mime)
	if mime != "image/jpeg" {
		t.Fatalf("фото задания не сохранено: %q", mime)
	}
}

func TestWritingValidation(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9105, "")
	with := func(k, v string) map[string]string {
		f := task2(words(260))
		f[k] = v
		return f
	}
	cases := []struct {
		name   string
		fields map[string]string
		files  []file
		want   string
	}{
		{"задача", with("task", "task3"), nil, "bad_task"},
		{"источник", with("source", "voice"), nil, "bad_source"},
		{"мало слов", task2(words(49)), nil, "too_short"},
		{"много слов", task2(words(601)), nil, "too_long"},
		{"Task 1 больше 400 слов", map[string]string{"task": "task1", "source": "text", "essay": words(401)}, []file{{"image", jpeg}}, "too_long"},
		{"своя тема слишком короткая", with("prompt", "Discuss."), nil, "bad_prompt"},
		{"своя тема слишком длинная", with("prompt", strings.Repeat("a", 1001)), nil, "bad_prompt"},
		{"нет темы", with("prompt", ""), nil, "bad_prompt"},
		{"тема не из банка", with("prompt_id", "999999"), nil, "bad_prompt"},
		{"тема банка для Task 1", map[string]string{"task": "task1", "source": "text", "essay": words(170), "prompt_id": "1"}, []file{{"image", jpeg}}, "bad_prompt"},
		{"не картинка", map[string]string{"task": "task1", "source": "text", "essay": words(170)}, []file{{"image", []byte("%PDF-1.4 not an image")}}, "bad_image"},
		{"фото слишком большое", map[string]string{"task": "task1", "source": "text", "essay": words(170)}, []file{{"image", append(jpeg, make([]byte, maxImageBytes)...)}}, "image_too_large"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := form(t, h, "/api/writing/checks", auth, c.fields, c.files...)
			if r.code != 400 || r.body["error"] != c.want {
				t.Fatalf("ждали 400 %s, получили %d %s", c.want, r.code, r.raw)
			}
		})
	}
	if r := do(t, h, "POST", "/api/writing/checks", auth, `{"task":"task2"}`); r.code != 400 || r.body["error"] != "bad_form" {
		t.Fatalf("не multipart: %d %s", r.code, r.raw)
	}
}

func TestWritingDailyLimit(t *testing.T) {
	h := newHandler(t, Deps{WritingDailyLimit: 2})
	auth := initData(9106, "")
	for i := range 2 {
		if r := form(t, h, "/api/writing/checks", auth, task2(words(260))); r.code != 202 {
			t.Fatalf("проверка %d: %d %s", i+1, r.code, r.raw)
		}
	}
	if r := form(t, h, "/api/writing/checks", auth, task2(words(260))); r.code != 429 || r.body["error"] != "daily_limit" {
		t.Fatalf("сверх лимита: %d %s", r.code, r.raw)
	}
}

// failingAI — распознавание всегда падает.
type failingAI struct{ ai.Fake }

func (failingAI) Recognize(context.Context, []ai.Image) (string, error) {
	return "", io.ErrUnexpectedEOF
}

func TestRecognize(t *testing.T) {
	h := newHandler(t, Deps{})
	auth := initData(9107, "")
	r := form(t, h, "/api/writing/recognize", auth, nil, file{"images", jpeg}, file{"images", jpeg})
	if r.code != 200 || !strings.Contains(r.body["text"].(string), "2 photo") || r.body["words"].(float64) < 10 {
		t.Fatalf("распознавание: %d %s", r.code, r.raw)
	}
	if r := form(t, h, "/api/writing/recognize", auth, nil); r.code != 400 || r.body["error"] != "bad_pages" {
		t.Fatalf("без фото: %d %s", r.code, r.raw)
	}
	four := []file{{"images", jpeg}, {"images", jpeg}, {"images", jpeg}, {"images", jpeg}}
	if r := form(t, h, "/api/writing/recognize", auth, nil, four...); r.code != 400 || r.body["error"] != "bad_pages" {
		t.Fatalf("4 страницы: %d %s", r.code, r.raw)
	}

	// Неудачный вызов AI не съедает лимит.
	bad := newHandler(t, Deps{AI: failingAI{}})
	for range recognizeDailyLimit + 2 {
		if r := form(t, bad, "/api/writing/recognize", auth, nil, file{"images", jpeg}); r.code != 502 {
			t.Fatalf("AI недоступен: %d %s", r.code, r.raw)
		}
	}
	// Уже был 1 удачный вызов: ещё 9 проходят, 11-й — нет.
	for i := range recognizeDailyLimit - 1 {
		if r := form(t, h, "/api/writing/recognize", auth, nil, file{"images", jpeg}); r.code != 200 {
			t.Fatalf("вызов %d: %d %s", i+2, r.code, r.raw)
		}
	}
	if r := form(t, h, "/api/writing/recognize", auth, nil, file{"images", jpeg}); r.code != 429 {
		t.Fatalf("сверх лимита распознаваний: %d %s", r.code, r.raw)
	}
}
