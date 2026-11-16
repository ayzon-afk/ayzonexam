-- Проверка Writing (этап 2). Эссе — личное: видит только автор (ответы фильтруются по user_id).

-- Банк тем. Темы Task 1 (графики) появятся с банком заданий в этапе 3; пока Task 1 — фото задания.
CREATE TABLE writing_prompts (
    id         bigserial PRIMARY KEY,
    task       text        NOT NULL CHECK (task IN ('task1', 'task2')),
    text       text        NOT NULL,
    active     boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Проверка — она же задание очереди: воркеры берут строки queued (и processing с истёкшей арендой).
CREATE TABLE writing_checks (
    id           bigserial PRIMARY KEY,
    user_id      bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    task         text        NOT NULL CHECK (task IN ('task1', 'task2')),
    prompt_id    bigint      REFERENCES writing_prompts (id),
    -- Копия текста темы: правка банка не меняет историю.
    prompt       text        NOT NULL DEFAULT '',
    essay        text        NOT NULL,
    words        integer     NOT NULL,
    source       text        NOT NULL CHECK (source IN ('text', 'photo')),
    -- Язык разбора — язык интерфейса в момент отправки.
    lang         text        NOT NULL CHECK (lang IN ('uz', 'ru', 'en')),
    status       text        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'done', 'failed')),
    attempts     integer     NOT NULL DEFAULT 0,
    run_after    timestamptz NOT NULL,
    locked_until timestamptz,
    -- Общий балл считает сервер из четырёх критериев (writing.Band), не AI.
    band         numeric(2,1) CHECK (band BETWEEN 0 AND 9),
    ta           smallint CHECK (ta BETWEEN 0 AND 9),
    cc           smallint CHECK (cc BETWEEN 0 AND 9),
    lr           smallint CHECK (lr BETWEEN 0 AND 9),
    gra          smallint CHECK (gra BETWEEN 0 AND 9),
    feedback     jsonb,
    model        text        NOT NULL DEFAULT '',
    last_error   text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL,
    finished_at  timestamptz
);

CREATE INDEX writing_checks_user ON writing_checks (user_id, created_at DESC);
CREATE INDEX writing_checks_queue ON writing_checks (run_after) WHERE status IN ('queued', 'processing');

-- Фото задания Task 1 с графиком. Живёт только до конца проверки (DECISIONS 2026-11-16).
CREATE TABLE writing_images (
    check_id bigint PRIMARY KEY REFERENCES writing_checks (id) ON DELETE CASCADE,
    mime     text   NOT NULL,
    data     bytea  NOT NULL
);

-- Платные вызовы AI вне очереди (распознавание фото) — для дневного лимита.
CREATE TABLE ai_usage (
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text        NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX ai_usage_user ON ai_usage (user_id, kind, created_at);

-- Свои темы Task 2 (не из книг Cambridge).
INSERT INTO writing_prompts (task, text) VALUES
('task2', 'Some people believe that university education should be free for every student, while others think students should pay for their own studies. Discuss both views and give your own opinion.'),
('task2', 'In many cities, people spend more and more time commuting to work. What are the causes of this problem, and what measures could be taken to reduce it?'),
('task2', 'Some people think that children should start learning a foreign language at primary school rather than at secondary school. Do the advantages of this outweigh the disadvantages?'),
('task2', 'Many young people today prefer to work for themselves rather than for a large company. Why is this the case? Is this a positive or negative development?'),
('task2', 'Some people say that the best way to improve public health is to increase the number of sports facilities. Others believe this would have little effect and other measures are needed. Discuss both views and give your opinion.'),
('task2', 'Online shopping is replacing traditional shops in many countries. Do you think this is a positive or negative development?'),
('task2', 'Governments should spend more money on public transport than on building new roads. To what extent do you agree or disagree?'),
('task2', 'In some countries, older people are increasingly living alone rather than with their families. What are the reasons for this, and what problems can it cause?'),
('task2', 'Some people believe that teenagers should be required to do unpaid community work in their free time. To what extent do you agree or disagree?'),
('task2', 'Social media has made it easier for people to stay in touch, but some argue that it has made relationships more superficial. Discuss both views and give your own opinion.'),
('task2', 'Many people believe that tourism does more harm than good to the places that tourists visit. To what extent do you agree or disagree?'),
('task2', 'Some people think that it is better for children to grow up in the countryside, while others believe that a big city offers more opportunities. Discuss both views and give your own opinion.');
