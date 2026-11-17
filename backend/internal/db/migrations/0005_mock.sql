-- Полный mock по частям (этап 18): Listening → Reading → Writing → Speaking, у каждой части свой таймер,
-- между частями можно закрыть приложение и продолжить позже.

CREATE TABLE mocks (
    id                 bigserial PRIMARY KEY,
    user_id            bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status             text        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'done')),
    -- Чем оплачен (этап 6): premium — из двух в месяц, credit — разовая покупка, trial — пробные дни.
    paid_with          text        NOT NULL DEFAULT 'premium',
    listening_band     numeric(2,1),
    reading_band       numeric(2,1),
    writing_band       numeric(2,1),
    speaking_band      numeric(2,1),
    overall            numeric(2,1),
    -- Writing: таймер с начала части; темы закрепляются при старте.
    writing_started_at timestamptz,
    task1_prompt_id    bigint,
    task2_prompt_id    bigint,
    writing_at         timestamptz,
    writing_ranked     boolean     NOT NULL DEFAULT false,
    speaking_started_at timestamptz,
    -- Вопросы Speaking mock: [{"part": 1, "text": "..."}], по одной записи на часть.
    speaking_questions jsonb,
    speaking_at        timestamptz,
    speaking_ranked    boolean     NOT NULL DEFAULT false,
    started_at         timestamptz NOT NULL,
    finished_at        timestamptz
);

CREATE INDEX mocks_user ON mocks (user_id, started_at DESC);
-- Один незаконченный mock на ученика.
CREATE UNIQUE INDEX mocks_active ON mocks (user_id) WHERE status = 'active';

ALTER TABLE attempts ADD FOREIGN KEY (mock_id) REFERENCES mocks (id) ON DELETE CASCADE;
ALTER TABLE writing_checks ADD COLUMN mock_id bigint REFERENCES mocks (id) ON DELETE SET NULL;

-- Темы Task 1 с графиком из банка (для mock): картинка задания хранится вместе с темой.
ALTER TABLE writing_prompts ADD COLUMN image_id bigint REFERENCES media (id);
ALTER TABLE mocks ADD FOREIGN KEY (task1_prompt_id) REFERENCES writing_prompts (id),
                  ADD FOREIGN KEY (task2_prompt_id) REFERENCES writing_prompts (id);

-- Writing и Speaking идут в рейтинг только из mock: там они с таймером (SPEC, «Честный рейтинг»).
CREATE OR REPLACE VIEW rated_parts AS
    SELECT user_id, kind, band, finished_at AS at FROM attempts WHERE ranked
    UNION ALL
    SELECT user_id, 'writing', writing_band, writing_at FROM mocks WHERE writing_ranked AND writing_band IS NOT NULL
    UNION ALL
    SELECT user_id, 'speaking', speaking_band, speaking_at FROM mocks WHERE speaking_ranked AND speaking_band IS NOT NULL;
