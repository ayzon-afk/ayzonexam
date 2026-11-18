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

## Админка и оплата

В `backend/.env`: `ADMIN_IDS=<свой Telegram id>` (узнать — у @userinfobot; помощники — через запятую).
Тогда в «Профиле» появится «Админка», а боту можно писать `/post` и отвечать на обращения reply. Локально
с `DEV_AUTH=1` вход — тестовый ученик `?uid=…`: чтобы зайти админом, его id (1000001 без `?uid`) тоже
впишите в `ADMIN_IDS`.

Оплата Premium: `PAY_CARD` и `PAY_CARD_NAME` — карта для переводов. Страница оплаты — `/pay` на том же
сервере; без карты она пишет «оплата недоступна».

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

То же проверяет GitHub на каждый коммит (`.github/workflows/ci.yml`). После зелёной проверки на `main`
GitHub собирает образы для сервера: `ghcr.io/ayzon-afk/ayzon-exam` и `…-backup` (`release.yml`).

## Сервер

Файлы — в `deploy/`. Сервер общий с ayzonstock: HTTPS даёт его Caddy, приложение подключается к его сети
`ayzon-web`. Всего три контейнера: база, приложение, служба копий и наблюдения. Лимит памяти на всё —
512 МБ.

**Первый запуск** (на сервере, в папке `deploy` этого репозитория):

1. Поддомен (например `exam.ayzon.uz`), запись A — на IP сервера.
2. `./make-env.sh exam.ayzon.uz`, затем вписать в `deploy/.env` `BOT_TOKEN`, `ADMIN_IDS`,
   `GEMINI_API_KEY`, `PAY_CARD`, `PAY_CARD_NAME`, а также `R2_*` — для копий вне сервера. Копию
   `deploy/.env` сохранить у себя.
3. Войти в реестр GitHub токеном только для чтения пакетов: `docker login ghcr.io`.
4. `docker compose up -d`.
5. В `deploy/Caddyfile` ayzonstock добавить блок и перечитать настройки:
   ```
   exam.ayzon.uz {
   	encode zstd gzip
   	reverse_proxy ayzon-exam-app:8080
   }
   ```
   `docker exec ayzon-caddy caddy reload --config /etc/caddy/Caddyfile`
6. Бот сам переходит на webhook при запуске. В BotFather → Bot Settings → Configure Mini App адрес
   указывать не нужно: кнопка «Ochish» берёт его из `WEBAPP_URL`.

**Каждый день:**

| Что | Команда (в `deploy`) |
|---|---|
| Обновить на последнюю версию: копия базы, при сбое — откат | `./update.sh` |
| Состояние и память | `docker compose ps`, `docker stats --no-stream` |
| Журнал сервера | `docker compose logs --tail 100 app` |
| Копии (на сервере и в R2) | `./restore.sh` |
| Копия сейчас | `docker compose exec backup backup now` |
| Вернуть базу из копии | `./restore.sh <файл> --yes` |

Служба копий каждую минуту проверяет сервер. Если он не отвечает 3 минуты, приходит сообщение в Telegram
всем из `ADMIN_IDS`, и ещё одно, когда сервер снова отвечает. Ночная копия делается в 03:00 по Ташкенту.
Если копия не сделалась или не ушла в R2, тоже приходит сообщение.

**Репетиция на компьютере** (Docker Desktop; в Git Bash — `export MSYS_NO_PATHCONV=1`, иначе пути
`/backups/…` подменяются):

```bash
cd deploy && cp local/.env.example local/.env
export EXAM_COMPOSE="docker compose -f compose.yml -f local/compose.local.yml --env-file local/.env"
$EXAM_COMPOSE up -d --build          # сервер на http://localhost:8088
UPDATE_PULL=off ./update.sh          # обновление с откатом
./restore.sh <файл> --yes            # восстановление
$EXAM_COMPOSE down -v                # убрать всё вместе с данными
```
