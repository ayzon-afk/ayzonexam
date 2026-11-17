-- Reading, Listening и тест уровня (этап 3). Проверка по ключу: ключ ответов уходит ученику только
-- после сдачи. Задания — свои, не из книг Cambridge (SPEC, «Позиционирование»).

-- Файлы: аудио Listening (открытые) и скрины оплат, фото обращений (private — только админке).
-- Байты в базе, пока нет R2: файлов мало, а копия базы сохраняет их вместе со всем остальным.
CREATE TABLE media (
    id         bigserial PRIMARY KEY,
    mime       text        NOT NULL,
    data       bytea       NOT NULL,
    private    boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tests (
    id         bigserial PRIMARY KEY,
    kind       text        NOT NULL CHECK (kind IN ('reading', 'listening', 'level')),
    title      text        NOT NULL,
    -- full — полноразмерная часть для mock (около 40 вопросов); иначе короткий сет практики.
    full_size  boolean     NOT NULL DEFAULT false,
    minutes    integer     NOT NULL CHECK (minutes BETWEEN 1 AND 90),
    -- Текст Reading, абзацы через пустую строку.
    passage    text        NOT NULL DEFAULT '',
    -- Сценарий Listening: из него синтезируется речь, ученик видит его после сдачи.
    transcript text        NOT NULL DEFAULT '',
    audio_id   bigint      REFERENCES media (id),
    -- [{"q": "...", "options": ["..."], "answer": 0, "type": "tfng|detail|main|grammar|vocab"}]
    questions  jsonb       NOT NULL,
    active     boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Попытка создаётся при старте (сервер знает, когда пошёл таймер) и закрывается при сдаче.
CREATE TABLE attempts (
    id            bigserial PRIMARY KEY,
    user_id       bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    test_id       bigint      NOT NULL REFERENCES tests (id),
    kind          text        NOT NULL,
    started_at    timestamptz NOT NULL,
    finished_at   timestamptz,
    answers       jsonb,
    correct       integer,
    total         integer,
    band          numeric(2,1) CHECK (band BETWEEN 0 AND 9),
    -- Сколько раз ученик уходил из приложения во время попытки (SPEC, «Честный рейтинг»).
    leaves        integer     NOT NULL DEFAULT 0,
    -- В рейтинг: сдана вовремя, без частых уходов; тест уровня — никогда.
    ranked        boolean     NOT NULL DEFAULT false,
    mock_id       bigint,
    tournament_id bigint
);

CREATE INDEX attempts_user ON attempts (user_id, finished_at DESC);
CREATE INDEX attempts_ranked ON attempts (user_id, kind, finished_at) WHERE ranked;

INSERT INTO tests (kind, title, minutes, questions) VALUES ('level', 'Level test', 15, $json$[
 {"q": "She ___ in Tashkent since 2020.", "options": ["lives", "has lived", "is living", "lived"], "answer": 1, "type": "grammar"},
 {"q": "If I ___ more time, I would travel more.", "options": ["have", "had", "will have", "am having"], "answer": 1, "type": "grammar"},
 {"q": "The report was written ___ the manager.", "options": ["from", "with", "by", "of"], "answer": 2, "type": "grammar"},
 {"q": "Closest meaning: “significant”", "options": ["important", "small", "quick", "strange"], "answer": 0, "type": "vocab"},
 {"q": "He is used to ___ early.", "options": ["get up", "getting up", "got up", "gets up"], "answer": 1, "type": "grammar"},
 {"q": "Neither the teacher nor the students ___ ready.", "options": ["was", "were", "is", "be"], "answer": 1, "type": "grammar"},
 {"q": "Closest meaning: “decline”", "options": ["decrease", "choose", "explain", "arrive"], "answer": 0, "type": "vocab"},
 {"q": "By the time we arrived, the film ___.", "options": ["started", "has started", "had started", "starts"], "answer": 2, "type": "grammar"},
 {"q": "I wish I ___ harder at school.", "options": ["studied", "had studied", "study", "would study"], "answer": 1, "type": "grammar"},
 {"q": "Closest meaning: “reluctant”", "options": ["eager", "unwilling", "careful", "famous"], "answer": 1, "type": "vocab"},
 {"q": "The number of tourists ___ steadily since 2015.", "options": ["rises", "has risen", "rose", "is rising"], "answer": 1, "type": "grammar"},
 {"q": "Hardly ___ the station when the train left.", "options": ["I had reached", "had I reached", "I reached", "did I reached"], "answer": 1, "type": "grammar"},
 {"q": "Closest meaning: “mitigate”", "options": ["make worse", "reduce the effect of", "describe", "postpone"], "answer": 1, "type": "vocab"},
 {"q": "It is high time the government ___ action.", "options": ["takes", "took", "will take", "has taken"], "answer": 1, "type": "grammar"},
 {"q": "Closest meaning: “ubiquitous”", "options": ["rare", "found everywhere", "expensive", "ancient"], "answer": 1, "type": "vocab"}
]$json$);

INSERT INTO tests (kind, title, minutes, passage, questions) VALUES ('reading', 'The Paper Makers of Samarkand', 20,
$text$For several centuries Samarkand was famous across Asia and Europe for its paper. According to a popular story, the technique reached the city in the eighth century, when Chinese craftsmen shared their knowledge after a battle. Whatever its origin, Samarkand paper was soon praised for being smooth, strong and pleasant to write on.

The paper was made from the bark of the mulberry tree. Workers soaked the bark in water, beat it into a soft pulp and spread it thinly on frames to dry. Finally, the sheets were polished with a stone until they shone. The process required patience: a skilled master could produce only a few hundred sheets a day.

By the nineteenth century, cheaper factory paper had almost replaced the traditional craft. In the 1990s, however, a local master, Zarif Mukhtarov, decided to revive it. Using old descriptions and years of experiments, he rebuilt a water-powered mill in the village of Konigil. Today the workshop attracts thousands of tourists, and its paper is used by artists, restorers and bookbinders.$text$,
$json$[
 {"q": "Samarkand paper was known outside Asia.", "options": ["True", "False", "Not Given"], "answer": 0, "type": "tfng"},
 {"q": "The paper was made from the leaves of the mulberry tree.", "options": ["True", "False", "Not Given"], "answer": 1, "type": "tfng"},
 {"q": "Polishing the sheets was the first stage of the process.", "options": ["True", "False", "Not Given"], "answer": 1, "type": "tfng"},
 {"q": "Samarkand paper was more expensive than Chinese paper.", "options": ["True", "False", "Not Given"], "answer": 2, "type": "tfng"},
 {"q": "The mill in Konigil uses water power.", "options": ["True", "False", "Not Given"], "answer": 0, "type": "tfng"},
 {"q": "A skilled master could make…", "options": ["thousands of sheets a day", "a few hundred sheets a day", "a few sheets a week"], "answer": 1, "type": "detail"},
 {"q": "Why did the traditional craft almost disappear?", "options": ["A war destroyed the mills", "Mulberry trees died out", "Factory paper was cheaper"], "answer": 2, "type": "detail"},
 {"q": "How did Zarif Mukhtarov learn the old method?", "options": ["From his grandfather", "From old descriptions and experiments", "At a factory in China"], "answer": 1, "type": "detail"},
 {"q": "What happened to the bark after it was soaked?", "options": ["It was dried in the sun", "It was beaten into pulp", "It was polished with a stone"], "answer": 1, "type": "detail"},
 {"q": "What is the passage mainly about?", "options": ["How paper came to China", "The decline and revival of a traditional craft", "Tourism in Konigil"], "answer": 1, "type": "main"}
]$json$);

-- Listening без аудио ученикам не показывается: запись владельца или синтез речи по сценарию
-- добавляются в админке.
INSERT INTO tests (kind, title, minutes, transcript, questions) VALUES ('listening', 'Booking a city tour', 10,
$text$Receptionist: Good morning, Tashkent City Tours, Nodira speaking. How can I help you?
Caller: Hi. I'd like to book a city tour for this weekend, please.
Receptionist: Of course. We have tours on Saturday and Sunday. Which day would you prefer?
Caller: Saturday would be better. On Sunday we're visiting friends.
Receptionist: Saturday, fine. The tour starts at nine o'clock, but we ask everyone to meet at half past eight.
Caller: Half past eight. And where do we meet?
Receptionist: At the main gate of Navoi Park. You'll see a guide with a green flag.
Caller: Great. How long does the tour last?
Receptionist: About four hours. We finish at around one o'clock at Chorsu Bazaar.
Caller: And how much is it?
Receptionist: It's twenty dollars for adults and twelve dollars for students.
Caller: I'm a student, and my brother is too. So that's two student tickets.
Receptionist: Two students — twenty-four dollars in total. Could I have your name, please?
Caller: Yes, it's David Miller. M-I-double L-E-R.
Receptionist: Thank you, Mr Miller. Is there anything else you'd like to know?
Caller: Should we bring anything?
Receptionist: Lunch isn't included, but you'll have free time at the bazaar to buy food. The most important thing is comfortable shoes — we walk a lot. And bring water if it's hot.
Caller: Can we pay by card?
Receptionist: Yes, you can pay by card on the day, or in cash.
Caller: Perfect. Thanks very much.
Receptionist: You're welcome. See you on Saturday!$text$,
$json$[
 {"q": "What does the caller want to book?", "options": ["A city tour", "A hotel room", "A train ticket"], "answer": 0, "type": "detail"},
 {"q": "Which day will the caller go on the tour?", "options": ["Friday", "Saturday", "Sunday"], "answer": 1, "type": "detail"},
 {"q": "What time should the group meet?", "options": ["8:30", "9:00", "9:30"], "answer": 0, "type": "detail"},
 {"q": "Where is the meeting point?", "options": ["The museum", "The main gate of the park", "The bus station"], "answer": 1, "type": "detail"},
 {"q": "How will they recognise the guide?", "options": ["By a green flag", "By a red cap", "By a name badge"], "answer": 0, "type": "detail"},
 {"q": "Where does the tour finish?", "options": ["At the park", "At the hotel", "At a bazaar"], "answer": 2, "type": "detail"},
 {"q": "How much is a student ticket?", "options": ["$12", "$15", "$20"], "answer": 0, "type": "detail"},
 {"q": "How much will the caller pay in total?", "options": ["$24", "$32", "$40"], "answer": 0, "type": "detail"},
 {"q": "What is the most important thing to bring?", "options": ["A camera", "Comfortable shoes", "Lunch"], "answer": 1, "type": "detail"},
 {"q": "How can they pay?", "options": ["Only in cash", "Only by card", "By card or in cash"], "answer": 2, "type": "detail"}
]$json$);
