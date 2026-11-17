// Package news — посты владельца во вкладке «Новости» (этап 14): пишутся боту командой /post или в админке.
package news

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("не найдено")
	ErrBad      = errors.New("пустой заголовок или текст")
)

const (
	MaxTitle = 200
	MaxBody  = 8000
)

var Tags = []string{"tip", "result", "tournament", "news"}

type Post struct {
	ID        int64     `json:"id"`
	Tag       string    `json:"tag"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Hidden    bool      `json:"hidden"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Valid проверяет и чистит пост.
func Valid(p *Post) error {
	p.Title, p.Body = strings.TrimSpace(p.Title), strings.TrimSpace(p.Body)
	if p.Title == "" || p.Body == "" || utf8.RuneCountInString(p.Title) > MaxTitle || utf8.RuneCountInString(p.Body) > MaxBody {
		return ErrBad
	}
	ok := false
	for _, t := range Tags {
		ok = ok || t == p.Tag
	}
	if !ok {
		p.Tag = "tip"
	}
	return nil
}

// FromMessage — пост из сообщения боту: первая строка — заголовок, остальное — текст; «#result» и
// подобные в заголовке задают метку.
func FromMessage(text string) (Post, error) {
	title, body, _ := strings.Cut(strings.TrimSpace(text), "\n")
	p := Post{Tag: "tip", Title: title, Body: body}
	for _, t := range Tags {
		if strings.Contains(p.Title, "#"+t) {
			p.Tag = t
			p.Title = strings.TrimSpace(strings.ReplaceAll(p.Title, "#"+t, ""))
		}
	}
	return p, Valid(&p)
}

func (s *Store) Create(ctx context.Context, p Post, authorTG int64) (Post, error) {
	if err := Valid(&p); err != nil {
		return p, err
	}
	err := s.pool.QueryRow(ctx, `INSERT INTO posts (tag, title, body, author_tg) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		p.Tag, p.Title, p.Body, authorTG).Scan(&p.ID, &p.CreatedAt)
	return p, err
}

func (s *Store) Update(ctx context.Context, p Post) (Post, error) {
	if err := Valid(&p); err != nil {
		return p, err
	}
	err := s.pool.QueryRow(ctx, `UPDATE posts SET tag = $2, title = $3, body = $4, hidden = $5, updated_at = now()
		WHERE id = $1 RETURNING created_at`, p.ID, p.Tag, p.Title, p.Body, p.Hidden).Scan(&p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Store) SetHidden(ctx context.Context, id int64, hidden bool) error {
	tag, err := s.pool.Exec(ctx, `UPDATE posts SET hidden = $2, updated_at = now() WHERE id = $1`, id, hidden)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// List — посты, новые первыми; скрытые — только для админки.
func (s *Store) List(ctx context.Context, limit int, withHidden bool) ([]Post, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, tag, title, body, hidden, created_at FROM posts
		WHERE $2 OR NOT hidden ORDER BY created_at DESC LIMIT $1`, limit, withHidden)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Post])
}

func (s *Store) Get(ctx context.Context, id int64, withHidden bool) (Post, error) {
	var p Post
	err := s.pool.QueryRow(ctx, `SELECT id, tag, title, body, hidden, created_at FROM posts WHERE id = $1 AND ($2 OR NOT hidden)`,
		id, withHidden).Scan(&p.ID, &p.Tag, &p.Title, &p.Body, &p.Hidden, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}
