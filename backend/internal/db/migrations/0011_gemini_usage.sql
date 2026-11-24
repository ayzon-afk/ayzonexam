-- Запросы к Gemini за сутки Gemini (полночь по тихоокеанскому времени) — для GEMINI_DAILY_LIMIT:
-- бесплатный ключ даёт 20 запросов в день на модель.
CREATE TABLE gemini_usage (
    day   date NOT NULL,
    model text NOT NULL,
    calls int  NOT NULL,
    PRIMARY KEY (day, model)
);
