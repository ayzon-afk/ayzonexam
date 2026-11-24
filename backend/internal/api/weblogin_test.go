package api

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"

	"ayzonexam/internal/tgauth"
)

func TestWebLogin(t *testing.T) {
	h := newHandler(t, Deps{Admins: []int64{adminTG}})
	login := func(id int64, forge bool) resp {
		v := tgauth.SignLogin(url.Values{"id": {strconv.FormatInt(id, 10)}, "first_name": {"Owner"},
			"auth_date": {strconv.FormatInt(now.Unix(), 10)}}, fakeBot)
		auth := map[string]any{"first_name": v.Get("first_name"), "auth_date": now.Unix(), "hash": v.Get("hash"), "id": id}
		if forge {
			auth["id"] = id + 1
		}
		body, _ := json.Marshal(map[string]any{"auth": auth, "lang": "ru"})
		return do(t, h, "POST", "/api/web/login", "", string(body))
	}
	r := login(adminTG, false)
	init, _ := r.body["init_data"].(string)
	if r.code != 200 || init == "" {
		t.Fatalf("вход на сайте: %d %s", r.code, r.raw)
	}
	if me := do(t, h, "GET", "/api/me", init, ""); me.code != 200 || me.body["admin"] != true {
		t.Fatalf("пропуск сайта как initData: %d %s", me.code, me.raw)
	}
	if a := do(t, h, "GET", "/api/admin/usage", init, ""); a.code != 200 {
		t.Fatalf("админка с сайта: %d", a.code)
	}
	if f := login(9991, true); f.code != 401 {
		t.Fatalf("подменённый id: %d %s", f.code, f.raw)
	}
}
