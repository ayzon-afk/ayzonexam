-- Деньги (этап 6): Premium до даты, перевод на карту со скрином, проверка владельцем.

-- premium_until — до какого момента Premium (пробные 3 дня, оплата, бонус за друга);
-- pay_code — код платежа для комментария к переводу; pay_trust — rejected после первого отказа:
-- дальше Premium открывается только после подтверждения; mock_credits — купленные разовые mock.
ALTER TABLE users
    ADD COLUMN premium_until timestamptz,
    ADD COLUMN pay_code      text    UNIQUE,
    ADD COLUMN pay_trust     text    NOT NULL DEFAULT 'new' CHECK (pay_trust IN ('new', 'rejected')),
    ADD COLUMN mock_credits  integer NOT NULL DEFAULT 0 CHECK (mock_credits >= 0);

-- Новым — 3 дня Premium (SPEC); тем, кто уже есть, — тоже от даты регистрации.
UPDATE users SET premium_until = created_at + interval '3 days';

CREATE TABLE payments (
    id            bigserial PRIMARY KEY,
    user_id       bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code          text        NOT NULL,
    plan          text        NOT NULL CHECK (plan IN ('month', 'quarter', 'mock')),
    amount        integer     NOT NULL CHECK (amount > 0),
    status        text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'rejected')),
    screenshot    bigint      NOT NULL REFERENCES media (id),
    -- Что открылось сразу, «на проверке»: при отказе это забирается.
    granted_days  integer     NOT NULL DEFAULT 0,
    granted_mocks integer     NOT NULL DEFAULT 0,
    reason        text        NOT NULL DEFAULT '',
    decided_by    bigint,
    created_at    timestamptz NOT NULL DEFAULT now(),
    decided_at    timestamptz
);

CREATE INDEX payments_status ON payments (status, created_at DESC);
CREATE INDEX payments_user ON payments (user_id, created_at DESC);
-- Одна непроверенная оплата за раз (SPEC, «Защита от обмана»).
CREATE UNIQUE INDEX payments_one_pending ON payments (user_id) WHERE status = 'pending';

-- Оплата центра и учителя (B2B): владелец прикладывает чек и подтверждает в админке.
CREATE TABLE center_payments (
    id         bigserial PRIMARY KEY,
    center_id  bigint      NOT NULL REFERENCES centers (id) ON DELETE CASCADE,
    amount     integer     NOT NULL CHECK (amount > 0),
    seats      integer     NOT NULL CHECK (seats > 0),
    paid_until date        NOT NULL,
    receipt    bigint      REFERENCES media (id),
    note       text        NOT NULL DEFAULT '',
    admin_tg   bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
