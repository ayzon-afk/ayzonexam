package api

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDailyQuestion(t *testing.T) {
	// 08:00 по Ташкенту — рано; в 09:30 — рассылка, один раз за день.
	c := &clock{t: time.Date(2026, 12, 1, 3, 0, 0, 0, time.UTC)}
	msg := &fakeBotAPI{}
	h := newHandler(t, Deps{Now: c.now, Msg: msg})
	const tg = 9981
	if r := do(t, h, "PATCH", "/api/me", initDataAt(tg, c.now()), `{"goal_band":7,"onboarded":true}`); r.code != 200 {
		t.Fatalf("онбординг: %d %s", r.code, r.raw)
	}
	ctx := t.Context()
	if err := h.SendDaily(ctx); err != nil || len(msg.sent) != 0 {
		t.Fatalf("до 9:00 не шлём: %v %d", err, len(msg.sent))
	}
	c.add(90 * time.Minute)
	if err := h.SendDaily(ctx); err != nil {
		t.Fatal(err)
	}
	var mine *sentMsg
	for i := range msg.sent {
		if msg.sent[i].chat == tg {
			mine = &msg.sent[i]
		}
	}
	if mine == nil || !strings.Contains(mine.text, "Вопрос дня") || len(mine.buttons) < 2 {
		t.Fatalf("вопрос ученику: %+v", msg.sent)
	}
	n := len(msg.sent)
	if err := h.SendDaily(ctx); err != nil || len(msg.sent) != n {
		t.Fatalf("второй раз за день не шлём: %v %d→%d", err, n, len(msg.sent))
	}

	q, err := h.dailyQuestion(ctx, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	wrong := (q.Answer + 1) % len(q.Options)
	press := func(data string) {
		h.HandleUpdate(ctx, update(t, fmt.Sprintf(`{"callback_query":{"id":"cb","data":%q,"from":{"id":%d,"first_name":"A"},
			"message":{"message_id":1,"chat":{"id":%d,"type":"private"}}}}`, data, tg, tg)))
	}
	press(fmt.Sprintf("dq:20261201:%d", wrong))
	press(fmt.Sprintf("dq:20261201:%d", q.Answer)) // второй ответ не считается
	var correct bool
	var answers int
	pool.QueryRow(ctx, `SELECT count(*), bool_or(correct) FROM daily_answers a JOIN users u ON u.id = a.user_id
		WHERE u.telegram_id = $1`, tg).Scan(&answers, &correct)
	if answers != 1 || correct {
		t.Fatalf("засчитан первый ответ (неверный): %d %v", answers, correct)
	}
	if !strings.Contains(strings.Join(msg.users, "|"), fmt.Sprintf("%d:daily.wrong", tg)) {
		t.Fatalf("после ответа — приглашение в приложение: %v", msg.users)
	}
	press("dq:bad")
	press("dq:off")
	var on bool
	pool.QueryRow(ctx, `SELECT daily_q FROM users WHERE telegram_id = $1`, tg).Scan(&on)
	if on {
		t.Fatal("«Отключить» не сработало")
	}
}
