-- Тест уровня, который подстраивается под ученика (DECISIONS 2026-11-27). Вопросы — из банка в коде
-- (internal/level/bank.json); попытка — обычная строка attempts вида level (история, бонус за друга, счёт
-- тестов), а ход теста — здесь. Задание-заглушка нужно для attempts.test_id; ученикам не показывается.

INSERT INTO tests (kind, title, minutes, questions, active, source)
VALUES ('level', 'Level test', 15, '[]', false, 'level-adaptive');

CREATE TABLE level_runs (
    attempt_id bigint      PRIMARY KEY REFERENCES attempts (id) ON DELETE CASCADE,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Заданные вопросы по порядку: [{"id": "c12", "a": 1, "ok": true}] — по ним потом уточняется сложность.
    items      jsonb       NOT NULL DEFAULT '[]',
    -- Вопрос на экране; пусто — идёт текст Reading.
    current    text        NOT NULL DEFAULT '',
    passage    text        NOT NULL DEFAULT '',
    reading    jsonb,
    low        numeric(2,1),
    high       numeric(2,1)
);

CREATE INDEX level_runs_user ON level_runs (user_id);
