// Package tgauth проверяет initData, который Telegram передаёт в Mini App.
// Алгоритм: https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
package tgauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalid = errors.New("initData: подпись или формат неверны")
	ErrExpired = errors.New("initData: устарел")
)

// User — поле user из initData.
type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	PhotoURL     string `json:"photo_url"`
}

// Validate проверяет подпись initData токеном бота и возраст auth_date.
func Validate(initData, botToken string, maxAge time.Duration, now time.Time) (User, error) {
	v, err := url.ParseQuery(initData)
	if err != nil {
		return User{}, ErrInvalid
	}
	for _, vals := range v {
		if len(vals) != 1 {
			return User{}, ErrInvalid
		}
	}
	got := v.Get("hash")
	if got == "" || !hmac.Equal([]byte(got), []byte(hash(v, botToken))) {
		return User{}, ErrInvalid
	}
	ts, err := strconv.ParseInt(v.Get("auth_date"), 10, 64)
	if err != nil {
		return User{}, ErrInvalid
	}
	age := now.Sub(time.Unix(ts, 0))
	if age > maxAge || age < -time.Minute {
		return User{}, ErrExpired
	}
	var u User
	if err := json.Unmarshal([]byte(v.Get("user")), &u); err != nil || u.ID <= 0 {
		return User{}, ErrInvalid
	}
	return u, nil
}

// Sign подписывает поля так же, как Telegram. Нужен тестам и локальному входу без Telegram.
func Sign(v url.Values, botToken string) string {
	out := url.Values{}
	for k, vals := range v {
		if k != "hash" {
			out[k] = vals
		}
	}
	out.Set("hash", hash(out, botToken))
	return out.Encode()
}

func hash(v url.Values, botToken string) string {
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	m := hmac.New(sha256.New, secret.Sum(nil))
	m.Write([]byte(dataCheckString(v)))
	return hex.EncodeToString(m.Sum(nil))
}

// dataCheckString — все поля, кроме hash, по алфавиту, «ключ=значение» через перевод строки.
func dataCheckString(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v.Get(k))
	}
	return b.String()
}
