-- Вопрос дня в боте (хакатон, DECISIONS 2026-11-24): каждое утро вопрос из банка теста уровня.
ALTER TABLE users ADD COLUMN daily_q boolean NOT NULL DEFAULT true; -- false — ученик отключил кнопкой

-- Рассылка за день — одна: строка занимается до отправки, перезапуск сервера не шлёт второй раз.
CREATE TABLE daily_sends (
    day  date        PRIMARY KEY,
    sent integer     NOT NULL DEFAULT 0,
    at   timestamptz NOT NULL DEFAULT now()
);

-- Засчитывается первый ответ ученика за день.
CREATE TABLE daily_answers (
    user_id bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    day     date        NOT NULL,
    correct boolean     NOT NULL,
    at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, day)
);
