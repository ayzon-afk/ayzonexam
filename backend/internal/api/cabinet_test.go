package api

import (
	"fmt"
	"strings"
	"testing"
)

// Кабинет центра: кто пускается, что видит учитель и аккаунт центра, чужой центр недоступен, личное не уходит.

// cabinetCenterFor — центр с аккаунтом центра (owner), учителем (teacher) и двумя учениками с баллом.
type cabinetSetup struct {
	cid                          int64
	owner, teacher, s1, s2, free string
	ownerID, teacherID, s1ID     int64
	code                         string
}

func userID(t *testing.T, tg int64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(t.Context(), `SELECT id FROM users WHERE telegram_id = $1`, tg).Scan(&id); err != nil {
		t.Fatalf("ученик %d: %v", tg, err)
	}
	return id
}

func newCabinet(t *testing.T, h *Server, name string, base int64) cabinetSetup {
	t.Helper()
	admin := initData(adminTG, "")
	c := do(t, h, "POST", "/api/admin/centers", admin, `{"name":"`+name+`","city":"tashkent","about":"old"}`)
	if c.code != 200 {
		t.Fatalf("центр: %d %s", c.code, c.raw)
	}
	s := cabinetSetup{cid: int64(c.body["id"].(float64))}
	pool.QueryRow(t.Context(), `SELECT invite_code FROM centers WHERE id = $1`, s.cid).Scan(&s.code)
	s.owner, s.teacher, s.s1, s.s2, s.free = initData(base, ""), initData(base+1, ""), initData(base+2, ""), initData(base+3, ""), initData(base+4, "")
	for _, auth := range []string{s.owner, s.teacher, s.s1, s.s2} {
		if r := do(t, h, "POST", "/api/centers/join", auth, `{"code":"`+s.code+`"}`); r.code != 200 {
			t.Fatalf("вход по ссылке: %d %s", r.code, r.raw)
		}
	}
	do(t, h, "GET", "/api/me", s.free, "")
	s.ownerID, s.teacherID, s.s1ID = userID(t, base), userID(t, base+1), userID(t, base+2)
	do(t, h, "PATCH", fmt.Sprintf("/api/admin/users/%d", s.ownerID), admin, `{"role":"admin"}`)
	do(t, h, "PATCH", fmt.Sprintf("/api/admin/users/%d", s.teacherID), admin, `{"role":"teacher"}`)
	solve(t, h, s.s1, fmt.Sprintf("/api/tests/%d/start", testID(t, h, s.s1, "reading")), "", 10)
	solve(t, h, s.s2, fmt.Sprintf("/api/tests/%d/start", testID(t, h, s.s2, "reading")), "", 4)
	return s
}

func TestCabinetAccess(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	s := newCabinet(t, h, "Cabinet Center 96", 9611)

	// Ученик центра и человек без центра — не сотрудники: весь кабинет закрыт.
	for _, auth := range []string{s.s1, s.free} {
		for _, rq := range [][3]string{
			{"GET", "/api/cabinet", ""},
			{"PATCH", "/api/cabinet", `{"about":"x"}`},
			{"POST", "/api/cabinet/code", ""},
			{"GET", "/api/cabinet/students", ""},
			{"DELETE", fmt.Sprintf("/api/cabinet/students/%d", s.s1ID), ""},
			{"PUT", fmt.Sprintf("/api/cabinet/students/%d/role", s.s1ID), `{"teacher":true}`},
		} {
			if r := do(t, h, rq[0], rq[1], auth, rq[2]); r.code != 403 {
				t.Fatalf("не сотрудник: %s %s → %d %s", rq[0], rq[1], r.code, r.raw)
			}
		}
	}
	if r := do(t, h, "GET", "/api/cabinet", "", ""); r.code != 401 {
		t.Fatalf("без входа: %d", r.code)
	}

	// Учитель: видит центр и учеников, но без денег и без управления.
	tv := do(t, h, "GET", "/api/cabinet", s.teacher, "")
	if tv.code != 200 || tv.body["role"] != "teacher" || tv.body["payments"] != nil {
		t.Fatalf("кабинет учителя: %d %s", tv.code, tv.raw)
	}
	if c := tv.body["center"].(map[string]any); c["seats"] != 0.0 || c["paid_until"] != nil || c["students"] != 4.0 {
		t.Fatalf("учителю — без тарифа: %v", c)
	}
	if r := do(t, h, "GET", "/api/cabinet/students", s.teacher, ""); r.code != 200 {
		t.Fatalf("учитель — ученики: %d", r.code)
	}
	for _, rq := range [][3]string{
		{"PATCH", "/api/cabinet", `{"about":"x"}`},
		{"POST", "/api/cabinet/code", ""},
		{"DELETE", fmt.Sprintf("/api/cabinet/students/%d", s.s1ID), ""},
		{"PUT", fmt.Sprintf("/api/cabinet/students/%d/role", s.s1ID), `{"teacher":true}`},
	} {
		if r := do(t, h, rq[0], rq[1], s.teacher, rq[2]); r.code != 403 {
			t.Fatalf("учитель управляет: %s %s → %d", rq[0], rq[1], r.code)
		}
	}

	// Ученик, который ещё ждёт подтверждения, — не сотрудник, даже если роль осталась в базе.
	pool.Exec(t.Context(), `UPDATE users SET center_status = 'pending' WHERE id = $1`, s.teacherID)
	if r := do(t, h, "GET", "/api/cabinet", s.teacher, ""); r.code != 403 {
		t.Fatalf("pending с ролью пущен: %d", r.code)
	}
}

func TestCabinetOwner(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	s := newCabinet(t, h, "Cabinet Center 97", 9621)
	other := newCabinet(t, h, "Cabinet Center 98", 9631)

	ov := do(t, h, "GET", "/api/cabinet", s.owner, "")
	c, _ := ov.body["center"].(map[string]any)
	if ov.code != 200 || ov.body["role"] != "admin" || c["invite_code"] != s.code || c["students"] != 4.0 || c["scored"] != 2.0 {
		t.Fatalf("обзор: %d %s", ov.code, ov.raw)
	}
	if c["place"] == nil || c["centers"].(float64) < 2 {
		t.Fatalf("место среди центров: %v", c)
	}
	if _, ok := ov.body["payments"].([]any); !ok {
		t.Fatalf("оплаты — списком, даже пустым: %s", ov.raw)
	}

	// Ученики: лучший — первым, с баллом по частям; без имени, @username и Telegram id.
	st := do(t, h, "GET", "/api/cabinet/students", s.owner, "")
	list, _ := st.body["students"].([]any)
	if st.code != 200 || len(list) != 4 {
		t.Fatalf("ученики: %d %s", st.code, st.raw)
	}
	first := list[0].(map[string]any)
	if first["id"] != float64(s.s1ID) || first["band"] != 9.0 || first["parts"].(map[string]any)["reading"] != 9.0 || first["month"] != 1.0 || first["last_at"] == nil {
		t.Fatalf("лучший ученик: %v", first)
	}
	if last := list[3].(map[string]any); last["band"] != nil || len(last["parts"].(map[string]any)) != 0 {
		t.Fatalf("без попыток — без балла: %v", last)
	}
	for _, leak := range []string{"Ali", "ali_tg", "9621", "9623", "telegram", "first_name", "username"} {
		if strings.Contains(st.raw, leak) || strings.Contains(ov.raw, leak) {
			t.Fatalf("в кабинете личное %q: %s", leak, st.raw)
		}
	}

	// Текст о центре: меняется, длинный — отказ.
	if r := do(t, h, "PATCH", "/api/cabinet", s.owner, `{"about":"  New about  "}`); r.code != 200 || r.body["center"].(map[string]any)["about"] != "New about" {
		t.Fatalf("о центре: %d %s", r.code, r.raw)
	}
	if r := do(t, h, "PATCH", "/api/cabinet", s.owner, `{"about":"`+strings.Repeat("я", 2001)+`"}`); r.code != 400 {
		t.Fatalf("длинный текст: %d", r.code)
	}
	if r := do(t, h, "PATCH", "/api/cabinet", s.owner, `{"name":"Hack"}`); r.code != 400 {
		t.Fatalf("название меняет только владелец: %d", r.code)
	}

	// Новая ссылка: старая больше не пускает.
	nc := do(t, h, "POST", "/api/cabinet/code", s.owner, "")
	code := nc.body["center"].(map[string]any)["invite_code"].(string)
	if nc.code != 200 || code == s.code || len(code) != 10 {
		t.Fatalf("новая ссылка: %d %s", nc.code, nc.raw)
	}
	if r := do(t, h, "POST", "/api/centers/join", s.free, `{"code":"`+s.code+`"}`); r.code != 404 {
		t.Fatalf("старая ссылка работает: %d", r.code)
	}

	// Роль учителя: даётся и снимается; себе, другому аккаунту центра и в чужом центре — нельзя.
	s2ID := userID(t, 9624)
	if r := do(t, h, "PUT", fmt.Sprintf("/api/cabinet/students/%d/role", s2ID), s.owner, `{"teacher":true}`); r.code != 204 {
		t.Fatalf("сделать учителем: %d %s", r.code, r.raw)
	}
	if me := do(t, h, "GET", "/api/me", s.s2, ""); !strings.Contains(me.raw, `"role":"teacher"`) {
		t.Fatalf("роль не встала: %s", me.raw)
	}
	if r := do(t, h, "GET", "/api/cabinet", s.s2, ""); r.code != 200 {
		t.Fatalf("новый учитель не вошёл: %d", r.code)
	}
	if r := do(t, h, "PUT", fmt.Sprintf("/api/cabinet/students/%d/role", s2ID), s.owner, `{"teacher":false}`); r.code != 204 {
		t.Fatalf("снять роль: %d", r.code)
	}
	if r := do(t, h, "GET", "/api/cabinet", s.s2, ""); r.code != 403 {
		t.Fatalf("роль снята, а кабинет открыт: %d", r.code)
	}
	if r := do(t, h, "PUT", fmt.Sprintf("/api/cabinet/students/%d/role", s.ownerID), s.owner, `{"teacher":true}`); r.code != 403 {
		t.Fatalf("себе роль: %d", r.code)
	}
	secondAdmin := userID(t, 9622) // учитель центра → владелец делает его вторым аккаунтом центра
	do(t, h, "PATCH", fmt.Sprintf("/api/admin/users/%d", secondAdmin), initData(adminTG, ""), `{"role":"admin"}`)
	for _, rq := range [][3]string{
		{"PUT", fmt.Sprintf("/api/cabinet/students/%d/role", secondAdmin), `{"teacher":false}`},
		{"DELETE", fmt.Sprintf("/api/cabinet/students/%d", secondAdmin), ""},
		{"DELETE", fmt.Sprintf("/api/cabinet/students/%d", s.ownerID), ""},
	} {
		if r := do(t, h, rq[0], rq[1], s.owner, rq[2]); r.code != 403 {
			t.Fatalf("над аккаунтом центра: %s %s → %d", rq[0], rq[1], r.code)
		}
	}
	for _, rq := range [][3]string{
		{"PUT", fmt.Sprintf("/api/cabinet/students/%d/role", other.s1ID), `{"teacher":true}`},
		{"DELETE", fmt.Sprintf("/api/cabinet/students/%d", other.s1ID), ""},
		{"DELETE", "/api/cabinet/students/999999999", ""},
	} {
		if r := do(t, h, rq[0], rq[1], s.owner, rq[2]); r.code != 404 {
			t.Fatalf("чужой ученик: %s %s → %d", rq[0], rq[1], r.code)
		}
	}
	if me := do(t, h, "GET", "/api/me", other.s1, ""); !strings.Contains(me.raw, `"status":"member"`) {
		t.Fatalf("ученик чужого центра задет: %s", me.raw)
	}

	// Убрать ученика: он больше не в центре, список короче, действие — в журнале владельца.
	if r := do(t, h, "DELETE", fmt.Sprintf("/api/cabinet/students/%d", s.s1ID), s.owner, ""); r.code != 204 {
		t.Fatalf("убрать ученика: %d %s", r.code, r.raw)
	}
	if me := do(t, h, "GET", "/api/me", s.s1, ""); strings.Contains(me.raw, `"center":{`) {
		t.Fatalf("ученик остался в центре: %s", me.raw)
	}
	if r := do(t, h, "DELETE", fmt.Sprintf("/api/cabinet/students/%d", s.s1ID), s.owner, ""); r.code != 404 {
		t.Fatalf("повторно убрать: %d", r.code)
	}
	var logged int
	pool.QueryRow(t.Context(), `SELECT count(*) FROM admin_log WHERE admin_tg = 9621 AND action LIKE 'cabinet.%'`).Scan(&logged)
	if logged != 5 {
		t.Fatalf("журнал: %d записей, ждали 5 (о центре, ссылка, роль ×2, убрать)", logged)
	}
}

func TestCabinetPayments(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	s := newCabinet(t, h, "Cabinet Center 99", 9641)
	pay := form(t, h, fmt.Sprintf("/api/admin/centers/%d/payments", s.cid), initData(adminTG, ""),
		map[string]string{"amount": "400000", "seats": "20", "paid_until": "2027-01-01", "note": "secret note"})
	if pay.code != 200 {
		t.Fatalf("оплата центра: %d %s", pay.code, pay.raw)
	}
	ov := do(t, h, "GET", "/api/cabinet", s.owner, "")
	pays, _ := ov.body["payments"].([]any)
	c := ov.body["center"].(map[string]any)
	if len(pays) != 1 || c["seats"] != 20.0 || c["paid_until"] == nil {
		t.Fatalf("тариф в кабинете: %s", ov.raw)
	}
	if strings.Contains(ov.raw, "admin_tg") || strings.Contains(ov.raw, "receipt") || strings.Contains(ov.raw, "secret note") {
		t.Fatalf("служебное владельца ушло центру: %s", ov.raw)
	}
}
