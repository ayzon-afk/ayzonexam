-- Speaking (этап 4): запись в приложении или голосовое боту → AI ставит балл по 4 критериям.

CREATE TABLE speaking_questions (
    id         bigserial PRIMARY KEY,
    part       smallint    NOT NULL CHECK (part IN (1, 2, 3)),
    -- Part 2 — карточка: тема и пункты «You should say», через перевод строки.
    text       text        NOT NULL,
    active     boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Проверка — она же задание очереди, как writing_checks. Одна проверка — один ответ (практика) или вся
-- часть Speaking mock (три записи).
CREATE TABLE speaking_checks (
    id           bigserial PRIMARY KEY,
    user_id      bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    mock_id      bigint      REFERENCES mocks (id) ON DELETE SET NULL,
    source       text        NOT NULL CHECK (source IN ('app', 'bot')),
    -- [{"part": 2, "text": "..."}] — вопросы по порядку записей.
    questions    jsonb       NOT NULL,
    seconds      integer     NOT NULL DEFAULT 0,
    lang         text        NOT NULL CHECK (lang IN ('uz', 'ru', 'en')),
    status       text        NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'done', 'failed')),
    attempts     integer     NOT NULL DEFAULT 0,
    run_after    timestamptz NOT NULL,
    locked_until timestamptz,
    -- Балл считает сервер из четырёх критериев (как Writing).
    band         numeric(2,1) CHECK (band BETWEEN 0 AND 9),
    fc           smallint CHECK (fc BETWEEN 0 AND 9),
    lr           smallint CHECK (lr BETWEEN 0 AND 9),
    gra          smallint CHECK (gra BETWEEN 0 AND 9),
    pr           smallint CHECK (pr BETWEEN 0 AND 9),
    transcript   text        NOT NULL DEFAULT '',
    feedback     jsonb,
    model        text        NOT NULL DEFAULT '',
    last_error   text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL,
    finished_at  timestamptz
);

CREATE INDEX speaking_checks_user ON speaking_checks (user_id, created_at DESC);
CREATE INDEX speaking_checks_queue ON speaking_checks (run_after) WHERE status IN ('queued', 'processing');

-- Записи живут только до конца проверки, как фото задания Writing.
CREATE TABLE speaking_audio (
    check_id bigint   NOT NULL REFERENCES speaking_checks (id) ON DELETE CASCADE,
    n        smallint NOT NULL,
    mime     text     NOT NULL,
    data     bytea    NOT NULL,
    PRIMARY KEY (check_id, n)
);

-- Свои вопросы в стиле IELTS.
INSERT INTO speaking_questions (part, text) VALUES
(1, 'Do you work or are you a student? What do you like about it?'),
(1, 'Where do you live? What do you like about your neighbourhood?'),
(1, 'How do you usually spend your weekends?'),
(1, 'Do you prefer reading books or watching films? Why?'),
(1, 'What kind of music do you enjoy? Has your taste changed since you were a child?'),
(1, 'Do you like cooking? What dishes can you make?'),
(1, 'How often do you use public transport? What do you think of it in your city?'),
(1, 'Do you enjoy spending time outdoors? What do you like to do?'),
(2, E'Describe a place in your city that you like to visit.\nYou should say: where it is, how often you go there, what you do there, and explain why you like it.'),
(2, E'Describe a person who has influenced you.\nYou should say: who this person is, how you know them, what they did, and explain how they influenced you.'),
(2, E'Describe a skill you would like to learn.\nYou should say: what the skill is, why you want to learn it, how you would learn it, and explain how it would help you.'),
(2, E'Describe a memorable journey you have taken.\nYou should say: where you went, who you went with, what happened, and explain why it was memorable.'),
(2, E'Describe a book or film that made you think.\nYou should say: what it was, when you read or watched it, what it was about, and explain why it made you think.'),
(2, E'Describe a celebration in your family.\nYou should say: what it was, who was there, what you did, and explain why it was special.'),
(3, 'Why do people enjoy visiting historical places? Has this changed in recent years?'),
(3, 'Do you think young people today learn more from their friends or from their parents? Why?'),
(3, 'Should schools teach practical skills like cooking and managing money? Why or why not?'),
(3, 'How has technology changed the way people travel?'),
(3, 'Why do some people prefer to live in big cities while others prefer small towns?'),
(3, 'Is it more important for a job to be well paid or to be interesting? Why?');
