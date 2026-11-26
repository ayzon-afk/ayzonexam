package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ayzonexam/internal/ai"
	"ayzonexam/internal/billing"
	"ayzonexam/internal/exam"
	"ayzonexam/internal/speaking"
	"ayzonexam/internal/users"
)

// Speaking (этап 4): запись в приложении, голосовое боту, Speaking в mock.

// audioMIME — тип записи по содержимому: Gemini принимает ogg, mp4/aac, mp3, wav, webm (DECISIONS).
func audioMIME(data []byte) string {
	switch {
	case len(data) > 4 && string(data[:4]) == "OggS":
		return "audio/ogg"
	case len(data) > 4 && string(data[:4]) == "\x1a\x45\xdf\xa3":
		return "audio/webm"
	case len(data) > 12 && string(data[4:8]) == "ftyp":
		return "audio/mp4"
	case len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE":
		return "audio/wav"
	case len(data) > 3 && (string(data[:3]) == "ID3" || data[0] == 0xff && data[1]&0xe0 == 0xe0):
		return "audio/mp3"
	}
	return ""
}

func readAudio(fh *multipart.FileHeader) (ai.Audio, string) {
	if fh.Size > speaking.MaxAudioBytes {
		return ai.Audio{}, "audio_too_large"
	}
	f, err := fh.Open()
	if err != nil {
		return ai.Audio{}, "bad_audio"
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, speaking.MaxAudioBytes+1))
	if err != nil || len(data) > speaking.MaxAudioBytes {
		return ai.Audio{}, "audio_too_large"
	}
	mime := audioMIME(data)
	if mime == "" || len(data) < 2000 {
		return ai.Audio{}, "bad_audio"
	}
	return ai.Audio{MIME: mime, Data: data}, ""
}

type speakQuota struct {
	Used   int    `json:"used"`
	Limit  int    `json:"limit"`
	Period string `json:"period"`
	Bonus  int    `json:"bonus"`
}

func (s *server) speakQuota(ctx context.Context, u users.User) (speakQuota, billing.Quota, error) {
	q := billing.SpeakingQuota(billing.Of(u, s.Now()), s.Now())
	used, err := s.speaking.Used(ctx, u.ID, q.Since)
	return speakQuota{used, q.Limit, q.Period, u.BonusChecks}, q, err
}

// getSpeaking — вопросы для практики (по одному на часть), история и лимит.
func (s *server) getSpeaking(w http.ResponseWriter, r *http.Request, u users.User) {
	ctx := r.Context()
	qs := map[string]speaking.Question{}
	for part := 1; part <= 3; part++ {
		list, err := s.speaking.Random(ctx, part, 1)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if len(list) > 0 {
			qs[strconv.Itoa(part)] = list[0]
		}
	}
	hist, err := s.speaking.List(ctx, u.ID, 20)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	q, _, err := s.speakQuota(ctx, u)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": qs, "history": hist, "quota": q})
}

// createSpeaking — ответ на вопрос из приложения: одна запись.
func (s *server) createSpeaking(w http.ResponseWriter, r *http.Request, u users.User) {
	if code := parseForm(w, r, speaking.MaxAudioBytes+64<<10); code != "" {
		writeError(w, http.StatusBadRequest, strings.Replace(code, "image", "audio", 1))
		return
	}
	defer r.MultipartForm.RemoveAll()
	f := r.MultipartForm
	qid, _ := strconv.ParseInt(first(f.Value["question_id"]), 10, 64)
	secs, _ := strconv.Atoi(first(f.Value["seconds"]))
	if len(f.File["audio"]) != 1 {
		writeError(w, http.StatusBadRequest, "bad_audio")
		return
	}
	ctx := r.Context()
	q, err := s.speaking.Question(ctx, qid)
	if errors.Is(err, speaking.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "bad_question")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	a, code := readAudio(f.File["audio"][0])
	if code != "" {
		writeError(w, http.StatusBadRequest, code)
		return
	}
	_, quota, err := s.speakQuota(ctx, u)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	id, err := s.speaking.Create(ctx, speaking.NewCheck{UserID: u.ID, Source: "app",
		Questions: []ai.SpeakingQuestion{{Part: q.Part, Text: q.Text}}, Audio: []ai.Audio{a},
		Seconds: min(max(secs, 0), speaking.MaxSeconds), Lang: u.Lang}, quota.Limit, quota.Since, s.Now())
	if errors.Is(err, speaking.ErrLimit) {
		writeError(w, http.StatusTooManyRequests, "daily_limit")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]int64{"id": id})
}

func first(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

func (s *server) getSpeakingCheck(w http.ResponseWriter, r *http.Request, u users.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := s.speaking.Get(r.Context(), u.ID, id)
	if errors.Is(err, speaking.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// speakingViaBot — запасной путь (SPEC): микрофон в Mini App не работает — бот присылает вопрос, ученик
// отвечает голосовым.
func (s *server) speakingViaBot(w http.ResponseWriter, r *http.Request, u users.User) {
	var b struct {
		QuestionID int64 `json:"question_id"`
	}
	if !readJSON(w, r, &b, 1<<10) {
		return
	}
	ctx := r.Context()
	q, err := s.speaking.Question(ctx, b.QuestionID)
	if errors.Is(err, speaking.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "bad_question")
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if s.Msg == nil {
		writeError(w, http.StatusServiceUnavailable, "no_bot")
		return
	}
	if err := s.Users.SetBotState(ctx, u.ID, "speaking:"+strconv.FormatInt(q.ID, 10)); err != nil {
		s.fail(w, r, err)
		return
	}
	s.say(ctx, u.TelegramID, u.Lang, "speaking.ask", map[string]string{"q": q.Text}, "-")
	w.WriteHeader(http.StatusNoContent)
}

// startMockSpeaking — таймер Speaking mock и вопросы трёх частей.
func (s *server) startMockSpeaking(w http.ResponseWriter, r *http.Request, u users.User) {
	m, ok := s.loadMock(w, r, u, true)
	if !ok {
		return
	}
	m, err := s.exam.StartSpeaking(r.Context(), m, s.Now())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var qs []ai.SpeakingQuestion
	json.Unmarshal(m.SpeakingQuestions, &qs)
	if len(qs) == 0 {
		writeError(w, http.StatusNotFound, "no_tests")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ends_at": m.SpeakingStartedAt.Add(exam.SpeakingMinutes * time.Minute), "questions": qs,
	})
}

// submitMockSpeaking — записи всех частей Speaking mock (audio0, audio1, …) одной проверкой.
func (s *server) submitMockSpeaking(w http.ResponseWriter, r *http.Request, u users.User) {
	m, ok := s.loadMock(w, r, u, true)
	if !ok {
		return
	}
	var qs []ai.SpeakingQuestion
	json.Unmarshal(m.SpeakingQuestions, &qs)
	if m.SpeakingStartedAt == nil || len(qs) == 0 {
		writeError(w, http.StatusConflict, "not_started")
		return
	}
	if code := parseForm(w, r, int64(len(qs))*speaking.MaxAudioBytes+64<<10); code != "" {
		writeError(w, http.StatusBadRequest, strings.Replace(code, "image", "audio", 1))
		return
	}
	defer r.MultipartForm.RemoveAll()
	ctx := r.Context()
	var busy bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM speaking_checks WHERE mock_id = $1 AND status <> 'failed')`, m.ID).Scan(&busy); err != nil {
		s.fail(w, r, err)
		return
	}
	if busy {
		writeError(w, http.StatusConflict, "part_done")
		return
	}
	var audio []ai.Audio
	for i := range qs {
		fhs := r.MultipartForm.File["audio"+strconv.Itoa(i)]
		if len(fhs) != 1 {
			writeError(w, http.StatusBadRequest, "bad_audio")
			return
		}
		a, code := readAudio(fhs[0])
		if code != "" {
			writeError(w, http.StatusBadRequest, code)
			return
		}
		audio = append(audio, a)
	}
	secs, _ := strconv.Atoi(first(r.MultipartForm.Value["seconds"]))
	// Нарушения честного режима (камера): с трёх Speaking не идёт в рейтинг; повторная отправка их не обнуляет.
	violations, _ := strconv.Atoi(first(r.MultipartForm.Value["violations"]))
	if _, err := s.DB.Exec(ctx, `UPDATE mocks SET speaking_violations = greatest(speaking_violations, $2) WHERE id = $1`, m.ID, max(violations, 0)); err != nil {
		s.fail(w, r, err)
		return
	}
	if _, err := s.speaking.Create(ctx, speaking.NewCheck{UserID: u.ID, MockID: &m.ID, Source: "app", Questions: qs,
		Audio: audio, Seconds: max(secs, 0), Lang: u.Lang}, 0, s.Now(), s.Now()); err != nil {
		s.fail(w, r, err)
		return
	}
	v, err := s.mockView(ctx, u, m)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
