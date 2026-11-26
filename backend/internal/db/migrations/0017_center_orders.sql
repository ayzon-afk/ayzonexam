-- Центр сам продлевает тариф (DECISIONS 2026-11-25): места и месяцы → сумма → перевод на карту со скрином →
-- владелец подтверждает кнопкой в боте → запись в center_payments, места и срок продлеваются сами.
-- Доступ до подтверждения не открывается: суммы центров крупнее, а откатить места и срок сложнее.
CREATE TABLE center_orders (
    id         bigserial PRIMARY KEY,
    center_id  bigint      NOT NULL REFERENCES centers (id) ON DELETE CASCADE,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE, -- кто отправил: аккаунт центра
    seats      integer     NOT NULL CHECK (seats > 0),
    months     integer     NOT NULL CHECK (months BETWEEN 1 AND 12),
    amount     integer     NOT NULL CHECK (amount > 0),
    screenshot bigint      NOT NULL REFERENCES media (id),
    status     text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'rejected')),
    reason     text        NOT NULL DEFAULT '',
    decided_by bigint,
    created_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz
);

-- Одна непроверенная заявка на центр: вторую — после ответа владельца.
CREATE UNIQUE INDEX center_orders_one_pending ON center_orders (center_id) WHERE status = 'pending';
CREATE INDEX center_orders_status ON center_orders (status, created_at DESC);
