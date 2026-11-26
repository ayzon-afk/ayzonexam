-- Заявка «Добавить свой центр / Я учитель» (DECISIONS 2026-11-25): кнопка на сайте → бот → анкета в
-- приложении (вход через Telegram) → владелец одобряет кнопкой в боте → центр создаётся сам, заявитель —
-- его аккаунт. pilot — одобрено с бесплатным месяцем (SPEC, «Деньги»: пилот — месяц бесплатно).
CREATE TABLE center_applications (
    id         bigserial PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text        NOT NULL CHECK (kind IN ('center', 'teacher')),
    name       text        NOT NULL,
    city       text        NOT NULL,
    students   integer     NOT NULL CHECK (students >= 0),
    phone      text        NOT NULL DEFAULT '',
    about      text        NOT NULL DEFAULT '',
    status     text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    pilot      boolean     NOT NULL DEFAULT false,
    center_id  bigint      REFERENCES centers (id) ON DELETE SET NULL,
    decided_by bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz
);

-- Одна заявка на проверке от человека: вторую — после ответа владельца.
CREATE UNIQUE INDEX center_applications_one_pending ON center_applications (user_id) WHERE status = 'pending';
