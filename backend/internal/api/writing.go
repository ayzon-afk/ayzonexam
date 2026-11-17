package api

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/billing"
	"ayzonexam/internal/users"
	"ayzonexam/internal/writing"
)

const (
	maxImageBytes       = 1536 << 10 // фото сжимается на телефоне (lib/photo.ts) до ~200–400 КБ
	maxPages            = 3
	maxEssayRunes       = 8000
	maxPromptRunes      = 1000
	minOwnPromptRunes   = 20
	recognizeDailyLimit = 10
	historyLimit        = 30
)

// dayStart — полночь сегодняшнего дня по Ташкенту: дневные лимиты считаются от неё.
func (s *server) dayStart() time.Time { return billing.DayStart(s.Now()) }

func (s *server) writingPrompts(w http.ResponseWriter, r *http.Request, _ users.User) {
	task := r.URL.Query().Get("task")
	if task != writing.Task1 && task != writing.Task2 {
		writeError(w, http.StatusBadRequest, "bad_task")
		return
	}
	list, err := s.Writing.Prompts(r.Context(), task)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if list == nil {
		list = []writing.Prompt{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"prompts": list})
}

// readImage читает одно фото из формы и проверяет, что это картинка нужного размера.
func readImage(fh *multipart.FileHeader) (ai.Image, string) {
	if fh.Size > maxImageBytes {
		return ai.Image{}, "image_too_large"
	}
	f, err := fh.Open()
	if err != nil {
		return ai.Image{}, "bad_image"
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
	if err != nil || len(data) > maxImageBytes {
		return ai.Image{}, "image_too_large"
	}
	// Тип — по содержимому, не по тому, что прислал телефон.
	switch mime := http.DetectContentType(data); mime {
	case "image/jpeg", "image/png", "image/webp":
		return ai.Image{MIME: mime, Data: data}, ""
	}
	return ai.Image{}, "bad_image"
}

// parseForm читает multipart-форму целиком в память, не больше limit байт.
func parseForm(w http.ResponseWriter, r *http.Request, limit int64) string {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := r.ParseMultipartForm(limit + 1<<20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return "image_too_large"
		}
		return "bad_form"
	}
	return ""
}

// recognize переписывает рукопись с фото в текст. Ученик видит текст и правит его до отправки.
func (s *server) recognize(w http.ResponseWriter, r *http.Request, u users.User) {
	if code := parseForm(w, r, maxPages*maxImageBytes+64<<10); code != "" {
		writeError(w, http.StatusBadRequest, code)
		return
	}
	defer r.MultipartForm.RemoveAll()
	files := r.MultipartForm.File["images"]
	if len(files) == 0 || len(files) > maxPages {
		writeError(w, http.StatusBadRequest, "bad_pages")
		return
	}
	pages := make([]ai.Image, 0, len(files))
	for _, fh := range files {
		img, code := readImage(fh)
		if code != "" {
			writeError(w, http.StatusBadRequest, code)
			return
		}
		pages = append(pages, img)
	}

	ok, err := s.Writing.TakeUsage(r.Context(), u.ID, "recognize", recognizeDailyLimit, s.dayStart(), s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !ok {
		writeError(w, http.StatusTooManyRequests, "daily_limit")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	text, err := s.AI.Recognize(ctx, pages)
	if err != nil {
		// Не получилось — попытка не считается.
		if rerr := s.Writing.ReturnUsage(context.WithoutCancel(r.Context()), u.ID, "recognize"); rerr != nil {
			s.Log.Error("лимит распознавания не возвращён", "err", rerr)
		}
		if errors.Is(err, ai.ErrNoText) {
			writeError(w, http.StatusUnprocessableEntity, "no_text")
			return
		}
		s.Log.Warn("распознавание не удалось", "err", err)
		writeError(w, http.StatusBadGateway, "ai_unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": text, "words": writing.CountWords(text)})
}

func (s *server) createCheck(w http.ResponseWriter, r *http.Request, u users.User) {
	if code := parseForm(w, r, maxImageBytes+64<<10); code != "" {
		writeError(w, http.StatusBadRequest, code)
		return
	}
	defer r.MultipartForm.RemoveAll()
	f := r.MultipartForm.Value
	get := func(k string) string {
		if v := f[k]; len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}

	c := writing.NewCheck{UserID: u.ID, Task: get("task"), Essay: get("essay"), Source: get("source"), Lang: u.Lang}
	if c.Task != writing.Task1 && c.Task != writing.Task2 {
		writeError(w, http.StatusBadRequest, "bad_task")
		return
	}
	if c.Source != "text" && c.Source != "photo" {
		writeError(w, http.StatusBadRequest, "bad_source")
		return
	}

	// Тема: Task 2 — из банка (prompt_id) или своя (prompt); Task 1 — фото задания и, по желанию, текст.
	prompt := get("prompt")
	if utf8.RuneCountInString(prompt) > maxPromptRunes {
		writeError(w, http.StatusBadRequest, "bad_prompt")
		return
	}
	switch {
	case get("prompt_id") != "":
		id, err := strconv.ParseInt(get("prompt_id"), 10, 64)
		if err != nil || c.Task != writing.Task2 {
			writeError(w, http.StatusBadRequest, "bad_prompt")
			return
		}
		p, err := s.Writing.Prompt(r.Context(), id)
		if errors.Is(err, writing.ErrNotFound) || (err == nil && p.Task != c.Task) {
			writeError(w, http.StatusBadRequest, "bad_prompt")
			return
		}
		if err != nil {
			s.fail(w, r, err)
			return
		}
		c.PromptID, c.Prompt = &p.ID, p.Text
	case c.Task == writing.Task2:
		if utf8.RuneCountInString(prompt) < minOwnPromptRunes {
			writeError(w, http.StatusBadRequest, "bad_prompt")
			return
		}
		c.Prompt = prompt
	default:
		c.Prompt = prompt
	}
	if fhs := r.MultipartForm.File["image"]; c.Task == writing.Task1 {
		if len(fhs) != 1 {
			writeError(w, http.StatusBadRequest, "image_required")
			return
		}
		img, code := readImage(fhs[0])
		if code != "" {
			writeError(w, http.StatusBadRequest, code)
			return
		}
		c.Image = &img
	}

	maxWords := writing.MaxWordsTask2
	if c.Task == writing.Task1 {
		maxWords = writing.MaxWordsTask1
	}
	c.Words = writing.CountWords(c.Essay)
	if c.Words < writing.MinWords {
		writeError(w, http.StatusBadRequest, "too_short")
		return
	}
	if c.Words > maxWords || utf8.RuneCountInString(c.Essay) > maxEssayRunes {
		writeError(w, http.StatusBadRequest, "too_long")
		return
	}

	q := billing.WritingQuota(billing.Of(u, s.Now()), s.WritingDailyLimit, s.Now())
	id, err := s.Writing.Create(r.Context(), c, q.Limit, q.Since, s.Now())
	if errors.Is(err, writing.ErrDailyLimit) {
		writeError(w, http.StatusTooManyRequests, "daily_limit")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]int64{"id": id})
}

// checkSummary — строка истории: без текста эссе и разбора.
type checkSummary struct {
	ID        int64     `json:"id"`
	Task      string    `json:"task"`
	Status    string    `json:"status"`
	Band      *float64  `json:"band"`
	Words     int       `json:"words"`
	Prompt    string    `json:"prompt"`
	CreatedAt time.Time `json:"created_at"`
}

type criteriaView struct {
	TA  int `json:"ta"`
	CC  int `json:"cc"`
	LR  int `json:"lr"`
	GRA int `json:"gra"`
}

type checkView struct {
	checkSummary
	Essay      string            `json:"essay"`
	Source     string            `json:"source"`
	Criteria   *criteriaView     `json:"criteria"`
	Feedback   *writing.Feedback `json:"feedback"`
	FinishedAt *time.Time        `json:"finished_at"`
}

func toSummary(c writing.Check) checkSummary {
	prompt := c.Prompt
	if utf8.RuneCountInString(prompt) > 140 {
		prompt = string([]rune(prompt)[:140]) + "…"
	}
	return checkSummary{ID: c.ID, Task: c.Task, Status: c.Status, Band: c.Band, Words: c.Words, Prompt: prompt, CreatedAt: c.CreatedAt}
}

func (s *server) listChecks(w http.ResponseWriter, r *http.Request, u users.User) {
	list, err := s.Writing.List(r.Context(), u.ID, historyLimit)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	q := billing.WritingQuota(billing.Of(u, s.Now()), s.WritingDailyLimit, s.Now())
	used, err := s.Writing.Used(r.Context(), u.ID, q.Since)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]checkSummary, 0, len(list))
	for _, c := range list {
		out = append(out, toSummary(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"checks": out, "used": used, "limit": q.Limit, "period": q.Period, "bonus": u.BonusChecks})
}

func (s *server) getCheck(w http.ResponseWriter, r *http.Request, u users.User) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	c, err := s.Writing.Get(r.Context(), u.ID, id)
	if errors.Is(err, writing.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	v := checkView{checkSummary: toSummary(c), Essay: c.Essay, Source: c.Source, Feedback: c.Feedback, FinishedAt: c.FinishedAt}
	v.Prompt = c.Prompt
	if c.TA != nil && c.CC != nil && c.LR != nil && c.GRA != nil {
		v.Criteria = &criteriaView{*c.TA, *c.CC, *c.LR, *c.GRA}
	}
	writeJSON(w, http.StatusOK, v)
}
