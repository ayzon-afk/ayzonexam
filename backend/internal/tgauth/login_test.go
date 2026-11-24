package tgauth

import (
	"errors"
	"net/url"
	"testing"
	"time"
)

func TestValidateLogin(t *testing.T) {
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	signed := SignLogin(url.Values{"id": {"42"}, "first_name": {"Ali"}, "username": {"ali"},
		"auth_date": {"1795521600"}}, fakeBot) // 2026-11-24 12:00 UTC
	u, err := ValidateLogin(signed, fakeBot, 24*time.Hour, now)
	if err != nil || u.ID != 42 || u.FirstName != "Ali" || u.Username != "ali" {
		t.Fatalf("верная подпись: %+v %v", u, err)
	}
	// Подпись Mini App («WebAppData») для виджета не подходит, и наоборот.
	if _, err := Validate(signed.Encode(), fakeBot, 24*time.Hour, now); err == nil {
		t.Fatal("данные виджета приняты как initData")
	}
	forged := url.Values{}
	for k, v := range signed {
		forged[k] = v
	}
	forged.Set("id", "43")
	if _, err := ValidateLogin(forged, fakeBot, 24*time.Hour, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("подменённый id: %v", err)
	}
	if _, err := ValidateLogin(signed, "654321:OTHER", 24*time.Hour, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("чужой бот: %v", err)
	}
	if _, err := ValidateLogin(signed, fakeBot, 24*time.Hour, now.Add(25*time.Hour)); !errors.Is(err, ErrExpired) {
		t.Fatalf("старый вход: %v", err)
	}
}
