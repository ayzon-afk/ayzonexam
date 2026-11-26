package api

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

// importForm — пакет теста канала: test.json, аудио и картинки с именами файлов.
func importForm(t *testing.T, h *Server, auth, testJSON string, files map[string][2]string) resp {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("test", testJSON)
	for name, f := range files { // f — поле и содержимое
		w, _ := mw.CreateFormFile(f[0], name)
		w.Write([]byte(f[1]))
	}
	mw.Close()
	req := httptest.NewRequest("POST", "/api/admin/tests/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "tma "+auth)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return resp{code: rec.Code, raw: rec.Body.String()}
}

func TestTestImport(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	admin, auth := initData(adminTG, ""), initData(9261, "")
	pkg := `{"source":"V10-T02-L","kind":"listening","title":"Volume 10 · Test 2 · Listening","minutes":30,"full_size":true,
		"groups":[{"title":"Questions 1–2. Label the map. Choose A–C.","options":["bank","cafe","park"],"image_file":"map.png"}],
		"questions":[{"q":"School","options":["A","B","C"],"answer":2,"group":1,"type":"map"},
			{"q":"Move ____ players","kind":"text","accept":["cassette"],"group":1,"type":"gap"}]}`
	audio := string(append([]byte("OggS"), make([]byte, 3000)...))
	png := "\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 100)
	files := map[string][2]string{"audio.ogg": {"audio", audio}, "map.png": {"image", png}}

	if r := importForm(t, h, auth, pkg, files); r.code != 404 {
		t.Fatalf("не админ загрузил: %d", r.code)
	}
	if r := importForm(t, h, admin, pkg, map[string][2]string{"map.png": files["map.png"]}); r.code != 400 || !strings.Contains(r.raw, "no_audio") {
		t.Fatalf("Listening без аудио: %d %s", r.code, r.raw)
	}
	if r := importForm(t, h, admin, pkg, map[string][2]string{"audio.ogg": files["audio.ogg"]}); r.code != 400 || !strings.Contains(r.raw, "no_image") {
		t.Fatalf("без картинки группы: %d %s", r.code, r.raw)
	}
	if r := importForm(t, h, admin, strings.Replace(pkg, "V10-T02-L", "../x", 1), files); r.code != 400 {
		t.Fatalf("плохой source: %d", r.code)
	}
	r := importForm(t, h, admin, pkg, files)
	if r.code != 200 || !strings.Contains(r.raw, `"replaced":false`) {
		t.Fatalf("загрузка: %d %s", r.code, r.raw)
	}
	var id int64
	var active bool
	var image int64
	pool.QueryRow(t.Context(), `SELECT id, active, (groups->0->>'image')::bigint FROM tests WHERE source = 'V10-T02-L'`).Scan(&id, &active, &image)
	if active || image == 0 {
		t.Fatalf("тест должен быть выключен и с картинкой: active=%v image=%d", active, image)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `UPDATE tests SET active = false WHERE id = $1`, id) })
	if g := do(t, h, "GET", fmt.Sprintf("/media/%d", image), "", ""); g.code != 200 {
		t.Fatalf("картинка группы не открыта: %d", g.code)
	}

	// Повтор — тот же тест, старые файлы удалены.
	again := importForm(t, h, admin, strings.Replace(pkg, `"minutes":30`, `"minutes":32`, 1), files)
	if again.code != 200 || !strings.Contains(again.raw, `"replaced":true`) || !strings.Contains(again.raw, fmt.Sprintf(`"id":%d`, id)) {
		t.Fatalf("повтор: %d %s", again.code, again.raw)
	}
	if g := do(t, h, "GET", fmt.Sprintf("/media/%d", image), "", ""); g.code != 404 {
		t.Fatalf("старая картинка осталась: %d", g.code)
	}
	full := do(t, h, "GET", fmt.Sprintf("/api/admin/tests/%d", id), admin, "")
	if full.code != 200 || !strings.Contains(full.raw, `"accept":["cassette"]`) || !strings.Contains(full.raw, `"minutes":32`) {
		t.Fatalf("задание целиком: %d %s", full.code, full.raw)
	}
	list := do(t, h, "GET", "/api/admin/tests", admin, "")
	if strings.Contains(list.raw, `"questions"`) || !strings.Contains(list.raw, `"source":"V10-T02-L"`) {
		t.Fatalf("список без вопросов, с source: %s", list.raw[:min(len(list.raw), 400)])
	}

	// После первой попытки ученика замена запрещена — иначе разбор старых попыток поедет.
	pool.Exec(t.Context(), `UPDATE tests SET active = true WHERE id = $1`, id)
	if st := do(t, h, "POST", fmt.Sprintf("/api/tests/%d/start", id), auth, ""); st.code != 200 {
		t.Fatalf("старт: %d %s", st.code, st.raw)
	}
	if r := importForm(t, h, admin, pkg, files); r.code != 409 {
		t.Fatalf("замена после попыток: %d %s", r.code, r.raw)
	}
}
