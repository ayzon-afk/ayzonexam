package ai

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestQuotaReset(t *testing.T) {
	utc := func(mo time.Month, d, h, mi int) time.Time { return time.Date(2026, mo, d, h, mi, 0, 0, time.UTC) }
	cases := []struct{ now, want time.Time }{
		// Зима: полночь PST — 08:00 UTC, 13:00 по Ташкенту.
		{utc(11, 19, 7, 59), utc(11, 19, 8, 0)},
		{utc(11, 19, 8, 0), utc(11, 20, 8, 0)},
		// Лето: полночь PDT — 07:00 UTC, 12:00 по Ташкенту.
		{utc(7, 1, 6, 59), utc(7, 1, 7, 0)},
		{utc(7, 1, 7, 0), utc(7, 2, 7, 0)},
		// 8 марта 2026 — переход на летнее в 2:00: сам день начинается ещё по PST, следующий — уже по PDT.
		{utc(3, 8, 7, 30), utc(3, 8, 8, 0)},
		{utc(3, 8, 8, 0), utc(3, 9, 7, 0)},
		// 1 ноября 2026 — возврат на зимнее: день начинается по PDT, следующий — по PST (в сутках 25 часов).
		{utc(11, 1, 6, 30), utc(11, 1, 7, 0)},
		{utc(11, 1, 7, 0), utc(11, 2, 8, 0)},
		{utc(11, 2, 7, 30), utc(11, 2, 8, 0)},
	}
	for _, c := range cases {
		if got := QuotaReset(c.now); !got.Equal(c.want) {
			t.Errorf("QuotaReset(%s) = %s, ждали %s", c.now.Format(time.RFC3339), got.UTC().Format(time.RFC3339), c.want.Format(time.RFC3339))
		}
	}
}

func TestQuotaStopsRequest(t *testing.T) {
	g, req, _ := fakeGemini(t, 200, candidate("text"))
	var asked []string
	g.Take = func(_ context.Context, model string) (bool, error) {
		asked = append(asked, model)
		return false, nil
	}
	if _, err := g.Recognize(context.Background(), []Image{{MIME: "image/jpeg", Data: []byte{1}}}); !errors.Is(err, ErrQuota) {
		t.Fatalf("ждали ErrQuota: %v", err)
	}
	if req.Method != "" {
		t.Fatal("сверх лимита запрос не должен уходить в Gemini")
	}
	if len(asked) != 1 || asked[0] != "gemini-test" {
		t.Fatalf("лимит спрашивают по модели запроса: %v", asked)
	}

	g.Take = func(context.Context, string) (bool, error) { return true, nil }
	if _, err := g.Recognize(context.Background(), []Image{{MIME: "image/jpeg", Data: []byte{1}}}); err != nil || req.Method != http.MethodPost {
		t.Fatalf("в пределах лимита запрос уходит: %v", err)
	}
}
