-- Рейтинг, центры и «Турнир месяца» (этап 8).

CREATE TABLE centers (
    id          bigserial PRIMARY KEY,
    name        text        NOT NULL UNIQUE,
    city        text        NOT NULL DEFAULT 'tashkent',
    about       text        NOT NULL DEFAULT '',
    -- Код ссылки-приглашения: по ней ученик входит в центр сразу, без подтверждения.
    invite_code text        NOT NULL UNIQUE,
    -- Тариф центра (этап 17): сколько учеников оплачено и до какого дня у них Premium.
    seats       integer     NOT NULL DEFAULT 0,
    paid_until  date,
    active      boolean     NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- Город — для рейтинга Ташкента; '' — не указан. Центр: pending — ученик попросился, центр ещё не
-- подтвердил; member — в центре. center_role — аккаунт центра (admin) или учитель центра.
ALTER TABLE users
    ADD COLUMN city             text        NOT NULL DEFAULT '' CHECK (city IN ('', 'tashkent', 'other')),
    ADD COLUMN center_id        bigint      REFERENCES centers (id) ON DELETE SET NULL,
    ADD COLUMN center_status    text        CHECK (center_status IN ('pending', 'member')),
    ADD COLUMN center_role      text        CHECK (center_role IN ('admin', 'teacher')),
    ADD COLUMN verified_teacher boolean     NOT NULL DEFAULT false,
    ADD COLUMN blocked_at       timestamptz;

CREATE INDEX users_center ON users (center_id) WHERE center_id IS NOT NULL;

-- Части, которые идут в рейтинг: таймерные попытки. Mock (этап 18) добавляет сюда Writing и Speaking.
CREATE VIEW rated_parts AS
    SELECT user_id, kind, band, finished_at AS at FROM attempts WHERE ranked;

CREATE TABLE tournaments (
    id                bigserial PRIMARY KEY,
    starts_at         timestamptz NOT NULL UNIQUE,
    ends_at           timestamptz NOT NULL,
    reading_test_id   bigint      REFERENCES tests (id),
    listening_test_id bigint      REFERENCES tests (id),
    -- Чемпион — после подтверждения тройки Speaking владельцем; стоит на главной месяц.
    champion_id       bigint      REFERENCES users (id) ON DELETE SET NULL,
    champion_band     numeric(2,1),
    created_at        timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE tournament_entries (
    tournament_id bigint      NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
    user_id       bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    joined_at     timestamptz NOT NULL DEFAULT now(),
    -- Тройка лучших подтверждает результат коротким Speaking; не подтвердил — место следующему.
    confirmed     boolean     NOT NULL DEFAULT false,
    disqualified  boolean     NOT NULL DEFAULT false,
    PRIMARY KEY (tournament_id, user_id)
);

ALTER TABLE attempts ADD FOREIGN KEY (tournament_id) REFERENCES tournaments (id);
CREATE INDEX attempts_tournament ON attempts (tournament_id) WHERE tournament_id IS NOT NULL;
