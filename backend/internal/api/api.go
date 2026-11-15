// Package api — HTTP-интерфейс Mini App.
package api

import (
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

	"ayzonexam/internal/tgauth"
	"ayzonexam/internal/users"
)

// Ташкент без перехода на летнее время: UTC+5 круглый год.
var tashkent = time.FixedZone("UZT", 5*3600)

// initDataMaxAge — сколько живёт вход: Telegram выдаёт свежий initData при каждом открытии приложения.
const initDataMaxAge = 24 * time.Hour

type Deps struct {
	Users    *users.Store
	BotToken string
	DevAuth  bool
	WebDir   string
	Webhook  http.Handler // nil — бот без webhook
	Log      *slog.Logger
	Now      func() time.Time // nil — time.Now
}

type server struct{ Deps }

func New(d Deps) http.Handler {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Log == nil {
		d.Log = slog.Default()
	}
	s := &server{d}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.Handle("GET /api/me", s.auth(s.getMe))
	mux.Handle("PATCH /api/me", s.auth(s.patchMe))
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
	return headers(mux)
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
		next(w, r, u)
	})
}

// meView — ученик о самом себе. Имя видит только он сам; другим оно не отдаётся никогда.
type meView struct {
	Nickname  string   `json:"nickname"`
	FirstName string   `json:"first_name"`
	PhotoURL  string   `json:"photo_url"`
	Lang      string   `json:"lang"`
	GoalBand  *float64 `json:"goal_band"`
	ExamDate  *string  `json:"exam_date"`
	Onboarded bool     `json:"onboarded"`
}

func toMe(u users.User) meView {
	v := meView{
		Nickname: u.Nickname, FirstName: u.FirstName, PhotoURL: u.PhotoURL,
		Lang: u.Lang, GoalBand: u.GoalBand, Onboarded: u.OnboardedAt != nil,
	}
	if u.ExamDate != nil {
		d := u.ExamDate.Format(time.DateOnly)
		v.ExamDate = &d
	}
	return v
}

func (s *server) getMe(w http.ResponseWriter, _ *http.Request, u users.User) {
	writeJSON(w, http.StatusOK, toMe(u))
}

// patchMeBody: exam_date — строка «ГГГГ-ММ-ДД» или null («пока не знаю»); отсутствие поля — не менять.
type patchMeBody struct {
	Lang      *string         `json:"lang"`
	GoalBand  *float64        `json:"goal_band"`
	ExamDate  json.RawMessage `json:"exam_date"`
	Onboarded bool            `json:"onboarded"`
}

func (s *server) patchMe(w http.ResponseWriter, r *http.Request, u users.User) {
	var b patchMeBody
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json")
		return
	}

	p := users.Patch{Lang: b.Lang, GoalBand: b.GoalBand, Onboarded: b.Onboarded}
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
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMe(updated))
}

func validLang(l string) bool { return l == "uz" || l == "ru" || l == "en" }

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
