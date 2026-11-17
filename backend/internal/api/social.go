package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"ayzonexam/internal/social"
	"ayzonexam/internal/users"
)

// Друзья, приглашения и дуэли (этап 9).

func (s *server) refer(w http.ResponseWriter, r *http.Request, u users.User) {
	var b struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	ref, isNew, err := s.social.Refer(r.Context(), u, b.Code, s.Now())
	switch {
	case errors.Is(err, social.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
		return
	case errors.Is(err, social.ErrSelf):
		writeError(w, http.StatusBadRequest, "self")
		return
	case err != nil:
		s.fail(w, r, err)
		return
	}
	if isNew {
		s.say(r.Context(), ref.TelegramID, ref.Lang, "friend.joined", map[string]string{"nick": u.Nickname}, "friends")
	}
	writeJSON(w, http.StatusOK, map[string]any{"friend": ref.Nickname, "referred": isNew})
}

func (s *server) getFriends(w http.ResponseWriter, r *http.Request, u users.User) {
	list, err := s.social.Friends(r.Context(), u.ID, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if list == nil {
		list = []social.Friend{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"friends": list})
}

func (s *server) removeFriend(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.social.Unfriend(r.Context(), u.ID, id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) createDuel(w http.ResponseWriter, r *http.Request, u users.User) {
	var b struct {
		FriendID *int64 `json:"friend_id"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	ctx := r.Context()
	if b.FriendID != nil {
		ok, err := s.social.AreFriends(ctx, u.ID, *b.FriendID)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if !ok {
			writeError(w, http.StatusForbidden, "not_friend")
			return
		}
	}
	d, err := s.social.CreateDuel(ctx, u.ID, b.FriendID, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if b.FriendID != nil {
		if f, err := s.social.WhoByID(ctx, *b.FriendID); err == nil {
			s.say(ctx, f.TelegramID, f.Lang, "duel.challenge", map[string]string{"nick": u.Nickname}, "duel-"+d.Code)
		}
	}
	v, err := s.social.View(ctx, d.ID, u.ID, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *server) listDuels(w http.ResponseWriter, r *http.Request, u users.User) {
	list, err := s.social.Duels(r.Context(), u.ID, s.Now(), 20)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"duels": list})
}

func (s *server) duelByCode(w http.ResponseWriter, r *http.Request, u users.User) {
	d, err := s.social.DuelByCode(r.Context(), r.PathValue("code"))
	if errors.Is(err, social.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeDuel(w, r, u, d.ID)
}

func (s *server) getDuel(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.writeDuel(w, r, u, id)
}

func (s *server) writeDuel(w http.ResponseWriter, r *http.Request, u users.User, id int64) {
	v, err := s.social.View(r.Context(), id, u.ID, s.Now())
	if errors.Is(err, social.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *server) playDuel(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	d, answered, accepted, err := s.social.Play(ctx, id, u.ID, s.Now())
	switch {
	case errors.Is(err, social.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
		return
	case errors.Is(err, social.ErrDuelBusy):
		writeError(w, http.StatusConflict, "busy")
		return
	case errors.Is(err, social.ErrDuelExpired):
		writeError(w, http.StatusGone, "expired")
		return
	case err != nil:
		s.fail(w, r, err)
		return
	}
	if accepted {
		if c, err := s.social.WhoByID(ctx, d.ChallengerID); err == nil {
			s.say(ctx, c.TelegramID, c.Lang, "duel.accepted", map[string]string{"nick": u.Nickname}, "duel-"+d.Code)
		}
	}
	v, err := s.social.View(ctx, id, u.ID, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"duel": v, "questions": social.Public(d.Questions), "answered": answered})
}

func (s *server) answerDuel(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		I      int `json:"i"`
		Choice int `json:"choice"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	res, err := s.social.Answer(r.Context(), id, u.ID, b.I, b.Choice, s.Now())
	switch {
	case errors.Is(err, social.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
		return
	case errors.Is(err, social.ErrDuelOver):
		writeError(w, http.StatusConflict, "over")
		return
	case errors.Is(err, social.ErrDuelOrder):
		writeError(w, http.StatusConflict, "order")
		return
	case err != nil:
		s.fail(w, r, err)
		return
	}
	if res.Finished {
		s.duelDone(r.Context(), id, u)
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *server) finishDuel(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.social.Finish(r.Context(), id, u.ID, s.Now()); err != nil {
		s.fail(w, r, err)
		return
	}
	s.duelDone(r.Context(), id, u)
	s.writeDuel(w, r, u, id)
}

// duelDone — ученик доиграл; если соперник доиграл раньше, ему приходит итог ботом.
func (s *server) duelDone(ctx context.Context, id int64, u users.User) {
	v, err := s.social.View(ctx, id, u.ID, s.Now())
	if err != nil || v.State != "done" || v.Opponent == nil {
		return
	}
	me, other := v.Challenger, *v.Opponent
	if other.ID == u.ID {
		me, other = other, me
	}
	o, err := s.social.WhoByID(ctx, other.ID)
	if err != nil {
		return
	}
	s.say(ctx, o.TelegramID, o.Lang, "duel.result", map[string]string{
		"nick": u.Nickname, "score": strconv.Itoa(other.Score) + ":" + strconv.Itoa(me.Score),
	}, "duel-"+v.Code)
}
