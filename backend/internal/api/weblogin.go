package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"ayzonexam/internal/tgauth"
)

// webLogin — вход на сайте (DECISIONS 2026-11-24): данные виджета «Войти через Telegram» меняются на
// подписанный сервером initData. Дальше браузер ходит с ним, как Mini App, — auth, админка, учёт работают
// без изменений. Пропуск живёт initDataMaxAge, потом — войти снова.
func (s *server) webLogin(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Auth map[string]json.RawMessage `json:"auth"` // поля виджета как есть: id — число, остальное — строки
		Lang string                     `json:"lang"` // язык браузера: у виджета его нет
	}
	if !readJSON(w, r, &b, 4<<10) {
		return
	}
	v := url.Values{}
	for k, raw := range b.Auth {
		var str string
		if json.Unmarshal(raw, &str) == nil {
			v.Set(k, str)
			continue
		}
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			writeError(w, http.StatusBadRequest, "bad_login")
			return
		}
		v.Set(k, n.String())
	}
	now := s.Now()
	u, err := tgauth.ValidateLogin(v, s.BotToken, initDataMaxAge, now)
	if err != nil {
		code := "unauthorized"
		if errors.Is(err, tgauth.ErrExpired) {
			code = "expired"
		}
		writeError(w, http.StatusUnauthorized, code)
		return
	}
	if validLang(b.Lang) {
		u.LanguageCode = b.Lang
	}
	user, _ := json.Marshal(u)
	init := tgauth.Sign(url.Values{"user": {string(user)}, "auth_date": {strconv.FormatInt(now.Unix(), 10)}}, s.BotToken)
	s.Log.Info("вход на сайте", "user", fmt.Sprint(u.ID))
	writeJSON(w, http.StatusOK, map[string]string{"init_data": init})
}
