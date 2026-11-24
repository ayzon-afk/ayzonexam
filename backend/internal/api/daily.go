package api

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"ayzonexam/internal/bot"
)

// Вопрос дня (хакатон, DECISIONS 2026-11-24): в 9:00 по Ташкенту бот шлёт вопрос из банка теста уровня всем,
// кто прошёл онбординг и не отключил; ответ — кнопкой, засчитывается первый.

const dailyHour = 9

type dailyQ struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Answer  int      `json:"answer"`
}

// dailyQuestion — вопрос дня day: по порядку из банка, по кругу. Банк меняется — меняется и порядок; ответ
// на вчерашний вопрос сверяется с тем, что лежит в банке сейчас, — для вопроса дня этого достаточно.
func (s *server) dailyQuestion(ctx context.Context, day time.Time) (dailyQ, error) {
	rows, err := s.DB.Query(ctx, `SELECT questions FROM tests WHERE kind = 'level' ORDER BY id`)
	if err != nil {
		return dailyQ{}, err
	}
	defer rows.Close()
	var all []dailyQ
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return dailyQ{}, err
		}
		var qs []dailyQ
		if err := json.Unmarshal(raw, &qs); err != nil {
			return dailyQ{}, err
		}
		for _, q := range qs {
			if q.Q != "" && len(q.Options) >= 2 && q.Answer >= 0 && q.Answer < len(q.Options) {
				all = append(all, q)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return dailyQ{}, err
	}
	if len(all) == 0 {
		return dailyQ{}, errors.New("банк теста уровня пуст")
	}
	n := int(day.Unix() / 86400)
	return all[n%len(all)], nil
}

func dayKey(d time.Time) string { return d.Format("20060102") }

// RunDaily раз в минуту проверяет, не пора ли разослать вопрос дня. До отмены ctx.
func (s *server) RunDaily(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		if err := s.SendDaily(ctx); err != nil {
			s.Log.Error("вопрос дня не разослан", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// SendDaily — рассылка за сегодня, если 9:00 прошло и сегодня ещё не слали.
func (s *server) SendDaily(ctx context.Context) error {
	if s.Msg == nil {
		return nil
	}
	now := s.Now().In(tashkent)
	if now.Hour() < dailyHour {
		return nil
	}
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tag, err := s.DB.Exec(ctx, `INSERT INTO daily_sends (day) VALUES ($1) ON CONFLICT DO NOTHING`, day)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	q, err := s.dailyQuestion(ctx, day)
	if err != nil {
		return err
	}
	rows, err := s.DB.Query(ctx, `SELECT telegram_id, lang FROM users
		WHERE daily_q AND onboarded_at IS NOT NULL AND blocked_at IS NULL ORDER BY id`)
	if err != nil {
		return err
	}
	type to struct {
		tg   int64
		lang string
	}
	var list []to
	for rows.Next() {
		var r to
		if err := rows.Scan(&r.tg, &r.lang); err != nil {
			rows.Close()
			return err
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	sent := 0
	for _, r := range list {
		if ctx.Err() != nil {
			break
		}
		// Кто не нажимал /start, тому Telegram писать не даёт — это не ошибка рассылки.
		if _, err := s.Msg.Send(ctx, r.tg, bot.T(r.lang, "daily.q")+"\n\n"+q.Q, dailyButtons(day, q, r.lang)); err == nil {
			sent++
		}
		time.Sleep(50 * time.Millisecond) // Telegram: не больше ~30 сообщений в секунду
	}
	_, err = s.DB.Exec(context.WithoutCancel(ctx), `UPDATE daily_sends SET sent = $2 WHERE day = $1`, day, sent)
	s.Log.Info("вопрос дня разослан", "day", day.Format(time.DateOnly), "sent", sent, "of", len(list))
	return err
}

func dailyButtons(day time.Time, q dailyQ, lang string) [][]bot.Button {
	var rows [][]bot.Button
	for i := 0; i < len(q.Options); i += 2 {
		row := []bot.Button{{Text: q.Options[i], Data: "dq:" + dayKey(day) + ":" + strconv.Itoa(i)}}
		if i+1 < len(q.Options) {
			row = append(row, bot.Button{Text: q.Options[i+1], Data: "dq:" + dayKey(day) + ":" + strconv.Itoa(i+1)})
		}
		rows = append(rows, row)
	}
	return append(rows, []bot.Button{{Text: bot.T(lang, "daily.offbtn"), Data: "dq:off"}})
}

// dailyCallback — нажата кнопка под вопросом дня (dq:<ГГГГММДД>:<вариант> или dq:off).
func (s *Server) dailyCallback(ctx context.Context, c *bot.Callback) {
	answer := func(t string) {
		if err := s.Msg.AnswerCallback(ctx, c.ID, t); err != nil {
			s.Log.Warn("бот: callback без ответа", "err", err)
		}
	}
	usr, err := s.botUser(ctx, &c.From)
	if err != nil {
		s.Log.Error("бот: ученик не найден", "err", err)
		answer("")
		return
	}
	if c.Data == "dq:off" {
		if _, err := s.DB.Exec(ctx, `UPDATE users SET daily_q = false WHERE id = $1`, usr.ID); err != nil {
			s.Log.Error("вопрос дня: база", "err", err)
		}
		answer(bot.T(usr.Lang, "daily.off"))
		return
	}
	parts := strings.Split(c.Data, ":")
	if len(parts) != 3 {
		answer("")
		return
	}
	day, err1 := time.Parse("20060102", parts[1])
	pick, err2 := strconv.Atoi(parts[2])
	// Ответ — только на вопрос последних 7 дней: старые кнопки и подделанные даты не считаются.
	if age := s.Now().Sub(day); err1 != nil || err2 != nil || age < -24*time.Hour || age > 7*24*time.Hour {
		answer("")
		return
	}
	q, err := s.dailyQuestion(ctx, day)
	if err != nil || pick < 0 || pick >= len(q.Options) {
		answer("")
		return
	}
	right := pick == q.Answer
	tag, err := s.DB.Exec(ctx, `INSERT INTO daily_answers (user_id, day, correct) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		usr.ID, day, right)
	if err != nil {
		s.Log.Error("вопрос дня: база", "err", err)
		answer("")
		return
	}
	mark := "✅ "
	if !right {
		mark = "❌ "
	}
	if c.Message != nil {
		b := [][]bot.Button{{{Text: mark + q.Options[pick], Data: "noop:0"}}}
		if !right {
			b = append(b, []bot.Button{{Text: "✅ " + q.Options[q.Answer], Data: "noop:0"}})
		}
		if err := s.Msg.SetButtons(ctx, c.Message.Chat.ID, c.Message.MessageID, b); err != nil {
			s.Log.Warn("бот: кнопки не обновлены", "err", err)
		}
	}
	if tag.RowsAffected() == 0 { // уже отвечал — второй раз не считается и приглашение не повторяется
		answer("")
		return
	}
	key := "daily.right"
	if !right {
		key = "daily.wrong"
	}
	answer(strings.TrimSpace(mark))
	s.say(ctx, usr.TelegramID, usr.Lang, key, map[string]string{"answer": q.Options[q.Answer]}, "")
}
