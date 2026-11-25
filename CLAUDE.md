# AYZON EXAM — инструкции для Claude

Telegram Mini App для подготовки к IELTS: mock-экзамены, AI-проверка Writing и Speaking, рейтинг, дуэли,
учителя и центры. Что должен делать продукт — [SPEC.md](SPEC.md), этапы и состояние —
[ROADMAP.md](ROADMAP.md), решения и почему — [DECISIONS.md](DECISIONS.md), запуск — [README.md](README.md).

## Общение

- Пользователь пишет по-русски — отвечать по-русски, простыми словами.
- Главное требование — мало памяти и быстро: приложение открывают на недорогих Android через мобильный
  интернет, сервер общий с ayzonstock. Любая новая зависимость — с причиной в DECISIONS.md.

## Устройство

- `backend/` — Go, один бинарник: API Mini App, бот, страница оплаты `/pay`, собранный интерфейс (если
  задан `WEB_DIR`).
  - `internal/api` — HTTP и логика бота (`HandleUpdate`), админка `/api/admin/*`; `internal/tgauth` —
    проверка initData; `internal/users` — ученики; `internal/bot` — только Bot API (транспорт, тексты);
    `internal/db` — пул и миграции.
  - Домены: `writing`, `speaking` (очереди AI), `exam` (задания, баллы, mock), `rating` (рейтинг, центры,
    турнир), `social` (друзья, дуэли), `billing` (тарифы, лимиты, оплаты), `news`, `support`; `ai` —
    Gemini и заглушка `Fake`.
  - `cmd/server` — сервер, `cmd/devdb` — встроенный PostgreSQL для разработки, `cmd/calibrate`.
- `frontend/` — Svelte 5 + Vite, SPA без роутера: `screens/` (онбординг, оболочка, экраны поверх вкладок
  по папкам `exam`, `mock`, `speaking`, `social`, `center`, `news`, `writing`, `admin`), `screens/demo/`
  (ещё не подключённое: группы, учитель, взаимная проверка), `tabs/`, `components/`, `lib/` (API,
  Telegram, i18n, навигация `nav.svelte.ts`, запись голоса, турнир, демо-данные `demo.ts`), `screens/cabinet/`
  (кабинет центра: сайт `/cabinet` и экран в Mini App, API — `internal/api/cabinet.go`).
  Этап подключает экран к серверу — убирает его данные из `lib/demo.ts` и пометку `<Demo />`.
- `deploy/` — сервер: образ, compose рядом с ayzonstock, служба копий и наблюдения (`backup/backup.sh`),
  обновление с откатом и восстановление; как пользоваться — README, «Сервер».
- Схема — только миграциями `backend/internal/db/migrations/NNNN_*.sql`; старые не редактировать.

## Команды (из корня)

| Что | Команда |
|---|---|
| Все тесты: Go на встроенном PostgreSQL, интерфейс, сборка с бюджетом веса, вёрстка в Chrome | `npm test` |
| Только вёрстка (после сборки); `cabinet` — фильтр по имени, `--shots папка` — снимки | `npm --prefix frontend run layout` |
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
  Исключение — админка (`screens/admin/`): только по-русски, для владельца (DECISIONS 2026-11-17).
- Тексты бота — `internal/bot/texts.go`, тоже uz, ru и en.
- Админ-маршрут — только через `s.admin(...)` и с записью в журнал (`s.audit`).
- Бюджет: первый экран (входной JS со статическими импортами и словарь одного языка) — меньше 60 КБ gzip,
  проверяет `npm test`. Новый экран поверх вкладок — через ленивую загрузку в `Shell`. Перед тяжёлой
  библиотекой — искать лёгкую или писать своё.

## Ловушки

- Встроенный PostgreSQL на русской Windows создаётся в WIN1251 — `Encoding("UTF8")` в `cmd/devdb` и
  `internal/testpg` не убирать.
- Ошибки `net/http` содержат адрес запроса, а в адресе Bot API — токен: `bot.call` отдаёт ошибку без адреса.
- `DEV_AUTH=1` открывает вход без Telegram; `config` запрещает его вместе с `BOT_MODE=webhook`.
- svelte-check пока не работает с TypeScript 7 — во `frontend` стоит TypeScript 6.
- Порты встроенной базы тестов — ниже 49152 (25431+): Windows резервирует диапазоны выше
  (`netsh int ipv4 show excludedportrange protocol=tcp`). Упавший тест может оставить свою базу
  (`ayzonexam-pg…` во временных файлах) — следующий запуск пишет «already listening».
- Правила рейтинга и турнира одинаковые на сервере и в интерфейсе: `rating.TourStart` ↔ `lib/tournament.ts`.
- Новый экран — добавить случай в `frontend/scripts/layout.mjs` (и ответы API в `layout/fixtures.mjs`), иначе
  его вёрстку никто не проверит. Сетка с длинным текстом — `minmax(0, 1fr)`, иначе она раздвигает экран.
