-- Тесты канала «IELTS Volume» (DECISIONS 2026-11-26): кроме выбора варианта — вписать слово, «выберите N»,
-- буквы из общего списка. Виды и ключи — в questions (jsonb, как раньше); общие блоки вопросов (инструкция,
-- заметки, список вариантов, картинка) — groups. source — id теста канала: повторная загрузка заменяет тест.
ALTER TABLE tests
    ADD COLUMN groups jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN source text  NOT NULL DEFAULT '';
CREATE UNIQUE INDEX tests_source ON tests (source) WHERE source <> '';
