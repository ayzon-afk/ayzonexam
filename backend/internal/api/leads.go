package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ayzonexam/internal/users"
)

// Открытая страница (хакатон, DECISIONS 2026-11-24): адрес бота для кнопки «Открыть в Telegram» и заявка
// учебного центра. Без входа — поэтому лимиты: с одного адреса и всего за час.

const (
	leadsPerIP   = 3  // заявок с одного адреса за час
	leadsPerHour = 30 // заявок всего за час: дальше — спам, владелец всё равно не обзвонит
)

type leadLimiter struct {
	mu    sync.Mutex
	since time.Time
	total int
	byIP  map[string]int
}

// allow — окно в час: сбрасывается целиком, точнее не нужно.
func (l *leadLimiter) allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.byIP == nil || now.Sub(l.since) > time.Hour {
		l.since, l.total, l.byIP = now, 0, map[string]int{}
	}
	if l.total >= leadsPerHour || l.byIP[ip] >= leadsPerIP {
		return false
	}
	l.total++
	l.byIP[ip]++
	return true
}

// clientIP — адрес посетителя: за прокси Railway — X-Real-IP или последний X-Forwarded-For (его дописывает
// прокси, начало строки может подделать кто угодно).
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *server) publicInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"bot": s.BotUsername})
}

func (s *server) createLead(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Center   string `json:"center"`
		Name     string `json:"name"`
		Phone    string `json:"phone"`
		Students *int   `json:"students"`
		Message  string `json:"message"`
		Lang     string `json:"lang"`
		Website  string `json:"website"` // ловушка: поле скрыто, человек его не заполняет
	}
	if !readJSON(w, r, &b, 8<<10) {
		return
	}
	if b.Website != "" {
		w.WriteHeader(http.StatusNoContent) // бот думает, что отправил
		return
	}
	b.Center, b.Name, b.Phone, b.Message = strings.TrimSpace(b.Center), strings.TrimSpace(b.Name),
		strings.TrimSpace(b.Phone), strings.TrimSpace(b.Message)
	digits := 0
	for _, c := range b.Phone {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	switch {
	case b.Center == "" || utf8.RuneCountInString(b.Center) > 120,
		b.Name == "" || utf8.RuneCountInString(b.Name) > 80,
		digits < 7 || utf8.RuneCountInString(b.Phone) > 30,
		utf8.RuneCountInString(b.Message) > 1000,
		b.Students != nil && (*b.Students < 0 || *b.Students > 100000):
		writeError(w, http.StatusBadRequest, "bad_lead")
		return
	}
	if !validLang(b.Lang) {
		b.Lang = "uz"
	}
	if !s.leads.allow(clientIP(r), s.Now()) {
		writeError(w, http.StatusTooManyRequests, "too_many")
		return
	}
	var id int64
	if err := s.DB.QueryRow(r.Context(), `INSERT INTO center_leads (center, name, phone, students, message, lang)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, b.Center, b.Name, b.Phone, b.Students, b.Message, b.Lang).Scan(&id); err != nil {
		s.fail(w, r, err)
		return
	}
	if s.Msg != nil {
		students := "—"
		if b.Students != nil {
			students = fmt.Sprint(*b.Students)
		}
		text := fmt.Sprintf("Заявка центра №%d с сайта\nЦентр: %s\nКонтакт: %s\nТелефон: %s\nУчеников: %s",
			id, b.Center, b.Name, b.Phone, students)
		if b.Message != "" {
			text += "\n\n" + b.Message
		}
		if _, err := s.Msg.ToAdmins(r.Context(), text, nil, nil); err != nil {
			s.Log.Warn("заявка центра не отправлена владельцу", "lead", id, "err", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminLeads(w http.ResponseWriter, r *http.Request, _ users.User) {
	s.listJSON(w, r, "leads", `SELECT id, center, name, phone, students, message, lang, created_at
		FROM center_leads ORDER BY id DESC LIMIT 200`)
}
