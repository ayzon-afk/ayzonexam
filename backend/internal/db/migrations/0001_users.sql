-- Ученики. Имя, фамилия и @username из Telegram — личные: в ответах о других людях их нет,
-- публично только nickname (DECISIONS 2026-11-15).
CREATE TABLE users (
    id           bigserial PRIMARY KEY,
    telegram_id  bigint      NOT NULL UNIQUE,
    first_name   text        NOT NULL DEFAULT '',
    last_name    text        NOT NULL DEFAULT '',
    username     text        NOT NULL DEFAULT '',
    photo_url    text        NOT NULL DEFAULT '',
    nickname     text        NOT NULL UNIQUE,
    lang         text        NOT NULL DEFAULT 'uz' CHECK (lang IN ('uz', 'ru', 'en')),
    -- Цель: 4.0–9.0 с шагом 0.5, как баллы IELTS.
    goal_band    numeric(2,1) CHECK (goal_band BETWEEN 4.0 AND 9.0 AND goal_band * 2 = trunc(goal_band * 2)),
    exam_date    date,
    onboarded_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
