-- Дуэли и друзья (этап 9): вызов ссылкой, 5 минут слов; друзья — для рейтинга друзей и бонусов.

-- Дружба — две строки (a→b и b→a): рейтинг друзей и список читаются одним условием.
CREATE TABLE friends (
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    friend_id  bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, friend_id),
    CHECK (user_id <> friend_id)
);

-- invite_code — ссылка «пригласить друга»; referred_by — кто пригласил (бонусы: SPEC, «Деньги»);
-- bonus_checks — бонусные проверки сверх лимита; referral_rewarded — бонус за тест уровня уже выдан.
ALTER TABLE users
    ADD COLUMN invite_code       text    UNIQUE,
    ADD COLUMN referred_by       bigint  REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN bonus_checks      integer NOT NULL DEFAULT 0 CHECK (bonus_checks >= 0),
    ADD COLUMN referral_rewarded boolean NOT NULL DEFAULT false;

-- Дуэль не в реальном времени: каждый играет свои 5 минут на одних и тех же словах, итог — когда
-- сыграли оба. Так дуэль работает, даже если друг откроет ссылку вечером.
CREATE TABLE duels (
    id            bigserial PRIMARY KEY,
    code          text        NOT NULL UNIQUE,
    challenger_id bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    opponent_id   bigint      REFERENCES users (id) ON DELETE CASCADE,
    -- [{"q": "rapid", "options": [...], "answer": 1}]
    questions     jsonb       NOT NULL,
    created_at    timestamptz NOT NULL,
    expires_at    timestamptz NOT NULL
);

CREATE INDEX duels_challenger ON duels (challenger_id, created_at DESC);
CREATE INDEX duels_opponent ON duels (opponent_id, created_at DESC);

CREATE TABLE duel_players (
    duel_id     bigint      NOT NULL REFERENCES duels (id) ON DELETE CASCADE,
    user_id     bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    started_at  timestamptz NOT NULL,
    finished_at timestamptz,
    -- Ответ на каждый вопрос по порядку; сервер сверяет время и ключ на каждом ответе.
    answers     jsonb       NOT NULL DEFAULT '[]',
    score       integer     NOT NULL DEFAULT 0,
    PRIMARY KEY (duel_id, user_id)
);
