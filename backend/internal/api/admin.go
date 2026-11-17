package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"ayzonexam/internal/billing"
	"ayzonexam/internal/exam"
	"ayzonexam/internal/news"
	"ayzonexam/internal/rating"
	"ayzonexam/internal/support"
	"ayzonexam/internal/users"
)

// Админка владельца (этап 17). Вход — только Telegram id из ADMIN_IDS; каждое изменение — в журнал.
// Здесь админ видит и личное (имя, @username, Telegram id): это его ученики и его поддержка.

// admin пускает только владельца и помощников; остальным — 404, как будто маршрута нет.
func (s *server) admin(next func(http.ResponseWriter, *http.Request, users.User)) http.Handler {
	return s.auth(func(w http.ResponseWriter, r *http.Request, u users.User) {
		if !s.isAdmin(u) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		next(w, r, u)
	})
}

// audit — запись в журнал действий: кто, что и когда изменил.
func (s *server) audit(ctx context.Context, adminTG int64, action, target string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	js, _ := json.Marshal(details)
	if _, err := s.DB.Exec(context.WithoutCancel(ctx), `INSERT INTO admin_log (admin_tg, action, target, details) VALUES ($1, $2, $3, $4)`,
		adminTG, action, target, js); err != nil {
		s.Log.Error("журнал админки не записан", "action", action, "err", err)
	}
}

// rowsJSON — результат запроса списком объектов: колонка → значение. Для простых таблиц админки.
func (s *server) rowsJSON(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	rows, err := s.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, pgx.RowToMap)
	if list == nil {
		list = []map[string]any{}
	}
	return list, err
}

func (s *server) listJSON(w http.ResponseWriter, r *http.Request, key, sql string, args ...any) {
	list, err := s.rowsJSON(r.Context(), sql, args...)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{key: list})
}

// Стоимость AI-проверок для обзора (SPEC, «Расходы»), в долларах.
const (
	costWriting  = 0.01
	costSpeaking = 0.08
)

func (s *server) adminOverview(w http.ResponseWriter, r *http.Request, _ users.User) {
	now := s.Now()
	day := billing.DayStart(now)
	month := billing.MonthStart(now)
	var o struct {
		Users        int     `json:"users"`
		NewDay       int     `json:"new_day"`
		NewWeek      int     `json:"new_week"`
		ActiveDay    int     `json:"active_day"`
		ActiveWeek   int     `json:"active_week"`
		Paying       int     `json:"paying"`
		MoneyMonth   int64   `json:"money_month"`
		Pending      int     `json:"pending_payments"`
		WritingDay   int     `json:"writing_day"`
		WritingMonth int     `json:"writing_month"`
		SpeakDay     int     `json:"speaking_day"`
		SpeakMonth   int     `json:"speaking_month"`
		CostMonth    float64 `json:"ai_cost_month"`
		OpenTickets  int     `json:"open_tickets"`
		Centers      int     `json:"centers"`
	}
	err := s.DB.QueryRow(r.Context(), `SELECT
		(SELECT count(*) FROM users),
		(SELECT count(*) FROM users WHERE created_at >= $1),
		(SELECT count(*) FROM users WHERE created_at >= $2::timestamptz - interval '7 days'),
		(SELECT count(*) FROM users WHERE seen_at >= $1),
		(SELECT count(*) FROM users WHERE seen_at >= $2::timestamptz - interval '7 days'),
		(SELECT count(DISTINCT user_id) FROM payments WHERE status = 'confirmed' AND plan <> 'mock'
			AND user_id IN (SELECT id FROM users WHERE premium_until > $2)),
		(SELECT coalesce(sum(amount), 0) FROM payments WHERE status = 'confirmed' AND decided_at >= $3)
			+ (SELECT coalesce(sum(amount), 0) FROM center_payments WHERE created_at >= $3),
		(SELECT count(*) FROM payments WHERE status = 'pending'),
		(SELECT count(*) FROM writing_checks WHERE created_at >= $1 AND status = 'done'),
		(SELECT count(*) FROM writing_checks WHERE created_at >= $3 AND status = 'done'),
		(SELECT count(*) FROM speaking_checks WHERE created_at >= $1 AND status = 'done'),
		(SELECT count(*) FROM speaking_checks WHERE created_at >= $3 AND status = 'done'),
		(SELECT count(*) FROM tickets WHERE status = 'open'),
		(SELECT count(*) FROM centers WHERE active)`, day, now, month).
		Scan(&o.Users, &o.NewDay, &o.NewWeek, &o.ActiveDay, &o.ActiveWeek, &o.Paying, &o.MoneyMonth, &o.Pending,
			&o.WritingDay, &o.WritingMonth, &o.SpeakDay, &o.SpeakMonth, &o.OpenTickets, &o.Centers)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	o.CostMonth = float64(o.WritingMonth)*costWriting + float64(o.SpeakMonth)*costSpeaking
	writeJSON(w, http.StatusOK, o)
}

// adminUsers — поиск по нику, имени, @username или Telegram id.
func (s *server) adminUsers(w http.ResponseWriter, r *http.Request, _ users.User) {
	q := strings.TrimSpace(strings.TrimPrefix(r.URL.Query().Get("q"), "@"))
	tg, _ := strconv.ParseInt(q, 10, 64)
	s.listJSON(w, r, "users", `SELECT u.id, u.telegram_id, u.nickname, u.first_name, u.last_name, u.username, u.lang, u.created_at,
			u.seen_at, u.premium_until, u.blocked_at, u.center_role, u.verified_teacher, u.center_status, c.name AS center
		FROM users u LEFT JOIN centers c ON c.id = u.center_id
		WHERE $1 = '' OR u.telegram_id = $2 OR u.nickname ILIKE '%' || $1 || '%' OR u.username ILIKE '%' || $1 || '%'
			OR (u.first_name || ' ' || u.last_name) ILIKE '%' || $1 || '%'
		ORDER BY u.created_at DESC LIMIT 50`, q, tg)
}

func (s *server) adminUser(w http.ResponseWriter, r *http.Request, _ users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	u, err := s.Users.Get(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	cur, _, err := s.rating.Current(ctx, id, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := map[string]any{"user": map[string]any{
		"id": u.ID, "telegram_id": u.TelegramID, "nickname": u.Nickname, "first_name": u.FirstName, "last_name": u.LastName,
		"username": u.Username, "lang": u.Lang, "created_at": u.CreatedAt, "seen_at": u.SeenAt, "goal_band": u.GoalBand,
		"premium_until": u.PremiumUntil, "pay_trust": u.PayTrust, "mock_credits": u.MockCredits, "bonus_checks": u.BonusChecks,
		"blocked_at": u.BlockedAt, "center_id": u.CenterID, "center_status": u.CenterStatus, "center_role": u.CenterRole,
		"verified_teacher": u.VerifiedTeacher, "city": u.City, "current": cur,
	}, "plan": billing.Of(u, s.Now())}
	for key, sql := range map[string]string{
		"attempts": `SELECT a.id, a.kind, t.title, a.band, a.ranked, a.leaves, a.finished_at FROM attempts a JOIN tests t ON t.id = a.test_id
			WHERE a.user_id = $1 AND a.finished_at IS NOT NULL ORDER BY a.finished_at DESC LIMIT 20`,
		"writing":  `SELECT id, task, status, band, words, created_at FROM writing_checks WHERE user_id = $1 ORDER BY id DESC LIMIT 20`,
		"speaking": `SELECT id, source, status, band, created_at FROM speaking_checks WHERE user_id = $1 ORDER BY id DESC LIMIT 20`,
		"mocks":    `SELECT id, status, overall, paid_with, started_at FROM mocks WHERE user_id = $1 ORDER BY id DESC LIMIT 10`,
		"payments": `SELECT id, plan, amount, status, reason, created_at FROM payments WHERE user_id = $1 ORDER BY id DESC LIMIT 10`,
		"tickets":  `SELECT id, status, updated_at FROM tickets WHERE user_id = $1 ORDER BY id DESC LIMIT 10`,
	} {
		list, err := s.rowsJSON(ctx, sql, id)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		out[key] = list
	}
	writeJSON(w, http.StatusOK, out)
}

// adminUserPatch — Premium (дни; отрицательное — забрать), блокировка, роль в центре, «Проверенный учитель».
func (s *server) adminUserPatch(w http.ResponseWriter, r *http.Request, a users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		PremiumDays *int    `json:"premium_days"`
		Blocked     *bool   `json:"blocked"`
		CenterID    *int64  `json:"center_id"` // 0 — убрать из центра
		Role        *string `json:"role"`      // admin, teacher, '' — снять
		Verified    *bool   `json:"verified"`  // «Проверенный учитель»
		MockCredits *int    `json:"mock_credits"`
	}
	if !readJSON(w, r, &b, 2<<10) {
		return
	}
	ctx := r.Context()
	target := "user:" + strconv.FormatInt(id, 10)
	if b.PremiumDays != nil {
		if err := s.billing.SetPremium(ctx, id, *b.PremiumDays, s.Now()); err != nil {
			s.fail(w, r, err)
			return
		}
		s.audit(ctx, a.TelegramID, "user.premium", target, map[string]any{"days": *b.PremiumDays})
	}
	if b.Blocked != nil {
		if _, err := s.DB.Exec(ctx, `UPDATE users SET blocked_at = CASE WHEN $2 THEN now() ELSE NULL END WHERE id = $1`, id, *b.Blocked); err != nil {
			s.fail(w, r, err)
			return
		}
		s.audit(ctx, a.TelegramID, "user.block", target, map[string]any{"blocked": *b.Blocked})
	}
	if b.CenterID != nil {
		var err error
		if *b.CenterID == 0 {
			err = s.rating.Leave(ctx, id)
		} else {
			_, err = s.DB.Exec(ctx, `UPDATE users SET center_id = $2, center_status = 'member' WHERE id = $1`, id, *b.CenterID)
		}
		if err != nil {
			s.fail(w, r, err)
			return
		}
		s.audit(ctx, a.TelegramID, "user.center", target, map[string]any{"center_id": *b.CenterID})
	}
	if b.Role != nil {
		var role any
		if *b.Role == "admin" || *b.Role == "teacher" {
			role = *b.Role
		} else if *b.Role != "" {
			writeError(w, http.StatusBadRequest, "bad_role")
			return
		}
		if _, err := s.DB.Exec(ctx, `UPDATE users SET center_role = $2 WHERE id = $1`, id, role); err != nil {
			s.fail(w, r, err)
			return
		}
		s.audit(ctx, a.TelegramID, "user.role", target, map[string]any{"role": *b.Role})
	}
	if b.Verified != nil {
		if _, err := s.DB.Exec(ctx, `UPDATE users SET verified_teacher = $2 WHERE id = $1`, id, *b.Verified); err != nil {
			s.fail(w, r, err)
			return
		}
		s.audit(ctx, a.TelegramID, "user.verified", target, map[string]any{"verified": *b.Verified})
	}
	if b.MockCredits != nil {
		if _, err := s.DB.Exec(ctx, `UPDATE users SET mock_credits = greatest($2, 0) WHERE id = $1`, id, *b.MockCredits); err != nil {
			s.fail(w, r, err)
			return
		}
		s.audit(ctx, a.TelegramID, "user.mocks", target, map[string]any{"mock_credits": *b.MockCredits})
	}
	s.adminUser(w, r, a)
}

func (s *server) adminCenters(w http.ResponseWriter, r *http.Request, _ users.User) {
	s.listJSON(w, r, "centers", `SELECT c.id, c.name, c.city, c.about, c.invite_code, c.seats, c.paid_until, c.active, c.created_at,
			(SELECT count(*) FROM users m WHERE m.center_id = c.id AND m.center_status = 'member') AS students,
			(SELECT count(*) FROM users m WHERE m.center_id = c.id AND m.center_status = 'pending') AS pending,
			(SELECT coalesce(json_agg(json_build_object('id', m.id, 'nickname', m.nickname, 'name', m.first_name, 'role', m.center_role)), '[]')
				FROM users m WHERE m.center_id = c.id AND m.center_role IS NOT NULL) AS staff
		FROM centers c ORDER BY c.active DESC, c.name`)
}

type centerBody struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	About   string `json:"about"`
	Active  *bool  `json:"active"`
	NewCode bool   `json:"new_code"`
}

func (s *server) adminCenterSave(w http.ResponseWriter, r *http.Request, a users.User) {
	var b centerBody
	if !readJSON(w, r, &b, 8<<10) {
		return
	}
	b.Name = strings.TrimSpace(b.Name)
	if b.City != "tashkent" && b.City != "other" {
		b.City = "tashkent"
	}
	if b.Name == "" || utf8.RuneCountInString(b.Name) > 80 || utf8.RuneCountInString(b.About) > 2000 {
		writeError(w, http.StatusBadRequest, "bad_center")
		return
	}
	ctx := r.Context()
	var id int64
	var err error
	if r.PathValue("id") == "" {
		err = s.DB.QueryRow(ctx, `INSERT INTO centers (name, city, about, invite_code) VALUES ($1, $2, $3, $4) RETURNING id`,
			b.Name, b.City, b.About, rating.NewCode()).Scan(&id)
		s.audit(ctx, a.TelegramID, "center.create", "", map[string]any{"name": b.Name})
	} else {
		var ok bool
		if id, ok = pathID(w, r); !ok {
			return
		}
		code := ""
		if b.NewCode {
			code = rating.NewCode()
		}
		_, err = s.DB.Exec(ctx, `UPDATE centers SET name = $2, city = $3, about = $4, active = coalesce($5, active),
			invite_code = CASE WHEN $6 = '' THEN invite_code ELSE $6 END WHERE id = $1`, id, b.Name, b.City, b.About, b.Active, code)
		s.audit(ctx, a.TelegramID, "center.update", "center:"+strconv.FormatInt(id, 10), map[string]any{"name": b.Name, "active": b.Active, "new_code": b.NewCode})
	}
	if isUnique(err) {
		writeError(w, http.StatusConflict, "name_taken")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// adminCenterPayment — владелец принял оплату центра (с чеком, если есть): тариф продлевается.
func (s *server) adminCenterPayment(w http.ResponseWriter, r *http.Request, a users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if code := parseForm(w, r, maxImageBytes+64<<10); code != "" {
		writeError(w, http.StatusBadRequest, code)
		return
	}
	defer r.MultipartForm.RemoveAll()
	f := r.MultipartForm.Value
	amount, _ := strconv.Atoi(first(f["amount"]))
	seats, _ := strconv.Atoi(first(f["seats"]))
	until, err := time.Parse(time.DateOnly, first(f["paid_until"]))
	if amount <= 0 || seats <= 0 || err != nil {
		writeError(w, http.StatusBadRequest, "bad_payment")
		return
	}
	ctx := r.Context()
	var receipt *int64
	if fhs := r.MultipartForm.File["receipt"]; len(fhs) > 0 {
		img, code := readImage(fhs[0])
		if code != "" {
			writeError(w, http.StatusBadRequest, code)
			return
		}
		var mid int64
		if err := s.DB.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, true) RETURNING id`, img.MIME, img.Data).Scan(&mid); err != nil {
			s.fail(w, r, err)
			return
		}
		receipt = &mid
	}
	note := first(f["note"])
	if err := s.billing.CenterPayment(ctx, id, amount, seats, until, receipt, note, a.TelegramID); errors.Is(err, billing.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	} else if err != nil {
		s.fail(w, r, err)
		return
	}
	s.audit(ctx, a.TelegramID, "center.payment", "center:"+strconv.FormatInt(id, 10),
		map[string]any{"amount": amount, "seats": seats, "paid_until": first(f["paid_until"])})
	s.listJSON(w, r, "payments", `SELECT id, amount, seats, paid_until, receipt, note, created_at FROM center_payments
		WHERE center_id = $1 ORDER BY id DESC`, id)
}

func (s *server) adminCenterPayments(w http.ResponseWriter, r *http.Request, _ users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	s.listJSON(w, r, "payments", `SELECT id, amount, seats, paid_until, receipt, note, created_at FROM center_payments
		WHERE center_id = $1 ORDER BY id DESC`, id)
}

func (s *server) adminTeachers(w http.ResponseWriter, r *http.Request, _ users.User) {
	s.listJSON(w, r, "teachers", `SELECT u.id, u.telegram_id, u.nickname, u.first_name, u.username, u.center_role, u.verified_teacher,
			c.name AS center, c.id AS center_id
		FROM users u LEFT JOIN centers c ON c.id = u.center_id
		WHERE u.center_role IS NOT NULL OR u.verified_teacher ORDER BY c.name NULLS LAST, u.nickname`)
}

func (s *server) adminPayments(w http.ResponseWriter, r *http.Request, _ users.User) {
	list, err := s.billing.List(r.Context(), r.URL.Query().Get("status"), 100)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": list})
}

func (s *server) adminPaymentDecide(w http.ResponseWriter, r *http.Request, a users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		Confirm bool   `json:"confirm"`
		Reason  string `json:"reason"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	var err error
	if b.Confirm {
		err = s.confirmPayment(r.Context(), id, a.TelegramID)
	} else {
		err = s.rejectPayment(r.Context(), id, a.TelegramID, b.Reason)
	}
	if errors.Is(err, billing.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	p, err := s.billing.Get(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// adminMedia — закрытый файл (скрин оплаты, фото обращения, чек): только админке.
func (s *server) adminMedia(w http.ResponseWriter, r *http.Request, _ users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var mime string
	var data []byte
	if err := s.DB.QueryRow(r.Context(), `SELECT mime, data FROM media WHERE id = $1`, id).Scan(&mime, &data); err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(data))
}

func (s *server) adminTickets(w http.ResponseWriter, r *http.Request, _ users.User) {
	list, err := s.support.List(r.Context(), 0, r.URL.Query().Get("status"), 100)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tickets": list})
}

func (s *server) adminTicket(w http.ResponseWriter, r *http.Request, _ users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := s.support.Get(r.Context(), id, 0)
	if errors.Is(err, support.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *server) adminTicketAct(w http.ResponseWriter, r *http.Request, a users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		Reply  string `json:"reply"`
		Status string `json:"status"`
	}
	if !readJSON(w, r, &b, 8<<10) {
		return
	}
	ctx := r.Context()
	if strings.TrimSpace(b.Reply) != "" {
		if err := s.replyTicket(ctx, id, a.TelegramID, b.Reply); errors.Is(err, support.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		} else if err != nil {
			s.Log.Warn("ответ поддержки не доставлен ботом", "ticket", id, "err", err)
		}
	}
	if b.Status == "open" || b.Status == "answered" || b.Status == "closed" {
		if err := s.support.SetStatus(ctx, id, b.Status); err != nil {
			s.fail(w, r, err)
			return
		}
		s.audit(ctx, a.TelegramID, "ticket.status", "ticket:"+strconv.FormatInt(id, 10), map[string]any{"status": b.Status})
	}
	s.adminTicket(w, r, a)
}

func (s *server) adminNews(w http.ResponseWriter, r *http.Request, _ users.User) {
	list, err := s.news.List(r.Context(), 100, true)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if list == nil {
		list = []news.Post{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"posts": list})
}

func (s *server) adminNewsSave(w http.ResponseWriter, r *http.Request, a users.User) {
	var p news.Post
	if !readJSON(w, r, &p, 32<<10) {
		return
	}
	ctx := r.Context()
	var err error
	if r.PathValue("id") == "" {
		p, err = s.news.Create(ctx, p, a.TelegramID)
		s.audit(ctx, a.TelegramID, "news.create", "", map[string]any{"title": p.Title})
	} else {
		var ok bool
		if p.ID, ok = pathID(w, r); !ok {
			return
		}
		p, err = s.news.Update(ctx, p)
		s.audit(ctx, a.TelegramID, "news.update", "post:"+strconv.FormatInt(p.ID, 10), map[string]any{"hidden": p.Hidden})
	}
	switch {
	case errors.Is(err, news.ErrBad):
		writeError(w, http.StatusBadRequest, "bad_post")
	case errors.Is(err, news.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
	case err != nil:
		s.fail(w, r, err)
	default:
		writeJSON(w, http.StatusOK, p)
	}
}

// adminTournaments — прошедшие турниры с итогами: тройку подтверждает владелец, назначает чемпиона.
func (s *server) adminTournaments(w http.ResponseWriter, r *http.Request, _ users.User) {
	ctx := r.Context()
	tours, err := s.rating.PastTours(ctx, s.Now(), 12)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := []map[string]any{}
	for _, t := range tours {
		res, err := s.rowsJSON(ctx, `WITH parts AS (
				SELECT a.user_id, avg(a.band) AS avg, count(*) AS n, max(a.finished_at) AS done
				FROM attempts a WHERE a.tournament_id = $1 AND a.ranked GROUP BY a.user_id)
			SELECT u.id, u.nickname, u.first_name, u.telegram_id, (floor(p.avg * 2 + 0.5) / 2)::float8 AS band, p.n AS parts,
				e.disqualified, c.name AS center
			FROM parts p JOIN users u ON u.id = p.user_id
			JOIN tournament_entries e ON e.tournament_id = $1 AND e.user_id = u.id
			LEFT JOIN centers c ON c.id = u.center_id AND u.center_status = 'member'
			ORDER BY e.disqualified, p.n DESC, band DESC, p.done LIMIT 10`, t.ID)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		var champ *int64
		var players int
		if err := s.DB.QueryRow(ctx, `SELECT champion_id, (SELECT count(*) FROM tournament_entries WHERE tournament_id = $1)
			FROM tournaments WHERE id = $1`, t.ID).Scan(&champ, &players); err != nil {
			s.fail(w, r, err)
			return
		}
		out = append(out, map[string]any{"id": t.ID, "starts_at": t.StartsAt, "ends_at": t.EndsAt, "players": players,
			"champion_id": champ, "top": res})
	}
	writeJSON(w, http.StatusOK, map[string]any{"tournaments": out})
}

func (s *server) adminTournamentAct(w http.ResponseWriter, r *http.Request, a users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var b struct {
		Champion   *int64 `json:"champion"` // 0 — снять
		Disqualify *int64 `json:"disqualify"`
		Requalify  *int64 `json:"requalify"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	ctx := r.Context()
	target := "tournament:" + strconv.FormatInt(id, 10)
	var err error
	switch {
	case b.Champion != nil:
		err = s.rating.SetChampion(ctx, id, *b.Champion)
		s.audit(ctx, a.TelegramID, "tournament.champion", target, map[string]any{"user_id": *b.Champion})
	case b.Disqualify != nil:
		err = s.rating.Disqualify(ctx, id, *b.Disqualify, true)
		s.audit(ctx, a.TelegramID, "tournament.disqualify", target, map[string]any{"user_id": *b.Disqualify})
	case b.Requalify != nil:
		err = s.rating.Disqualify(ctx, id, *b.Requalify, false)
		s.audit(ctx, a.TelegramID, "tournament.requalify", target, map[string]any{"user_id": *b.Requalify})
	}
	if errors.Is(err, rating.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.adminTournaments(w, r, a)
}

// Задания: Reading, Listening, тест уровня; темы Writing; вопросы Speaking.

func (s *server) adminTests(w http.ResponseWriter, r *http.Request, _ users.User) {
	s.listJSON(w, r, "tests", `SELECT t.id, t.kind, t.title, t.full_size, t.minutes, t.passage, t.transcript, t.audio_id, t.questions, t.active,
			(SELECT count(*) FROM attempts a WHERE a.test_id = t.id AND a.finished_at IS NOT NULL) AS attempts
		FROM tests t ORDER BY t.kind, t.id`)
}

type testBody struct {
	Kind       string          `json:"kind"`
	Title      string          `json:"title"`
	FullSize   bool            `json:"full_size"`
	Minutes    int             `json:"minutes"`
	Passage    string          `json:"passage"`
	Transcript string          `json:"transcript"`
	Questions  []exam.Question `json:"questions"`
	Active     bool            `json:"active"`
}

func validTest(b *testBody) bool {
	b.Title = strings.TrimSpace(b.Title)
	if !validKind(b.Kind) || b.Title == "" || b.Minutes < 1 || b.Minutes > 90 || len(b.Questions) == 0 || len(b.Questions) > 60 {
		return false
	}
	if utf8.RuneCountInString(b.Passage) > 20000 || utf8.RuneCountInString(b.Transcript) > 20000 {
		return false
	}
	for _, q := range b.Questions {
		if strings.TrimSpace(q.Q) == "" || len(q.Options) < 2 || len(q.Options) > 6 || q.Answer < 0 || q.Answer >= len(q.Options) {
			return false
		}
	}
	return true
}

func (s *server) adminTestSave(w http.ResponseWriter, r *http.Request, a users.User) {
	var b testBody
	if !readJSON(w, r, &b, 256<<10) {
		return
	}
	if !validTest(&b) {
		writeError(w, http.StatusBadRequest, "bad_test")
		return
	}
	qs, _ := json.Marshal(b.Questions)
	ctx := r.Context()
	var id int64
	var err error
	if r.PathValue("id") == "" {
		err = s.DB.QueryRow(ctx, `INSERT INTO tests (kind, title, full_size, minutes, passage, transcript, questions, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`, b.Kind, b.Title, b.FullSize, b.Minutes, b.Passage, b.Transcript, qs, b.Active).Scan(&id)
		s.audit(ctx, a.TelegramID, "test.create", "", map[string]any{"title": b.Title})
	} else {
		var ok bool
		if id, ok = pathID(w, r); !ok {
			return
		}
		_, err = s.DB.Exec(ctx, `UPDATE tests SET kind = $2, title = $3, full_size = $4, minutes = $5, passage = $6, transcript = $7,
			questions = $8, active = $9 WHERE id = $1`, id, b.Kind, b.Title, b.FullSize, b.Minutes, b.Passage, b.Transcript, qs, b.Active)
		s.audit(ctx, a.TelegramID, "test.update", "test:"+strconv.FormatInt(id, 10), map[string]any{"active": b.Active})
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// adminTestAudio — аудио Listening: запись владельца или озвучка, пережатая в браузере админки.
func (s *server) adminTestAudio(w http.ResponseWriter, r *http.Request, a users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	const maxAudio = 15 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxAudio)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "audio_too_large")
		return
	}
	mime := audioMIME(data)
	if mime == "" {
		writeError(w, http.StatusBadRequest, "bad_audio")
		return
	}
	ctx := r.Context()
	var mid int64
	if err := s.DB.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, false) RETURNING id`, mime, data).Scan(&mid); err != nil {
		s.fail(w, r, err)
		return
	}
	if _, err := s.DB.Exec(ctx, `UPDATE tests SET audio_id = $2 WHERE id = $1`, id, mid); err != nil {
		s.fail(w, r, err)
		return
	}
	s.audit(ctx, a.TelegramID, "test.audio", "test:"+strconv.FormatInt(id, 10), map[string]any{"bytes": len(data), "mime": mime})
	writeJSON(w, http.StatusOK, map[string]any{"audio_id": mid})
}

// adminTestSpeak — озвучка сценария Listening (Gemini, часть Б). Отдаёт WAV: админка пережимает его в
// браузере и сохраняет через adminTestAudio.
func (s *server) adminTestSpeak(w http.ResponseWriter, r *http.Request, a users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	var script string
	if err := s.DB.QueryRow(ctx, `SELECT transcript FROM tests WHERE id = $1`, id).Scan(&script); err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if strings.TrimSpace(script) == "" {
		writeError(w, http.StatusBadRequest, "no_transcript")
		return
	}
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	wav, err := s.AI.Speak(callCtx, script)
	if err != nil {
		s.Log.Warn("озвучка не удалась", "test", id, "err", err)
		writeError(w, http.StatusBadGateway, "ai_unavailable")
		return
	}
	s.audit(ctx, a.TelegramID, "test.speak", "test:"+strconv.FormatInt(id, 10), map[string]any{"bytes": len(wav)})
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(wav)
}

func (s *server) adminPrompts(w http.ResponseWriter, r *http.Request, _ users.User) {
	s.listJSON(w, r, "prompts", `SELECT id, task, text, image_id, active FROM writing_prompts ORDER BY task, id`)
}

// adminPromptSave — тема Writing; Task 1 — с картинкой графика (для mock).
func (s *server) adminPromptSave(w http.ResponseWriter, r *http.Request, a users.User) {
	if code := parseForm(w, r, maxImageBytes+64<<10); code != "" {
		writeError(w, http.StatusBadRequest, code)
		return
	}
	defer r.MultipartForm.RemoveAll()
	f := r.MultipartForm.Value
	task, text, active := first(f["task"]), strings.TrimSpace(first(f["text"])), first(f["active"]) != "false"
	if (task != "task1" && task != "task2") || utf8.RuneCountInString(text) > maxPromptRunes || (task == "task2" && utf8.RuneCountInString(text) < minOwnPromptRunes) {
		writeError(w, http.StatusBadRequest, "bad_prompt")
		return
	}
	ctx := r.Context()
	var imageID *int64
	if fhs := r.MultipartForm.File["image"]; len(fhs) > 0 {
		img, code := readImage(fhs[0])
		if code != "" {
			writeError(w, http.StatusBadRequest, code)
			return
		}
		var mid int64
		if err := s.DB.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, false) RETURNING id`, img.MIME, img.Data).Scan(&mid); err != nil {
			s.fail(w, r, err)
			return
		}
		imageID = &mid
	}
	var id int64
	var err error
	if r.PathValue("id") == "" {
		err = s.DB.QueryRow(ctx, `INSERT INTO writing_prompts (task, text, image_id, active) VALUES ($1, $2, $3, $4) RETURNING id`,
			task, text, imageID, active).Scan(&id)
		s.audit(ctx, a.TelegramID, "prompt.create", "", map[string]any{"task": task})
	} else {
		var ok bool
		if id, ok = pathID(w, r); !ok {
			return
		}
		_, err = s.DB.Exec(ctx, `UPDATE writing_prompts SET task = $2, text = $3, image_id = coalesce($4, image_id), active = $5 WHERE id = $1`,
			id, task, text, imageID, active)
		s.audit(ctx, a.TelegramID, "prompt.update", "prompt:"+strconv.FormatInt(id, 10), map[string]any{"active": active})
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *server) adminQuestions(w http.ResponseWriter, r *http.Request, _ users.User) {
	s.listJSON(w, r, "questions", `SELECT id, part, text, active FROM speaking_questions ORDER BY part, id`)
}

func (s *server) adminQuestionSave(w http.ResponseWriter, r *http.Request, a users.User) {
	var b struct {
		Part   int    `json:"part"`
		Text   string `json:"text"`
		Active bool   `json:"active"`
	}
	if !readJSON(w, r, &b, 4<<10) {
		return
	}
	b.Text = strings.TrimSpace(b.Text)
	if b.Part < 1 || b.Part > 3 || b.Text == "" || utf8.RuneCountInString(b.Text) > 1000 {
		writeError(w, http.StatusBadRequest, "bad_question")
		return
	}
	ctx := r.Context()
	var id int64
	var err error
	if r.PathValue("id") == "" {
		err = s.DB.QueryRow(ctx, `INSERT INTO speaking_questions (part, text, active) VALUES ($1, $2, $3) RETURNING id`, b.Part, b.Text, b.Active).Scan(&id)
		s.audit(ctx, a.TelegramID, "question.create", "", nil)
	} else {
		var ok bool
		if id, ok = pathID(w, r); !ok {
			return
		}
		_, err = s.DB.Exec(ctx, `UPDATE speaking_questions SET part = $2, text = $3, active = $4 WHERE id = $1`, id, b.Part, b.Text, b.Active)
		s.audit(ctx, a.TelegramID, "question.update", "question:"+strconv.FormatInt(id, 10), map[string]any{"active": b.Active})
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

func (s *server) adminLog(w http.ResponseWriter, r *http.Request, _ users.User) {
	s.listJSON(w, r, "log", `SELECT id, admin_tg, action, target, details, created_at FROM admin_log ORDER BY id DESC LIMIT 200`)
}
