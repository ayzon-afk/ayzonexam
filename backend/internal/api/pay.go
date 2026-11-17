package api

import (
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ayzonexam/internal/billing"
	"ayzonexam/internal/users"
)

// Страница оплаты (этап 6, SPEC «Как ученик платит»): «Перейти на Premium» открывает её в браузере
// телефона; кто платит, сервер знает по подписанной ссылке — отдельный вход не нужен. Страница — простой
// HTML без интерфейса приложения: открывается быстро и работает в любом браузере.

const payLinkTTL = 24 * time.Hour

func (s *server) payKey() []byte {
	k := sha256.Sum256([]byte("ayzon-pay:" + s.BotToken))
	return k[:]
}

// payToken — «id.срок.подпись»: ссылка годится только этому ученику и только сутки.
func (s *server) payToken(userID int64, now time.Time) string {
	payload := strconv.FormatInt(userID, 10) + "." + strconv.FormatInt(now.Add(payLinkTTL).Unix(), 10)
	m := hmac.New(sha256.New, s.payKey())
	m.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *server) payUser(r *http.Request, token string) (users.User, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return users.User{}, errors.New("bad token")
	}
	m := hmac.New(sha256.New, s.payKey())
	m.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, m.Sum(nil)) {
		return users.User{}, errors.New("bad token")
	}
	exp, _ := strconv.ParseInt(parts[1], 10, 64)
	if s.Now().Unix() > exp {
		return users.User{}, errors.New("expired")
	}
	id, _ := strconv.ParseInt(parts[0], 10, 64)
	u, err := s.Users.Get(r.Context(), id)
	if err == nil && u.BlockedAt != nil {
		return u, errors.New("blocked")
	}
	return u, err
}

func (s *server) payLink(w http.ResponseWriter, r *http.Request, u users.User) {
	writeJSON(w, http.StatusOK, map[string]string{"path": "/pay?t=" + s.payToken(u.ID, s.Now())})
}

//go:embed pay.html
var payHTML string

var payPage = template.Must(template.New("pay").Parse(payHTML))

var payTexts = map[string]map[string]string{
	"uz": {
		"title": "Premium uchun to‘lov", "plan": "Tarif", "month": "1 oy", "quarter": "3 oy", "mock": "To‘liq mock (1 ta)",
		"discount": "Birinchi oy uchun chegirma", "card": "Karta raqami", "holder": "Qabul qiluvchi", "code": "To‘lov kodi",
		"codeHint": "Kodni o‘tkazma izohiga yozing — shu bo‘yicha to‘lovingizni topamiz.", "copy": "Nusxa olish", "copied": "Nusxalandi",
		"steps": "1. Tarifni tanlang. 2. Summani kartaga o‘tkazing, izohga kodni yozing. 3. Chek skrinshotini biriktiring va «To‘ladim»ni bosing.",
		"shot":  "Chek skrinshoti", "submit": "To‘ladim", "sum": "so‘m",
		"instant": "Premium darhol ochiladi — «tekshiruvda» belgisi bilan. Egasi to‘lovni ko‘rgach, tasdiqlaydi.",
		"later":   "Premium egasi to‘lovni tasdiqlagandan keyin ochiladi.",
		"pending": "Sizda tekshiruvdagi to‘lov bor. Tasdiq kelguncha yangi to‘lov qilib bo‘lmaydi.",
		"done":    "Rahmat! To‘lov tekshiruvga yuborildi. Ilovaga qayting.", "doneLater": "Rahmat! To‘lov tekshiruvga yuborildi — tasdiqlangach Premium ochiladi.",
		"expired": "Havola eskirgan. Ilovada «Premiumga o‘tish»ni qayta bosing.", "noCard": "To‘lov hozircha mavjud emas. Keyinroq urinib ko‘ring.",
		"error": "Nimadir xato ketdi. Qayta urinib ko‘ring.", "badShot": "Skrinshot rasm bo‘lishi kerak (JPG yoki PNG, 1.5 MB gacha).",
		"back": "Ilovaga qaytish",
	},
	"ru": {
		"title": "Оплата Premium", "plan": "Тариф", "month": "1 месяц", "quarter": "3 месяца", "mock": "Полный mock (1 шт.)",
		"discount": "Скидка на первый месяц", "card": "Номер карты", "holder": "Получатель", "code": "Код платежа",
		"codeHint": "Впишите код в комментарий к переводу — по нему мы найдём ваш платёж.", "copy": "Копировать", "copied": "Скопировано",
		"steps": "1. Выберите тариф. 2. Переведите сумму на карту, в комментарии — код. 3. Приложите скрин чека и нажмите «Я оплатил».",
		"shot":  "Скрин чека", "submit": "Я оплатил", "sum": "сум",
		"instant": "Premium откроется сразу — с пометкой «на проверке». Владелец подтвердит, когда увидит перевод.",
		"later":   "Premium откроется после того, как владелец подтвердит перевод.",
		"pending": "У вас уже есть оплата на проверке. Новую можно отправить после ответа.",
		"done":    "Спасибо! Оплата отправлена на проверку. Вернитесь в приложение.", "doneLater": "Спасибо! Оплата отправлена на проверку — Premium откроется после подтверждения.",
		"expired": "Ссылка устарела. Нажмите «Перейти на Premium» в приложении ещё раз.", "noCard": "Оплата пока недоступна. Попробуйте позже.",
		"error": "Что-то пошло не так. Попробуйте ещё раз.", "badShot": "Скрин должен быть картинкой (JPG или PNG, до 1.5 МБ).",
		"back": "Вернуться в приложение",
	},
	"en": {
		"title": "Premium payment", "plan": "Plan", "month": "1 month", "quarter": "3 months", "mock": "Full mock (1)",
		"discount": "First-month discount", "card": "Card number", "holder": "Recipient", "code": "Payment code",
		"codeHint": "Put the code in the transfer comment — that's how we find your payment.", "copy": "Copy", "copied": "Copied",
		"steps": "1. Choose a plan. 2. Transfer the amount to the card with the code in the comment. 3. Attach a screenshot of the receipt and tap “I've paid”.",
		"shot":  "Receipt screenshot", "submit": "I've paid", "sum": "UZS",
		"instant": "Premium opens right away, marked “under review”. The owner confirms it once the transfer is found.",
		"later":   "Premium opens once the owner confirms the transfer.",
		"pending": "You already have a payment under review. You can send a new one after it's checked.",
		"done":    "Thank you! Your payment is sent for review. Go back to the app.", "doneLater": "Thank you! Your payment is sent for review — Premium opens after confirmation.",
		"expired": "This link has expired. Tap “Go Premium” in the app again.", "noCard": "Payment isn't available yet. Please try later.",
		"error": "Something went wrong. Please try again.", "badShot": "The screenshot must be an image (JPG or PNG, up to 1.5 MB).",
		"back": "Back to the app",
	},
}

type payPlan struct {
	ID      string
	Name    string
	Amount  string
	Checked bool
}

type payData struct {
	Lang     string
	T        map[string]string
	State    string // form, done, doneLater, expired, noCard, pending, error
	Error    string
	Plans    []payPlan
	Discount bool
	Card     string
	Holder   string
	Code     string
	Instant  bool
	Token    string
	App      string
}

func money(n int) string {
	s := strconv.Itoa(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func (s *server) payRender(w http.ResponseWriter, d payData, status int) {
	if d.T == nil {
		d.T = payTexts["uz"]
		d.Lang = "uz"
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src blob: data:; form-action 'self'; base-uri 'none'")
	w.WriteHeader(status)
	if err := payPage.Execute(w, d); err != nil {
		s.Log.Error("страница оплаты", "err", err)
	}
}

// payPageGet — страница оплаты: тарифы, карта, код, форма со скрином.
func (s *server) payPageGet(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	u, err := s.payUser(r, token)
	if err != nil {
		s.payRender(w, payData{State: "expired"}, http.StatusForbidden)
		return
	}
	d, err := s.payForm(r, u, token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.payRender(w, d, http.StatusOK)
}

func (s *server) payForm(r *http.Request, u users.User, token string) (payData, error) {
	ctx := r.Context()
	d := payData{Lang: u.Lang, T: payTexts[u.Lang], State: "form", Card: s.PayCard, Holder: s.PayCardName, Token: token}
	if s.BotUsername != "" {
		d.App = "https://t.me/" + s.BotUsername
	}
	if d.T == nil {
		d.Lang, d.T = "uz", payTexts["uz"]
	}
	if s.PayCard == "" {
		d.State = "noCard"
		return d, nil
	}
	pending, err := s.billing.Pending(ctx, u.ID)
	if err != nil {
		return d, err
	}
	if pending != nil {
		d.State = "pending"
		return d, nil
	}
	paid, err := s.billing.PaidBefore(ctx, u.ID)
	if err != nil {
		return d, err
	}
	d.Discount = billing.DiscountUntil(u, paid, s.Now()) != nil
	if d.Code, err = s.billing.PayCode(ctx, u.ID); err != nil {
		return d, err
	}
	d.Instant = u.PayTrust != "rejected"
	for _, p := range []string{"month", "quarter", "mock"} {
		amount, _, _, _ := billing.Price(p, d.Discount)
		d.Plans = append(d.Plans, payPlan{ID: p, Name: d.T[p], Amount: money(amount), Checked: p == "month"})
	}
	return d, nil
}

// payPagePost — «Я оплатил»: оплата на проверку, Premium сразу (кроме тех, кому уже отказывали),
// владельцу — скрин с кнопками «Подтвердить» и «Отказать».
func (s *server) payPagePost(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	u, err := s.payUser(r, token)
	if err != nil {
		s.payRender(w, payData{State: "expired"}, http.StatusForbidden)
		return
	}
	d, err := s.payForm(r, u, token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if d.State != "form" {
		s.payRender(w, d, http.StatusConflict)
		return
	}
	if code := parseForm(w, r, maxImageBytes+64<<10); code != "" {
		d.Error = d.T["badShot"]
		s.payRender(w, d, http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()
	plan := first(r.MultipartForm.Value["plan"])
	fhs := r.MultipartForm.File["screenshot"]
	if len(fhs) != 1 {
		d.Error = d.T["badShot"]
		s.payRender(w, d, http.StatusBadRequest)
		return
	}
	img, code := readImage(fhs[0])
	if code != "" {
		d.Error = d.T["badShot"]
		s.payRender(w, d, http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	p, err := s.billing.Submit(ctx, u.ID, plan, img.MIME, img.Data, s.Now())
	if errors.Is(err, billing.ErrPending) {
		d.State = "pending"
		s.payRender(w, d, http.StatusConflict)
		return
	}
	if errors.Is(err, billing.ErrPlan) {
		d.Error = d.T["error"]
		s.payRender(w, d, http.StatusBadRequest)
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	opened := "открыт сразу, «на проверке»"
	if p.GrantedDays == 0 && p.GrantedMocks == 0 {
		opened = "НЕ открыт: ученику уже отказывали — откроется после подтверждения"
	}
	s.toAdmins(ctx, fmt.Sprintf("💳 Оплата #%d\n%s · %s · %s сум\nКод: %s\nДоступ %s.",
		p.ID, u.Nickname, plan, money(p.Amount), p.Code, opened), &Photo{MIME: img.MIME, Data: img.Data}, payButtons(p.ID), "payment", p.ID)
	d.State = "done"
	if !d.Instant {
		d.State = "doneLater"
	}
	s.payRender(w, d, http.StatusOK)
}
