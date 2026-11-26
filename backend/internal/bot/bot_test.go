package bot

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeBot — выдуманный бот для тестов.
const fakeBot = "555:TEST"

type call struct {
	method string
	body   map[string]any
}

// fakeTelegram записывает вызовы Bot API и отвечает ok.
func fakeTelegram(t *testing.T) (*Bot, *[]call) {
	t.Helper()
	var mu sync.Mutex
	calls := &[]call{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		*calls = append(*calls, call{strings.TrimPrefix(r.URL.Path, "/bot"+fakeBot+"/"), body})
		mu.Unlock()
		io.WriteString(w, `{"ok":true,"result":{"message_id":7,"username":"ayzon_test_bot"}}`)
	}))
	t.Cleanup(srv.Close)
	b := New(fakeBot, "https://exam.example/app", slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.apiBase = srv.URL
	return b, calls
}

func msg(chatType, text, lang string) Update {
	var u Update
	json.Unmarshal([]byte(`{"update_id":1,"message":{"chat":{"id":10,"type":"`+chatType+`"},"from":{"language_code":"`+lang+`"},"text":"`+text+`"}}`), &u)
	return u
}

func TestStartSendsWebAppButton(t *testing.T) {
	b, calls := fakeTelegram(t)
	b.Handle(context.Background(), msg("private", "/start", "ru"))
	if len(*calls) != 1 || (*calls)[0].method != "sendMessage" {
		t.Fatalf("вызовы: %+v", *calls)
	}
	body := (*calls)[0].body
	raw, _ := json.Marshal(body["reply_markup"])
	if !strings.Contains(string(raw), `"web_app":{"url":"https://exam.example/app"}`) || !strings.Contains(body["text"].(string), "Привет") {
		t.Fatalf("неверное сообщение: %v", body)
	}
}

func TestDefaultLanguageIsUzbek(t *testing.T) {
	b, calls := fakeTelegram(t)
	b.Handle(context.Background(), msg("private", "/start", "de"))
	if !strings.Contains((*calls)[0].body["text"].(string), "Salom") {
		t.Fatalf("язык по умолчанию должен быть узбекский: %v", (*calls)[0].body["text"])
	}
}

func TestGroupsIgnored(t *testing.T) {
	b, calls := fakeTelegram(t)
	b.Handle(context.Background(), msg("group", "/start", "ru"))
	b.Handle(context.Background(), Update{UpdateID: 2})
	if len(*calls) != 0 {
		t.Fatalf("в группах и без сообщения бот молчит: %+v", *calls)
	}
}

func TestWebhookSecret(t *testing.T) {
	b, calls := fakeTelegram(t)
	h := b.WebhookHandler("right-secret-right-secret-right-secret")
	body := `{"update_id":1,"message":{"chat":{"id":10,"type":"private"},"text":"/start"}}`

	req := httptest.NewRequest("POST", "/tg/webhook", strings.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 401 || len(*calls) != 0 {
		t.Fatalf("чужой запрос прошёл: %d, вызовов %d", rec.Code, len(*calls))
	}

	req = httptest.NewRequest("POST", "/tg/webhook", strings.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "right-secret-right-secret-right-secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || len(*calls) != 1 {
		t.Fatalf("настоящий запрос не обработан: %d, вызовов %d", rec.Code, len(*calls))
	}
}

func TestErrorsHideToken(t *testing.T) {
	b := New(fakeBot, "https://exam.example/app", slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.apiBase = "http://127.0.0.1:1" // никто не слушает
	err := b.call(context.Background(), "getMe", map[string]any{}, nil)
	if err == nil || strings.Contains(err.Error(), fakeBot) {
		t.Fatalf("токен попал в текст ошибки: %v", err)
	}
}

func TestSetupModes(t *testing.T) {
	b, calls := fakeTelegram(t)
	if err := b.Setup(context.Background(), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := b.Setup(context.Background(), "https://exam.example/tg/webhook", "s3cret-s3cret-s3cret-s3cret-s3cret"); err != nil {
		t.Fatal(err)
	}
	var methods []string
	for _, c := range *calls {
		methods = append(methods, c.method)
	}
	setup := "getMe setChatMenuButton setMyCommands setMyCommands setMyCommands"
	want := setup + " deleteWebhook " + setup + " setWebhook"
	if strings.Join(methods, " ") != want {
		t.Fatalf("вызовы %v, ждали %s", methods, want)
	}
	last := (*calls)[len(*calls)-1].body
	if last["secret_token"] != "s3cret-s3cret-s3cret-s3cret-s3cret" {
		t.Fatalf("webhook без секрета: %v", last)
	}
	if raw, _ := json.Marshal(last["allowed_updates"]); string(raw) != `["message","callback_query"]` {
		t.Fatalf("бот должен получать и нажатия кнопок: %s", raw)
	}
	if b.Username() != "ayzon_test_bot" {
		t.Fatalf("username из getMe: %q", b.Username())
	}
}

func TestWritingChecked(t *testing.T) {
	b, calls := fakeTelegram(t)
	band := 6.5
	if err := b.WritingChecked(context.Background(), 42, "ru", 17, &band); err != nil {
		t.Fatal(err)
	}
	body := (*calls)[0].body
	raw, _ := json.Marshal(body["reply_markup"])
	if body["chat_id"] != float64(42) || !strings.Contains(body["text"].(string), "6.5") ||
		!strings.Contains(string(raw), `"url":"https://exam.example/app?check=17"`) {
		t.Fatalf("сообщение о проверке: %v %s", body, raw)
	}

	if err := b.WritingChecked(context.Background(), 42, "uz", 18, nil); err != nil {
		t.Fatal(err)
	}
	if text := (*calls)[1].body["text"].(string); !strings.Contains(text, "limitga kirmaydi") {
		t.Fatalf("неудача на узбекском: %q", text)
	}
}

func TestStartParamOpensScreen(t *testing.T) {
	b, calls := fakeTelegram(t)
	b.Handle(context.Background(), msg("private", "/start d-abc123", "ru"))
	body := (*calls)[0].body
	raw, _ := json.Marshal(body["reply_markup"])
	if !strings.Contains(string(raw), `https://exam.example/app?open=d-abc123`) || !strings.Contains(body["text"].(string), "дуэль") {
		t.Fatalf("ссылка-приглашение на дуэль: %v %s", body, raw)
	}
	// Кнопка сайта «Я учитель» — анкета в приложении.
	b.Handle(context.Background(), msg("private", "/start apply-teacher", "ru"))
	body = (*calls)[1].body
	raw, _ = json.Marshal(body["reply_markup"])
	if !strings.Contains(string(raw), `open=apply-teacher`) || !strings.Contains(body["text"].(string), "анкету") {
		t.Fatalf("заявка учителя: %v %s", body, raw)
	}
}

func TestOnUpdateTakesOver(t *testing.T) {
	b, calls := fakeTelegram(t)
	var got []Update
	b.OnUpdate = func(_ context.Context, u Update) bool {
		got = append(got, u)
		return u.Message != nil && u.Message.Text == "/help"
	}
	b.Handle(context.Background(), msg("private", "/help", "ru"))
	if len(*calls) != 0 || len(got) != 1 {
		t.Fatalf("/help обрабатывает OnUpdate, бот молчит: %+v", *calls)
	}
	b.Handle(context.Background(), msg("private", "привет", "ru"))
	if len(*calls) != 1 {
		t.Fatalf("необработанное — кнопка запуска: %+v", *calls)
	}
	var cb Update
	json.Unmarshal([]byte(`{"update_id":3,"callback_query":{"id":"q","from":{"id":1},"data":"pay:ok:5"}}`), &cb)
	b.Handle(context.Background(), cb)
	if len(got) != 3 || got[2].CallbackQuery.Data != "pay:ok:5" {
		t.Fatalf("нажатие кнопки не дошло до OnUpdate: %+v", got)
	}
}

func TestButtonsAndAdmins(t *testing.T) {
	b, calls := fakeTelegram(t)
	b.Admins = []int64{1, 2}
	sent, err := b.ToAdmins(context.Background(), "оплата", nil, [][]Button{{{Text: "ok", Data: "pay:ok:1"}, {Text: "app", Open: "premium"}}})
	if err != nil || len(sent) != 2 || sent[0].MessageID != 7 {
		t.Fatalf("сообщения админам: %+v %v", sent, err)
	}
	raw, _ := json.Marshal((*calls)[0].body["reply_markup"])
	if !strings.Contains(string(raw), `"callback_data":"pay:ok:1"`) || !strings.Contains(string(raw), `app?open=premium`) {
		t.Fatalf("кнопки: %s", raw)
	}
	if err := b.ToUser(context.Background(), 5, "en", "duel.result", map[string]string{"nick": "ali", "score": "3:2"}, "-"); err != nil {
		t.Fatal(err)
	}
	last := (*calls)[len(*calls)-1].body
	if last["text"] != "Your duel with ali is over: 3:2." || last["reply_markup"] != nil {
		t.Fatalf("сообщение ученику без кнопки: %v", last)
	}
}

func TestTextsComplete(t *testing.T) {
	for k, v := range texts {
		for i, s := range v {
			if strings.TrimSpace(s) == "" {
				t.Errorf("%s: пустой текст на языке %d", k, i)
			}
		}
	}
	if T("de", "btn.open") != "Ochish" {
		t.Fatal("неизвестный язык — узбекский")
	}
}
