-- Регионы Узбекистана вместо «Ташкент / другой город» (DECISIONS 2026-11-25): рейтинг своего региона.
-- Прежний other значил «не Ташкент» — такие ученики выбирают регион заново ('' — не указан);
-- теперь other — другая страна. Список — как rating.Regions и frontend/src/lib/cities.ts.

ALTER TABLE users DROP CONSTRAINT users_city_check;
UPDATE users SET city = '' WHERE city = 'other';
ALTER TABLE users ADD CONSTRAINT users_city_check CHECK (city IN ('', 'tashkent', 'tashkent_region', 'andijan',
    'bukhara', 'fergana', 'jizzakh', 'kashkadarya', 'khorezm', 'namangan', 'navoi', 'samarkand', 'surkhandarya',
    'syrdarya', 'karakalpakstan', 'other'));

-- Центр — только в регионе Узбекистана; '' — владелец ещё не указал.
UPDATE centers SET city = '' WHERE city = 'other';
ALTER TABLE centers ADD CONSTRAINT centers_city_check CHECK (city IN ('', 'tashkent', 'tashkent_region', 'andijan',
    'bukhara', 'fergana', 'jizzakh', 'kashkadarya', 'khorezm', 'namangan', 'navoi', 'samarkand', 'surkhandarya',
    'syrdarya', 'karakalpakstan'));
