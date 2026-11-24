// Package bot — Bot API напрямую по HTTP, без библиотеки: кнопка запуска Mini App, уведомления ученикам,
// сообщения владельцу (обращения, оплаты). Что делать с входящими сообщениями, решает OnUpdate (api);
// без него бот на всё отвечает кнопкой запуска.
package bot

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Bot struct {
	token     string
	apiBase   string
	webAppURL string
	client    *http.Client
	log       *slog.Logger
	username  string

	// Admins — Telegram id владельца и помощников: им уходят обращения и оплаты.
	Admins []int64
	// OnUpdate обрабатывает входящее; false — не его случай, бот отвечает кнопкой запуска.
	OnUpdate func(ctx context.Context, u Update) bool
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

// Username — @username бота без «@» (из getMe при Setup); ” — Telegram не ответил.
func (b *Bot) Username() string { return b.username }

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
	return b.do(ctx, method, "application/json", bytes.NewReader(body), out)
}

func (b *Bot) do(ctx context.Context, method, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.apiBase+"/bot"+b.token+"/"+method, body)
	if err != nil {
		return fmt.Errorf("telegram %s: запрос не собран", method)
	}
	req.Header.Set("Content-Type", contentType)
	res, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, stripURL(err))
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

// stripURL убирает из ошибки net/http адрес запроса — в нём токен.
func stripURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// Setup узнаёт @username бота, ставит кнопку меню «открыть приложение», команды и включает webhook или
// снимает его для polling.
func (b *Bot) Setup(ctx context.Context, webhookURL, secret string) error {
	var me struct {
		Username string `json:"username"`
	}
	if err := b.call(ctx, "getMe", map[string]any{}, &me); err != nil {
		return err
	}
	b.username = me.Username
	if err := b.call(ctx, "setChatMenuButton", map[string]any{
		"menu_button": map[string]any{"type": "web_app", "text": "AYZON EXAM", "web_app": map[string]string{"url": b.webAppURL}},
	}, nil); err != nil {
		return err
	}
	for _, lang := range []string{"", "ru", "en"} {
		cmds := []map[string]string{{"command": "start", "description": T(lang, "cmd.start")}, {"command": "help", "description": T(lang, "cmd.help")}}
		p := map[string]any{"commands": cmds}
		if lang != "" {
			p["language_code"] = lang
		}
		if err := b.call(ctx, "setMyCommands", p, nil); err != nil {
			return err
		}
	}
	if webhookURL == "" {
		return b.call(ctx, "deleteWebhook", map[string]any{}, nil)
	}
	return b.call(ctx, "setWebhook", map[string]any{
		"url":             webhookURL,
		"secret_token":    secret,
		"allowed_updates": []string{"message", "callback_query"},
	}, nil)
}

type Update struct {
	UpdateID      int64     `json:"update_id"`
	Message       *Message  `json:"message"`
	CallbackQuery *Callback `json:"callback_query"`
}

type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type PhotoSize struct {
	FileID   string `json:"file_id"`
	FileSize int    `json:"file_size"`
	Width    int    `json:"width"`
}

type File struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type"`
	FileSize int    `json:"file_size"`
}

type Message struct {
	MessageID int64       `json:"message_id"`
	Chat      Chat        `json:"chat"`
	From      *User       `json:"from"`
	Text      string      `json:"text"`
	Caption   string      `json:"caption"`
	Photo     []PhotoSize `json:"photo"`
	Voice     *File       `json:"voice"`
	Audio     *File       `json:"audio"`
	ReplyTo   *Message    `json:"reply_to_message"`
}

type Callback struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Data    string   `json:"data"`
	Message *Message `json:"message"`
}

// RunPolling получает обновления long polling'ом до отмены ctx. Для локального запуска.
func (b *Bot) RunPolling(ctx context.Context) {
	var offset int64
	for ctx.Err() == nil {
		var updates []Update
		err := b.call(ctx, "getUpdates", map[string]any{"offset": offset, "timeout": 50, "allowed_updates": []string{"message", "callback_query"}}, &updates)
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

// Handle — входящее обновление. Группы бот не обслуживает.
func (b *Bot) Handle(ctx context.Context, u Update) {
	if u.CallbackQuery != nil {
		if b.OnUpdate != nil {
			b.OnUpdate(ctx, u)
		}
		return
	}
	m := u.Message
	if m == nil || m.Chat.Type != "private" {
		return
	}
	if b.OnUpdate != nil && b.OnUpdate(ctx, u) {
		return
	}
	lang := ""
	if m.From != nil {
		lang = m.From.LanguageCode
	}
	if err := b.Greet(ctx, m.Chat.ID, lang, m.Text); err != nil {
		b.log.Warn("бот: ответ не отправлен", "err", err)
	}
}

// Greet отвечает кнопкой запуска. /start с параметром (ссылка-приглашение t.me/<бот>?start=…)
// передаёт его приложению как ?open=…
func (b *Bot) Greet(ctx context.Context, chatID int64, lang, text string) error {
	start := strings.HasPrefix(text, "/start")
	param := ""
	if start {
		param = strings.TrimSpace(strings.TrimPrefix(text, "/start"))
	}
	key := "greet"
	if start {
		key = "greet.start"
	}
	if strings.HasPrefix(param, "d-") {
		key = "greet.duel"
	}
	_, err := b.Send(ctx, chatID, T(lang, key), [][]Button{{{Text: T(lang, "btn.open"), Open: param}}})
	return err
}

// Photo — картинка для отправки.
type Photo struct {
	MIME string
	Data []byte
}

// Button — кнопка под сообщением: Data — callback, URL — ссылка, иначе — запуск приложения с ?open=Open.
type Button struct {
	Text string
	Data string
	URL  string
	Open string
}

// Sent — отправленное сообщение: на него админ может ответить reply.
type Sent struct {
	ChatID    int64
	MessageID int64
}

// AppURL — адрес Mini App с ?open=… (что открыть сразу); ” — просто приложение.
func (b *Bot) AppURL(open string) string {
	if open == "" {
		return b.webAppURL
	}
	u, err := url.Parse(b.webAppURL)
	if err != nil {
		return b.webAppURL
	}
	q := u.Query()
	q.Set("open", open)
	u.RawQuery = q.Encode()
	return u.String()
}

func (b *Bot) keyboard(rows [][]Button) map[string]any {
	kb := make([][]map[string]any, 0, len(rows))
	for _, row := range rows {
		r := make([]map[string]any, 0, len(row))
		for _, btn := range row {
			switch {
			case btn.Data != "":
				r = append(r, map[string]any{"text": btn.Text, "callback_data": btn.Data})
			case btn.URL != "":
				r = append(r, map[string]any{"text": btn.Text, "url": btn.URL})
			default:
				r = append(r, map[string]any{"text": btn.Text, "web_app": map[string]string{"url": b.AppURL(btn.Open)}})
			}
		}
		kb = append(kb, r)
	}
	return map[string]any{"inline_keyboard": kb}
}

// Send — текстовое сообщение с кнопками (nil — без кнопок).
func (b *Bot) Send(ctx context.Context, chatID int64, text string, buttons [][]Button) (Sent, error) {
	p := map[string]any{"chat_id": chatID, "text": text}
	if len(buttons) > 0 {
		p["reply_markup"] = b.keyboard(buttons)
	}
	var m Message
	err := b.call(ctx, "sendMessage", p, &m)
	return Sent{chatID, m.MessageID}, err
}

// SendPhoto — картинка с подписью (подпись Telegram режет до 1024 знаков).
func (b *Bot) SendPhoto(ctx context.Context, chatID int64, photo Photo, caption string, buttons [][]Button) (Sent, error) {
	if r := []rune(caption); len(r) > 1000 {
		caption = string(r[:1000]) + "…"
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	mw.WriteField("caption", caption)
	if len(buttons) > 0 {
		kb, _ := json.Marshal(b.keyboard(buttons))
		mw.WriteField("reply_markup", string(kb))
	}
	ext := ".jpg"
	if photo.MIME == "image/png" {
		ext = ".png"
	}
	fw, err := mw.CreateFormFile("photo", "photo"+ext)
	if err != nil {
		return Sent{}, err
	}
	fw.Write(photo.Data)
	mw.Close()
	var m Message
	err = b.do(ctx, "sendPhoto", mw.FormDataContentType(), &body, &m)
	return Sent{chatID, m.MessageID}, err
}

// SendDocument — файл в чат (выгрузка CSV для владельца).
func (b *Bot) SendDocument(ctx context.Context, chatID int64, name string, data []byte, caption string) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	mw.WriteField("caption", caption)
	fw, err := mw.CreateFormFile("document", name)
	if err != nil {
		return err
	}
	fw.Write(data)
	mw.Close()
	return b.do(ctx, "sendDocument", mw.FormDataContentType(), &body, nil)
}

// ToAdmins — сообщение владельцу и помощникам; возвращает отправленные (для ответов reply).
func (b *Bot) ToAdmins(ctx context.Context, text string, photo *Photo, buttons [][]Button) ([]Sent, error) {
	var sent []Sent
	var errs []error
	for _, id := range b.Admins {
		var s Sent
		var err error
		if photo != nil {
			s, err = b.SendPhoto(ctx, id, *photo, text, buttons)
		} else {
			s, err = b.Send(ctx, id, text, buttons)
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		sent = append(sent, s)
	}
	return sent, errors.Join(errs...)
}

// ToUser — ученику текст key на его языке; open — кнопка, открывающая приложение на нужном экране.
func (b *Bot) ToUser(ctx context.Context, tgID int64, lang, key string, args map[string]string, open string) error {
	var buttons [][]Button
	if open != "-" {
		buttons = [][]Button{{{Text: T(lang, "btn.open"), Open: open}}}
	}
	_, err := b.Send(ctx, tgID, Format(T(lang, key), args), buttons)
	return err
}

// AnswerCallback убирает «часики» на нажатой кнопке; text — всплывающая подсказка.
func (b *Bot) AnswerCallback(ctx context.Context, id, text string) error {
	return b.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": id, "text": text}, nil)
}

// SetButtons меняет кнопки под сообщением (nil — убрать).
func (b *Bot) SetButtons(ctx context.Context, chatID, messageID int64, buttons [][]Button) error {
	kb := b.keyboard(buttons)
	return b.call(ctx, "editMessageReplyMarkup", map[string]any{"chat_id": chatID, "message_id": messageID, "reply_markup": kb}, nil)
}

// Download скачивает файл из Telegram (голосовое, фото), не больше max байт.
func (b *Bot) Download(ctx context.Context, fileID string, max int64) ([]byte, error) {
	var f struct {
		FilePath string `json:"file_path"`
		FileSize int64  `json:"file_size"`
	}
	if err := b.call(ctx, "getFile", map[string]any{"file_id": fileID}, &f); err != nil {
		return nil, err
	}
	if f.FileSize > max {
		return nil, errors.New("telegram: файл слишком большой")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.apiBase+"/file/bot"+b.token+"/"+f.FilePath, nil)
	if err != nil {
		return nil, errors.New("telegram file: запрос не собран")
	}
	res, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram file: %w", stripURL(err))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram file: ответ %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, max+1))
	if err != nil {
		return nil, fmt.Errorf("telegram file: %w", stripURL(err))
	}
	if int64(len(data)) > max {
		return nil, errors.New("telegram: файл слишком большой")
	}
	return data, nil
}

// WritingChecked сообщает ученику итог проверки Writing (band == nil — проверить не удалось) и даёт
// кнопку, которая открывает разбор: ?check=<id> читает App.svelte.
func (b *Bot) WritingChecked(ctx context.Context, chatID int64, lang string, checkID int64, band *float64) error {
	u, err := url.Parse(b.webAppURL)
	if err != nil {
		return fmt.Errorf("WEBAPP_URL: %w", err)
	}
	q := u.Query()
	q.Set("check", strconv.FormatInt(checkID, 10))
	u.RawQuery = q.Encode()
	text, button := writingText(lang, band)
	return b.call(ctx, "sendMessage", map[string]any{
		"chat_id": chatID,
		"text":    text,
		"reply_markup": map[string]any{
			"inline_keyboard": [][]map[string]any{{{"text": button, "web_app": map[string]string{"url": u.String()}}}},
		},
	}, nil)
}

func writingText(lang string, band *float64) (text, button string) {
	if band == nil {
		return T(lang, "writing.failed"), T(lang, "btn.open")
	}
	return Format(T(lang, "writing.done"), map[string]string{"band": strconv.FormatFloat(*band, 'f', 1, 64)}), T(lang, "btn.feedback")
}

// SpeakingChecked — итог проверки Speaking (band nil — не удалось): кнопка открывает разбор.
func (b *Bot) SpeakingChecked(ctx context.Context, chatID int64, lang string, checkID int64, band *float64) error {
	key, args := "speaking.failed", map[string]string(nil)
	if band != nil {
		key, args = "speaking.done", map[string]string{"band": strconv.FormatFloat(*band, 'f', 1, 64)}
	}
	return b.ToUser(ctx, chatID, lang, key, args, "speaking-"+strconv.FormatInt(checkID, 10))
}
