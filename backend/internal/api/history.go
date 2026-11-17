package api

import (
	"net/http"
	"sort"
	"time"

	"ayzonexam/internal/users"
)

// historyItem — строка «Истории попыток»: задание, Writing, Speaking, mock.
type historyItem struct {
	Kind       string    `json:"kind"` // reading, listening, level, writing, speaking, mock
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Band       *float64  `json:"band"`
	Status     string    `json:"status"` // done, pending, failed, active
	Ranked     bool      `json:"ranked"`
	Tournament bool      `json:"tournament"`
	Mock       bool      `json:"mock"`
	At         time.Time `json:"at"`
}

func (s *server) getHistory(w http.ResponseWriter, r *http.Request, u users.User) {
	ctx := r.Context()
	rows, err := s.DB.Query(ctx, `
		SELECT a.kind, a.id, t.title, a.band::float8, 'done', a.ranked, a.tournament_id IS NOT NULL, a.mock_id IS NOT NULL, a.finished_at
		FROM attempts a JOIN tests t ON t.id = a.test_id WHERE a.user_id = $1 AND a.finished_at IS NOT NULL
		UNION ALL
		SELECT 'writing', id, task, band::float8, CASE WHEN status IN ('queued', 'processing') THEN 'pending' ELSE status END,
			false, false, mock_id IS NOT NULL, created_at
		FROM writing_checks WHERE user_id = $1
		UNION ALL
		SELECT 'speaking', id, source, band::float8, CASE WHEN status IN ('queued', 'processing') THEN 'pending' ELSE status END,
			false, false, mock_id IS NOT NULL, created_at
		FROM speaking_checks WHERE user_id = $1
		UNION ALL
		SELECT 'mock', id, '', overall::float8, status, false, false, true, coalesce(finished_at, started_at)
		FROM mocks WHERE user_id = $1
		ORDER BY 9 DESC LIMIT 100`, u.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer rows.Close()
	list := []historyItem{}
	for rows.Next() {
		var h historyItem
		if err := rows.Scan(&h.Kind, &h.ID, &h.Title, &h.Band, &h.Status, &h.Ranked, &h.Tournament, &h.Mock, &h.At); err != nil {
			s.fail(w, r, err)
			return
		}
		list = append(list, h)
	}
	if err := rows.Err(); err != nil {
		s.fail(w, r, err)
		return
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].At.After(list[j].At) })
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}
