// Package api — HTTP-интерфейс Mini App.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/billing"
	"ayzonexam/internal/exam"
	"ayzonexam/internal/news"
	"ayzonexam/internal/rating"
	"ayzonexam/internal/social"
	"ayzonexam/internal/speaking"
	"ayzonexam/internal/support"
	"ayzonexam/internal/tgauth"
	"ayzonexam/internal/users"
	"ayzonexam/internal/writing"
)

// Ташкент без перехода на летнее время: UTC+5 круглый год.
var tashkent = time.FixedZone("UZT", 5*3600)

// initDataMaxAge — сколько живёт вход: Telegram выдаёт свежий initData при каждом открытии приложения.
const initDataMaxAge = 24 * time.Hour

type Deps struct {
	DB                *pgxpool.Pool // остальные хранилища (задания, рейтинг, …) строятся из него
	Users             *users.Store
	Writing           *writing.Store
	AI                ai.Checker // распознавание рукописи и озвучка; оценку делают очереди
	WritingDailyLimit int        // проверок Writing в день с Premium
	BotToken          string
	DevAuth           bool
	WebDir            string
	Webhook           http.Handler // nil — бот без webhook
	Msg               BotAPI       // nil — бот выключен
	BotUsername       string       // @username бота без «@»: ссылки-приглашения t.me/<bot>?start=…
	Admins            []int64      // Telegram id владельца и помощников — вход в админку
	PayCard           string       // номер карты для переводов; пусто — оплата недоступна
	PayCardName       string       // получатель
	Log               *slog.Logger
	Now               func() time.Time // nil — time.Now
}

// Server — HTTP-интерфейс и обработчик входящих сообщений бота (HandleUpdate).
type Server = server

type server struct {
	Deps
	h        http.Handler
	exam     *exam.Store
	rating   *rating.Store
	billing  *billing.Store
	social   *social.Store
	news     *news.Store
	support  *support.Store
	speaking *speaking.Store
	leads    leadLimiter
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.h.ServeHTTP(w, r) }

// Speaking — хранилище проверок Speaking: очередь (speaking.Queue) строится на нём в cmd/server.
func (s *server) Speaking() *speaking.Store { return s.speaking }

// AfterCheck — проверка Writing или Speaking закрыта: если она из mock, mock пересчитывается.
func (s *server) AfterCheck(ctx context.Context, table string, checkID int64) {
	mockID, err := s.exam.MockOfCheck(ctx, table, checkID)
	if err != nil || mockID == nil {
		return
	}
	if err := s.mockPartDone(ctx, *mockID); err != nil {
		s.Log.Error("mock не пересчитан", "mock", *mockID, "err", err)
	}
}

func New(d Deps) *Server {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Log == nil {
		d.Log = slog.Default()
	}
	s := &server{Deps: d, exam: exam.NewStore(d.DB), rating: rating.NewStore(d.DB), billing: billing.NewStore(d.DB),
		social: social.NewStore(d.DB), news: news.NewStore(d.DB), support: support.NewStore(d.DB), speaking: speaking.NewStore(d.DB)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	a := func(pattern string, h func(http.ResponseWriter, *http.Request, users.User)) {
		mux.Handle(pattern, s.auth(h))
	}
	adm := func(pattern string, h func(http.ResponseWriter, *http.Request, users.User)) {
		mux.Handle(pattern, s.admin(h))
	}

	// Открытая страница: без входа (DECISIONS 2026-11-24).
	mux.HandleFunc("GET /api/info", s.publicInfo)
	mux.HandleFunc("POST /api/leads", s.createLead)
	mux.HandleFunc("POST /api/web/login", s.webLogin)

	a("GET /api/me", s.getMe)
	a("PATCH /api/me", s.patchMe)
	a("GET /api/me/stats", s.getStats)
	a("GET /api/history", s.getHistory)
	a("POST /api/beat", s.beat)

	a("GET /api/writing/prompts", s.writingPrompts)
	a("POST /api/writing/recognize", s.recognize)
	a("GET /api/writing/checks", s.listChecks)
	a("POST /api/writing/checks", s.createCheck)
	a("GET /api/writing/checks/{id}", s.getCheck)

	a("GET /api/tests", s.listTests)
	a("POST /api/tests/{id}/start", s.startTest)
	a("POST /api/attempts/{id}/submit", s.submitAttempt)
	a("GET /api/attempts/{id}", s.getAttempt)
	mux.HandleFunc("GET /media/{id}", s.media)

	a("GET /api/rating", s.getRating)
	a("GET /api/rating/centers", s.getCenterRating)
	a("GET /api/centers", s.listCenters)
	a("GET /api/centers/{id}", s.getCenter)
	a("POST /api/centers/join", s.joinCenter)
	a("POST /api/centers/{id}/request", s.requestCenter)
	a("DELETE /api/centers/me", s.leaveCenter)
	a("GET /api/center/requests", s.centerRequests)
	a("POST /api/center/requests/{id}", s.decideRequest)
	a("GET /api/cabinet", s.getCabinet)
	a("PATCH /api/cabinet", s.patchCabinet)
	a("POST /api/cabinet/code", s.cabinetCode)
	a("GET /api/cabinet/students", s.cabinetStudents)
	a("DELETE /api/cabinet/students/{id}", s.cabinetRemove)
	a("PUT /api/cabinet/students/{id}/role", s.cabinetRole)
	a("GET /api/tournament", s.getTournament)
	a("POST /api/tournament/join", s.joinTournament)
	a("POST /api/tournament/start", s.startTournamentPart)

	a("GET /api/mock", s.getMocks)
	a("POST /api/mock", s.createMock)
	a("GET /api/mock/{id}", s.getMock)
	a("POST /api/mock/{id}/part", s.startMockPart)
	a("POST /api/mock/{id}/writing/start", s.startMockWriting)
	a("POST /api/mock/{id}/writing", s.submitMockWriting)
	a("POST /api/mock/{id}/speaking/start", s.startMockSpeaking)
	a("POST /api/mock/{id}/speaking", s.submitMockSpeaking)

	a("POST /api/referral", s.refer)
	a("GET /api/friends", s.getFriends)
	a("DELETE /api/friends/{id}", s.removeFriend)
	a("GET /api/duels", s.listDuels)
	a("POST /api/duels", s.createDuel)
	a("GET /api/duels/code/{code}", s.duelByCode)
	a("GET /api/duels/{id}", s.getDuel)
	a("POST /api/duels/{id}/play", s.playDuel)
	a("POST /api/duels/{id}/answer", s.answerDuel)
	a("POST /api/duels/{id}/finish", s.finishDuel)

	a("GET /api/news", s.listNews)
	a("GET /api/news/{id}", s.getPost)
	a("GET /api/support", s.mySupport)
	a("POST /api/support", s.writeSupport)

	a("GET /api/speaking", s.getSpeaking)
	a("POST /api/speaking", s.createSpeaking)
	a("GET /api/speaking/{id}", s.getSpeakingCheck)
	a("POST /api/speaking/bot", s.speakingViaBot)

	a("GET /api/pay/link", s.payLink)
	mux.HandleFunc("GET /pay", s.payPageGet)
	mux.HandleFunc("POST /pay", s.payPagePost)

	adm("GET /api/admin/overview", s.adminOverview)
	adm("GET /api/admin/users", s.adminUsers)
	adm("GET /api/admin/users/{id}", s.adminUser)
	adm("PATCH /api/admin/users/{id}", s.adminUserPatch)
	adm("GET /api/admin/centers", s.adminCenters)
	adm("POST /api/admin/centers", s.adminCenterSave)
	adm("PUT /api/admin/centers/{id}", s.adminCenterSave)
	adm("GET /api/admin/centers/{id}/payments", s.adminCenterPayments)
	adm("POST /api/admin/centers/{id}/payments", s.adminCenterPayment)
	adm("GET /api/admin/teachers", s.adminTeachers)
	adm("GET /api/admin/payments", s.adminPayments)
	adm("POST /api/admin/payments/{id}", s.adminPaymentDecide)
	adm("GET /api/admin/media/{id}", s.adminMedia)
	adm("GET /api/admin/tickets", s.adminTickets)
	adm("GET /api/admin/tickets/{id}", s.adminTicket)
	adm("POST /api/admin/tickets/{id}", s.adminTicketAct)
	adm("GET /api/admin/news", s.adminNews)
	adm("POST /api/admin/news", s.adminNewsSave)
	adm("PUT /api/admin/news/{id}", s.adminNewsSave)
	adm("GET /api/admin/tournaments", s.adminTournaments)
	adm("POST /api/admin/tournaments/{id}", s.adminTournamentAct)
	adm("GET /api/admin/tests", s.adminTests)
	adm("POST /api/admin/tests", s.adminTestSave)
	adm("PUT /api/admin/tests/{id}", s.adminTestSave)
	adm("POST /api/admin/tests/{id}/audio", s.adminTestAudio)
	adm("POST /api/admin/tests/{id}/speak", s.adminTestSpeak)
	adm("GET /api/admin/prompts", s.adminPrompts)
	adm("POST /api/admin/prompts", s.adminPromptSave)
	adm("POST /api/admin/prompts/{id}", s.adminPromptSave)
	adm("GET /api/admin/questions", s.adminQuestions)
	adm("POST /api/admin/questions", s.adminQuestionSave)
	adm("PUT /api/admin/questions/{id}", s.adminQuestionSave)
	adm("GET /api/admin/log", s.adminLog)
	adm("GET /api/admin/usage", s.adminUsage)
	adm("GET /api/admin/leads", s.adminLeads)
	adm("GET /api/admin/usage.csv", s.adminUsageCSV)
	adm("POST /api/admin/usage/send", s.adminUsageSend)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found")
	})
	if d.DevAuth {
		mux.HandleFunc("GET /api/dev/init-data", s.devInitData)
	}
	if d.Webhook != nil {
		mux.Handle("POST /tg/webhook", d.Webhook)
	}
	if d.WebDir != "" {
		mux.Handle("/", spa(d.WebDir))
	}
	s.h = headers(mux)
	return s
}

// auth пускает дальше только запрос с подписанным Telegram initData и находит (или создаёт) ученика.
func (s *server) auth(next func(http.ResponseWriter, *http.Request, users.User)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "tma ")
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		tg, err := tgauth.Validate(raw, s.BotToken, initDataMaxAge, s.Now())
		if err != nil {
			code := "unauthorized"
			if errors.Is(err, tgauth.ErrExpired) {
				code = "expired"
			}
			writeError(w, http.StatusUnauthorized, code)
			return
		}
		u, err := s.Users.Ensure(r.Context(), tg)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if u.BlockedAt != nil {
			writeError(w, http.StatusForbidden, "blocked")
			return
		}
		if err := s.Users.Seen(r.Context(), u, s.Now()); err != nil {
			s.Log.Warn("seen_at не записан", "err", err)
		}
		next(w, r, u)
	})
}

// meView — ученик о самом себе. Имя видит только он сам; другим оно не отдаётся никогда.
type meView struct {
	Nickname  string    `json:"nickname"`
	FirstName string    `json:"first_name"`
	PhotoURL  string    `json:"photo_url"`
	Lang      string    `json:"lang"`
	GoalBand  *float64  `json:"goal_band"`
	ExamDate  *string   `json:"exam_date"`
	Onboarded bool      `json:"onboarded"`
	City      string    `json:"city"`
	Center    *myCenter `json:"center"`
	Bot       string    `json:"bot"` // @username бота для ссылок-приглашений; '' — бот не настроен
	Admin     bool      `json:"admin"`
	Teacher   bool      `json:"verified_teacher"`
	Plan      *planView `json:"plan"`
	Invite    string    `json:"invite"` // код приглашения друзей
}

type myCenter struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Status string  `json:"status"` // pending, member
	Role   *string `json:"role"`   // admin, teacher
}

func (s *server) toMe(r *http.Request, u users.User) (meView, error) {
	v := meView{
		Nickname: u.Nickname, FirstName: u.FirstName, PhotoURL: u.PhotoURL,
		Lang: u.Lang, GoalBand: u.GoalBand, Onboarded: u.OnboardedAt != nil, City: u.City,
		Bot: s.BotUsername, Admin: s.isAdmin(u), Teacher: u.VerifiedTeacher,
	}
	if u.ExamDate != nil {
		d := u.ExamDate.Format(time.DateOnly)
		v.ExamDate = &d
	}
	if u.CenterID != nil && u.CenterStatus != nil {
		c := myCenter{ID: *u.CenterID, Status: *u.CenterStatus, Role: u.CenterRole}
		if err := s.DB.QueryRow(r.Context(), `SELECT name FROM centers WHERE id = $1`, c.ID).Scan(&c.Name); err != nil {
			return v, err
		}
		v.Center = &c
	}
	return v, s.fillMe(r, u, &v)
}

func (s *server) writeMe(w http.ResponseWriter, r *http.Request, u users.User) {
	v, err := s.toMe(r, u)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *server) getMe(w http.ResponseWriter, r *http.Request, u users.User) {
	s.writeMe(w, r, u)
}

// patchMeBody: exam_date — строка «ГГГГ-ММ-ДД» или null («пока не знаю»); отсутствие поля — не менять.
type patchMeBody struct {
	Lang      *string         `json:"lang"`
	GoalBand  *float64        `json:"goal_band"`
	ExamDate  json.RawMessage `json:"exam_date"`
	Onboarded bool            `json:"onboarded"`
	Nickname  *string         `json:"nickname"`
	City      *string         `json:"city"`
}

func (s *server) patchMe(w http.ResponseWriter, r *http.Request, u users.User) {
	var b patchMeBody
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json")
		return
	}

	p := users.Patch{Lang: b.Lang, GoalBand: b.GoalBand, Onboarded: b.Onboarded, City: b.City}
	if b.Nickname != nil {
		nick := strings.ToLower(strings.TrimSpace(*b.Nickname))
		if !validNick(nick) {
			writeError(w, http.StatusBadRequest, "bad_nickname")
			return
		}
		p.Nickname = &nick
	}
	if b.City != nil && *b.City != "tashkent" && *b.City != "other" {
		writeError(w, http.StatusBadRequest, "bad_city")
		return
	}
	if b.Lang != nil && !validLang(*b.Lang) {
		writeError(w, http.StatusBadRequest, "bad_lang")
		return
	}
	if b.GoalBand != nil && !validBand(*b.GoalBand) {
		writeError(w, http.StatusBadRequest, "bad_goal_band")
		return
	}
	if b.ExamDate != nil {
		p.SetExamDate = true
		if string(b.ExamDate) != "null" {
			var str string
			if json.Unmarshal(b.ExamDate, &str) != nil {
				writeError(w, http.StatusBadRequest, "bad_exam_date")
				return
			}
			d, err := time.ParseInLocation(time.DateOnly, str, tashkent)
			n := s.Now().In(tashkent)
			today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, tashkent)
			if err != nil || d.Before(today) || d.After(today.AddDate(2, 0, 0)) {
				writeError(w, http.StatusBadRequest, "bad_exam_date")
				return
			}
			p.ExamDate = &d
		}
	}
	// Онбординг не закончен, пока нет цели: без неё нечего показывать на главной.
	if b.Onboarded && b.GoalBand == nil && u.GoalBand == nil {
		writeError(w, http.StatusBadRequest, "goal_required")
		return
	}

	updated, err := s.Users.Update(r.Context(), u.ID, p)
	if errors.Is(err, users.ErrNickTaken) {
		writeError(w, http.StatusConflict, "nick_taken")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeMe(w, r, updated)
}

func validLang(l string) bool { return l == "uz" || l == "ru" || l == "en" }

// validNick: 3–20 знаков a–z, 0–9, «_» и «.». Вид user<число> занят за ником по умолчанию — его взял бы
// новый ученик с этим номером, и вход у него сломался бы.
func validNick(n string) bool {
	if len(n) < 3 || len(n) > 20 {
		return false
	}
	for _, r := range n {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.') {
			return false
		}
	}
	if rest, ok := strings.CutPrefix(n, "user"); ok && rest != "" && strings.Trim(rest, "0123456789") == "" {
		return false
	}
	return true
}

func validBand(b float64) bool {
	return b >= 4 && b <= 9 && b*2 == float64(int(b*2))
}

// devInitData подписывает initData выдуманного ученика, чтобы открыть приложение в обычном браузере.
// Маршрут есть только при DEV_AUTH=1, а config не пускает DEV_AUTH на боевой сервер.
func (s *server) devInitData(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id <= 0 {
		id = 1000001
	}
	lang := r.URL.Query().Get("lang")
	if !validLang(lang) {
		lang = "ru"
	}
	user, _ := json.Marshal(tgauth.User{ID: id, FirstName: "Dev", LanguageCode: lang})
	init := tgauth.Sign(url.Values{
		"user":      {string(user)},
		"auth_date": {strconv.FormatInt(s.Now().Unix(), 10)},
	}, s.BotToken)
	writeJSON(w, http.StatusOK, map[string]string{"init_data": init})
}

func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	s.Log.Error("запрос не выполнен", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// spa отдаёт собранный интерфейс: файлы из assets/ кэшируются навсегда (в имени хэш),
// index.html — каждый раз свежий, неизвестный путь — тоже index.html.
func spa(dir string) http.Handler {
	root := os.DirFS(dir)
	files := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if st, err := fs.Stat(root, name); name == "" || err != nil || st.IsDir() {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, root, "index.html")
			return
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}

// isUnique — ошибка нарушения уникальности (имя занято).
func isUnique(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}
