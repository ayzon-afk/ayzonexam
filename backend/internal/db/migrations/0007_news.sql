-- Новости (этап 14): владелец пишет пост боту (/post) или в админке, пост появляется во вкладке.
CREATE TABLE posts (
    id         bigserial PRIMARY KEY,
    tag        text        NOT NULL DEFAULT 'tip' CHECK (tag IN ('tip', 'result', 'tournament', 'news')),
    title      text        NOT NULL,
    body       text        NOT NULL,
    hidden     boolean     NOT NULL DEFAULT false,
    author_tg  bigint      NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX posts_feed ON posts (created_at DESC) WHERE NOT hidden;
