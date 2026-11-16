# AYZON EXAM — инструкции для Claude

Telegram Mini App для подготовки к IELTS: mock-экзамены, AI-проверка Writing и Speaking, рейтинг, дуэли,
учителя и центры. Что должен делать продукт — [SPEC.md](SPEC.md), этапы и состояние —
[ROADMAP.md](ROADMAP.md), решения и почему — [DECISIONS.md](DECISIONS.md), запуск — [README.md](README.md).

## Общение

- Пользователь пишет по-русски — отвечать по-русски, простыми словами.
- Главное требование — мало памяти и быстро: приложение открывают на недорогих Android через мобильный
  интернет, сервер общий с ayzonstock. Любая новая зависимость — с причиной в DECISIONS.md.

## Устройство

- `backend/` — Go, один бинарник: API Mini App, бот, собранный интерфейс (если задан `WEB_DIR`).
  - `internal/tgauth` — проверка initData; `internal/api` — HTTP; `internal/users` — ученики;
    `internal/bot` — Bot API напрямую по HTTP; `internal/db` — пул и миграции.
  - `cmd/server` — сервер, `cmd/devdb` — встроенный PostgreSQL для разработки.
- `frontend/` — Svelte 5 + Vite, SPA без роутера: `screens/` (онбординг, оболочка с вкладками, Writing),
  `screens/demo/` (экраны этапа 12 на демо-данных, грузятся при переходе), `tabs/`, `components/`, `lib/`
  (API, Telegram, i18n, навигация `nav.svelte.ts`, баллы и даты, демо-данные `demo.ts` и `demo-exam.ts`).
  Этап подключает экран к серверу — убирает его данные из `lib/demo*.ts` и пометку `<Demo />`.
- Схема — только миграциями `backend/internal/db/migrations/NNNN_*.sql`; старые не редактировать.

## Команды (из корня)

| Что | Команда |
|---|---|
| Все тесты: Go на встроенном PostgreSQL, интерфейс, сборка с бюджетом веса | `npm test` |
| gofmt, go vet, svelte-check | `npm run lint` |
| Локальная база (localhost:5433, данные в `backend/.devdb`) | `npm run dev:db` |
| Сервер (читает `backend/.env`) | `npm run dev:server` |
| Интерфейс в браузере с горячей перезагрузкой | `npm run dev:web` |
| Боевая сборка: `frontend/dist` и `dist/ayzon-exam` | `npm run build` |

## Правила кода

- Вход: каждый `/api/` запрос несёт `Authorization: tma <initData>`, проверка в `api.auth`. Новый
  маршрут ученика — только через `s.auth(...)`.
- Личное не уходит другим: имя, @username, Telegram id отдаются только самому ученику (`meView`).
  Для ответов о других людях — отдельная структура с ником, аватаром и результатами.
- Правила баллов и дат одинаковые на сервере (`api.validBand`, проверка `exam_date` по Ташкенту) и в
  интерфейсе (`lib/exam.ts`). Меняется одно — меняется и другое.
- Тексты интерфейса — только через `t('ключ')`; ключ добавляется сразу в uz, ru и en (тест сверяет).
- Бюджет: первый экран (входной JS со статическими импортами и словарь одного языка) — меньше 60 КБ gzip,
  проверяет `npm test`. Новый экран поверх вкладок — через ленивую загрузку в `Shell`. Перед тяжёлой библиотекой — искать лёгкую
  или писать своё.

## Ловушки

- Встроенный PostgreSQL на русской Windows создаётся в WIN1251 — `Encoding("UTF8")` в `cmd/devdb` и
  `internal/testpg` не убирать.
- Ошибки `net/http` содержат адрес запроса, а в адресе Bot API — токен: `bot.call` отдаёт ошибку без адреса.
- `DEV_AUTH=1` открывает вход без Telegram; `config` запрещает его вместе с `BOT_MODE=webhook`.
- svelte-check пока не работает с TypeScript 7 — во `frontend` стоит TypeScript 6.
