-- Честный mock на сайте (DECISIONS 2026-11-25): нарушения в Writing и Speaking mock — на сервере, как уходы
-- (leaves) у Reading и Listening. С MaxLeaves (3) часть не идёт в рейтинг; отчёт показывает их по частям.
ALTER TABLE mocks
    ADD COLUMN writing_violations  integer NOT NULL DEFAULT 0 CHECK (writing_violations >= 0),
    ADD COLUMN speaking_violations integer NOT NULL DEFAULT 0 CHECK (speaking_violations >= 0);
