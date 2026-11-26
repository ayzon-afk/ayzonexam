-- Учитель без центра — как центр (SPEC, «Деньги»; DECISIONS 2026-11-25): свои ученики, ссылка, заявки,
-- кабинет и оплата; минимум 10 мест вместо 20; в рейтинг центров не входит — там только центры.
ALTER TABLE centers ADD COLUMN kind text NOT NULL DEFAULT 'center' CHECK (kind IN ('center', 'teacher'));
