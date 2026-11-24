-- Учёт реального использования (хакатон, DECISIONS 2026-11-24): одна строка — одно открытие приложения.
-- Приложение шлёт пульс раз в 30 с, пока открыто и активно; active_seconds — сумма засчитанных пульсов.
CREATE TABLE app_sessions (
    user_id        bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    sid            text        NOT NULL, -- случайный id открытия от приложения
    started_at     timestamptz NOT NULL,
    last_beat      timestamptz NOT NULL,
    active_seconds integer     NOT NULL DEFAULT 0,
    beats          integer     NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, sid)
);

CREATE INDEX app_sessions_started ON app_sessions (started_at);
