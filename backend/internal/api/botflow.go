package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/billing"
	"ayzonexam/internal/bot"
	"ayzonexam/internal/news"
	"ayzonexam/internal/speaking"
	"ayzonexam/internal/support"
	"ayzonexam/internal/tgauth"
	"ayzonexam/internal/users"
)

// HandleUpdate — что бот делает с входящим (bot.OnUpdate). false — не наш случай: бот ответит кнопкой
// запуска приложения.
//
// Ученик: /help — следующее сообщение уходит в поддержку; голосовое — ответ Speaking; /cancel.
// Админ: ответ (reply) на обращение — уходит ученику; /post — новость; /hide <id>; кнопки оплат.
func (s *Server) HandleUpdate(ctx context.Context, u bot.Update) bool {
	if s.Msg == nil {
		return false
	}
	if c := u.CallbackQuery; c != nil && strings.HasPrefix(c.Data, "dq:") {
		s.dailyCallback(ctx, c)
		return true
	}
	if c := u.CallbackQuery; c != nil {
		s.callback(ctx, c)
		return true
	}
	m := u.Message
	if m == nil || m.From == nil {
		return false
	}
	if slices.Contains(s.Admins, m.From.ID) && s.adminMessage(ctx, m) {
		return true
	}
	text := strings.TrimSpace(m.Text)
	if strings.HasPrefix(text, "/start") {
		return false
	}
	usr, err := s.botUser(ctx, m.From)
	if err != nil {
		s.Log.Error("бот: ученик не найден", "err", err)
		return false
	}
	if usr.BlockedAt != nil {
		return true
	}
	switch {
	case text == "/help":
		s.setState(ctx, usr, "help")
		s.say(ctx, usr.TelegramID, usr.Lang, "help.ask", nil, "-")
		return true
	case text == "/daily":
		if _, err := s.DB.Exec(ctx, `UPDATE users SET daily_q = true WHERE id = $1`, usr.ID); err != nil {
			s.Log.Error("вопрос дня: база", "err", err)
		}
		s.say(ctx, usr.TelegramID, usr.Lang, "daily.on", nil, "-")
		return true
	case text == "/cancel":
		s.setState(ctx, usr, "")
		s.say(ctx, usr.TelegramID, usr.Lang, "cancelled", nil, "-")
		return true
	case usr.BotState == "help":
		s.botSupport(ctx, usr, m)
		return true
	case m.Voice != nil || m.Audio != nil:
		s.botVoice(ctx, usr, m)
		return true
	case strings.HasPrefix(usr.BotState, "speaking:"):
		s.say(ctx, usr.TelegramID, usr.Lang, "speaking.voiceonly", nil, "-")
		return true
	}
	return false
}

// botUser — ученик по Telegram; кто ещё не открывал приложение — создаётся (как при первом входе).
func (s *Server) botUser(ctx context.Context, from *bot.User) (users.User, error) {
	u, err := s.Users.ByTelegram(ctx, from.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.Users.Ensure(ctx, tgauth.User{ID: from.ID, FirstName: from.FirstName, Username: from.Username, LanguageCode: from.LanguageCode})
	}
	return u, err
}

func (s *Server) setState(ctx context.Context, u users.User, state string) {
	if err := s.Users.SetBotState(ctx, u.ID, state); err != nil {
		s.Log.Error("бот: состояние не записано", "err", err)
	}
}

// botSupport — сообщение в поддержку из бота: текст или фото с подписью.
func (s *Server) botSupport(ctx context.Context, u users.User, m *bot.Message) {
	text := m.Text
	var photo *Photo
	var mediaID *int64
	if len(m.Photo) > 0 {
		text = m.Caption
		big := m.Photo[len(m.Photo)-1]
		data, err := s.Msg.Download(ctx, big.FileID, maxImageBytes)
		if err == nil {
			if img, code := imageFromBytes(data); code == "" {
				var id int64
				if err := s.DB.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, true) RETURNING id`, img.MIME, img.Data).Scan(&id); err == nil {
					mediaID, photo = &id, &Photo{MIME: img.MIME, Data: img.Data}
				}
			}
		}
	}
	id, err := s.support.Write(ctx, u.ID, text, mediaID)
	if errors.Is(err, support.ErrEmpty) {
		s.say(ctx, u.TelegramID, u.Lang, "help.ask", nil, "-")
		return
	}
	if err != nil {
		s.Log.Error("бот: обращение не записано", "err", err)
		return
	}
	s.setState(ctx, u, "")
	s.ticketToAdmins(ctx, u, id, text, photo)
	s.say(ctx, u.TelegramID, u.Lang, "help.sent", nil, "-")
}

// botVoice — голосовое: ответ на вопрос, присланный ботом (speaking:<id>), или свободный ответ.
func (s *Server) botVoice(ctx context.Context, u users.User, m *bot.Message) {
	f := m.Voice
	if f == nil {
		f = m.Audio
	}
	if f.Duration > speaking.MaxSeconds {
		s.say(ctx, u.TelegramID, u.Lang, "speaking.toolong", nil, "-")
		return
	}
	q := ai.SpeakingQuestion{Part: 0, Text: ""}
	if id, ok := strings.CutPrefix(u.BotState, "speaking:"); ok {
		qid, _ := strconv.ParseInt(id, 10, 64)
		if sq, err := s.speaking.Question(ctx, qid); err == nil {
			q = ai.SpeakingQuestion{Part: sq.Part, Text: sq.Text}
		}
	}
	data, err := s.Msg.Download(ctx, f.FileID, speaking.MaxAudioBytes)
	if err != nil {
		s.Log.Warn("бот: голосовое не скачано", "err", err)
		s.say(ctx, u.TelegramID, u.Lang, "speaking.failed", nil, "-")
		return
	}
	mime := audioMIME(data)
	if mime == "" {
		mime = "audio/ogg"
	}
	quota := billing.SpeakingQuota(billing.Of(u, s.Now()), s.Now())
	_, err = s.speaking.Create(ctx, speaking.NewCheck{UserID: u.ID, Source: "bot", Questions: []ai.SpeakingQuestion{q},
		Audio: []ai.Audio{{MIME: mime, Data: data}}, Seconds: f.Duration, Lang: u.Lang}, quota.Limit, quota.Since, s.Now())
	if errors.Is(err, speaking.ErrLimit) {
		s.say(ctx, u.TelegramID, u.Lang, "speaking.limit", nil, "premium")
		return
	}
	if err != nil {
		s.Log.Error("бот: проверка Speaking не создана", "err", err)
		return
	}
	s.setState(ctx, u, "")
	s.say(ctx, u.TelegramID, u.Lang, "speaking.received", nil, "-")
}

// adminMessage — сообщения админа боту. false — обычное сообщение, бот ответит как ученику.
func (s *Server) adminMessage(ctx context.Context, m *bot.Message) bool {
	text := strings.TrimSpace(m.Text)
	admin := m.From.ID
	reply := func(t string) {
		if _, err := s.Msg.Send(ctx, m.Chat.ID, t, nil); err != nil {
			s.Log.Warn("бот: ответ админу не отправлен", "err", err)
		}
	}
	switch {
	case m.ReplyTo != nil:
		kind, ref, err := s.support.Ref(ctx, m.Chat.ID, m.ReplyTo.MessageID)
		if err != nil || kind != "ticket" {
			return false
		}
		if text == "" {
			reply("Ответ — только текстом.")
			return true
		}
		if err := s.replyTicket(ctx, ref, admin, text); err != nil {
			reply("Не отправлено: " + err.Error())
			return true
		}
		reply(fmt.Sprintf("✅ Ответ на обращение #%d отправлен.", ref))
		return true
	case strings.HasPrefix(text, "/post"):
		p, err := news.FromMessage(strings.TrimSpace(strings.TrimPrefix(text, "/post")))
		if err != nil {
			reply("Формат: /post Заголовок (можно с #result, #tournament, #news)\nТекст поста со следующей строки.")
			return true
		}
		p, err = s.news.Create(ctx, p, admin)
		if err != nil {
			reply("Не сохранено: " + err.Error())
			return true
		}
		s.audit(ctx, admin, "news.create", "post:"+strconv.FormatInt(p.ID, 10), map[string]any{"title": p.Title})
		reply(fmt.Sprintf("✅ Пост #%d опубликован. Скрыть: /hide %d", p.ID, p.ID))
		return true
	case strings.HasPrefix(text, "/hide"):
		id, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(text, "/hide")), 10, 64)
		if err != nil || s.news.SetHidden(ctx, id, true) != nil {
			reply("Формат: /hide <номер поста>")
			return true
		}
		s.audit(ctx, admin, "news.hide", "post:"+strconv.FormatInt(id, 10), nil)
		reply(fmt.Sprintf("Пост #%d скрыт.", id))
		return true
	}
	return false
}

// replyTicket — ответ админа ученику (из бота или админки).
func (s *Server) replyTicket(ctx context.Context, ticketID, admin int64, text string) error {
	tg, lang, err := s.support.Reply(ctx, ticketID, admin, text)
	if err != nil {
		return err
	}
	s.audit(ctx, admin, "ticket.reply", "ticket:"+strconv.FormatInt(ticketID, 10), nil)
	if s.Msg == nil {
		return nil
	}
	return s.Msg.ToUser(context.WithoutCancel(ctx), tg, lang, "support.reply", map[string]string{"text": text}, "support")
}

// Причины отказа в оплате — кнопками, чтобы ученик получил понятный текст на своём языке.
var rejectReasons = map[string]string{"notfound": "перевод не найден", "amount": "не та сумма", "screenshot": "скрин не подходит"}

// callback — кнопки под сообщениями админам: оплаты и обращения.
func (s *Server) callback(ctx context.Context, c *bot.Callback) {
	answer := func(t string) {
		if err := s.Msg.AnswerCallback(ctx, c.ID, t); err != nil {
			s.Log.Warn("бот: callback без ответа", "err", err)
		}
	}
	if !slices.Contains(s.Admins, c.From.ID) {
		answer("")
		return
	}
	parts := strings.Split(c.Data, ":")
	id, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	setButtons := func(b [][]bot.Button) {
		if c.Message != nil {
			if err := s.Msg.SetButtons(ctx, c.Message.Chat.ID, c.Message.MessageID, b); err != nil {
				s.Log.Warn("бот: кнопки не обновлены", "err", err)
			}
		}
	}
	switch {
	case len(parts) == 3 && parts[0] == "pay" && parts[1] == "ok":
		if err := s.confirmPayment(ctx, id, c.From.ID); err != nil {
			answer(err.Error())
			return
		}
		setButtons([][]bot.Button{{{Text: "✅ Подтверждено", Data: "noop:0"}}})
		answer("Подтверждено")
	case len(parts) == 3 && parts[0] == "pay" && parts[1] == "no":
		setButtons([][]bot.Button{
			{{Text: "Перевод не найден", Data: "pay:r:notfound:" + parts[2]}},
			{{Text: "Не та сумма", Data: "pay:r:amount:" + parts[2]}},
			{{Text: "Скрин не подходит", Data: "pay:r:screenshot:" + parts[2]}},
			{{Text: "‹ Назад", Data: "pay:back:" + parts[2]}},
		})
		answer("Выберите причину")
	case len(parts) == 3 && parts[0] == "pay" && parts[1] == "back":
		setButtons(payButtons(id))
		answer("")
	case len(parts) == 4 && parts[0] == "pay" && parts[1] == "r":
		if err := s.rejectPayment(ctx, id, c.From.ID, parts[2]); err != nil {
			answer(err.Error())
			return
		}
		setButtons([][]bot.Button{{{Text: "❌ Отказано: " + rejectReasons[parts[2]], Data: "noop:0"}}})
		answer("Отказано")
	case len(parts) == 3 && parts[0] == "tk" && parts[1] == "close":
		if err := s.support.SetStatus(ctx, id, "closed"); err != nil {
			answer(err.Error())
			return
		}
		s.audit(ctx, c.From.ID, "ticket.close", "ticket:"+parts[2], nil)
		setButtons([][]bot.Button{{{Text: "Закрыто", Data: "noop:0"}}})
		answer("Закрыто")
	default:
		answer("")
	}
}

func payButtons(id int64) [][]bot.Button {
	sid := strconv.FormatInt(id, 10)
	return [][]bot.Button{{{Text: "✅ Подтвердить", Data: "pay:ok:" + sid}, {Text: "❌ Отказать", Data: "pay:no:" + sid}}}
}

// confirmPayment — «Подтвердить» (кнопка в Telegram или админка): ученику — сообщение, пригласившему —
// +7 дней за первую оплату друга.
func (s *Server) confirmPayment(ctx context.Context, id, admin int64) error {
	d, err := s.billing.Confirm(ctx, id, admin, s.Now())
	if errors.Is(err, billing.ErrDecided) {
		return errors.New("оплата уже проверена")
	}
	if err != nil {
		return err
	}
	s.audit(ctx, admin, "payment.confirm", "payment:"+strconv.FormatInt(id, 10), map[string]any{"amount": d.Amount, "plan": d.Plan})
	key := "pay.confirmed"
	if d.Plan == "mock" {
		key = "pay.confirmed.mock"
	}
	s.say(ctx, d.TelegramID, d.Lang, key, nil, "premium")
	if d.ReferrerTG != 0 {
		s.say(ctx, d.ReferrerTG, d.ReferrerLang, "bonus.premium", nil, "premium")
	}
	return nil
}

// rejectPayment — «Отказать»: Premium выключается, ученику — причина на его языке.
func (s *Server) rejectPayment(ctx context.Context, id, admin int64, reason string) error {
	if _, ok := rejectReasons[reason]; !ok {
		return errors.New("неизвестная причина")
	}
	p, err := s.billing.Reject(ctx, id, admin, reason, s.Now())
	if errors.Is(err, billing.ErrDecided) {
		return errors.New("оплата уже проверена")
	}
	if err != nil {
		return err
	}
	s.audit(ctx, admin, "payment.reject", "payment:"+strconv.FormatInt(id, 10), map[string]any{"reason": reason})
	s.say(ctx, p.TelegramID, p.Lang, "pay.rejected", map[string]string{"reason": bot.T(p.Lang, "pay.reason."+reason)}, "premium")
	return nil
}

// imageFromBytes — картинка из байтов (фото из Telegram): тип по содержимому.
func imageFromBytes(data []byte) (ai.Image, string) {
	switch mime := http.DetectContentType(data); mime {
	case "image/jpeg", "image/png", "image/webp":
		return ai.Image{MIME: mime, Data: data}, ""
	}
	return ai.Image{}, "bad_image"
}
