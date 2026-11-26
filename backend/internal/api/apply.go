package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"ayzonexam/internal/bot"
	"ayzonexam/internal/rating"
	"ayzonexam/internal/users"
)

// Заявка «Добавить свой центр / Я учитель» (DECISIONS 2026-11-25): кнопка на сайте открывает бота
// (start=apply-center / apply-teacher), бот — анкету в приложении; владелец одобряет кнопкой в боте —
// кабинет создаётся сам.

func (s *server) getApply(w http.ResponseWriter, r *http.Request, u users.User) {
	a, err := s.rating.MyApplication(r.Context(), u.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"application": a})
}

func (s *server) postApply(w http.ResponseWriter, r *http.Request, u users.User) {
	var b struct {
		Kind     string `json:"kind"`
		Name     string `json:"name"`
		City     string `json:"city"`
		Students int    `json:"students"`
		Phone    string `json:"phone"`
		About    string `json:"about"`
	}
	if !readJSON(w, r, &b, 8<<10) {
		return
	}
	b.Name, b.Phone, b.About = strings.TrimSpace(b.Name), strings.TrimSpace(b.Phone), strings.TrimSpace(b.About)
	switch {
	case b.Kind != rating.KindCenter && b.Kind != rating.KindTeacher,
		utf8.RuneCountInString(b.Name) < 2 || utf8.RuneCountInString(b.Name) > 80,
		!slices.Contains(rating.Regions, b.City),
		b.Students < 0 || b.Students > 100_000,
		utf8.RuneCountInString(b.Phone) > 30, utf8.RuneCountInString(b.About) > 1000:
		writeError(w, http.StatusBadRequest, "bad_application")
		return
	}
	ctx := r.Context()
	a, err := s.rating.Apply(ctx, u.ID, rating.Application{Kind: b.Kind, Name: b.Name, City: b.City, Students: b.Students, Phone: b.Phone, About: b.About}, s.Now())
	switch {
	case errors.Is(err, rating.ErrApplyStaff):
		writeError(w, http.StatusConflict, "already_staff")
		return
	case errors.Is(err, rating.ErrApplyPending):
		writeError(w, http.StatusConflict, "pending")
		return
	case err != nil:
		s.fail(w, r, err)
		return
	}
	what := "Центр"
	if a.Kind == rating.KindTeacher {
		what = "Учитель без центра"
	}
	contact := a.Nickname
	if a.Username != "" {
		contact += " · @" + a.Username
	}
	text := fmt.Sprintf("🆕 Заявка #%d: %s\n%s · %s · учеников: %d\nКонтакт: %s", a.ID, what, a.Name, a.City, a.Students, contact)
	if a.Phone != "" {
		text += " · " + a.Phone
	}
	if a.About != "" {
		text += "\n\n" + a.About
	}
	s.toAdmins(ctx, text, nil, applyButtons(a.ID), "application", a.ID)
	writeJSON(w, http.StatusOK, map[string]any{"application": a})
}

func applyButtons(id int64) [][]bot.Button {
	sid := strconv.FormatInt(id, 10)
	return [][]bot.Button{
		{{Text: "✅ Одобрить", Data: "ap:ok:" + sid}, {Text: "🎁 + месяц бесплатно", Data: "ap:pilot:" + sid}},
		{{Text: "❌ Отказать", Data: "ap:no:" + sid}},
	}
}

// decideApplication — кнопка владельца под заявкой: ok, pilot (одобрить с бесплатным месяцем) или no.
func (s *Server) decideApplication(ctx context.Context, id, admin int64, action string) (string, error) {
	if action == "no" {
		a, err := s.rating.Reject(ctx, id, admin, s.Now())
		if errors.Is(err, rating.ErrApplyDecided) {
			return "", errors.New("заявка уже рассмотрена")
		}
		if err != nil {
			return "", err
		}
		s.audit(ctx, admin, "application.reject", "application:"+strconv.FormatInt(id, 10), nil)
		s.say(ctx, a.TelegramID, a.Lang, "apply.rejected", nil, "support")
		return "❌ Отказано", nil
	}
	a, err := s.rating.Approve(ctx, id, admin, action == "pilot", s.Now())
	if errors.Is(err, rating.ErrApplyDecided) {
		return "", errors.New("заявка уже рассмотрена")
	}
	if err != nil {
		return "", err
	}
	s.audit(ctx, admin, "application.approve", "application:"+strconv.FormatInt(id, 10), map[string]any{"center_id": a.CenterID, "pilot": a.Pilot})
	if a.Pilot && a.PaidUntil != nil {
		s.say(ctx, a.TelegramID, a.Lang, "apply.pilot", map[string]string{"date": a.PaidUntil.Format("02.01.2006")}, "cabinet")
		return "✅ Одобрено, месяц бесплатно", nil
	}
	s.say(ctx, a.TelegramID, a.Lang, "apply.approved", nil, "cabinet")
	return "✅ Одобрено", nil
}
