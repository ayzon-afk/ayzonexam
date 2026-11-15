package tgauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// fakeBot — выдуманный бот для тестов, не настоящий.
const fakeBot = "123456:TEST"

var now = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

func signed(t *testing.T, user string, at time.Time) string {
	t.Helper()
	return Sign(url.Values{
		"user":      {user},
		"auth_date": {strconv.FormatInt(at.Unix(), 10)},
		"query_id":  {"AAH"},
	}, fakeBot)
}

func TestValidateOK(t *testing.T) {
	init := signed(t, `{"id":42,"first_name":"Ali","username":"ali","language_code":"ru"}`, now.Add(-time.Hour))
	u, err := Validate(init, fakeBot, 24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 42 || u.FirstName != "Ali" || u.Username != "ali" || u.LanguageCode != "ru" {
		t.Fatalf("неверный пользователь: %+v", u)
	}
}

// Подпись считается вручную по документации Telegram, без кода пакета:
// если dataCheckString или ключ собраны не так, тест это поймает, а пара Sign/Validate — нет.
func TestValidateAgainstManualSignature(t *testing.T) {
	dcs := "auth_date=1763200000\nquery_id=AAH\nuser={\"id\":7,\"first_name\":\"Bek\"}"
	key := hmac.New(sha256.New, []byte("WebAppData"))
	key.Write([]byte(fakeBot))
	m := hmac.New(sha256.New, key.Sum(nil))
	m.Write([]byte(dcs))
	sig := hex.EncodeToString(m.Sum(nil))

	init := "query_id=AAH&user=" + url.QueryEscape(`{"id":7,"first_name":"Bek"}`) + "&auth_date=1763200000&hash=" + sig
	u, err := Validate(init, fakeBot, 24*time.Hour, time.Unix(1763200000, 0).Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 7 {
		t.Fatalf("id = %d", u.ID)
	}
}

func TestValidateRejects(t *testing.T) {
	good := signed(t, `{"id":42,"first_name":"Ali"}`, now)
	tampered, _ := url.ParseQuery(good)
	tampered.Set("user", `{"id":43,"first_name":"Ali"}`)

	cases := []struct {
		name string
		init string
		bot  string
		want error
	}{
		{"чужой пользователь подставлен", tampered.Encode(), fakeBot, ErrInvalid},
		{"другой бот", good, "999:other", ErrInvalid},
		{"без hash", "auth_date=1&user=%7B%7D", fakeBot, ErrInvalid},
		{"пусто", "", fakeBot, ErrInvalid},
		{"мусор", "%zz", fakeBot, ErrInvalid},
		{"повтор поля", good + "&user=x", fakeBot, ErrInvalid},
		{"устарел", signed(t, `{"id":42}`, now.Add(-25*time.Hour)), fakeBot, ErrExpired},
		{"из будущего", signed(t, `{"id":42}`, now.Add(time.Hour)), fakeBot, ErrExpired},
		{"нет id", signed(t, `{"first_name":"Ali"}`, now), fakeBot, ErrInvalid},
		{"user не JSON", signed(t, `ali`, now), fakeBot, ErrInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Validate(c.init, c.bot, 24*time.Hour, now)
			if !errors.Is(err, c.want) {
				t.Fatalf("ждали %v, получили %v", c.want, err)
			}
		})
	}
}
