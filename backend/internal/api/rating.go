package api

import (
	"errors"
	"net/http"

	"ayzonexam/internal/exam"
	"ayzonexam/internal/rating"
	"ayzonexam/internal/users"
)

// Рейтинг, центры и «Турнир месяца» (этап 8).

func (s *server) getRating(w http.ResponseWriter, r *http.Request, u users.User) {
	scope := r.URL.Query().Get("scope")
	switch scope {
	case rating.City, rating.Center, rating.Friends, rating.Country:
	default:
		writeError(w, http.StatusBadRequest, "bad_scope")
		return
	}
	var center *int64
	if u.Member() {
		center = u.CenterID
	}
	b, err := s.rating.Students(r.Context(), scope, u.ID, center, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *server) getCenterRating(w http.ResponseWriter, r *http.Request, u users.User) {
	list, err := s.rating.Centers(r.Context(), s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var mine *int64
	if u.Member() {
		mine = u.CenterID
	}
	writeJSON(w, http.StatusOK, map[string]any{"centers": list, "mine": mine})
}

func (s *server) listCenters(w http.ResponseWriter, r *http.Request, _ users.User) {
	list, err := s.rating.List(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"centers": list})
}

func (s *server) getCenter(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := s.rating.Center(r.Context(), id, s.Now())
	if errors.Is(err, rating.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var status *string
	if u.CenterID != nil && *u.CenterID == id {
		status = u.CenterStatus
	}
	writeJSON(w, http.StatusOK, map[string]any{"center": c, "status": status})
}

func (s *server) joinCenter(w http.ResponseWriter, r *http.Request, u users.User) {
	var b struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	_, err := s.rating.JoinByCode(r.Context(), u.ID, b.Code)
	if errors.Is(err, rating.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	s.meAfter(w, r, u, err)
}

func (s *server) requestCenter(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	err := s.rating.Request(r.Context(), u.ID, id)
	if errors.Is(err, rating.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err == nil {
		s.notifyCenterStaff(r, id)
	}
	s.meAfter(w, r, u, err)
}

func (s *server) leaveCenter(w http.ResponseWriter, r *http.Request, u users.User) {
	s.meAfter(w, r, u, s.rating.Leave(r.Context(), u.ID))
}

// meAfter — ответ после изменения профиля: свежий профиль ученика.
func (s *server) meAfter(w http.ResponseWriter, r *http.Request, u users.User, err error) {
	if err == nil {
		u, err = s.Users.Get(r.Context(), u.ID)
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeMe(w, r, u)
}

// staffCenter — центр, в котором ученик — аккаунт центра или учитель; иначе 403.
func staffCenter(w http.ResponseWriter, u users.User) (int64, bool) {
	if !u.Member() || u.CenterRole == nil {
		writeError(w, http.StatusForbidden, "forbidden")
		return 0, false
	}
	return *u.CenterID, true
}

func (s *server) centerRequests(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, ok := staffCenter(w, u)
	if !ok {
		return
	}
	list, err := s.rating.Requests(r.Context(), cid)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if list == nil {
		list = []rating.Pending{}
	}
	// Код ссылки-приглашения: аккаунт центра рассылает её своим ученикам.
	var code string
	if err := s.DB.QueryRow(r.Context(), `SELECT invite_code FROM centers WHERE id = $1`, cid).Scan(&code); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": list, "invite_code": code})
}

func (s *server) decideRequest(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, ok := staffCenter(w, u)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		Accept bool `json:"accept"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	err := s.rating.Decide(r.Context(), cid, id, b.Accept)
	if errors.Is(err, rating.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) getTournament(w http.ResponseWriter, r *http.Request, u users.User) {
	now := s.Now()
	start := rating.TourStart(now)
	cur, err := s.rating.State(r.Context(), u.ID, start, now)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// Прошлый турнир — окно которого кончилось до текущего: итоги и «ваш результат».
	var last *rating.TourState
	{
		st, err := s.rating.State(r.Context(), u.ID, rating.PrevTourStart(start), now)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if st.ID != 0 {
			last = &st
		}
	}
	winners, err := s.rating.Winners(r.Context(), 6)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if winners == nil {
		winners = []rating.Winner{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": cur, "last": last, "winners": winners})
}

func (s *server) joinTournament(w http.ResponseWriter, r *http.Request, u users.User) {
	err := s.rating.Join(r.Context(), u.ID, s.Now())
	if errors.Is(err, rating.ErrTourClosed) {
		writeError(w, http.StatusConflict, "closed")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) startTournamentPart(w http.ResponseWriter, r *http.Request, u users.User) {
	var b struct {
		Kind string `json:"kind"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	if b.Kind != exam.Reading && b.Kind != exam.Listening {
		writeError(w, http.StatusBadRequest, "bad_kind")
		return
	}
	tourID, testID, err := s.rating.PartTest(r.Context(), u.ID, b.Kind, s.Now(), func() (int64, error) {
		t, err := s.exam.Pick(r.Context(), b.Kind, 0)
		return t.ID, err
	})
	switch {
	case errors.Is(err, rating.ErrNotLive):
		writeError(w, http.StatusConflict, "not_live")
	case errors.Is(err, rating.ErrNotJoined):
		writeError(w, http.StatusForbidden, "not_joined")
	case errors.Is(err, rating.ErrPartDone):
		writeError(w, http.StatusConflict, "part_done")
	case errors.Is(err, exam.ErrNotFound):
		writeError(w, http.StatusNotFound, "no_tests")
	case err != nil:
		s.fail(w, r, err)
	default:
		s.start(w, r, u, testID, nil, &tourID, s.Now())
	}
}
