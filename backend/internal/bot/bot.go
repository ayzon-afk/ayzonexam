// Package bot — бот, который открывает Mini App и (в следующих этапах) шлёт уведомления.
// Bot API вызывается напрямую по HTTP, без библиотеки.
package bot

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Bot struct {
	token     string
	apiBase   string
	webAppURL string
	client    *http.Client
	log       *slog.Logger
}

func New(token, webAppURL string, log *slog.Logger) *Bot {
	return &Bot{
		token:     token,
		apiBase:   "https://api.telegram.org",
		webAppURL: webAppURL,
		client:    &http.Client{Timeout: 70 * time.Second}, // дольше long polling (50 с)
		log:       log,
	}
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	Description string          `json:"description"`
}

// call вызывает метод Bot API. Адрес запроса содержит токен — в ошибку он не попадает.
func (b *Bot) call(ctx context.Context, method string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.apiBase+"/bot"+b.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram %s: запрос не собран", method)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := b.client.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer res.Body.Close()
	var r apiResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return fmt.Errorf("telegram %s: ответ %d не JSON", method, res.StatusCode)
	}
	if !r.OK {
		return fmt.Errorf("telegram %s: %s", method, r.Description)
	}
	if out != nil {
		return json.Unmarshal(r.Result, out)
	}
	return nil
}

// Setup ставит кнопку меню «открыть приложение» и включает webhook или снимает его для polling.
func (b *Bot) Setup(ctx context.Context, webhookURL, secret string) error {
	if err := b.call(ctx, "setChatMenuButton", map[string]any{
		"menu_button": map[string]any{"type": "web_app", "text": "AYZON EXAM", "web_app": map[string]string{"url": b.webAppURL}},
	}, nil); err != nil {
		return err
	}
	if webhookURL == "" {
		return b.call(ctx, "deleteWebhook", map[string]any{}, nil)
	}
	return b.call(ctx, "setWebhook", map[string]any{
		"url":             webhookURL,
		"secret_token":    secret,
		"allowed_updates": []string{"message"},
	}, nil)
}

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type Message struct {
	Chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
	From *struct {
		LanguageCode string `json:"language_code"`
	} `json:"from"`
	Text string `json:"text"`
}

// RunPolling получает обновления long polling'ом до отмены ctx. Для локального запуска.
func (b *Bot) RunPolling(ctx context.Context) {
	var offset int64
	for ctx.Err() == nil {
		var updates []Update
		err := b.call(ctx, "getUpdates", map[string]any{"offset": offset, "timeout": 50, "allowed_updates": []string{"message"}}, &updates)
		if err != nil {
			if ctx.Err() == nil {
				b.log.Warn("бот: getUpdates", "err", err)
				select {
				case <-ctx.Done():
				case <-time.After(5 * time.Second):
				}
			}
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			b.Handle(ctx, u)
		}
	}
}

// WebhookHandler принимает обновления от Telegram; чужой запрос без секрета получает 401.
func (b *Bot) WebhookHandler(secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var u Update
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&u); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b.Handle(r.Context(), u)
		w.WriteHeader(http.StatusOK)
	})
}

// Handle отвечает на сообщение в личке кнопкой запуска приложения. Группы бот не обслуживает.
func (b *Bot) Handle(ctx context.Context, u Update) {
	m := u.Message
	if m == nil || m.Chat.Type != "private" {
		return
	}
	lang := ""
	if m.From != nil {
		lang = m.From.LanguageCode
	}
	text, button := greeting(lang, strings.HasPrefix(m.Text, "/start"))
	err := b.call(ctx, "sendMessage", map[string]any{
		"chat_id": m.Chat.ID,
		"text":    text,
		"reply_markup": map[string]any{
			"inline_keyboard": [][]map[string]any{{{"text": button, "web_app": map[string]string{"url": b.webAppURL}}}},
		},
	}, nil)
	if err != nil {
		b.log.Warn("бот: ответ не отправлен", "err", err)
	}
}

func greeting(lang string, start bool) (text, button string) {
	switch lang {
	case "ru":
		if start {
			return "Привет! Это AYZON EXAM — IELTS-style mock с AI-проверкой Writing и Speaking. Жми кнопку, чтобы начать.", "Открыть"
		}
		return "Всё происходит в приложении — открой его кнопкой ниже.", "Открыть"
	case "en":
		if start {
			return "Hi! This is AYZON EXAM — IELTS-style mocks with AI feedback on Writing and Speaking. Tap the button to start.", "Open"
		}
		return "Everything happens in the app — open it with the button below.", "Open"
	}
	if start {
		return "Salom! Bu AYZON EXAM — Writing va Speaking AI tekshiruvi bilan IELTS-style mock. Boshlash uchun tugmani bosing.", "Ochish"
	}
	return "Hammasi ilova ichida — pastdagi tugma orqali oching.", "Ochish"
}
