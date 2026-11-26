-- Экзамен как компьютерный IELTS (DECISIONS 2026-11-26): у теста части — Listening Part 1–4, Reading
-- Passage 1–3 со своим текстом; без частей (свои задания, тест уровня) — одна часть из passage, как раньше.
ALTER TABLE tests ADD COLUMN parts jsonb NOT NULL DEFAULT '[]';

-- Mock как на экзамене: Listening → Reading → Writing подряд, таймер следующей части идёт с конца предыдущей.
-- Начатые до выкладки mock — по-старому (strict = false), новые — подряд.
ALTER TABLE mocks ADD COLUMN strict boolean NOT NULL DEFAULT false;
ALTER TABLE mocks ALTER COLUMN strict SET DEFAULT true;
