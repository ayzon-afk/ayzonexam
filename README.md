# AYZON EXAM

Telegram Mini App для подготовки к IELTS-style экзаменам. Продукт — [SPEC.md](SPEC.md), этапы —
[ROADMAP.md](ROADMAP.md).

## Что нужно на компьютере

- Go 1.27+ (`go version`)
- Node.js 24+ (`node -v`)

Docker и установленный PostgreSQL не нужны: база для разработки и для тестов запускается встроенной.

## Первый запуск

```bash
npm --prefix frontend install
cp backend/.env.example backend/.env
```

В трёх окнах терминала из корня проекта:

```bash
npm run dev:db       # база на localhost:5433, остановка — Ctrl+C
npm run dev:server   # сервер на localhost:8080
npm run dev:web      # интерфейс: адрес покажет Vite
```

В обычном браузере вход работает без Telegram (`DEV_AUTH=1` в `.env`). Другой ученик —
`?uid=2` в адресе, другой язык Telegram — `?lang=en`.

## Проверка Writing (Gemini)

Ключ — в [Google AI Studio](https://aistudio.google.com/apikey), в `backend/.env`:
`GEMINI_API_KEY=...`. Без ключа проверки идут заглушкой (`ai.Fake`) — так можно только локально.

Калибровка: эссе с баллами учителей прогоняются через Gemini, видно расхождение. Формат файлов — в
начале `backend/cmd/calibrate/main.go`.

```bash
npm run calibrate -- -dir "C:/путь/к/эссе"            # таблица «учитель / AI / разница»
npm run calibrate -- -list-models                      # какие модели доступны ключу
```

Бесплатный тариф Gemini — 5 запросов в минуту: калибровка сама ждёт и повторяет.

## Проверка на телефоне в Telegram

Telegram открывает Mini App только по HTTPS, поэтому нужен туннель к компьютеру.

1. Бот в [@BotFather](https://t.me/BotFather): `/newbot`, токен — в `BOT_TOKEN` в `backend/.env`.
2. Собрать интерфейс: `npm --prefix frontend run build`.
3. Туннель (без регистрации): `cloudflared tunnel --url http://localhost:8080` — он покажет адрес
   `https://….trycloudflare.com`.
4. В `backend/.env`: `BOT_MODE=polling`, `WEBAPP_URL=<адрес туннеля>`, `WEB_DIR=../frontend/dist`,
   `DEV_AUTH=0`. Перезапустить `npm run dev:server`.
5. В Telegram: бот → `/start` → «Ochish».

## Проверка проекта

```bash
npm test       # тесты сервера и интерфейса, сборка с бюджетом веса
npm run lint   # форматирование и типы
```
