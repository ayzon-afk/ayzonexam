package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"ayzonexam/internal/billing"
	"ayzonexam/internal/news"
	"ayzonexam/internal/support"
	"ayzonexam/internal/users"
)

// Новости (этап 14) и поддержка глазами ученика (этап 17).

func (s *server) listNews(w http.ResponseWriter, r *http.Request, _ users.User) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	list, err := s.news.List(r.Context(), limit, false)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if list == nil {
		list = []news.Post{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"posts": list})
}

func (s *server) getPost(w http.ResponseWriter, r *http.Request, _ users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := s.news.Get(r.Context(), id, false)
	if errors.Is(err, news.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *server) mySupport(w http.ResponseWriter, r *http.Request, u users.User) {
	list, err := s.support.List(r.Context(), u.ID, "", 10)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	for i := range list {
		t, err := s.support.Get(r.Context(), list[i].ID, u.ID)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		list[i] = t
	}
	writeJSON(w, http.StatusOK, map[string]any{"tickets": list})
}

// writeSupport — сообщение в поддержку из приложения: текст и, по желанию, одно фото.
func (s *server) writeSupport(w http.ResponseWriter, r *http.Request, u users.User) {
	if code := parseForm(w, r, maxImageBytes+64<<10); code != "" {
		writeError(w, http.StatusBadRequest, code)
		return
	}
	defer r.MultipartForm.RemoveAll()
	text := ""
	if v := r.MultipartForm.Value["text"]; len(v) > 0 {
		text = v[0]
	}
	ctx := r.Context()
	var photo *Photo
	var mediaID *int64
	if fhs := r.MultipartForm.File["photo"]; len(fhs) > 0 {
		img, code := readImage(fhs[0])
		if code != "" {
			writeError(w, http.StatusBadRequest, code)
			return
		}
		var id int64
		if err := s.DB.QueryRow(ctx, `INSERT INTO media (mime, data, private) VALUES ($1, $2, true) RETURNING id`, img.MIME, img.Data).Scan(&id); err != nil {
			s.fail(w, r, err)
			return
		}
		mediaID, photo = &id, &Photo{MIME: img.MIME, Data: img.Data}
	}
	id, err := s.support.Write(ctx, u.ID, text, mediaID)
	if errors.Is(err, support.ErrEmpty) {
		writeError(w, http.StatusBadRequest, "empty")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.ticketToAdmins(ctx, u, id, text, photo)
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// ticketToAdmins — обращение приходит владельцу в Telegram: ответ на это сообщение уйдёт ученику.
func (s *server) ticketToAdmins(ctx context.Context, u users.User, ticketID int64, text string, photo *Photo) {
	plan := "free"
	if a := billing.Of(u, s.Now()); a.Premium {
		plan = "Premium (" + a.Source + ")"
	}
	center := "—"
	if u.Member() {
		s.DB.QueryRow(ctx, `SELECT name FROM centers WHERE id = $1`, *u.CenterID).Scan(&center)
	}
	msg := fmt.Sprintf("🆘 Обращение #%d\n%s · %s · центр: %s\n\n%s\n\n↩️ Ответьте на это сообщение — ответ уйдёт ученику.",
		ticketID, u.Nickname, plan, center, text)
	s.toAdmins(ctx, msg, photo, [][]Button{{{Text: "✅ Закрыть", Data: "tk:close:" + strconv.FormatInt(ticketID, 10)}}}, "ticket", ticketID)
}
