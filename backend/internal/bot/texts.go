package bot

import "strings"

// Тексты бота: узбекский — основной (SPEC), ru и en — по языку Telegram ученика.
var texts = map[string][3]string{ // uz, ru, en
	"cmd.start":    {"Ilovani ochish", "Открыть приложение", "Open the app"},
	"cmd.help":     {"Yordam — savol yozish", "Поддержка — написать вопрос", "Support — ask a question"},
	"btn.open":     {"Ochish", "Открыть", "Open"},
	"btn.feedback": {"Tahlilni ochish", "Открыть разбор", "Open feedback"},

	"greet": {"Hammasi ilova ichida — pastdagi tugma orqali oching. Savol bo‘lsa — /help.",
		"Всё происходит в приложении — открой его кнопкой ниже. Есть вопрос — /help.",
		"Everything happens in the app — open it with the button below. Questions — /help."},
	"greet.start": {"Salom! Bu AYZON EXAM — Writing va Speaking AI tekshiruvi bilan IELTS-style mock. Boshlash uchun tugmani bosing.",
		"Привет! Это AYZON EXAM — IELTS-style mock с AI-проверкой Writing и Speaking. Жми кнопку, чтобы начать.",
		"Hi! This is AYZON EXAM — IELTS-style mocks with AI feedback on Writing and Speaking. Tap the button to start."},
	"greet.duel": {"Sizni duelga chaqirishdi! 5 daqiqa, so‘zlar. Boshlash uchun tugmani bosing.",
		"Тебя вызвали на дуэль! 5 минут, слова. Жми кнопку, чтобы начать.",
		"You've been challenged to a duel! 5 minutes, words. Tap the button to start."},

	"writing.done": {"Insho tekshirildi: {band} ball. Xatolar tahlili va tuzatilgan variant — ilovada.",
		"Эссе проверено: {band}. Разбор ошибок и исправленная версия — в приложении.",
		"Your essay is checked: band {band}. The feedback and corrected version are in the app."},
	"writing.failed": {"Inshoni tekshirib bo‘lmadi. Bu urinish limitga kirmaydi — qayta yuboring.",
		"Не получилось проверить эссе. В лимит эта попытка не засчитана — отправьте ещё раз.",
		"We couldn't check your essay. This attempt doesn't count towards your limit — please send it again."},
	"speaking.done": {"Speaking baholandi: {band} ball. Tahlil — ilovada.",
		"Speaking оценён: {band}. Разбор — в приложении.",
		"Your Speaking is assessed: band {band}. The feedback is in the app."},
	"speaking.failed": {"Javobni baholab bo‘lmadi. Bu urinish limitga kirmaydi — qayta yozib yuboring.",
		"Не получилось оценить ответ. В лимит попытка не засчитана — запишите ещё раз.",
		"We couldn't assess your answer. It doesn't count towards your limit — please record it again."},
	"speaking.ask": {"Savolga ovozli xabar bilan javob bering (2 daqiqagacha):\n\n{q}",
		"Ответьте на вопрос голосовым сообщением (до 2 минут):\n\n{q}",
		"Answer the question with a voice message (up to 2 minutes):\n\n{q}"},
	"speaking.voiceonly": {"Ovozli xabar kutilmoqda. Bekor qilish — /cancel.",
		"Жду голосовое сообщение. Отменить — /cancel.",
		"Waiting for a voice message. Cancel — /cancel."},
	"speaking.received": {"Qabul qilindi! Baholash odatda bir daqiqagacha davom etadi.",
		"Принято! Оценка обычно занимает до минуты.",
		"Received! Assessment usually takes up to a minute."},
	"speaking.limit": {"Speaking limiti tugadi. Premium bilan — kuniga 3 ta.",
		"Лимит Speaking исчерпан. С Premium — до 3 в день.",
		"Your Speaking limit is used up. With Premium — up to 3 a day."},
	"speaking.toolong": {"Javob juda uzun: 4 daqiqagacha bo‘lsin.",
		"Слишком длинный ответ: не больше 4 минут.",
		"The answer is too long: up to 4 minutes, please."},
	"cancelled": {"Bekor qilindi.", "Отменено.", "Cancelled."},

	"help.ask": {"Savolingizni bitta xabar bilan yozing (rasm ham bo‘ladi) — javob shu yerga keladi.",
		"Напишите вопрос одним сообщением (можно с фото) — ответ придёт сюда.",
		"Write your question in one message (a photo is fine too) — the answer will come here."},
	"help.sent": {"Rahmat! Xabaringiz yuborildi, tez orada javob beramiz.",
		"Спасибо! Сообщение отправлено, скоро ответим.",
		"Thank you! Your message is sent, we'll reply soon."},
	"support.reply": {"Qo‘llab-quvvatlash javobi:\n\n{text}\n\nYana savol bo‘lsa — /help.",
		"Ответ поддержки:\n\n{text}\n\nЕщё вопрос — /help.",
		"Support reply:\n\n{text}\n\nAnother question — /help."},

	"bonus.level": {"Do‘stingiz {nick} daraja testini topshirdi — sizga +1 bonus tekshiruv!",
		"Ваш друг {nick} прошёл тест уровня — вам +1 бонусная проверка!",
		"Your friend {nick} took the level test — you get +1 bonus check!"},
	"bonus.premium": {"Do‘stingiz Premium sotib oldi — sizga +7 kun Premium!",
		"Ваш друг купил Premium — вам +7 дней Premium!",
		"Your friend bought Premium — you get +7 days of Premium!"},
	"friend.joined": {"{nick} sizning havolangiz orqali qo‘shildi — endi siz do‘stsiz.",
		"{nick} пришёл по вашей ссылке — теперь вы друзья.",
		"{nick} joined with your link — you're friends now."},
	"center.request": {"Markazga yangi ariza bor — ilovada tasdiqlang.",
		"Новая заявка в центр — подтвердите в приложении.",
		"A new request to join your center — confirm it in the app."},
	"duel.challenge": {"{nick} sizni duelga chaqirdi! 5 daqiqa, so‘zlar.",
		"{nick} вызывает тебя на дуэль! 5 минут, слова.",
		"{nick} challenges you to a duel! 5 minutes, words."},
	"duel.accepted": {"{nick} chaqiruvingizni qabul qildi! Natija ikkalangiz o‘ynab bo‘lgach chiqadi.",
		"{nick} принял вызов! Итог — когда сыграете оба.",
		"{nick} accepted your challenge! The result comes when you've both played."},
	"duel.result": {"{nick} bilan duel tugadi: {score}.",
		"Дуэль с {nick} закончилась: {score}.",
		"Your duel with {nick} is over: {score}."},
	"mock.done": {"To‘liq mock tayyor: Overall Band {band}. Hisobot va 2 haftalik reja — ilovada.",
		"Полный mock готов: Overall Band {band}. Отчёт и план на 2 недели — в приложении.",
		"Your full mock is ready: Overall Band {band}. The report and 2-week plan are in the app."},

	"pay.confirmed": {"To‘lov tasdiqlandi — rahmat! Premium faol.",
		"Оплата подтверждена — спасибо! Premium активен.",
		"Payment confirmed — thank you! Premium is active."},
	"pay.confirmed.mock": {"To‘lov tasdiqlandi — to‘liq mock ochildi.",
		"Оплата подтверждена — полный mock открыт.",
		"Payment confirmed — your full mock is unlocked."},
	"pay.rejected": {"To‘lov tasdiqlanmadi: {reason}. Premium o‘chirildi. Xato bo‘lsa — /help.",
		"Оплата не подтверждена: {reason}. Premium выключен. Если это ошибка — /help.",
		"Payment not confirmed: {reason}. Premium is turned off. If this is a mistake — /help."},
	"pay.reason.notfound":   {"o‘tkazma topilmadi", "перевод не найден", "the transfer wasn't found"},
	"pay.reason.amount":     {"summa mos emas", "сумма не совпадает", "the amount doesn't match"},
	"pay.reason.screenshot": {"skrinshot to‘g‘ri kelmaydi", "скрин не подходит", "the screenshot isn't valid"},
}

func langIndex(lang string) int {
	switch lang {
	case "ru":
		return 1
	case "en":
		return 2
	}
	return 0
}

// T — текст key на языке lang; неизвестный язык — узбекский.
func T(lang, key string) string {
	t, ok := texts[key]
	if !ok {
		return key
	}
	return t[langIndex(lang)]
}

// Format подставляет {name} из args.
func Format(s string, args map[string]string) string {
	for k, v := range args {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}
