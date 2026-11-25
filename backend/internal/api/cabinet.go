package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"ayzonexam/internal/rating"
	"ayzonexam/internal/users"
)

// Кабинет центра (DECISIONS 2026-11-25): сайт /cabinet и тот же кабинет в Mini App. Пускает аккаунт
// центра (center_role admin) и учителей центра (teacher) — только в свой центр. Учитель видит учеников
// и заявки; тариф, оплаты, текст о центре, новая ссылка и роли — только аккаунт центра. Ученики в
// ответах — ник, аватар и результаты. Действия пишутся в журнал владельца (s.audit).

// cabinetCenter — центр сотрудника и его роль; не сотрудник — 403.
func cabinetCenter(w http.ResponseWriter, u users.User) (int64, bool, bool) {
	cid, ok := staffCenter(w, u)
	if !ok {
		return 0, false, false
	}
	return cid, *u.CenterRole == "admin", true
}

// cabinetAdmin — только аккаунт центра; учитель — 403.
func cabinetAdmin(w http.ResponseWriter, u users.User) (int64, bool) {
	cid, admin, ok := cabinetCenter(w, u)
	if ok && !admin {
		writeError(w, http.StatusForbidden, "forbidden")
		return 0, false
	}
	return cid, ok
}

type cabinetPayment struct {
	Amount    int       `json:"amount"`
	Seats     int       `json:"seats"`
	PaidUntil time.Time `json:"paid_until"`
	At        time.Time `json:"at"`
}

func (s *server) getCabinet(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, admin, ok := cabinetCenter(w, u)
	if !ok {
		return
	}
	ctx := r.Context()
	var c struct {
		ID        int64      `json:"id"`
		Name      string     `json:"name"`
		City      string     `json:"city"`
		About     string     `json:"about"`
		Code      string     `json:"invite_code"`
		Seats     int        `json:"seats"`
		PaidUntil *time.Time `json:"paid_until"`
		Active    bool       `json:"active"`
		Students  int        `json:"students"`
		Pending   int        `json:"pending"`
		Week      int        `json:"active_week"`
		Avg       *float64   `json:"avg"`
		Scored    int        `json:"scored"`
		Place     *int       `json:"place"`
		Centers   int        `json:"centers"`
		Delta     int        `json:"delta"`
	}
	err := s.DB.QueryRow(ctx, `SELECT c.id, c.name, c.city, c.about, c.invite_code, c.seats, c.paid_until, c.active,
			(SELECT count(*) FROM users m WHERE m.center_id = c.id AND m.center_status = 'member'),
			(SELECT count(*) FROM users m WHERE m.center_id = c.id AND m.center_status = 'pending'),
			(SELECT count(*) FROM users m WHERE m.center_id = c.id AND m.center_status = 'member' AND m.seen_at > $2::timestamptz - interval '7 days')
		FROM centers c WHERE c.id = $1`, cid, s.Now()).
		Scan(&c.ID, &c.Name, &c.City, &c.About, &c.Code, &c.Seats, &c.PaidUntil, &c.Active, &c.Students, &c.Pending, &c.Week)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	all, err := s.rating.Centers(ctx, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	c.Centers = len(all)
	for _, row := range all {
		if row.ID == cid {
			avg, place := row.Avg, row.Place
			c.Avg, c.Place, c.Scored, c.Delta = &avg, &place, row.Scored, row.Delta
		}
	}
	out := map[string]any{"center": c, "role": *u.CenterRole}
	if admin {
		rows, err := s.DB.Query(ctx, `SELECT amount, seats, paid_until, created_at FROM center_payments
			WHERE center_id = $1 ORDER BY created_at DESC LIMIT 24`, cid)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		pays := []cabinetPayment{}
		for rows.Next() {
			var p cabinetPayment
			if err := rows.Scan(&p.Amount, &p.Seats, &p.PaidUntil, &p.At); err != nil {
				rows.Close()
				s.fail(w, r, err)
				return
			}
			pays = append(pays, p)
		}
		rows.Close()
		out["payments"] = pays
	} else {
		// Учителю — без денег: тариф и оплаты видит аккаунт центра.
		c.Seats, c.PaidUntil = 0, nil
		out["center"] = c
	}
	writeJSON(w, http.StatusOK, out)
}

// patchCabinet — аккаунт центра меняет текст «О центре» (название и город — у владельца).
func (s *server) patchCabinet(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, ok := cabinetAdmin(w, u)
	if !ok {
		return
	}
	var b struct {
		About string `json:"about"`
	}
	if !readJSON(w, r, &b, 8<<10) {
		return
	}
	b.About = strings.TrimSpace(b.About)
	if utf8.RuneCountInString(b.About) > 2000 {
		writeError(w, http.StatusBadRequest, "too_long")
		return
	}
	if _, err := s.DB.Exec(r.Context(), `UPDATE centers SET about = $2 WHERE id = $1`, cid, b.About); err != nil {
		s.fail(w, r, err)
		return
	}
	s.audit(r.Context(), u.TelegramID, "cabinet.about", centerTarget(cid), nil)
	s.getCabinet(w, r, u)
}

// cabinetCode — новая ссылка-приглашение: старая перестаёт работать (если её переслали не туда).
func (s *server) cabinetCode(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, ok := cabinetAdmin(w, u)
	if !ok {
		return
	}
	if _, err := s.DB.Exec(r.Context(), `UPDATE centers SET invite_code = $2 WHERE id = $1`, cid, rating.NewCode()); err != nil {
		s.fail(w, r, err)
		return
	}
	s.audit(r.Context(), u.TelegramID, "cabinet.code", centerTarget(cid), nil)
	s.getCabinet(w, r, u)
}

func (s *server) cabinetStudents(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, _, ok := cabinetCenter(w, u)
	if !ok {
		return
	}
	list, err := s.rating.Members(r.Context(), cid, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"students": list, "me": u.ID})
}

func (s *server) cabinetRemove(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, ok := cabinetAdmin(w, u)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if !s.cabinetDone(w, r, s.rating.Remove(r.Context(), cid, u.ID, id)) {
		return
	}
	s.audit(r.Context(), u.TelegramID, "cabinet.remove", "user:"+strconv.FormatInt(id, 10), map[string]any{"center_id": cid})
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) cabinetRole(w http.ResponseWriter, r *http.Request, u users.User) {
	cid, ok := cabinetAdmin(w, u)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		Teacher bool `json:"teacher"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	if !s.cabinetDone(w, r, s.rating.SetTeacher(r.Context(), cid, u.ID, id, b.Teacher)) {
		return
	}
	s.audit(r.Context(), u.TelegramID, "cabinet.teacher", "user:"+strconv.FormatInt(id, 10), map[string]any{"center_id": cid, "teacher": b.Teacher})
	w.WriteHeader(http.StatusNoContent)
}

// cabinetDone — ответ на ошибку действия над учеником; true — ошибки нет.
func (s *server) cabinetDone(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case errors.Is(err, rating.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
	case errors.Is(err, rating.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case err != nil:
		s.fail(w, r, err)
	default:
		return true
	}
	return false
}

func centerTarget(id int64) string { return "center:" + strconv.FormatInt(id, 10) }
