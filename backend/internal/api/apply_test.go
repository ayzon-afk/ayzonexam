package api

import (
	"fmt"
	"strings"
	"testing"
)

// Заявка «Добавить свой центр / Я учитель»: анкета → владелец одобряет кнопкой → кабинет создан сам.
func TestApply(t *testing.T) {
	msg := &fakeBotAPI{}
	h := newHandler(t, Deps{Admins: []int64{adminTG}, Msg: msg})
	aziz := initData(9671, "")
	callback := func(from int64, data string) {
		h.HandleUpdate(t.Context(), update(t, fmt.Sprintf(`{"callback_query":{"id":"q","from":{"id":%d},"data":%q}}`, from, data)))
	}

	for _, bad := range []string{
		`{"kind":"boss","name":"Aziz","city":"samarkand","students":15}`,
		`{"kind":"teacher","name":"A","city":"samarkand","students":15}`,
		`{"kind":"teacher","name":"Aziz","city":"other","students":15}`,
		`{"kind":"teacher","name":"Aziz","city":"samarkand","students":-1}`,
	} {
		if r := do(t, h, "POST", "/api/apply", aziz, bad); r.code != 400 {
			t.Fatalf("плохая анкета %s: %d", bad, r.code)
		}
	}
	r := do(t, h, "POST", "/api/apply", aziz, `{"kind":"teacher","name":"Aziz Apply 8.0","city":"samarkand","students":15,"phone":"+998 90 111 22 33","about":"IELTS 8.0"}`)
	if r.code != 200 || r.body["application"].(map[string]any)["status"] != "pending" {
		t.Fatalf("заявка: %d %s", r.code, r.raw)
	}
	if strings.Contains(r.raw, "111 22 33") {
		t.Fatalf("телефон — только владельцу: %s", r.raw)
	}
	if len(msg.admins) != 1 || !strings.Contains(msg.admins[0], "Aziz Apply 8.0") || !strings.Contains(msg.admins[0], "Учитель без центра") {
		t.Fatalf("владельцу — заявка: %v", msg.admins)
	}
	if r := do(t, h, "POST", "/api/apply", aziz, `{"kind":"teacher","name":"Aziz 2","city":"samarkand","students":15}`); r.code != 409 {
		t.Fatalf("вторая заявка на проверке: %d", r.code)
	}
	id := int64(r.body["application"].(map[string]any)["id"].(float64))

	callback(9672, fmt.Sprintf("ap:pilot:%d", id)) // не владелец — ничего
	if g := do(t, h, "GET", "/api/apply", aziz, ""); g.body["application"].(map[string]any)["status"] != "pending" {
		t.Fatalf("одобрил не владелец: %s", g.raw)
	}
	callback(adminTG, fmt.Sprintf("ap:pilot:%d", id))
	cab := do(t, h, "GET", "/api/cabinet", aziz, "")
	c, _ := cab.body["center"].(map[string]any)
	if cab.code != 200 || c["kind"] != "teacher" || c["name"] != "Aziz Apply 8.0" || c["city"] != "samarkand" || c["seats"] != 15.0 || c["paid_until"] == nil {
		t.Fatalf("кабинет после одобрения: %d %s", cab.code, cab.raw)
	}
	if !strings.Contains(strings.Join(msg.users, "|"), "apply.pilot:cabinet") {
		t.Fatalf("заявителю — «кабинет готов»: %v", msg.users)
	}
	callback(adminTG, fmt.Sprintf("ap:ok:%d", id)) // второе нажатие — второй центр не создаётся
	var n int
	pool.QueryRow(t.Context(), `SELECT count(*) FROM centers WHERE name LIKE 'Aziz Apply 8.0%'`).Scan(&n)
	if n != 1 {
		t.Fatalf("центров после двух нажатий: %d", n)
	}
	if r := do(t, h, "POST", "/api/apply", aziz, `{"kind":"center","name":"Aziz Center","city":"samarkand","students":40}`); r.code != 409 || r.body["error"] != "already_staff" {
		t.Fatalf("уже управляет — новая заявка: %d %s", r.code, r.raw)
	}

	// Отказ: тот же человек может подать снова; занятое название получает номер заявки.
	bek := initData(9673, "")
	r = do(t, h, "POST", "/api/apply", bek, `{"kind":"center","name":"Aziz Apply 8.0","city":"tashkent","students":30}`)
	bid := int64(r.body["application"].(map[string]any)["id"].(float64))
	callback(adminTG, fmt.Sprintf("ap:no:%d", bid))
	if g := do(t, h, "GET", "/api/apply", bek, ""); g.body["application"].(map[string]any)["status"] != "rejected" {
		t.Fatalf("отказ: %s", g.raw)
	}
	r = do(t, h, "POST", "/api/apply", bek, `{"kind":"center","name":"Aziz Apply 8.0","city":"tashkent","students":30}`)
	if r.code != 200 {
		t.Fatalf("снова после отказа: %d %s", r.code, r.raw)
	}
	callback(adminTG, fmt.Sprintf("ap:ok:%d", int64(r.body["application"].(map[string]any)["id"].(float64))))
	b := do(t, h, "GET", "/api/cabinet", bek, "")
	bc := b.body["center"].(map[string]any)
	if bc["kind"] != "center" || !strings.HasPrefix(bc["name"].(string), "Aziz Apply 8.0 #") || bc["seats"] != 0.0 || bc["paid_until"] != nil {
		t.Fatalf("центр без пилота, имя с номером: %s", b.raw)
	}
}
