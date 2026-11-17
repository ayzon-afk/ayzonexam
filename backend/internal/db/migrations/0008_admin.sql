-- Админка владельца и поддержка (этап 17).

-- Журнал: кто из админов что и когда изменил (SPEC, «Админка»).
CREATE TABLE admin_log (
    id         bigserial PRIMARY KEY,
    admin_tg   bigint      NOT NULL,
    action     text        NOT NULL,
    target     text        NOT NULL DEFAULT '',
    details    jsonb       NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX admin_log_time ON admin_log (created_at DESC);

-- Обращения в поддержку: из приложения («Поддержка» в профиле) и из бота (/help).
CREATE TABLE tickets (
    id         bigserial PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status     text        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'answered', 'closed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tickets_user ON tickets (user_id, updated_at DESC);
CREATE INDEX tickets_status ON tickets (status, updated_at DESC);

CREATE TABLE ticket_messages (
    id         bigserial PRIMARY KEY,
    ticket_id  bigint      NOT NULL REFERENCES tickets (id) ON DELETE CASCADE,
    -- NULL — пишет ученик; иначе Telegram id админа, который ответил.
    admin_tg   bigint,
    text       text        NOT NULL DEFAULT '',
    media_id   bigint      REFERENCES media (id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ticket_messages_ticket ON ticket_messages (ticket_id, id);

-- Какое сообщение бота у админа к какому обращению относится: ответ «reply» в Telegram уходит
-- ученику. Так же — оплаты: сообщение со скрином и кнопками.
CREATE TABLE bot_refs (
    chat_id    bigint      NOT NULL,
    message_id bigint      NOT NULL,
    kind       text        NOT NULL CHECK (kind IN ('ticket', 'payment')),
    ref_id     bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, message_id)
);

-- Что бот ждёт от ученика следующим сообщением: help — текст обращения; speaking:<id> — голосовое
-- ответа на вопрос Speaking (этап 4).
ALTER TABLE users ADD COLUMN bot_state text NOT NULL DEFAULT '';

-- Когда ученик последний раз открывал приложение (с точностью до часа) — «активные» в обзоре админки.
ALTER TABLE users ADD COLUMN seen_at timestamptz;
CREATE INDEX users_seen ON users (seen_at);
