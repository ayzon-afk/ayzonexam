-- Заявки учебных центров с открытой страницы (хакатон, DECISIONS 2026-11-24): без входа, владелец получает
-- уведомление в бот и связывается сам.
CREATE TABLE center_leads (
    id         bigserial PRIMARY KEY,
    center     text        NOT NULL,
    name       text        NOT NULL,
    phone      text        NOT NULL,
    students   integer,
    message    text        NOT NULL DEFAULT '',
    lang       text        NOT NULL DEFAULT 'uz',
    created_at timestamptz NOT NULL DEFAULT now()
);
