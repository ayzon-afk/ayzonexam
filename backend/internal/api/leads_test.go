package api

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLeads(t *testing.T) {
	msg := &fakeBotAPI{}
	h := newHandler(t, Deps{Admins: []int64{adminTG}, Msg: msg, BotUsername: "ayzon_exam_bot"})
	if r := do(t, h, "GET", "/api/info", "", ""); r.code != 200 || r.body["bot"] != "ayzon_exam_bot" {
		t.Fatalf("адрес бота без входа: %d %s", r.code, r.raw)
	}
	post := func(ip, body string) int {
		req := httptest.NewRequest("POST", "/api/leads", strings.NewReader(body))
		req.Header.Set("X-Forwarded-For", "1.1.1.1, "+ip)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	ok := `{"center":"Smart Academy","name":"Dilnoza","phone":"+998 90 123 45 67","students":120,"message":"Хотим пилот","lang":"ru"}`
	if c := post("10.0.0.1", ok); c != 204 {
		t.Fatalf("заявка: %d", c)
	}
	if len(msg.admins) != 1 || !strings.Contains(msg.admins[0], "Smart Academy") || !strings.Contains(msg.admins[0], "+998 90 123 45 67") {
		t.Fatalf("владельцу: %v", msg.admins)
	}
	if c := post("10.0.0.2", `{"center":"X","name":"Y","phone":"12"}`); c != 400 {
		t.Fatalf("короткий телефон: %d", c)
	}
	if c := post("10.0.0.3", `{"center":"Spam","name":"Bot","phone":"+998901234567","website":"http://spam"}`); c != 204 {
		t.Fatalf("ловушка: %d", c)
	}
	var n int
	pool.QueryRow(t.Context(), `SELECT count(*) FROM center_leads WHERE center = 'Spam'`).Scan(&n)
	if n != 0 {
		t.Fatal("заявка из ловушки сохранена")
	}
	post("10.0.0.1", ok)
	post("10.0.0.1", ok)
	if c := post("10.0.0.1", ok); c != 429 {
		t.Fatalf("четвёртая с одного адреса за час: %d", c)
	}
	if r := do(t, h, "GET", "/api/admin/leads", initData(adminTG, ""), ""); r.code != 200 || len(r.body["leads"].([]any)) < 3 {
		t.Fatalf("заявки в админке: %d %s", r.code, r.raw)
	}
}
