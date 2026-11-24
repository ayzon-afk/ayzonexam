package api

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestUsage(t *testing.T) {
	c := &clock{t: now}
	msg := &fakeBotAPI{}
	h := newHandler(t, Deps{Now: c.now, Admins: []int64{adminTG}, Msg: msg})
	const tg = 9971
	beat := func(sid string, start bool) {
		t.Helper()
		r := do(t, h, "POST", "/api/beat", initDataAt(tg, c.now()), fmt.Sprintf(`{"sid":%q,"start":%v}`, sid, start))
		if r.code != 204 {
			t.Fatalf("пульс: %d %s", r.code, r.raw)
		}
	}
	beat("sessionA1", true)
	c.add(30 * time.Second)
	beat("sessionA1", false) // +30
	c.add(20 * time.Second)
	beat("sessionA1", false) // +20
	c.add(10 * time.Minute)
	beat("sessionA1", false) // пауза — телефон спал: 0
	c.add(30 * time.Second)
	beat("sessionA1", true) // возврат из фона: 0
	c.add(40 * time.Second)
	beat("sessionA1", false) // больше 30 с не засчитывается: +30
	var secs, beats int
	pool.QueryRow(t.Context(), `SELECT active_seconds, beats FROM app_sessions s JOIN users u ON u.id = s.user_id
		WHERE u.telegram_id = $1`, tg).Scan(&secs, &beats)
	if secs != 80 || beats != 6 {
		t.Fatalf("активные секунды %d, пульсов %d; ждали 80 и 6", secs, beats)
	}
	if r := do(t, h, "POST", "/api/beat", initDataAt(tg, c.now()), `{"sid":"x"}`); r.code != 400 {
		t.Fatalf("короткий sid: %d", r.code)
	}

	// Второй день — второе открытие: повторный заход. Владелец не считается.
	c.add(24 * time.Hour)
	beat("sessionB2", true)
	if r := do(t, h, "POST", "/api/beat", initDataAt(adminTG, c.now()), `{"sid":"adminsess","start":true}`); r.code != 204 {
		t.Fatalf("пульс админа: %d", r.code)
	}

	if r := do(t, h, "GET", "/api/admin/usage", initDataAt(tg, c.now()), ""); r.code != 404 {
		t.Fatalf("ученику статистика закрыта: %d", r.code)
	}
	admin := initDataAt(adminTG, c.now())
	u := do(t, h, "GET", "/api/admin/usage", admin, "")
	if u.code != 200 || u.body["returning"].(float64) < 1 || u.body["sessions"].(float64) < 2 {
		t.Fatalf("сводка: %d %s", u.code, u.raw)
	}

	var id int64
	pool.QueryRow(t.Context(), `SELECT id FROM users WHERE telegram_id = $1`, tg).Scan(&id)
	csv := do(t, h, "GET", "/api/admin/usage.csv", admin, "").raw
	day1 := now.In(tashkent).Format(time.DateOnly)
	if !strings.Contains(csv, fmt.Sprintf("%s,%d,", day1, id)) || !strings.Contains(csv, ",1,1.3,0,0,0") {
		t.Fatalf("CSV без строки ученика за первый день:\n%s", csv)
	}
	var adminID int64
	pool.QueryRow(t.Context(), `SELECT id FROM users WHERE telegram_id = $1`, adminTG).Scan(&adminID)
	if strings.Contains(csv, fmt.Sprintf(",%d,", adminID)) {
		t.Fatalf("владелец попал в CSV:\n%s", csv)
	}

	if r := do(t, h, "POST", "/api/admin/usage/send", admin, ""); r.code != 204 {
		t.Fatalf("CSV в Telegram: %d %s", r.code, r.raw)
	}
	if len(msg.docs) != 1 || !strings.HasPrefix(msg.docs[0], fmt.Sprintf("%d:ayzon-usage-", adminTG)) {
		t.Fatalf("файл владельцу: %v", msg.docs)
	}
}
