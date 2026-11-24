package api

import (
	"context"
	"net/http"
	"slices"
	"time"

	"ayzonexam/internal/billing"
	"ayzonexam/internal/bot"
	"ayzonexam/internal/exam"
	"ayzonexam/internal/rating"
	"ayzonexam/internal/users"
)

// BotAPI — то, чем API пользуется в боте. nil — бот выключен (тесты, локально без токена): сообщения не
// уходят, приложение работает.
type BotAPI interface {
	// ToUser — ученику текст key (bot/texts.go) на его языке; open — что открыть кнопкой ('-' — без кнопки).
	ToUser(ctx context.Context, tgID int64, lang, key string, args map[string]string, open string) error
	// ToAdmins — владельцу и помощникам; ответ админа reply на это сообщение свяжется через bot_refs.
	ToAdmins(ctx context.Context, text string, photo *bot.Photo, buttons [][]bot.Button) ([]bot.Sent, error)
	Send(ctx context.Context, chatID int64, text string, buttons [][]bot.Button) (bot.Sent, error)
	Greet(ctx context.Context, chatID int64, lang, text string) error
	AnswerCallback(ctx context.Context, id, text string) error
	SetButtons(ctx context.Context, chatID, messageID int64, buttons [][]bot.Button) error
	Download(ctx context.Context, fileID string, max int64) ([]byte, error)
	SendDocument(ctx context.Context, chatID int64, name string, data []byte, caption string) error
}

type (
	Photo  = bot.Photo
	Button = bot.Button
)

// say — сообщение ученику, ошибка только в журнал: приложение работает и без бота.
func (s *server) say(ctx context.Context, tgID int64, lang, key string, args map[string]string, open string) {
	if s.Msg == nil {
		return
	}
	if err := s.Msg.ToUser(context.WithoutCancel(ctx), tgID, lang, key, args, open); err != nil {
		s.Log.Warn("бот: сообщение ученику не отправлено", "key", key, "err", err)
	}
}

func (s *server) isAdmin(u users.User) bool { return slices.Contains(s.Admins, u.TelegramID) }

// planView — тариф ученика: Premium, лимиты, цены. Цены одинаковые везде (SPEC, «Деньги»).
type planView struct {
	Premium       bool           `json:"premium"`
	Until         *time.Time     `json:"until"`
	Source        string         `json:"source"` // trial, paid, center
	Pending       bool           `json:"pending"`
	Rejected      bool           `json:"rejected"`
	DiscountUntil *time.Time     `json:"discount_until"`
	MockCredits   int            `json:"mock_credits"`
	Bonus         int            `json:"bonus"`
	Prices        map[string]int `json:"prices"`
}

// fillMe дополняет профиль тарифом и кодом приглашения друзей.
func (s *server) fillMe(r *http.Request, u users.User, v *meView) error {
	ctx := r.Context()
	now := s.Now()
	a := billing.Of(u, now)
	pending, err := s.billing.Pending(ctx, u.ID)
	if err != nil {
		return err
	}
	paid, err := s.billing.PaidBefore(ctx, u.ID)
	if err != nil {
		return err
	}
	p := &planView{Premium: a.Premium, Until: a.Until, Source: a.Source, Pending: pending != nil, Rejected: u.PayTrust == "rejected",
		DiscountUntil: billing.DiscountUntil(u, paid, now), MockCredits: u.MockCredits, Bonus: u.BonusChecks,
		Prices: map[string]int{"month": billing.PriceMonth, "month_discount": billing.PriceMonthDiscount,
			"quarter": billing.PriceQuarter, "mock": billing.PriceMock}}
	v.Plan = p
	code, err := s.social.InviteCode(ctx, u)
	v.Invite = code
	return err
}

// statsView — профиль в цифрах: текущий балл, график, значки, друзья.
type statsView struct {
	Current  *float64        `json:"current"`
	Parts    int             `json:"parts"`
	Progress []float64       `json:"progress"`
	Badges   map[string]bool `json:"badges"`
	Friends  int             `json:"friends"`
	Invited  int             `json:"invited"`
	Place    *int            `json:"place"` // место в стране
}

func (s *server) getStats(w http.ResponseWriter, r *http.Request, u users.User) {
	ctx := r.Context()
	now := s.Now()
	var v statsView
	var err error
	if v.Current, v.Parts, err = s.rating.Current(ctx, u.ID, now); err != nil {
		s.fail(w, r, err)
		return
	}
	if v.Progress, err = s.rating.Progress(ctx, u.ID, 12); err != nil {
		s.fail(w, r, err)
		return
	}
	if v.Progress == nil {
		v.Progress = []float64{}
	}
	var best *float64
	var duels, mocks, essays, tours int
	err = s.DB.QueryRow(ctx, `SELECT
			greatest((SELECT max(band) FROM rated_parts WHERE user_id = $1), (SELECT max(overall) FROM mocks WHERE user_id = $1))::float8,
			(SELECT count(*) FROM duel_players WHERE user_id = $1 AND finished_at IS NOT NULL),
			(SELECT count(*) FROM mocks WHERE user_id = $1 AND status = 'done'),
			(SELECT count(*) FROM writing_checks WHERE user_id = $1 AND status = 'done'),
			(SELECT count(DISTINCT tournament_id) FROM attempts WHERE user_id = $1 AND tournament_id IS NOT NULL),
			(SELECT count(*) FROM friends WHERE user_id = $1),
			(SELECT count(*) FROM users WHERE referred_by = $1)`, u.ID).
		Scan(&best, &duels, &mocks, &essays, &tours, &v.Friends, &v.Invited)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	b := 0.0
	if best != nil {
		b = *best
	}
	v.Badges = map[string]bool{
		"first6": b >= 6, "first7": b >= 7, "duel10": duels >= 10,
		"tour": tours > 0, "mock10": mocks >= 10, "essay100": essays >= 100,
	}
	if v.Current != nil {
		board, err := s.rating.Students(ctx, rating.Country, u.ID, nil, now)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if board.Mine != nil {
			v.Place = &board.Mine.Place
		}
	}
	writeJSON(w, http.StatusOK, v)
}

// afterAttempt — что ещё меняет сданная попытка: бонус пригласившему за тест уровня друга, часть mock.
func (s *server) afterAttempt(r *http.Request, u users.User, a exam.Attempt) {
	ctx := context.WithoutCancel(r.Context())
	if a.Kind == exam.Level && u.ReferredBy != nil {
		ref, err := s.social.RewardLevel(ctx, u.ID)
		if err != nil {
			s.Log.Error("бонус за друга не выдан", "user", u.ID, "err", err)
		} else if ref != nil {
			s.say(ctx, ref.TelegramID, ref.Lang, "bonus.level", map[string]string{"nick": u.Nickname}, "premium")
		}
	}
	if a.MockID != nil {
		if err := s.mockPartDone(ctx, *a.MockID); err != nil {
			s.Log.Error("часть mock не записана", "mock", *a.MockID, "err", err)
		}
	}
}

// notifyCenterStaff — аккаунту центра и учителям: ученик просится в центр.
func (s *server) notifyCenterStaff(r *http.Request, centerID int64) {
	if s.Msg == nil {
		return
	}
	rows, err := s.DB.Query(r.Context(), `SELECT telegram_id, lang FROM users
		WHERE center_id = $1 AND center_status = 'member' AND center_role IS NOT NULL`, centerID)
	if err != nil {
		s.Log.Error("сотрудники центра не прочитаны", "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var tg int64
		var lang string
		if rows.Scan(&tg, &lang) == nil {
			s.say(r.Context(), tg, lang, "center.request", nil, "center-requests")
		}
	}
}

// toAdmins — сообщение владельцу и помощникам; kind/refID — к чему относится (ответ reply, кнопки).
func (s *server) toAdmins(ctx context.Context, text string, photo *Photo, buttons [][]Button, kind string, refID int64) {
	if s.Msg == nil {
		return
	}
	ctx = context.WithoutCancel(ctx)
	sent, err := s.Msg.ToAdmins(ctx, text, photo, buttons)
	if err != nil {
		s.Log.Warn("бот: сообщение админам не отправлено", "kind", kind, "err", err)
	}
	for _, m := range sent {
		if err := s.support.SaveRef(ctx, m.ChatID, m.MessageID, kind, refID); err != nil {
			s.Log.Error("bot_refs не записан", "err", err)
		}
	}
}
