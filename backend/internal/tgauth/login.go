package tgauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

// Вход на сайте виджетом «Войти через Telegram» (DECISIONS 2026-11-24).
// Алгоритм: https://core.telegram.org/widgets/login#checking-authorization — ключ HMAC — SHA-256 токена
// бота (у Mini App — HMAC «WebAppData»), поля — плоские: id, first_name, username, photo_url, auth_date.

// ValidateLogin проверяет подпись данных виджета и возраст auth_date.
func ValidateLogin(v url.Values, botToken string, maxAge time.Duration, now time.Time) (User, error) {
	for _, vals := range v {
		if len(vals) != 1 {
			return User{}, ErrInvalid
		}
	}
	got := v.Get("hash")
	if got == "" || !hmac.Equal([]byte(got), []byte(loginHash(v, botToken))) {
		return User{}, ErrInvalid
	}
	ts, err := strconv.ParseInt(v.Get("auth_date"), 10, 64)
	if err != nil {
		return User{}, ErrInvalid
	}
	if age := now.Sub(time.Unix(ts, 0)); age > maxAge || age < -time.Minute {
		return User{}, ErrExpired
	}
	id, err := strconv.ParseInt(v.Get("id"), 10, 64)
	if err != nil || id <= 0 {
		return User{}, ErrInvalid
	}
	return User{ID: id, FirstName: v.Get("first_name"), LastName: v.Get("last_name"), Username: v.Get("username"),
		PhotoURL: v.Get("photo_url")}, nil
}

// SignLogin подписывает поля так же, как виджет Telegram. Нужен тестам.
func SignLogin(v url.Values, botToken string) url.Values {
	out := url.Values{}
	for k, vals := range v {
		if k != "hash" {
			out[k] = vals
		}
	}
	out.Set("hash", loginHash(out, botToken))
	return out
}

func loginHash(v url.Values, botToken string) string {
	secret := sha256.Sum256([]byte(botToken))
	m := hmac.New(sha256.New, secret[:])
	m.Write([]byte(dataCheckString(v)))
	return hex.EncodeToString(m.Sum(nil))
}
